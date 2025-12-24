package tls

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// CertificateStatus represents the status of a managed certificate
type CertificateStatus struct {
	Domain          string
	Issuer          string
	NotBefore       time.Time
	NotAfter        time.Time
	DaysUntilExpiry int
}

// GetCertificateStatus returns the status of all managed certificates
func (m *Manager) GetCertificateStatus() ([]CertificateStatus, error) {
	var statuses []CertificateStatus

	// Get all allowed domains
	domains, err := m.GetAllowedDomains()
	if err != nil {
		return nil, fmt.Errorf("failed to get domains: %w", err)
	}

	// Walk the certificates directory to find actual certificate files
	certsBaseDir := filepath.Join(m.cfg.CertDir, "certificates")

	for _, domain := range domains {
		// Search for certificate file by walking the certificates directory
		var certPath string

		// Walk the certificates directory looking for this domain's cert
		filepath.Walk(certsBaseDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}

			// Look for files named {domain}.crt
			if filepath.Base(path) == domain+".crt" {
				// Verify parent directory matches domain
				parentDir := filepath.Base(filepath.Dir(path))
				if parentDir == domain {
					certPath = path
					return filepath.SkipAll // Found it, stop walking
				}
			}
			return nil
		})

		if certPath == "" {
			// No certificate found for this domain
			continue
		}

		// Read and parse certificate
		certPEM, err := os.ReadFile(certPath)
		if err != nil {
			log.Printf("Warning: Failed to read certificate for %s: %v", domain, err)
			continue
		}

		// Parse first PEM block (leaf certificate - Let's Encrypt places this first)
		block, _ := pem.Decode(certPEM)
		if block == nil {
			log.Printf("Warning: No PEM block found in certificate for %s", domain)
			continue
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			log.Printf("Warning: Failed to parse certificate for %s: %v", domain, err)
			continue
		}

		// Calculate days until expiry
		daysUntilExpiry := int(time.Until(cert.NotAfter).Hours() / 24)

		status := CertificateStatus{
			Domain:          domain,
			Issuer:          cert.Issuer.CommonName,
			NotBefore:       cert.NotBefore,
			NotAfter:        cert.NotAfter,
			DaysUntilExpiry: daysUntilExpiry,
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}
