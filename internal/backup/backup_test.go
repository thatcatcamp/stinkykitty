package backup

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
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

func TestCreateBackup(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	manager := NewBackupManager(tmpDir)

	// Test that CreateBackup returns a valid filename even without database
	filename, err := manager.CreateBackup("")
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

func TestCreateBackupFilenameFormat(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewBackupManager(tmpDir)

	// Create an empty tar to satisfy the method (no media dir, no db)
	filename, err := manager.CreateBackup("")
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	// Verify filename format: stinkykitty-YYYY-MM-DD-HHMMSS.tar.gz
	if !strings.HasPrefix(filename, "stinkykitty-") {
		t.Errorf("filename should start with 'stinkykitty-', got: %s", filename)
	}
	if !strings.HasSuffix(filename, ".tar.gz") {
		t.Errorf("filename should end with '.tar.gz', got: %s", filename)
	}

	// Verify file was actually created
	backupPath := filepath.Join(tmpDir, "system", filename)
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Errorf("backup file not created at expected location: %s", backupPath)
	}
}

func TestRestoreBackup(t *testing.T) {
	// Setup: create a temporary directory structure
	tmpDir := t.TempDir()
	manager := NewBackupManager(tmpDir)
	// Override BasePath for testing to avoid permission issues
	manager.BasePath = tmpDir

	// Create test media files
	testMediaDir := filepath.Join(tmpDir, "test-media")
	if err := os.MkdirAll(testMediaDir, 0755); err != nil {
		t.Fatalf("failed to create test media directory: %v", err)
	}
	testFile := filepath.Join(testMediaDir, "test-image.jpg")
	if err := os.WriteFile(testFile, []byte("fake image data"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create a test backup manually
	backupFilename := "test-restore-backup.tar.gz"
	backupPath := filepath.Join(tmpDir, "system", backupFilename)
	if err := os.MkdirAll(filepath.Join(tmpDir, "system"), 0755); err != nil {
		t.Fatalf("failed to create system directory: %v", err)
	}

	if err := createTestBackup(backupPath, testMediaDir); err != nil {
		t.Fatalf("failed to create test backup: %v", err)
	}

	// Test restore operation - now uses single parameter
	err := manager.RestoreBackup(backupFilename)
	if err != nil {
		t.Fatalf("RestoreBackup failed: %v", err)
	}

	// Verify the uploads directory was created
	uploadsDir := filepath.Join(tmpDir, "uploads")
	if _, err := os.Stat(uploadsDir); os.IsNotExist(err) {
		t.Errorf("uploads directory not created at: %s", uploadsDir)
	}

	// Verify the test file was restored
	restoredFile := filepath.Join(uploadsDir, "test-image.jpg")
	if _, err := os.Stat(restoredFile); os.IsNotExist(err) {
		t.Errorf("test file not restored at: %s", restoredFile)
	}
}

func TestRestoreBackupFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewBackupManager(tmpDir)

	// Test with non-existent backup file - now uses single parameter
	err := manager.RestoreBackup("nonexistent-backup.tar.gz")
	if err == nil {
		t.Fatal("expected error for non-existent backup file, got nil")
	}
	if !strings.Contains(err.Error(), "failed to open backup file") {
		t.Errorf("expected 'failed to open backup file' error, got: %v", err)
	}
}

func TestRestoreBackupExtractsFiles(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	manager := NewBackupManager(tmpDir)
	// Override BasePath for testing to avoid permission issues
	manager.BasePath = tmpDir

	// Create test media directory with files
	testMediaDir := filepath.Join(tmpDir, "source-media")
	testSubDir := filepath.Join(testMediaDir, "subfolder")
	if err := os.MkdirAll(testSubDir, 0755); err != nil {
		t.Fatalf("failed to create test subdirectory: %v", err)
	}

	testFiles := map[string]string{
		"image1.jpg":           "image1 content",
		"subfolder/image2.png": "image2 content",
	}
	for path, content := range testFiles {
		fullPath := filepath.Join(testMediaDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file %s: %v", path, err)
		}
	}

	// Create backup - note: we need to modify CreateBackup or manually create a tar
	// For now, manually create a proper tar.gz with our test structure
	backupFilename := "test-backup.tar.gz"
	backupPath := filepath.Join(tmpDir, "system", backupFilename)
	if err := os.MkdirAll(filepath.Join(tmpDir, "system"), 0755); err != nil {
		t.Fatalf("failed to create system directory: %v", err)
	}

	// Create the tar.gz file manually
	if err := createTestBackup(backupPath, testMediaDir); err != nil {
		t.Fatalf("failed to create test backup: %v", err)
	}

	// Test restore - now uses single parameter
	err := manager.RestoreBackup(backupFilename)
	if err != nil {
		t.Fatalf("RestoreBackup failed: %v", err)
	}

	// Verify files were extracted
	uploadsDir := filepath.Join(tmpDir, "uploads")
	for path, expectedContent := range testFiles {
		restoredPath := filepath.Join(uploadsDir, path)
		content, err := os.ReadFile(restoredPath)
		if err != nil {
			t.Errorf("failed to read restored file %s: %v", path, err)
			continue
		}
		if string(content) != expectedContent {
			t.Errorf("restored file %s content mismatch: expected %q, got %q",
				path, expectedContent, string(content))
		}
	}
}

// TestRestoreDatabaseFile verifies that RestoreBackup extracts and restores the database.db file
func TestRestoreDatabaseFile(t *testing.T) {
	// Setup: Create a temporary backup with database.db
	tmpDir := t.TempDir()
	manager := NewBackupManager(tmpDir)
	// Override BasePath for testing to avoid permission issues
	manager.BasePath = tmpDir

	// Create a test database file
	testDBPath := filepath.Join(tmpDir, "test_db.sqlite3")
	testDBFile, err := os.Create(testDBPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	testDBFile.WriteString("test database content v1")
	testDBFile.Close()

	// Create a backup containing the test database
	backupFile, err := manager.CreateBackup(testDBPath)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Modify the original database to simulate a different state
	os.WriteFile(testDBPath, []byte("modified content"), 0644)

	// Restore from backup - database.db should be extracted and restored
	err = manager.RestoreBackup(backupFile)
	if err != nil {
		t.Fatalf("RestoreBackup failed: %v", err)
	}

	// Verify: The restored database.db should exist in BasePath
	restoredDBPath := filepath.Join(manager.BasePath, "database.db")
	content, err := os.ReadFile(restoredDBPath)
	if err != nil {
		t.Fatalf("Restored database.db not found at %s: %v", restoredDBPath, err)
	}

	// Verify content matches original backup
	if string(content) != "test database content v1" {
		t.Errorf("Restored database content mismatch. Expected 'test database content v1', got '%s'", string(content))
	}
}

// Helper function to create a test backup
func createTestBackup(backupPath, sourceDir string) error {
	out, err := os.Create(backupPath)
	if err != nil {
		return err
	}
	defer out.Close()

	gz := gzip.NewWriter(out)
	defer gz.Close()

	tw := tar.NewWriter(gz)
	defer tw.Close()

	// Add source directory contents as "uploads" in the tar
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		header := &tar.Header{
			Name: filepath.Join("uploads", relPath),
			Size: info.Size(),
			Mode: 0644,
		}

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		_, err = io.Copy(tw, file)
		return err
	})
}
