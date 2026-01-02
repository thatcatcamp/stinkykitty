// SPDX-License-Identifier: MIT
package handlers

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/config"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/media"
	"github.com/thatcatcamp/stinkykitty/internal/models"
)

// UploadImageHandler handles image file uploads
func UploadImageHandler(c *gin.Context) {
	// Get site from context
	siteVal, exists := c.Get("site")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Site not found"})
		return
	}
	site := siteVal.(*models.Site)

	// Get user from context
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	user := userVal.(*models.User)

	// Get uploaded file from form
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No image file provided"})
		return
	}

	// Save the file to centralized storage
	filename, err := media.SaveToCentralizedStorage(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save image: %v", err)})
		return
	}

	// Create media item record for tracking
	mediaItem := models.MediaItem{
		SiteID:             site.ID,
		UploadedFromSiteID: &site.ID,
		Filename:           filename,
		OriginalName:       file.Filename,
		FileSize:           file.Size,
		MimeType:           file.Header.Get("Content-Type"),
		UploadedBy:         user.ID,
	}
	db.GetDB().Create(&mediaItem) // Ignore error - not critical

	// Generate thumbnail
	mediaDir := config.GetString("storage.media_dir")
	if mediaDir == "" {
		mediaDir = "/var/lib/stinkykitty/media"
	}
	srcPath := filepath.Join(mediaDir, "uploads", filename)
	thumbPath := filepath.Join(mediaDir, "uploads", "thumbs", filename)
	_ = media.GenerateThumbnail(srcPath, thumbPath, 200, 200)

	// Return the web-accessible URL
	c.JSON(http.StatusOK, gin.H{
		"url": "/assets/" + filename,
	})
}

// ServeUploadedFile serves uploaded files from the site's uploads directory
func ServeUploadedFile(c *gin.Context) {
	// Get site from context
	siteVal, exists := c.Get("site")
	if !exists {
		c.String(http.StatusInternalServerError, "Site not found")
		return
	}
	site := siteVal.(*models.Site)

	// Get filename from URL (strip leading slash if present)
	filename := c.Param("filepath")
	if len(filename) > 0 && filename[0] == '/' {
		filename = filename[1:]
	}

	// Build full path to file using site's directory
	filePath := filepath.Join(site.SiteDir, "uploads", filename)

	// Serve the file
	c.File(filePath)
}
