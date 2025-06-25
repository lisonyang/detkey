

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"crypto/x509/pkix"
	"github.com/lisonyang/detkey/internal/crypto"
	"github.com/lisonyang/detkey/internal/mtls"
	"github.com/lisonyang/detkey/internal/output"
	"golang.org/x/term"
)

func main() {
	// --- 1. Parse command line arguments ---
	context := flag.String("context", "", "Context string for key derivation (e.g. 'ssh/server-a/v1' or 'mtls/ca/v1') (required)")
	isPublicKey := flag.Bool("pub", false, "If specified, output public key only, otherwise output private key.")
	keyType := flag.String("type", "ed25519", "Key type to generate (ed25519, rsa2048, rsa4096)")
	outputFormat := flag.String("format", "auto", "Output format (auto, ssh, pem). auto will automatically choose based on context")
	salt := flag.String("salt", "", "Salt for key derivation. (default: a-unique-salt-for-detkey-v1)")
	action := flag.String("action", "", "Action to perform (e.g. 'create-ca-cert', 'sign-cert')")
	caContext := flag.String("ca-context", "", "Context of the CA for signing certificates")
	subj := flag.String("subj", "", "Subject for the certificate (e.g. '/CN=My CA')")
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

	// Determine salt
	finalSalt := crypto.SALT
	if *salt != "" {
		finalSalt = *salt
	} else if envSalt := os.Getenv("DETKEY_SALT"); envSalt != "" {
		finalSalt = envSalt
	}

	// --- 3. Derive key and generate ---
	privateKey, err := crypto.DeriveAndGenerateKey(password, []byte(finalSalt), *context, *keyType)
	if err != nil {
		log.Fatalf("Error: key generation failed: %v", err)
	}

	// --- 4. Determine output format ---
	format := determineOutputFormat(*outputFormat, *context, *keyType)

	// --- 5. Output result based on parameters ---
	if *isPublicKey {
		err = output.OutputPublicKey(privateKey, format)
	} else {
		err = output.OutputPrivateKey(privateKey, format)
	}
	
	if err != nil {
		log.Fatalf("Error: output failed: %v", err)
	}

	switch *action {
	case "create-ca-cert":
		if *subj == "" {
			log.Fatal("Error: --subj is required for creating a CA certificate.")
		}
		subject, err := parseSubject(*subj)
		if err != nil {
			log.Fatalf("Error parsing subject: %v", err)
		}
		cert, err := mtls.CreateCACertificate(privateKey, subject)
		if err != nil {
			log.Fatalf("Error creating CA certificate: %v", err)
		}
		fmt.Print(string(cert))
	case "sign-cert":
		if *caContext == "" {
			log.Fatal("Error: --ca-context is required for signing a certificate.")
		}
		if *subj == "" {
			log.Fatal("Error: --subj is required for signing a certificate.")
		}
		// ... sign cert logic
		caCertBytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("Error reading CA certificate from stdin: %v", err)
		}

		fmt.Fprint(os.Stderr, "Enter your CA master password: ")
		caPassword, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			log.Fatalf("Error reading CA password: %v", err)
		}
		fmt.Fprintln(os.Stderr)

		caKey, err := crypto.DeriveAndGenerateKey(caPassword, []byte(finalSalt), *caContext, *keyType)
		if err != nil {
			log.Fatalf("Error deriving CA key: %v", err)
		}

		subject, err := parseSubject(*subj)
		if err != nil {
			log.Fatalf("Error parsing subject: %v", err)
		}

		cert, err := mtls.SignCertificate(privateKey, caCertBytes, caKey, subject)
		if err != nil {
			log.Fatalf("Error signing certificate: %v", err)
		}
		fmt.Print(string(cert))
	}
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

func parseSubject(subj string) (pkix.Name, error) {
	name := pkix.Name{}
	parts := strings.Split(subj, "/")
	for _, part := range parts {
		if part == "" {
			continue
		}
		value := strings.Split(part, "=")
		if len(value) != 2 {
			return name, fmt.Errorf("invalid subject part: %s", part)
		}
		key := value[0]
		val := value[1]
		switch key {
		case "CN":
			name.CommonName = val
		case "O":
			name.Organization = []string{val}
		case "OU":
			name.OrganizationalUnit = []string{val}
		case "L":
			name.Locality = []string{val}
		case "ST":
			name.Province = []string{val}
		case "C":
			name.Country = []string{val}
		}
	}
	return name, nil
}

