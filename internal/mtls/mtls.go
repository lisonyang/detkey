
package mtls

import (
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

// CreateCACertificate creates a self-signed CA certificate.
func CreateCACertificate(priv crypto.PrivateKey, subj pkix.Name) ([]byte, error) {
	template := x509.Certificate{
		SerialNumber: big.NewInt(2024),
		Subject:      subj,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		IsCA:         true,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	}

	pub := priv.(interface{ Public() crypto.PublicKey }).Public()

	caBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, pub, priv)
	if err != nil {
		return nil, fmt.Errorf("failed to create CA certificate: %w", err)
	}

	caPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})

	return caPEM, nil
}

// SignCertificate signs a certificate with a CA.
func SignCertificate(priv crypto.PrivateKey, caCertPEM []byte, caPrivKey crypto.PrivateKey, subj pkix.Name) ([]byte, error) {
	caCert, err := parseCertificate(caCertPEM)
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(2024),
		Subject:      subj,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	pub := priv.(interface{ Public() crypto.PublicKey }).Public()

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, caCert, pub, caPrivKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	return certPEM, nil
}

func parseCertificate(certPEM []byte) (*x509.Certificate, error) {
	pemBlock, _ := pem.Decode(certPEM)
	if pemBlock == nil {
		return nil, fmt.Errorf("failed to decode PEM block containing certificate")
	}

	cert, err := x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return cert, nil
}
