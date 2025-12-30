// SPDX-License-Identifier: MIT
package backup

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SiteExporter handles site-specific exports
type SiteExporter struct {
	BackupPath string
}

// NewSiteExporter creates a new site exporter
func NewSiteExporter(backupPath string) *SiteExporter {
	return &SiteExporter{
		BackupPath: backupPath,
	}
}

// CreateSiteExport creates an export tarball for a specific site
// It includes the site's pages, menus, and uploaded media
func (se *SiteExporter) CreateSiteExport(siteID uint, siteName string) (filename string, retErr error) {
	// Ensure export directory exists
	exportDir := filepath.Join(se.BackupPath, "site-exports")
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create export directory: %w", err)
	}

	// Generate filename: site-{ID}-YYYY-MM-DD-HHMMSS.tar.gz
	timestamp := time.Now().Format("2006-01-02-150405")
	filename = fmt.Sprintf("site-%d-%s.tar.gz", siteID, timestamp)
	exportPath := filepath.Join(exportDir, filename)

	// Create export file
	out, err := os.Create(exportPath)
	if err != nil {
		return "", fmt.Errorf("failed to create export file: %w", err)
	}
	defer func() {
		if err := out.Close(); err != nil && retErr == nil {
			retErr = fmt.Errorf("failed to close export file: %w", err)
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

	// Create metadata file for this site export
	metadata := fmt.Sprintf("site_id=%d\nsite_name=%s\nexport_timestamp=%s\n", siteID, siteName, timestamp)
	metaHeader := &tar.Header{
		Name: "EXPORT_INFO",
		Size: int64(len(metadata)),
		Mode: 0644,
	}
	if err := tw.WriteHeader(metaHeader); err != nil {
		os.Remove(exportPath)
		return "", fmt.Errorf("failed to write metadata header: %w", err)
	}
	if _, err := tw.Write([]byte(metadata)); err != nil {
		os.Remove(exportPath)
		return "", fmt.Errorf("failed to write metadata content: %w", err)
	}

	// TODO: Export site pages, menus, and media
	// This will be integrated with GORM and handlers in later tasks

	return filename, nil
}
