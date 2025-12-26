package backup

import (
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
