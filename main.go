package main

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// SALT is a fixed, public string. Changing it will cause all keys to change.
// Ideally, each user should use their own unique salt.
const SALT = "a-unique-salt-for-detkey-v1"

func main() {
	// --- 1. Parse command line arguments ---
	context := flag.String("context", "", "Context string for key derivation (e.g. 'ssh/server-a/v1' or 'mtls/ca/v1') (required)")
	isPublicKey := flag.Bool("pub", false, "If specified, output public key only, otherwise output private key.")
	keyType := flag.String("type", "ed25519", "Key type to generate (ed25519, rsa2048, rsa4096)")
	outputFormat := flag.String("format", "auto", "Output format (auto, ssh, pem). auto will automatically choose based on context")
	flag.Parse()

	if *context == "" {
		flag.Usage()
		log.Fatal("Error: --context parameter is required.")
	}

	// Validate key type
	if !isValidKeyType(*keyType) {
		log.Fatalf("Error: unsupported key type '%s'. Supported types: ed25519, rsa2048, rsa4096", *keyType)
	}

	// --- 2. Securely read master password ---
	var password []byte
	var err error
	
	// Check if it's a terminal environment
	if term.IsTerminal(int(os.Stdin.Fd())) {
		// Securely read password in terminal environment (no echo)
		fmt.Fprint(os.Stderr, "Enter your master password: ")
		password, err = term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr) // Newline after reading
		if err != nil {
			log.Fatalf("Error: unable to read password: %v", err)
		}
	} else {
		// Read from standard input in non-terminal environment (like pipes, scripts)
		var line string
		line, err = readLine(os.Stdin)
		if err != nil {
			log.Fatalf("Error: unable to read password: %v", err)
		}
		password = []byte(line)
	}
	
	if len(password) == 0 {
		log.Fatal("Error: password cannot be empty.")
	}

	// --- 3. Derive key and generate ---
	privateKey, err := deriveAndGenerateKey(password, []byte(SALT), *context, *keyType)
	if err != nil {
		log.Fatalf("Error: key generation failed: %v", err)
	}

	// --- 4. Determine output format ---
	format := determineOutputFormat(*outputFormat, *context, *keyType)

	// --- 5. Output result based on parameters ---
	if *isPublicKey {
		err = outputPublicKey(privateKey, format)
	} else {
		err = outputPrivateKey(privateKey, format)
	}
	
	if err != nil {
		log.Fatalf("Error: output failed: %v", err)
	}
}

// deriveAndGenerateKey is the core logic function, now supports multiple key types
func deriveAndGenerateKey(password, salt []byte, context, keyType string) (crypto.PrivateKey, error) {
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

// isValidKeyType checks if the key type is valid
func isValidKeyType(keyType string) bool {
	validTypes := []string{"ed25519", "rsa2048", "rsa4096"}
	for _, t := range validTypes {
		if t == keyType {
			return true
		}
	}
	return false
}

// determineOutputFormat determines output format based on context and parameters
func determineOutputFormat(format, context, keyType string) string {
	if format != "auto" {
		return format
	}
	
	// If context contains "mtls", default to PEM format
	if containsString(context, "mtls") {
		return "pem"
	}
	
	// If context contains "ssh", default to SSH format
	if containsString(context, "ssh") {
		return "ssh"
	}
	
	// For RSA keys, default to PEM format when context is unclear
	if keyType == "rsa2048" || keyType == "rsa4096" {
		return "pem"
	}
	
	// Default to SSH format
	return "ssh"
}

// containsString checks if string contains substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    (len(s) > len(substr) && (s[:len(substr)+1] == substr+"/" || 
		     s[len(s)-len(substr)-1:] == "/"+substr || 
		     containsSubstring(s, "/"+substr+"/"))))
}

func containsSubstring(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// outputPublicKey outputs public key
func outputPublicKey(privateKey crypto.PrivateKey, format string) error {
	switch format {
	case "ssh":
		return outputSSHPublicKey(privateKey)
	case "pem":
		return outputPEMPublicKey(privateKey)
	default:
		return fmt.Errorf("unsupported public key format: %s", format)
	}
}

// outputPrivateKey outputs private key
func outputPrivateKey(privateKey crypto.PrivateKey, format string) error {
	switch format {
	case "ssh":
		return outputSSHPrivateKey(privateKey)
	case "pem":
		return outputPEMPrivateKey(privateKey)
	default:
		return fmt.Errorf("unsupported private key format: %s", format)
	}
}

// outputSSHPublicKey outputs SSH format public key
func outputSSHPublicKey(privateKey crypto.PrivateKey) error {
	publicKey := privateKey.(interface{ Public() crypto.PublicKey }).Public()
	sshPubKey, err := ssh.NewPublicKey(publicKey)
	if err != nil {
		return fmt.Errorf("unable to create SSH public key: %w", err)
	}
	fmt.Print(string(ssh.MarshalAuthorizedKey(sshPubKey)))
	return nil
}

// outputPEMPublicKey outputs PEM format public key
func outputPEMPublicKey(privateKey crypto.PrivateKey) error {
	publicKey := privateKey.(interface{ Public() crypto.PublicKey }).Public()
	
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return fmt.Errorf("unable to serialize public key: %w", err)
	}
	
	pemBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	}
	
	return pem.Encode(os.Stdout, pemBlock)
}

// outputSSHPrivateKey outputs SSH format private key
func outputSSHPrivateKey(privateKey crypto.PrivateKey) error {
	pemBlock, err := ssh.MarshalPrivateKey(privateKey, "")
	if err != nil {
		return fmt.Errorf("unable to serialize private key: %w", err)
	}
	return pem.Encode(os.Stdout, pemBlock)
}

// outputPEMPrivateKey outputs PEM format private key
func outputPEMPrivateKey(privateKey crypto.PrivateKey) error {
	var pemBlock *pem.Block
	
	switch key := privateKey.(type) {
	case *rsa.PrivateKey:
		keyBytes := x509.MarshalPKCS1PrivateKey(key)
		pemBlock = &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: keyBytes,
		}
	case ed25519.PrivateKey:
		keyBytes, err := x509.MarshalPKCS8PrivateKey(key)
		if err != nil {
			return fmt.Errorf("unable to serialize Ed25519 private key: %w", err)
		}
		pemBlock = &pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: keyBytes,
		}
	default:
		return fmt.Errorf("unsupported private key type")
	}
	
	return pem.Encode(os.Stdout, pemBlock)
}

// readLine reads a line of text from the given io.Reader
// Used for reading passwords in non-terminal environments
func readLine(reader io.Reader) (string, error) {
	var line []byte
	buffer := make([]byte, 1)
	
	for {
		n, err := reader.Read(buffer)
		if err != nil {
			if err == io.EOF && len(line) > 0 {
				break
			}
			return "", err
		}
		if n > 0 {
			if buffer[0] == '\n' {
				break
			}
			if buffer[0] != '\r' { // Ignore carriage return
				line = append(line, buffer[0])
			}
		}
	}
	
	return string(line), nil
} 