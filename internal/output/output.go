
package output

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	"golang.org/x/crypto/ssh"
)

// OutputPublicKey outputs public key
func OutputPublicKey(privateKey crypto.PrivateKey, format string) error {
	switch format {
	case "ssh":
		return outputSSHPublicKey(privateKey)
	case "pem":
		return outputPEMPublicKey(privateKey)
	default:
		return fmt.Errorf("unsupported public key format: %s", format)
	}
}

// OutputPrivateKey outputs private key
func OutputPrivateKey(privateKey crypto.PrivateKey, format string) error {
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
