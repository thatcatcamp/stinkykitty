// SPDX-License-Identifier: MIT
// internal/handlers/admin_delete_site.go
package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/models"
	"github.com/thatcatcamp/stinkykitty/internal/sites"
)

var _ = sites.DeleteSite // keep import

// DeleteSiteHandler handles soft-delete of a site
func DeleteSiteHandler(c *gin.Context) {
	// Get user from context
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}
	user := userVal.(*models.User)

	// Get site ID from URL parameter (not query param)
	siteIDStr := c.Param("id")
	if siteIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "site ID required"})
		return
	}

	siteID, err := strconv.ParseUint(siteIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid site ID"})
		return
	}

	// Verify user is owner/admin of site
	// Use Unscoped to find the site even if already soft-deleted
	var site models.Site
	if err := db.GetDB().Unscoped().First(&site, uint(siteID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "site not found"})
		return
	}

	// Check permissions: owner or global admin only
	if site.OwnerID != user.ID && !user.IsGlobalAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	// Soft-delete the site
	if err := sites.DeleteSite(db.GetDB(), uint(siteID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to delete site: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "site deleted"})
}
