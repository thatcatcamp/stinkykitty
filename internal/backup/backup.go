// Package backup provides backup and export functionality for the StinkyKitty CMS.
package backup

import (
	"time"
)

// BackupMetadata represents metadata about a backup
type BackupMetadata struct {
	Timestamp    time.Time
	DatabaseDump string // path to database dump file
	MediaDir     string // path to media files
	Version      string
	Note         string
}

// BackupManager handles all backup operations
type BackupManager struct {
	BackupPath string // /var/lib/stinkykitty/backups/
}

// NewBackupManager creates a new backup manager
func NewBackupManager(backupPath string) *BackupManager {
	return &BackupManager{
		BackupPath: backupPath,
	}
}
