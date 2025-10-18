package browser

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"path"

	"github.com/glitchedgitz/grroxy-db/rawproxy"
)

// GenerateCert generates or loads the CA certificate using rawproxy's cert system
// This ensures we use a single unified certificate system (rawproxy)
func GenerateCert(configPath string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Use the same directory as rawproxy for certificates
	certDir := path.Join(homeDir, ".config", "grroxy")
	if err := os.MkdirAll(certDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create certificate directory: %w", err)
	}

	// Ensure the config directory exists
	if err := os.MkdirAll(configPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	// Check if rawproxy certificates already exist
	caCrtPath := path.Join(certDir, "ca.crt")
	caKeyPath := path.Join(certDir, "ca.key")

	// If certificates don't exist, generate them using rawproxy
	if !fileExists(caCrtPath) || !fileExists(caKeyPath) {
		_, certPath, _, err := rawproxy.GenerateMITMCA(certDir)
		if err != nil {
			return "", fmt.Errorf("failed to generate MITM CA: %w", err)
		}
		return certPath, nil
	}

	// Return existing certificate path
	return caCrtPath, nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func GetSPKIFingerprint(certPath string) (string, error) {
	// Read the certificate file
	certData, err := os.ReadFile(certPath)
	if err != nil {
		return "", fmt.Errorf("failed to read certificate file: %v", err)
	}

	// Decode the PEM block
	block, _ := pem.Decode(certData)
	if block == nil || block.Type != "CERTIFICATE" {
		return "", fmt.Errorf("failed to decode PEM block containing certificate")
	}

	// Parse the certificate
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse certificate: %v", err)
	}

	// Calculate SHA-256 hash of the SubjectPublicKeyInfo
	spkiHash := sha256.Sum256(cert.RawSubjectPublicKeyInfo)

	// Convert hash to base64 encoded string (the format Chrome expects)
	fingerprint := base64.StdEncoding.EncodeToString(spkiHash[:])

	return fingerprint, nil
}
