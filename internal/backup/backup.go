// Package backup provides backup and export functionality for the StinkyKitty CMS.
package backup

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

// CreateBackup creates a new system backup with database and media files
func (bm *BackupManager) CreateBackup(dbType string, dbPath string) (string, error) {
	// Ensure backup directory exists
	if err := os.MkdirAll(filepath.Join(bm.BackupPath, "system"), 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("2006-01-02-150405")
	filename := fmt.Sprintf("stinkykitty-%s.tar.gz", timestamp)
	backupPath := filepath.Join(bm.BackupPath, "system", filename)

	// Create tar.gz file
	out, err := os.Create(backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer out.Close()

	// Create gzip writer
	gz := gzip.NewWriter(out)
	defer gz.Close()

	// Create tar writer
	tw := tar.NewWriter(gz)
	defer tw.Close()

	// Add database dump to tar
	if dbPath != "" {
		if err := addFileToTar(tw, dbPath, "database.db"); err != nil {
			os.Remove(backupPath)
			return "", fmt.Errorf("failed to add database to backup: %w", err)
		}
	}

	// Add media directory to tar
	mediaPath := filepath.Join("var", "lib", "stinkykitty", "uploads")
	if _, err := os.Stat(mediaPath); err == nil {
		if err := addDirToTar(tw, mediaPath, "uploads"); err != nil {
			os.Remove(backupPath)
			return "", fmt.Errorf("failed to add media to backup: %w", err)
		}
	}

	return filename, nil
}

// addFileToTar adds a single file to tar archive
func addFileToTar(tw *tar.Writer, filePath string, tarPath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	header := &tar.Header{
		Name: tarPath,
		Size: stat.Size(),
		Mode: 0644,
	}

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	_, err = io.Copy(tw, file)
	return err
}

// addDirToTar recursively adds a directory to tar archive
func addDirToTar(tw *tar.Writer, dirPath string, tarPath string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(dirPath, entry.Name())
		name := filepath.Join(tarPath, entry.Name())

		if entry.IsDir() {
			if err := addDirToTar(tw, path, name); err != nil {
				return err
			}
		} else {
			if err := addFileToTar(tw, path, name); err != nil {
				return err
			}
		}
	}

	return nil
}
