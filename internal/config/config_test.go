package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitConfig(t *testing.T) {
	// Create temp directory for test config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	err := InitConfig(configPath)
	if err != nil {
		t.Fatalf("InitConfig failed: %v", err)
	}

	// Verify config file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}
}

func TestGetConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	InitConfig(configPath)

	// Test getting a default value
	value := GetString("server.http_port")
	if value != "80" {
		t.Errorf("Expected default http_port to be 80, got %s", value)
	}
}

func TestSetConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	InitConfig(configPath)

	err := Set("server.http_port", "8080")
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	value := GetString("server.http_port")
	if value != "8080" {
		t.Errorf("Expected http_port to be 8080, got %s", value)
	}
}
