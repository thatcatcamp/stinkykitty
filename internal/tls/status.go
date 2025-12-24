package tls

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
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

	// For each domain, check if certificate exists
	for _, domain := range domains {
		// Look for certificate file in cert storage
		// Certmagic stores certs in: {certDir}/certificates/{ca}/{domain}/{domain}.crt
		// Try both staging and production CA paths
		certPath := ""

		// Try production CA first
		prodPath := filepath.Join(m.cfg.CertDir, "certificates", "acme-v02.api.letsencrypt.org-directory", domain, domain+".crt")
		if _, err := os.Stat(prodPath); err == nil {
			certPath = prodPath
		} else {
			// Try staging CA
			stagingPath := filepath.Join(m.cfg.CertDir, "certificates", "acme-staging-v02.api.letsencrypt.org-directory", domain, domain+".crt")
			if _, err := os.Stat(stagingPath); err == nil {
				certPath = stagingPath
			}
		}

		if certPath == "" {
			// No certificate found for this domain (not yet provisioned)
			continue
		}

		// Read and parse certificate
		certPEM, err := os.ReadFile(certPath)
		if err != nil {
			continue // Skip if can't read
		}

		// Parse PEM block
		block, _ := pem.Decode(certPEM)
		if block == nil {
			continue // Not a valid PEM block
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			continue // Skip if can't parse
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
