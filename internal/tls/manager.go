package tls

import (
	"crypto/tls"
	"fmt"
	"log"

	"github.com/caddyserver/certmagic"
	"gorm.io/gorm"
	"github.com/thatcatcamp/stinkykitty/internal/models"
)

// Manager handles certificate provisioning and management
type Manager struct {
	cfg         *Config
	db          *gorm.DB
	certmagic   *certmagic.Config
}

// NewManager creates a new TLS manager
func NewManager(db *gorm.DB, cfg *Config) (*Manager, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if db == nil {
		return nil, fmt.Errorf("database is required")
	}

	// Create certmagic config
	cache := certmagic.NewCache(certmagic.CacheOptions{
		GetConfigForCert: func(certmagic.Certificate) (*certmagic.Config, error) {
			return &certmagic.Default, nil
		},
	})

	magicCfg := certmagic.New(cache, certmagic.Config{
		Storage: &certmagic.FileStorage{Path: cfg.CertDir},
	})

	// Configure ACME issuer
	if cfg.Staging {
		magicCfg.Issuers = []certmagic.Issuer{
			certmagic.NewACMEIssuer(magicCfg, certmagic.ACMEIssuer{
				CA:     certmagic.LetsEncryptStagingCA,
				Email:  cfg.Email,
				Agreed: true,
			}),
		}
	} else {
		magicCfg.Issuers = []certmagic.Issuer{
			certmagic.NewACMEIssuer(magicCfg, certmagic.ACMEIssuer{
				CA:     certmagic.LetsEncryptProductionCA,
				Email:  cfg.Email,
				Agreed: true,
			}),
		}
	}

	m := &Manager{
		cfg:       cfg,
		db:        db,
		certmagic: magicCfg,
	}

	// Load and manage allowed domains
	if err := m.RefreshDomains(); err != nil {
		return nil, fmt.Errorf("failed to load domains: %w", err)
	}

	return m, nil
}

// GetAllowedDomains queries database for all domains that should have certificates
func (m *Manager) GetAllowedDomains() ([]string, error) {
	domains := []string{
		m.cfg.BaseDomain,
	}

	// Get all sites for subdomains
	var sites []models.Site
	if err := m.db.Find(&sites).Error; err != nil {
		return nil, fmt.Errorf("failed to query sites: %w", err)
	}

	for _, site := range sites {
		// Add subdomain
		subdomain := fmt.Sprintf("%s.%s", site.Subdomain, m.cfg.BaseDomain)
		domains = append(domains, subdomain)

		// Add custom domain if set
		if site.CustomDomain != nil && *site.CustomDomain != "" {
			domains = append(domains, *site.CustomDomain)
		}
	}

	return domains, nil
}

// RefreshDomains reloads the allowed domains list from database
func (m *Manager) RefreshDomains() error {
	domains, err := m.GetAllowedDomains()
	if err != nil {
		return err
	}

	log.Printf("TLS: Managing certificates for %d domains", len(domains))
	for _, domain := range domains {
		log.Printf("TLS: - %s", domain)
	}

	// Tell certmagic to manage these domains
	if err := m.certmagic.ManageAsync(m.db.Statement.Context, domains); err != nil {
		return fmt.Errorf("failed to manage domains: %w", err)
	}

	return nil
}

// GetTLSConfig returns TLS config for HTTPS server
func (m *Manager) GetTLSConfig() *tls.Config {
	return m.certmagic.TLSConfig()
}
