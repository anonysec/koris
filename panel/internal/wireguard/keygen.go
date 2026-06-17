package wireguard

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/curve25519"
)

// GenerateKeyPair generates a WireGuard private/public key pair.
// Returns base64-encoded private key and public key strings.
func GenerateKeyPair() (privateKey, publicKey string, err error) {
	var privBytes [32]byte
	if _, err := rand.Read(privBytes[:]); err != nil {
		return "", "", fmt.Errorf("generate random bytes: %w", err)
	}

	// Apply Curve25519 clamping
	privBytes[0] &= 248
	privBytes[31] &= 127
	privBytes[31] |= 64

	pubBytes, err := curve25519.X25519(privBytes[:], curve25519.Basepoint)
	if err != nil {
		return "", "", fmt.Errorf("derive public key: %w", err)
	}

	privateKey = base64.StdEncoding.EncodeToString(privBytes[:])
	publicKey = base64.StdEncoding.EncodeToString(pubBytes)
	return privateKey, publicKey, nil
}

// GeneratePresharedKey generates a random 32-byte preshared key for WireGuard.
// Returns the key as a base64-encoded string.
func GeneratePresharedKey() (string, error) {
	var key [32]byte
	if _, err := rand.Read(key[:]); err != nil {
		return "", fmt.Errorf("generate preshared key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(key[:]), nil
}
