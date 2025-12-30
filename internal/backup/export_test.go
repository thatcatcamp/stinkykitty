// SPDX-License-Identifier: MIT
package backup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateSiteExport(t *testing.T) {
	tmpDir := t.TempDir()
	exporter := NewSiteExporter(tmpDir)

	// Create a site export
	filename, err := exporter.CreateSiteExport(1, "test-site")
	if err != nil {
		t.Fatalf("CreateSiteExport failed: %v", err)
	}

	// Verify filename format
	if filename == "" {
		t.Fatal("CreateSiteExport returned empty filename")
	}

	// Verify file was created
	exportPath := filepath.Join(tmpDir, "site-exports", filename)
	if _, err := os.Stat(exportPath); os.IsNotExist(err) {
		t.Errorf("export file not created at: %s", exportPath)
	}
}

func TestCreateSiteExportFilenameFormat(t *testing.T) {
	tmpDir := t.TempDir()
	exporter := NewSiteExporter(tmpDir)

	filename, err := exporter.CreateSiteExport(1, "test-site")
	if err != nil {
		t.Fatalf("CreateSiteExport failed: %v", err)
	}

	// Format: site-1-2025-12-25-143022.tar.gz
	if !strings.HasPrefix(filename, "site-1-") {
		t.Errorf("filename should start with 'site-1-', got: %s", filename)
	}
	if !strings.HasSuffix(filename, ".tar.gz") {
		t.Errorf("filename should end with '.tar.gz', got: %s", filename)
	}
}
