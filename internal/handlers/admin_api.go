package handlers

import (
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/models"
)

// validSubdomainRegex matches DNS-compliant subdomains (RFC 1123)
// Format: starts and ends with alphanumeric, may contain hyphens in middle
// Valid: "mycamp", "my-camp", "camp123", "a"
// Invalid: "-camp", "camp-", "--camp"
var validSubdomainRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// Reserved subdomains that cannot be used
var reservedSubdomains = map[string]bool{
	"admin":  true,
	"api":    true,
	"www":    true,
	"mail":   true,
	"ftp":    true,
	"smtp":   true,
	"pop":    true,
	"imap":   true,
	"stinky": true,
	"status": true,
}

// SubdomainCheckHandler checks if a subdomain is available
func SubdomainCheckHandler(c *gin.Context) {
	subdomain := c.Query("subdomain")

	if subdomain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subdomain required"})
		return
	}

	// Normalize subdomain
	subdomain = strings.TrimSpace(subdomain)

	// Validate length
	if len(subdomain) < 2 || len(subdomain) > 63 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subdomain must be 2-63 characters"})
		return
	}

	// Validate format (lowercase alphanumeric and hyphens only)
	if !validSubdomainRegex.MatchString(subdomain) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subdomain contains invalid characters"})
		return
	}

	// Check if subdomain is reserved
	if reservedSubdomains[subdomain] {
		c.JSON(http.StatusOK, gin.H{"available": false})
		return
	}

	// Check if subdomain exists in database (including soft-deleted)
	var site models.Site
	result := db.GetDB().Unscoped().Where("subdomain = ?", subdomain).First(&site)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Subdomain is available
			c.JSON(http.StatusOK, gin.H{"available": true})
			return
		}
		// Actual database error occurred
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	// Subdomain exists
	c.JSON(http.StatusOK, gin.H{"available": false})
}
