// SPDX-License-Identifier: MIT
package handlers

import (
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/config"
)

// ServeAssetHandler serves uploaded media from centralized storage
func ServeAssetHandler(c *gin.Context) {
	// Get filename from URL
	filename := c.Param("filename")
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

	// Build full path to file
	filePath := filepath.Join(mediaDir, filename)

	// Serve the file
	c.File(filePath)
}
