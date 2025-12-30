// SPDX-License-Identifier: MIT
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

var v *viper.Viper

// InitConfig initializes the configuration system
func InitConfig(configPath string) error {
	v = viper.New()

	// Set defaults
	setDefaults()

	// Set config file path
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Create config directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Try to read existing config
	if err := v.ReadInConfig(); err != nil {
		// If config doesn't exist, create it with defaults
		if os.IsNotExist(err) {
			if err := v.WriteConfigAs(configPath); err != nil {
				return fmt.Errorf("failed to write config: %w", err)
			}
		} else {
			return fmt.Errorf("failed to read config: %w", err)
		}
	}

	return nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	v.SetDefault("server.http_port", "80")
	v.SetDefault("server.https_port", "443")
	v.SetDefault("server.behind_proxy", false)
	v.SetDefault("server.base_domain", "localhost")

	// Storage defaults
	v.SetDefault("storage.data_dir", "/var/lib/stinkykitty")
	v.SetDefault("storage.sites_dir", "/var/lib/stinkykitty/sites")
	v.SetDefault("storage.backups_dir", "/var/lib/stinkykitty/backups")

	// Backup defaults
	v.SetDefault("backups.path", "/var/lib/stinkykitty/backups")
	v.SetDefault("backups.interval", "24h")          // Daily backups
	v.SetDefault("backups.retention", 10)            // Keep last 10 backups
	v.SetDefault("backups.enable_auto_backup", true) // Enabled by default
	v.SetDefault("backups.schedule", "0 3 * * *")    // 3am daily (cron format)
	v.SetDefault("backups.retention.daily", 7)
	v.SetDefault("backups.retention.weekly", 4)
	v.SetDefault("backups.retention.monthly", 12)

	// Database defaults
	v.SetDefault("database.type", "sqlite")
	v.SetDefault("database.path", "/var/lib/stinkykitty/stinkykitty.db")

	// Auth defaults
	v.SetDefault("auth.jwt_secret", "CHANGE_ME_IN_PRODUCTION_USE_ENV_VAR")
	v.SetDefault("auth.jwt_expiry_hours", 8)
	v.SetDefault("auth.bcrypt_cost", 12)

	// TLS defaults
	v.SetDefault("server.tls_enabled", false)
	v.SetDefault("tls.email", "")
	v.SetDefault("tls.cert_dir", "/var/lib/stinkykitty/certs")
	v.SetDefault("tls.staging", false)
}

// GetString returns a config value as string
func GetString(key string) string {
	if v == nil {
		return ""
	}
	return v.GetString(key)
}

// GetInt returns a config value as int
func GetInt(key string) int {
	if v == nil {
		return 0
	}
	return v.GetInt(key)
}

// GetBool returns a config value as bool
func GetBool(key string) bool {
	if v == nil {
		return false
	}
	return v.GetBool(key)
}

// GetDuration returns a config value as time.Duration
func GetDuration(key string) time.Duration {
	if v == nil {
		return 0
	}
	return v.GetDuration(key)
}

// Set sets a config value and saves to file
func Set(key string, value interface{}) error {
	if v == nil {
		return fmt.Errorf("config not initialized")
	}

	v.Set(key, value)

	if err := v.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// GetAll returns all config values as a map
func GetAll() map[string]interface{} {
	if v == nil {
		return nil
	}
	return v.AllSettings()
}
