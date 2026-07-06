package updater

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"testing/quick"
)

// Property 1: SHA-256 Checksum Verification
// For any random byte sequence, computing SHA-256 and calling VerifyChecksum
// with the correct hash should return true.
// **Validates: Requirements 1.2**
func TestProperty_VerifyChecksum_CorrectHash_ReturnsTrue(t *testing.T) {
	f := func(data []byte) bool {
		h := sha256.Sum256(data)
		expected := hex.EncodeToString(h[:])
		return VerifyChecksum(data, expected)
	}

	cfg := &quick.Config{MaxCount: 200}
	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("property violated: correct hash should always return true: %v", err)
	}
}

// Property 1 (negative): For any random byte sequence, calling VerifyChecksum
// with ANY different hash should return false.
// **Validates: Requirements 1.2**
func TestProperty_VerifyChecksum_WrongHash_ReturnsFalse(t *testing.T) {
	f := func(data []byte) bool {
		// Compute the correct hash then flip a byte to make it wrong
		h := sha256.Sum256(data)
		correct := hex.EncodeToString(h[:])

		// Create a wrong hash by modifying the first character
		wrongHash := make([]byte, len(correct))
		copy(wrongHash, correct)
		if wrongHash[0] == 'a' {
			wrongHash[0] = 'b'
		} else {
			wrongHash[0] = 'a'
		}

		return !VerifyChecksum(data, string(wrongHash))
	}

	cfg := &quick.Config{MaxCount: 200}
	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("property violated: wrong hash should always return false: %v", err)
	}
}

// Property 1 (random wrong hash): For any random byte sequence, calling VerifyChecksum
// with a completely random 64-char hex string (different from actual hash) should return false.
// **Validates: Requirements 1.2**
func TestProperty_VerifyChecksum_RandomHash_ReturnsFalse(t *testing.T) {
	for i := 0; i < 200; i++ {
		// Generate random data
		dataLen := 1 + i%256
		data := make([]byte, dataLen)
		if _, err := rand.Read(data); err != nil {
			t.Fatalf("failed to generate random data: %v", err)
		}

		// Generate a random 32-byte hash (as hex)
		randomBytes := make([]byte, 32)
		if _, err := rand.Read(randomBytes); err != nil {
			t.Fatalf("failed to generate random hash bytes: %v", err)
		}
		randomHash := hex.EncodeToString(randomBytes)

		// Compute actual hash
		h := sha256.Sum256(data)
		actualHash := hex.EncodeToString(h[:])

		// If by extreme chance randomHash matches, skip this iteration
		if randomHash == actualHash {
			continue
		}

		if VerifyChecksum(data, randomHash) {
			t.Errorf("iteration %d: VerifyChecksum returned true for random hash that doesn't match actual", i)
		}
	}
}

// Property 3: VerifyChecksum is deterministic — calling it twice with the same
// data and expected hash returns the same result.
// **Validates: Requirements 1.1**
func TestProperty_VerifyChecksum_Deterministic(t *testing.T) {
	f := func(data []byte) bool {
		h := sha256.Sum256(data)
		expected := hex.EncodeToString(h[:])

		result1 := VerifyChecksum(data, expected)
		result2 := VerifyChecksum(data, expected)
		return result1 == result2
	}

	cfg := &quick.Config{MaxCount: 200}
	if err := quick.Check(f, cfg); err != nil {
		t.Errorf("property violated: VerifyChecksum should be deterministic: %v", err)
	}
}

// Property 3 (empty data): Empty data has a valid checksum (the SHA-256 of empty bytes).
// **Validates: Requirements 2.2**
func TestProperty_VerifyChecksum_EmptyData(t *testing.T) {
	emptyData := []byte{}
	h := sha256.Sum256(emptyData)
	expected := hex.EncodeToString(h[:])

	// Known SHA-256 of empty input: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
	knownEmpty := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	if expected != knownEmpty {
		t.Errorf("SHA-256 of empty bytes should be %s, got %s", knownEmpty, expected)
	}

	if !VerifyChecksum(emptyData, expected) {
		t.Error("VerifyChecksum should return true for empty data with correct hash")
	}

	if !VerifyChecksum(emptyData, knownEmpty) {
		t.Error("VerifyChecksum should return true for empty data with known empty hash")
	}
}
