package middleware

import (
	"encoding/json"
	"net"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/models"
)

// IPFilterMiddleware blocks requests based on IP address
// Uses global blocklist and optional per-site allowlist
func IPFilterMiddleware(globalBlocklist []string) gin.HandlerFunc {
	// Parse global blocklist into CIDR ranges
	blockedCIDRs := make([]*net.IPNet, 0, len(globalBlocklist))
	for _, cidr := range globalBlocklist {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err == nil {
			blockedCIDRs = append(blockedCIDRs, ipNet)
		}
	}

	return func(c *gin.Context) {
		// Extract client IP
		clientIP := extractIP(c)
		if clientIP == nil {
			c.AbortWithStatus(403)
			return
		}

		// Check global blocklist
		for _, ipNet := range blockedCIDRs {
			if ipNet.Contains(clientIP) {
				c.AbortWithStatus(403)
				return
			}
		}

		// Get site from context
		siteVal, exists := c.Get("site")
		if !exists {
			// No site context, allow (for system routes)
			c.Next()
			return
		}

		site := siteVal.(*models.Site)

		// If site has allowlist, enforce it
		if site.AllowedIPs != "" {
			var allowedRanges []string
			if err := json.Unmarshal([]byte(site.AllowedIPs), &allowedRanges); err != nil {
				// Invalid JSON, block access
				c.AbortWithStatus(403)
				return
			}

			// Check if client IP is in allowlist
			allowed := false
			for _, cidr := range allowedRanges {
				_, ipNet, err := net.ParseCIDR(cidr)
				if err != nil {
					continue
				}
				if ipNet.Contains(clientIP) {
					allowed = true
					break
				}
			}

			if !allowed {
				c.AbortWithStatus(403)
				return
			}
		}

		c.Next()
	}
}

// extractIP extracts the client IP from the request
// Handles X-Forwarded-For header if behind proxy
func extractIP(c *gin.Context) net.IP {
	// Check X-Forwarded-For header (if behind proxy)
	forwarded := c.GetHeader("X-Forwarded-For")
	if forwarded != "" {
		// Take first IP in the list
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			return net.ParseIP(ip)
		}
	}

	// Fall back to RemoteAddr
	// Use SplitHostPort to properly handle IPv6 addresses with brackets
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		// If no port, use the whole string
		host = c.Request.RemoteAddr
	}

	return net.ParseIP(host)
}
