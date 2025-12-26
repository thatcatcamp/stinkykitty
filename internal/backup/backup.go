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
	BasePath   string // Base path for system files (default: /var/lib/stinkykitty)
}

// NewBackupManager creates a new backup manager
func NewBackupManager(backupPath string) *BackupManager {
	return &BackupManager{
		BackupPath: backupPath,
		BasePath:   filepath.Join("/", "var", "lib", "stinkykitty"), // Default production path
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
	mediaPath := filepath.Join(bm.BasePath, "uploads")
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

// RestoreBackup restores the system from a backup tarball.
// It extracts the database dump and media files to their standard locations.
// Database restoration via GORM will be handled in a separate task.
func (bm *BackupManager) RestoreBackup(filename string) error {
	// Construct full path to backup file
	backupPath := filepath.Join(bm.BackupPath, "system", filename)

	// Open backup file
	file, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()

	// Create gzip reader
	gz, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gz.Close()

	// Create tar reader
	tr := tar.NewReader(gz)

	// Extract all files from the tar
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Handle database.db - extract to BasePath
		if header.Name == "database.db" {
			targetPath := filepath.Join(bm.BasePath, header.Name)

			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for %s: %w", targetPath, err)
			}

			// Create file
			outFile, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("failed to create database file %s: %w", targetPath, err)
			}

			// Copy file contents
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to extract database file: %w", err)
			}

			// Close file
			if err := outFile.Close(); err != nil {
				return fmt.Errorf("failed to close database file: %w", err)
			}

			// Set file permissions
			if err := os.Chmod(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to set permissions for database file: %w", err)
			}

			continue
		}

		// Only process files that start with "uploads/" prefix
		if !filepath.HasPrefix(header.Name, "uploads/") && header.Name != "uploads" {
			continue
		}

		// Construct target path: BasePath + full path from tar
		// e.g., tar has "uploads/photo.jpg" -> extract to "/var/lib/stinkykitty/uploads/photo.jpg"
		targetPath := filepath.Join(bm.BasePath, header.Name)

		// Handle directories
		if header.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}
			continue
		}

		// Handle regular files
		if header.Typeflag == tar.TypeReg {
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for %s: %w", targetPath, err)
			}

			// Create file
			outFile, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", targetPath, err)
			}

			// Copy file contents
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to extract file %s: %w", targetPath, err)
			}

			// Close file
			if err := outFile.Close(); err != nil {
				return fmt.Errorf("failed to close file %s: %w", targetPath, err)
			}

			// Set file permissions
			if err := os.Chmod(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to set permissions for %s: %w", targetPath, err)
			}
		}
	}

	return nil
}
