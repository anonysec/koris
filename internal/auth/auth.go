package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	DB     *sql.DB
	Secret string
}

// HashPassword hashes a password using bcrypt.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(hash), err
}

// CheckPassword verifies a password against a bcrypt hash.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// GenerateToken creates a random hex token.
func GenerateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// HashToken creates a SHA-256 hash of a token for storage.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// Session represents an admin session.
type Session struct {
	Token     string
	Username  string
	Role      string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// CreateSession creates a new admin session.
func (s *Service) CreateSession(username, role string) (*Session, error) {
	token := GenerateToken()
	tokenHash := HashToken(token)
	expires := time.Now().Add(24 * time.Hour)

	_, err := s.DB.Exec(`INSERT INTO admin_sessions (token_hash, username, role, expires_at) VALUES (?, ?, ?, ?)`,
		tokenHash, username, role, expires)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &Session{
		Token:     token,
		Username:  username,
		Role:      role,
		ExpiresAt: expires,
	}, nil
}

// ValidateSession checks if a session token is valid.
func (s *Service) ValidateSession(token string) (*Session, error) {
	tokenHash := HashToken(token)
	var sess Session
	err := s.DB.QueryRow(`SELECT username, role, expires_at FROM admin_sessions WHERE token_hash=? AND expires_at > NOW()`,
		tokenHash).Scan(&sess.Username, &sess.Role, &sess.ExpiresAt)
	if err != nil {
		return nil, err
	}
	sess.Token = token
	return &sess, nil
}

// DeleteSession removes a session.
func (s *Service) DeleteSession(token string) error {
	tokenHash := HashToken(token)
	_, err := s.DB.Exec(`DELETE FROM admin_sessions WHERE token_hash=?`, tokenHash)
	return err
}

// GetSessionFromRequest extracts the session token from cookie or header.
func GetSessionFromRequest(r *http.Request) string {
	// Try cookie first
	if c, err := r.Cookie("session"); err == nil {
		return c.Value
	}
	// Try Authorization header
	if h := r.Header.Get("Authorization"); len(h) > 7 && h[:7] == "Bearer " {
		return h[7:]
	}
	return ""
}
