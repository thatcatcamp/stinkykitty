package backup

import (
	"fmt"
	"log"
	"time"
)

// Scheduler handles automatic backup scheduling
type Scheduler struct {
	Manager        *BackupManager
	ticker         *time.Ticker
	done           chan bool
	stopChan       chan bool
	BackupInterval time.Duration // For testing
}

// NewScheduler creates a new backup scheduler
func NewScheduler(manager *BackupManager) *Scheduler {
	return &Scheduler{
		Manager:        manager,
		BackupInterval: 24 * time.Hour, // Default: daily
		done:           make(chan bool, 1),
		stopChan:       make(chan bool, 1),
	}
}

// Start begins the backup scheduler in a goroutine
// Returns a done channel that will be closed when scheduler stops
func (s *Scheduler) Start() chan bool {
	go func() {
		// Create ticker for backup interval
		s.ticker = time.NewTicker(s.BackupInterval)
		defer s.ticker.Stop()

		// Run initial backup immediately
		if err := s.runBackup(); err != nil {
			log.Printf("initial backup failed: %v\n", err)
		}

		// Loop until stopped
		for {
			select {
			case <-s.stopChan:
				s.done <- true
				return
			case <-s.ticker.C:
				if err := s.runBackup(); err != nil {
					log.Printf("scheduled backup failed: %v\n", err)
				}
			}
		}
	}()

	return s.done
}

// Stop stops the backup scheduler
func (s *Scheduler) Stop() {
	select {
	case s.stopChan <- true:
	default:
	}
}

// runBackup performs a single backup operation
func (s *Scheduler) runBackup() error {
	// TODO: Integrate with database for actual backup
	// For now, create empty backup for testing
	_, err := s.Manager.CreateBackup("")
	if err != nil {
		return fmt.Errorf("backup creation failed: %w", err)
	}

	// TODO: Delete old backups (keep last 10)
	return nil
}

// SetInterval sets the backup interval for testing
func (s *Scheduler) SetInterval(interval time.Duration) {
	s.BackupInterval = interval
}
