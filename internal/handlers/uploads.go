// SPDX-License-Identifier: MIT
package handlers

import (
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/models"
	"github.com/thatcatcamp/stinkykitty/internal/uploads"
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

	// Get uploaded file from form
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No image file provided"})
		return
	}

	// Validate file size (max 5MB)
	const maxFileSize = 5 * 1024 * 1024 // 5MB in bytes
	if file.Size > maxFileSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File size must be less than 5MB"})
		return
	}

	// Validate it's an image file
	if !uploads.IsImageFile(file.Filename) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File must be an image (jpg, jpeg, png, gif, webp)"})
		return
	}

	// Use site directory from database
	siteDir := site.SiteDir

	// Save the file
	webPath, err := uploads.SaveUploadedFile(file, siteDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image"})
		return
	}

	// Return the web-accessible URL
	c.JSON(http.StatusOK, gin.H{
		"url": webPath,
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
