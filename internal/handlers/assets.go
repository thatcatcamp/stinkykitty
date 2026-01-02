// SPDX-License-Identifier: MIT
package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/config"
)

// ServeAssetHandler serves uploaded media from centralized storage
func ServeAssetHandler(c *gin.Context) {
	// Get filename from URL
	filename := c.Param("filepath")
	if filename == "" {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// Strip leading slash if present
	if len(filename) > 0 && filename[0] == '/' {
		filename = filename[1:]
	}

	// Get centralized media directory
	mediaDir := config.GetString("storage.media_dir")
	if mediaDir == "" {
		mediaDir = "/var/lib/stinkykitty/media"
	}

	// Build full path to file (files are in uploads subdirectory)
	uploadsDir := filepath.Join(mediaDir, "uploads")
	filePath := filepath.Join(uploadsDir, filename)

	// CRITICAL: Prevent path traversal attacks
	cleanPath := filepath.Clean(filePath)
	cleanUploadsDir := filepath.Clean(uploadsDir)
	if !strings.HasPrefix(cleanPath, cleanUploadsDir+string(os.PathSeparator)) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	// Serve the file
	c.File(cleanPath)
}
