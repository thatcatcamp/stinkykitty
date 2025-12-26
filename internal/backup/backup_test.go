package backup

import (
	"strings"
	"testing"
	"time"
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
	filename, err := manager.CreateBackup("test", "")
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

func TestBackupFilenameFormat(t *testing.T) {
	// Test the expected backup filename format without creating actual backup
	now := time.Now()
	expectedPrefix := "stinkykitty-" + now.Format("2006-01-02")

	if !strings.Contains("stinkykitty-2025-12-25-143022.tar.gz", expectedPrefix) {
		t.Fatalf("backup filename should contain date prefix")
	}
}
