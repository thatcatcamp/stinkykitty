package media

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/thatcatcamp/stinkykitty/internal/models"
	"gorm.io/gorm"
)

// ImportExistingUploads scans uploads directory and creates media_items records
// Returns count of imported files
func ImportExistingUploads(db *gorm.DB, site models.Site) (int, error) {
	uploadsDir := filepath.Join(site.SiteDir, "uploads")

	// Check if directory exists
	if _, err := os.Stat(uploadsDir); os.IsNotExist(err) {
		return 0, nil // No uploads directory, nothing to import
	}

	// Read directory
	files, err := os.ReadDir(uploadsDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read uploads directory: %w", err)
	}

	count := 0
	for _, file := range files {
		if file.IsDir() {
			continue // Skip subdirectories (like thumbs/)
		}

		filename := file.Name()

		// Only import image files
		ext := strings.ToLower(filepath.Ext(filename))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" && ext != ".webp" {
			continue
		}

		// Check if already imported
		var existing models.MediaItem
		err := db.Where("site_id = ? AND filename = ?", site.ID, filename).First(&existing).Error
		if err == nil {
			continue // Already imported
		}

		// Get file info
		fileInfo, err := file.Info()
		if err != nil {
			continue // Skip if can't get info
		}

		// Detect mime type from extension
		mimeType := "image/jpeg"
		switch ext {
		case ".png":
			mimeType = "image/png"
		case ".gif":
			mimeType = "image/gif"
		case ".webp":
			mimeType = "image/webp"
		}

		// Create media item
		mediaItem := models.MediaItem{
			SiteID:       site.ID,
			Filename:     filename,
			OriginalName: filename, // Best guess
			FileSize:     fileInfo.Size(),
			MimeType:     mimeType,
			UploadedBy:   site.OwnerID, // Assume owner uploaded
		}

		if err := db.Create(&mediaItem).Error; err != nil {
			return count, fmt.Errorf("failed to create media item for %s: %w", filename, err)
		}

		count++
	}

	return count, nil
}
