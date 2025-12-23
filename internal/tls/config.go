package tls

import (
	"fmt"
	"os"

	"github.com/thatcatcamp/stinkykitty/internal/config"
)

// Config holds TLS configuration
type Config struct {
	Email      string
	CertDir    string
	Staging    bool
	BaseDomain string
	Enabled    bool
}

// LoadConfig loads TLS configuration from config system
func LoadConfig() (*Config, error) {
	cfg := &Config{
		Email:      config.GetString("tls.email"),
		CertDir:    config.GetString("tls.cert_dir"),
		Staging:    config.GetBool("tls.staging"),
		BaseDomain: config.GetString("server.base_domain"),
		Enabled:    config.GetBool("server.tls_enabled"),
	}

	// Validate required fields if TLS is enabled
	if cfg.Enabled {
		if cfg.Email == "" {
			return nil, fmt.Errorf("tls.email is required when TLS is enabled")
		}
		if cfg.BaseDomain == "" {
			return nil, fmt.Errorf("server.base_domain is required when TLS is enabled")
		}
	}

	// Create cert directory if it doesn't exist
	if cfg.CertDir != "" {
		if err := os.MkdirAll(cfg.CertDir, 0700); err != nil {
			return nil, fmt.Errorf("failed to create cert directory: %w", err)
		}
	}

	return cfg, nil
}
