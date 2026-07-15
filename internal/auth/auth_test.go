package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "modernc.org/sqlite"
	"golang.org/x/crypto/bcrypt"
)

func newTestAuth(t *testing.T) *Service {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Create admins table
	_, err = db.Exec(`
		CREATE TABLE admins (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'admin',
			is_active BOOLEAN NOT NULL DEFAULT 1
		)
	`)
	if err != nil {
		t.Fatalf("create admins table: %v", err)
	}

	return &Service{DB: db}
}

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("mypassword123")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword returned empty string")
	}
	if len(hash) < 20 {
		t.Fatalf("hash seems too short: %s", hash)
	}
	// Verify it's a bcrypt hash
	if hash[:4] != "$2a$" && hash[:4] != "$2b$" && hash[:4] != "$2y$" {
		t.Errorf("hash doesn't look like bcrypt: %s", hash[:10])
	}
}

func TestCheckPassword(t *testing.T) {
	hash, _ := HashPassword("mypassword123")
	if !CheckPassword(hash, "mypassword123") {
		t.Fatal("CheckPassword failed for correct password")
	}
	if CheckPassword(hash, "wrongpassword") {
		t.Fatal("CheckPassword succeeded for wrong password")
	}
	// Empty password
	if CheckPassword(hash, "") {
		t.Fatal("CheckPassword accepted empty password")
	}
	// Invalid hash
	if CheckPassword("not-a-hash", "password") {
		t.Fatal("CheckPassword accepted invalid hash")
	}
}

func TestRandomToken(t *testing.T) {
	t1 := RandomToken(16)
	t2 := RandomToken(16)
	if t1 == t2 {
		t.Fatal("RandomToken returned same value twice")
	}
	if len(t1) != 32 { // 16 bytes = 32 hex chars
		t.Errorf("RandomToken(16) length %d, want 32", len(t1))
	}
}

func TestAdminCount(t *testing.T) {
	s := newTestAuth(t)
	count, err := s.AdminCount()
	if err != nil {
		t.Fatalf("AdminCount failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 admins, got %d", count)
	}
}

func TestCreateOwner(t *testing.T) {
	s := newTestAuth(t)
	err := s.CreateOwner("admin", "password123")
	if err != nil {
		t.Fatalf("CreateOwner failed: %v", err)
	}
	// Duplicate user should fail
	err = s.CreateOwner("admin", "password123")
	if err == nil {
		t.Fatal("CreateOwner should fail for duplicate user")
	}
	// Short password
	s2 := newTestAuth(t)
	err = s2.CreateOwner("admin2", "short")
	if err == nil || err.Error() != "invalid owner" {
		t.Errorf("CreateOwner with short password should error: got %v", err)
	}
}

func TestLoginAdmin(t *testing.T) {
	s := newTestAuth(t)
	// First create an admin
	_ = s.CreateOwner("admin", "password123")

	// Correct password
	ok, err := s.LoginAdmin("admin", "password123")
	if err != nil {
		t.Fatalf("LoginAdmin failed: %v", err)
	}
	if !ok {
		t.Fatal("LoginAdmin rejected correct password")
	}

	// Wrong password
	ok, err = s.LoginAdmin("admin", "wrong")
	if err != nil {
		t.Fatalf("LoginAdmin error: %v", err)
	}
	if ok {
		t.Fatal("LoginAdmin accepted wrong password")
	}

	// Non-existent user
	ok, err = s.LoginAdmin("nonexistent", "password")
	if err != nil {
		t.Fatalf("LoginAdmin error: %v", err)
	}
	if ok {
		t.Fatal("LoginAdmin accepted non-existent user")
	}
}

func TestMakeSession_ReadSession(t *testing.T) {
	secret := "test-secret-key"
	username := "adminuser"
	ttl := time.Hour

	session, err := MakeSession(username, secret, ttl)
	if err != nil {
		t.Fatalf("MakeSession failed: %v", err)
	}

	// Create a request with the session cookie
	req := httptest.NewRequest("GET", "/", nil)
	cookie := &http.Cookie{Name: AdminCookieName, Value: session}
	req.AddCookie(cookie)

	readUser, ok := ReadSession(req, AdminCookieName, secret)
	if !ok {
		t.Fatal("ReadSession failed to read valid session")
	}
	if readUser != username {
		t.Errorf("ReadSession returned %q, want %q", readUser, username)
	}
}

func TestReadSession_MissingCookie(t *testing.T) {
	secret := "test-secret"
	req := httptest.NewRequest("GET", "/", nil)
	_, ok := ReadSession(req, AdminCookieName, secret)
	if ok {
		t.Fatal("ReadSession should fail when cookie is missing")
	}
}

func TestReadSession_EmptyCookie(t *testing.T) {
	secret := "test-secret"
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: AdminCookieName, Value: ""})
	_, ok := ReadSession(req, AdminCookieName, secret)
	if ok {
		t.Fatal("ReadSession should fail when cookie value is empty")
	}
}

func TestValidateToken(t *testing.T) {
	secret := "test-secret"
	username := "testuser"
	ttl := time.Hour

	token, err := MakeSession(username, secret, ttl)
	if err != nil {
		t.Fatalf("MakeSession failed: %v", err)
	}

	// Valid token
	user, ok := ValidateToken(token, secret)
	if !ok {
		t.Fatal("ValidateToken rejected valid token")
	}
	if user != username {
		t.Errorf("ValidateToken returned %q, want %q", user, username)
	}

	// Wrong secret
	_, ok = ValidateToken(token, "wrong-secret")
	if ok {
		t.Fatal("ValidateToken accepted token with wrong secret")
	}

	// Tampered token
	tampered := token[:len(token)-1] + "x"
	_, ok = ValidateToken(tampered, secret)
	if ok {
		t.Fatal("ValidateToken accepted tampered token")
	}

	// Expired token
	expiredToken, err := MakeSession(username, secret, -time.Hour)
	if err != nil {
		t.Fatalf("MakeSession with negative TTL failed: %v", err)
	}
	_, ok = ValidateToken(expiredToken, secret)
	if ok {
		t.Fatal("ValidateToken accepted expired token")
	}

	// Malformed token
	_, ok = ValidateToken("not.a.valid.token", secret)
	if ok {
		t.Fatal("ValidateToken accepted malformed token")
	}
}

func TestSetSession_ClearSession(t *testing.T) {
	w := httptest.NewRecorder()
	secret := "test-secret"

	SetSession(w, AdminCookieName, "user1", secret, true)
	cookie := w.Result().Cookies()[0]
	if cookie == nil {
		t.Fatal("SetSession did not set cookie")
	}
	if cookie.Name != AdminCookieName {
		t.Errorf("cookie name %q, want %q", cookie.Name, AdminCookieName)
	}
	if cookie.Value == "" {
		t.Fatal("cookie value is empty")
	}
	if !cookie.HttpOnly {
		t.Fatal("cookie should be HttpOnly")
	}
	if !cookie.Secure {
		t.Fatal("cookie should be Secure (secure=true)")
	}
	if cookie.Path != "/" {
		t.Errorf("cookie path %q, want /", cookie.Path)
	}

	// Clear session
	w2 := httptest.NewRecorder()
	ClearSession(w2, AdminCookieName, true)
	cookie2 := w2.Result().Cookies()[0]
	if cookie2.Value != "" {
		t.Fatal("ClearSession did not clear cookie value")
	}
	if cookie2.MaxAge != -1 {
		t.Errorf("ClearSession MaxAge %d, want -1", cookie2.MaxAge)
	}
}

func TestSessionSecretValidation(t *testing.T) {
	// Empty secret should error
	_, err := MakeSession("user", "", time.Hour)
	if err == nil {
		t.Fatal("MakeSession with empty secret should error")
	}

	// ValidateToken with empty secret should return false
	_, ok := ValidateToken("payload.sig", "")
	if ok {
		t.Fatal("ValidateToken with empty secret should return false")
	}
}

func TestSign(t *testing.T) {
	secret := "test-secret"
	payload := "test.payload"

	sig, err := sign(payload, secret)
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}
	if sig == "" {
		t.Fatal("sign returned empty signature")
	}

	// Verify with hmac.Equal
	expected := hmac.New(sha256.New, []byte(secret))
	expected.Write([]byte(payload))
	expectedHex := hex.EncodeToString(expected.Sum(nil))
	if sig != expectedHex {
		t.Errorf("sign output doesn't match manual HMAC: got %s, want %s", sig, expectedHex)
	}
}

func TestCookieNames(t *testing.T) {
	if AdminCookieName == "" {
		t.Fatal("AdminCookieName is empty")
	}
	if CustomerCookieName == "" {
		t.Fatal("CustomerCookieName is empty")
	}
	if AdminCookieName == CustomerCookieName {
		t.Fatal("Admin and Customer cookie names should differ")
	}
}

func TestSessionTTL(t *testing.T) {
	secret := "test-secret"
	username := "ttluser"

	// Short TTL - should expire immediately
	token, err := MakeSession(username, secret, -time.Second)
	if err != nil {
		t.Fatalf("MakeSession failed: %v", err)
	}
	_, ok := ValidateToken(token, secret)
	if ok {
		t.Fatal("ValidateToken accepted token with negative TTL")
	}

	// Valid TTL
	token2, err := MakeSession(username, secret, time.Hour)
	if err != nil {
		t.Fatalf("MakeSession failed: %v", err)
	}
	user, ok := ValidateToken(token2, secret)
	if !ok {
		t.Fatal("ValidateToken rejected valid TTL token")
	}
	if user != username {
		t.Errorf("username mismatch: %s vs %s", user, username)
	}
}

// bcrypt default cost sanity check
func TestBcryptCost(t *testing.T) {
	hash, _ := HashPassword("password")
	// bcrypt cost 10 is default (100000 iterations)
	// We just verify it hashes correctly
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("password")); err != nil {
		t.Fatalf("bcrypt default cost hash failed: %v", err)
	}
}