package backup

import (
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
