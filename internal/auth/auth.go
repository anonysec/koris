package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	AdminCookieName    = "koris_admin_session"
	CustomerCookieName = "koris_customer_session"
)

type Service struct{ DB *sql.DB }

func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

func RandomToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s Service) AdminCount() (int, error) {
	var c int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM admins`).Scan(&c)
	return c, err
}

func (s Service) CreateOwner(username, password string) error {
	username = strings.TrimSpace(username)
	if username == "" || len(password) < 6 {
		return errors.New("invalid owner")
	}
	h, err := HashPassword(password)
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(`INSERT INTO admins(username,password_hash,role) VALUES($1,$2, 'owner')`, username, h)
	return err
}

func (s Service) LoginAdmin(username, password string) (bool, error) {
	username = strings.TrimSpace(username)
	var hash string
	var active bool
	err := s.DB.QueryRow(`SELECT password_hash,is_active FROM admins WHERE username=$1 LIMIT 1`, username).Scan(&hash, &active)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return active && CheckPassword(hash, password), nil
}

func MakeSession(username, secret string, ttl time.Duration) (string, error) {
	encodedUser := base64.RawURLEncoding.EncodeToString([]byte(username))
	expires := time.Now().Add(ttl).Unix()
	payload := fmt.Sprintf("%s.%d", encodedUser, expires)
	sig, err := sign(payload, secret)
	if err != nil {
		return "", err
	}
	return payload + "." + sig, nil
}

func ReadSession(r *http.Request, cookieName, secret string) (string, bool) {
	cookie, err := r.Cookie(cookieName)
	if err != nil || cookie.Value == "" {
		return "", false
	}
	return ValidateToken(cookie.Value, secret)
}

// ValidateToken validates a raw session token string (same format as cookie value)
// and returns the username if valid and not expired.
func ValidateToken(token, secret string) (string, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", false
	}
	payload := parts[0] + "." + parts[1]
	expectedSig, err := sign(payload, secret)
	if err != nil {
		return "", false
	}
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return "", false
	}
	expires, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || time.Now().Unix() > expires {
		return "", false
	}
	userBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", false
	}
	username := strings.TrimSpace(string(userBytes))
	return username, username != ""
}

func SetSession(w http.ResponseWriter, cookieName, username, secret string, secure bool) {
	session, err := MakeSession(username, secret, 24*time.Hour)
	if err != nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    session,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(24 * time.Hour),
	})
}

func ClearSession(w http.ResponseWriter, cookieName string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}

func sign(payload, secret string) (string, error) {
	if secret == "" {
		return "", errors.New("auth: session secret must not be empty")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil)), nil
}
