package wireguard

import (
	"encoding/base64"
	"testing"
)

func TestGenerateKeyPair(t *testing.T) {
	priv, pub, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}

	// Base64-encoded 32 bytes = 44 characters
	if len(priv) != 44 {
		t.Errorf("private key length = %d, want 44", len(priv))
	}
	if len(pub) != 44 {
		t.Errorf("public key length = %d, want 44", len(pub))
	}

	// Verify valid base64 decoding to 32 bytes
	privBytes, err := base64.StdEncoding.DecodeString(priv)
	if err != nil {
		t.Fatalf("private key base64 decode error: %v", err)
	}
	if len(privBytes) != 32 {
		t.Errorf("decoded private key length = %d, want 32", len(privBytes))
	}

	pubBytes, err := base64.StdEncoding.DecodeString(pub)
	if err != nil {
		t.Fatalf("public key base64 decode error: %v", err)
	}
	if len(pubBytes) != 32 {
		t.Errorf("decoded public key length = %d, want 32", len(pubBytes))
	}

	// Verify different keys each call
	priv2, pub2, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() second call error: %v", err)
	}
	if priv == priv2 {
		t.Error("two calls produced the same private key")
	}
	if pub == pub2 {
		t.Error("two calls produced the same public key")
	}
}

func TestGenerateKeyPair_PrivateKeyIsClamped(t *testing.T) {
	priv, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}

	privBytes, _ := base64.StdEncoding.DecodeString(priv)
	// Check clamping bits
	if privBytes[0]&7 != 0 {
		t.Error("private key byte 0 not properly clamped (low 3 bits should be 0)")
	}
	if privBytes[31]&128 != 0 {
		t.Error("private key byte 31 high bit should be 0")
	}
	if privBytes[31]&64 == 0 {
		t.Error("private key byte 31 bit 6 should be 1")
	}
}

func TestGeneratePresharedKey(t *testing.T) {
	psk, err := GeneratePresharedKey()
	if err != nil {
		t.Fatalf("GeneratePresharedKey() error: %v", err)
	}

	// Base64-encoded 32 bytes = 44 characters
	if len(psk) != 44 {
		t.Errorf("preshared key length = %d, want 44", len(psk))
	}

	// Verify valid base64 decoding to 32 bytes
	pskBytes, err := base64.StdEncoding.DecodeString(psk)
	if err != nil {
		t.Fatalf("preshared key base64 decode error: %v", err)
	}
	if len(pskBytes) != 32 {
		t.Errorf("decoded preshared key length = %d, want 32", len(pskBytes))
	}

	// Verify different keys each call
	psk2, err := GeneratePresharedKey()
	if err != nil {
		t.Fatalf("GeneratePresharedKey() second call error: %v", err)
	}
	if psk == psk2 {
		t.Error("two calls produced the same preshared key")
	}
}
