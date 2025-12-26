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
func (bm *BackupManager) CreateBackup(dbPath string) (filename string, retErr error) {
	// Ensure backup directory exists
	if err := os.MkdirAll(filepath.Join(bm.BackupPath, "system"), 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("2006-01-02-150405")
	filename = fmt.Sprintf("stinkykitty-%s.tar.gz", timestamp)
	backupPath := filepath.Join(bm.BackupPath, "system", filename)

	// Create tar.gz file
	out, err := os.Create(backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer func() {
		if err := out.Close(); err != nil && retErr == nil {
			retErr = fmt.Errorf("failed to close backup file: %w", err)
		}
	}()

	// Create gzip writer
	gz := gzip.NewWriter(out)
	defer func() {
		if err := gz.Close(); err != nil && retErr == nil {
			retErr = fmt.Errorf("failed to close gzip writer: %w", err)
		}
	}()

	// Create tar writer
	tw := tar.NewWriter(gz)
	defer func() {
		if err := tw.Close(); err != nil && retErr == nil {
			retErr = fmt.Errorf("failed to close tar writer: %w", err)
		}
	}()

	// Add database dump to tar
	if dbPath != "" {
		if err := addFileToTar(tw, dbPath, "database.db"); err != nil {
			os.Remove(backupPath)
			return "", fmt.Errorf("failed to add database to backup: %w", err)
		}
	}

	// Add media directory to tar
	// Media files are stored at a standard location
	mediaPath := filepath.Join("/", "var", "lib", "stinkykitty", "uploads")
	if _, err := os.Stat(mediaPath); err == nil {
		if err := addDirToTar(tw, mediaPath, "uploads"); err != nil {
			os.Remove(backupPath)
			return "", fmt.Errorf("failed to add media to backup: %w", err)
		}
	}

	return filename, retErr
}

// addFileToTar adds a single file to tar archive
func addFileToTar(tw *tar.Writer, filePath string, tarPath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}

	header := &tar.Header{
		Name: tarPath,
		Size: stat.Size(),
		Mode: 0644,
	}

	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write tar header for %s: %w", filePath, err)
	}

	if _, err := io.Copy(tw, file); err != nil {
		return fmt.Errorf("failed to copy file %s to tar: %w", filePath, err)
	}

	return nil
}

// addDirToTar recursively adds a directory to tar archive
func addDirToTar(tw *tar.Writer, dirPath string, tarPath string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", dirPath, err)
	}

	for _, entry := range entries {
		path := filepath.Join(dirPath, entry.Name())
		name := filepath.Join(tarPath, entry.Name())

		if entry.IsDir() {
			if err := addDirToTar(tw, path, name); err != nil {
				return fmt.Errorf("failed to add subdirectory %s: %w", name, err)
			}
		} else {
			if err := addFileToTar(tw, path, name); err != nil {
				return fmt.Errorf("failed to add file %s: %w", name, err)
			}
		}
	}

	return nil
}
