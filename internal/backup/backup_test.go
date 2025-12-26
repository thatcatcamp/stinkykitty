package backup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewBackupManager(t *testing.T) {
	manager := NewBackupManager("/tmp/backups")
	if manager == nil {
		t.Fatal("NewBackupManager returned nil")
	}
	if manager.BackupPath != "/tmp/backups" {
		t.Errorf("expected /tmp/backups, got %s", manager.BackupPath)
	}
}

func TestNewSiteExporter(t *testing.T) {
	exporter := NewSiteExporter("/tmp/backups")
	if exporter == nil {
		t.Fatal("NewSiteExporter returned nil")
	}
	if exporter.BackupPath != "/tmp/backups" {
		t.Errorf("expected /tmp/backups, got %s", exporter.BackupPath)
	}
}

func TestNewScheduler(t *testing.T) {
	manager := NewBackupManager("/tmp/backups")
	scheduler := NewScheduler(manager)
	if scheduler == nil {
		t.Fatal("NewScheduler returned nil")
	}
	if scheduler.Manager != manager {
		t.Fatal("scheduler manager not set correctly")
	}
}

func TestCreateBackup(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	manager := NewBackupManager(tmpDir)

	// Test that CreateBackup returns a valid filename even without database
	filename, err := manager.CreateBackup("")
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}
	if filename == "" {
		t.Fatal("CreateBackup should return a filename")
	}
	if !strings.Contains(filename, "stinkykitty-") {
		t.Fatalf("filename should contain 'stinkykitty-', got: %s", filename)
	}
	if !strings.HasSuffix(filename, ".tar.gz") {
		t.Fatalf("filename should end with '.tar.gz', got: %s", filename)
	}
}

func TestCreateBackupFilenameFormat(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewBackupManager(tmpDir)

	// Create an empty tar to satisfy the method (no media dir, no db)
	filename, err := manager.CreateBackup("")
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	// Verify filename format: stinkykitty-YYYY-MM-DD-HHMMSS.tar.gz
	if !strings.HasPrefix(filename, "stinkykitty-") {
		t.Errorf("filename should start with 'stinkykitty-', got: %s", filename)
	}
	if !strings.HasSuffix(filename, ".tar.gz") {
		t.Errorf("filename should end with '.tar.gz', got: %s", filename)
	}

	// Verify file was actually created
	backupPath := filepath.Join(tmpDir, "system", filename)
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Errorf("backup file not created at expected location: %s", backupPath)
	}
}
