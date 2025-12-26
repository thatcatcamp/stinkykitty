package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/models"
)

// SubdomainCheckHandler checks if a subdomain is available
func SubdomainCheckHandler(c *gin.Context) {
	subdomain := c.Query("subdomain")

	if subdomain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subdomain required"})
		return
	}

	// Check if subdomain exists (including soft-deleted)
	var site models.Site
	result := db.GetDB().Unscoped().Where("subdomain = ?", subdomain).First(&site)

	if result.Error == nil {
		// Subdomain exists
		c.JSON(http.StatusOK, gin.H{"available": false})
		return
	}

	// Subdomain is available
	c.JSON(http.StatusOK, gin.H{"available": true})
}
