
package crypto

import (
	"crypto/ed25519"
	"testing"
)

func TestDeriveAndGenerateKey_Deterministic(t *testing.T) {
	password := []byte("test-password")
	salt := []byte("test-salt")
	context := "test/context"

	// Generate first key
	key1, err := DeriveAndGenerateKey(password, salt, context, "ed25519")
	if err != nil {
		t.Fatalf("Failed to generate first key: %v", err)
	}

	// Generate second key with same parameters
	key2, err := DeriveAndGenerateKey(password, salt, context, "ed25519")
	if err != nil {
		t.Fatalf("Failed to generate second key: %v", err)
	}

	// Check if keys are equal
	ed25519Key1, ok1 := key1.(ed25519.PrivateKey)
	ed25519Key2, ok2 := key2.(ed25519.PrivateKey)

	if !ok1 || !ok2 {
		t.Fatal("Failed to cast keys to ed25519.PrivateKey")
	}

	if !ed25519Key1.Equal(ed25519Key2) {
		t.Error("Generated keys are not deterministic")
	}
}
