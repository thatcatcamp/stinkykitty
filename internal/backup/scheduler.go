package backup

// Scheduler handles automatic backup scheduling
type Scheduler struct {
	Manager *BackupManager
}

// NewScheduler creates a new backup scheduler
func NewScheduler(manager *BackupManager) *Scheduler {
	return &Scheduler{
		Manager: manager,
	}
}
