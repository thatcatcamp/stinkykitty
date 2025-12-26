package middleware

import (
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/models"
	"github.com/thatcatcamp/stinkykitty/internal/sites"
	"gorm.io/gorm"
)

// CacheEntry represents a cached site with expiration
type CacheEntry struct {
	Site      *models.Site
	ExpiresAt time.Time
}

var (
	siteCache sync.Map
	cacheTTL  = 60 * time.Second
)

// SiteResolutionMiddleware resolves the site based on Host header
func SiteResolutionMiddleware(db *gorm.DB, baseDomain string) gin.HandlerFunc {
	return func(c *gin.Context) {
		host := strings.ToLower(c.Request.Host)

		// Remove port if present
		if idx := strings.Index(host, ":"); idx != -1 {
			host = host[:idx]
		}

		// Check cache first
		if entry, ok := siteCache.Load(host); ok {
			cacheEntry := entry.(CacheEntry)
			if time.Now().Before(cacheEntry.ExpiresAt) {
				// Cache hit and not expired
				c.Set("site", cacheEntry.Site)
				c.Next()
				return
			}
			// Cache expired, remove it
			siteCache.Delete(host)
		}

		// Cache miss - query database
		var site *models.Site
		var err error

		// Try custom domain first
		site, err = sites.GetSiteByDomain(db, host)

		// If not found by custom domain, try subdomain
		if err != nil {
			subdomain := extractSubdomain(host, baseDomain)
			if subdomain != "" {
				site, err = sites.GetSiteBySubdomain(db, subdomain)
			}
		}

		// Site not found - return 404
		if err != nil || site == nil {
			c.AbortWithStatus(404)
			return
		}

		// Cache the site
		siteCache.Store(host, CacheEntry{
			Site:      site,
			ExpiresAt: time.Now().Add(cacheTTL),
		})

		// Set site in context
		c.Set("site", site)
		c.Next()
	}
}

// extractSubdomain extracts the subdomain from a host
// e.g., "testcamp.stinkykitty.org" with baseDomain "stinkykitty.org" returns "testcamp"
func extractSubdomain(host, baseDomain string) string {
	if !strings.HasSuffix(host, "."+baseDomain) {
		return ""
	}

	// Remove base domain and trailing dot
	subdomain := strings.TrimSuffix(host, "."+baseDomain)

	// Check if there are any remaining dots (nested subdomains not supported)
	if strings.Contains(subdomain, ".") {
		return ""
	}

	return subdomain
}

// ClearSiteCache clears the entire site cache (useful for testing)
func ClearSiteCache() {
	siteCache = sync.Map{}
}
