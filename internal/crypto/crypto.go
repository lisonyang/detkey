
package crypto

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/hkdf"
)

// SALT is a fixed, public string. Changing it will cause all keys to change.
// Ideally, each user should use their own unique salt.
const SALT = "a-unique-salt-for-detkey-v1"

// DeriveAndGenerateKey is the core logic function, now supports multiple key types
func DeriveAndGenerateKey(password, salt []byte, context, keyType string) (crypto.PrivateKey, error) {
	// --- Core Step A: Key Stretching ---
	// Use Argon2id to perform "slow hash" on user input password, generating a high-strength 32-byte master seed.
	// This makes offline brute force attacks against the master password extremely expensive.
	// Argon2id parameters can be adjusted, higher values are more secure but slower to generate.
	masterSeed := argon2.IDKey(password, salt, 1, 64*1024, 4, 32)

	// --- Core Step B: Key Derivation ---
	// Use HKDF to derive the final seed for key generation from the master seed and context.
	// Using SHA256 as the hash function.
	hkdfReader := hkdf.New(sha256.New, masterSeed, salt, []byte(context))

	// --- Core Step C: Generate key based on type ---
	var privateKey crypto.PrivateKey
	var err error

	switch keyType {
	case "rsa2048":
		// Create unlimited entropy source for RSA key generation
		deterministicReader := newDeterministicReader(hkdfReader)
		privateKey, err = rsa.GenerateKey(deterministicReader, 2048)
	case "rsa4096":
		deterministicReader := newDeterministicReader(hkdfReader)
		privateKey, err = rsa.GenerateKey(deterministicReader, 4096)
	case "ed25519":
		finalSeed := make([]byte, ed25519.SeedSize) // Ed25519 requires 32 bytes of seed
		if _, err = io.ReadFull(hkdfReader, finalSeed); err != nil {
			return nil, fmt.Errorf("unable to read final seed from HKDF: %w", err)
		}
		privateKey = ed25519.NewKeyFromSeed(finalSeed)
	default:
		return nil, fmt.Errorf("unsupported key type: %s", keyType)
	}

	if err != nil {
		return nil, fmt.Errorf("unable to generate %s key: %w", keyType, err)
	}

	return privateKey, nil
}

// deterministicReader provides fast deterministic entropy source
type deterministicReader struct {
	seed    [32]byte
	counter uint64
	buffer  []byte
	bufPos  int
}

// newDeterministicReader creates a new efficient deterministic reader
func newDeterministicReader(hkdf io.Reader) *deterministicReader {
	dr := &deterministicReader{
		buffer: make([]byte, 8192), // 8KB buffer
		bufPos: 0,
	}
	
	// Read seed from HKDF
	_, err := io.ReadFull(hkdf, dr.seed[:])
	if err != nil {
		// If reading fails, use default seed (should not happen)
		copy(dr.seed[:], []byte("default-seed-for-rsa-generation"))
	}
	
	// Pre-fill buffer
	dr.refillBuffer()
	
	return dr
}

// refillBuffer refills the buffer using fast hash algorithm
func (dr *deterministicReader) refillBuffer() {
	hasher := sha256.New()
	for i := 0; i < len(dr.buffer)/32; i++ {
		hasher.Reset()
		hasher.Write(dr.seed[:])
		hasher.Write([]byte{byte(dr.counter), byte(dr.counter >> 8), byte(dr.counter >> 16), byte(dr.counter >> 24),
			byte(dr.counter >> 32), byte(dr.counter >> 40), byte(dr.counter >> 48), byte(dr.counter >> 56)})
		chunk := hasher.Sum(nil)
		copy(dr.buffer[i*32:(i+1)*32], chunk)
		dr.counter++
	}
	dr.bufPos = 0
}

// Read implements io.Reader interface, providing fast deterministic entropy
func (dr *deterministicReader) Read(p []byte) (n int, err error) {
	totalRead := 0
	
	for totalRead < len(p) {
		// If buffer is exhausted, refill it
		if dr.bufPos >= len(dr.buffer) {
			dr.refillBuffer()
		}
		
		// Copy data from buffer
		toCopy := len(p) - totalRead
		remaining := len(dr.buffer) - dr.bufPos
		if toCopy > remaining {
			toCopy = remaining
		}
		
		copy(p[totalRead:totalRead+toCopy], dr.buffer[dr.bufPos:dr.bufPos+toCopy])
		dr.bufPos += toCopy
		totalRead += toCopy
	}
	
	return totalRead, nil
}
