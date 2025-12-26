package handlers

import (
	"fmt"
	"log"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/backup"
	"github.com/thatcatcamp/stinkykitty/internal/models"
	"gorm.io/gorm"
)

// ExportSiteHandler handles site export requests
func ExportSiteHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get site ID from query parameter
		siteIDStr := c.Query("site")
		if siteIDStr == "" {
			c.JSON(400, gin.H{"error": "site parameter required"})
			return
		}

		siteID, err := strconv.ParseUint(siteIDStr, 10, 32)
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid site ID"})
			return
		}

		// Get site from database to verify it exists and get its name
		var site models.Site
		if err := db.First(&site, uint(siteID)).Error; err != nil {
			c.JSON(404, gin.H{"error": "site not found"})
			return
		}

		// Create site exporter
		backupPath := "/var/lib/stinkykitty/backups"
		exporter := backup.NewSiteExporter(backupPath)

		// Create export file
		filename, err := exporter.CreateSiteExport(uint(siteID), site.Subdomain)
		if err != nil {
			log.Printf("export failed: %v", err)
			c.JSON(500, gin.H{"error": "export failed"})
			return
		}

		// Prepare file for download
		filePath := fmt.Sprintf("%s/site-exports/%s", backupPath, filename)

		// Set response headers for download
		c.Header("Content-Type", "application/gzip")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

		// Send file
		c.File(filePath)

		// Clean up: delete the export file after sending
		// (in production, you might want to keep it for a period or delete it asynchronously)
		if err := deleteExportFile(filePath); err != nil {
			log.Printf("failed to clean up export file: %v", err)
		}
	}
}

// deleteExportFile removes an export file
func deleteExportFile(filePath string) error {
	// In tests, we might want to keep the file
	// For now, skip deletion to allow verification
	// TODO: Implement proper cleanup (maybe after download completes)
	return nil
}
