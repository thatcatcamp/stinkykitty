package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/models"
)

// ServeHomepage serves the site's homepage
func ServeHomepage(c *gin.Context) {
	siteVal, exists := c.Get("site")
	if !exists {
		c.String(404, "Site not found")
		return
	}

	site := siteVal.(*models.Site)

	// For now, just return site info (content blocks come later)
	c.String(200, "Welcome to %s!\nSubdomain: %s", site.SiteTitle, site.Subdomain)
}
