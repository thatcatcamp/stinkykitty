// SPDX-License-Identifier: MIT
package backup

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSchedulerStart(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewBackupManager(tmpDir)
	scheduler := NewScheduler(manager)

	// Start scheduler - should not error
	done := scheduler.Start()
	if done == nil {
		t.Fatal("Start returned nil done channel")
	}

	// Stop scheduler after short time
	time.Sleep(100 * time.Millisecond)
	scheduler.Stop()
}

func TestSchedulerStop(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewBackupManager(tmpDir)
	scheduler := NewScheduler(manager)

	done := scheduler.Start()
	time.Sleep(50 * time.Millisecond)
	scheduler.Stop()

	// Wait for done signal with timeout
	select {
	case <-done:
		// Successfully stopped
	case <-time.After(1 * time.Second):
		t.Fatal("scheduler did not stop within timeout")
	}
}

func TestSetInterval(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewBackupManager(tmpDir)
	scheduler := NewScheduler(manager)

	// Test SetInterval
	newInterval := 100 * time.Millisecond
	scheduler.SetInterval(newInterval)

	if scheduler.BackupInterval != newInterval {
		t.Errorf("expected interval %v, got %v", newInterval, scheduler.BackupInterval)
	}
}

func TestSchedulerInitialBackup(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewBackupManager(tmpDir)
	manager.BasePath = tmpDir
	scheduler := NewScheduler(manager)
	scheduler.SetInterval(10 * time.Millisecond)

	// Start scheduler - should create initial backup
	done := scheduler.Start()
	time.Sleep(50 * time.Millisecond)
	scheduler.Stop()

	// Wait for stop
	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("scheduler did not stop")
	}

	// Verify backup directory exists and has backup file
	backupDir := filepath.Join(tmpDir, "system")
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		t.Errorf("backup directory not created: %s", backupDir)
	}
}
