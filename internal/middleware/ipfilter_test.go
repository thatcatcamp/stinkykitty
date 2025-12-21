package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/models"
)

func TestIPFilterGlobalBlocklist(t *testing.T) {
	gin.SetMode(gin.TestMode)

	blocklist := []string{"192.168.1.0/24"}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/admin", nil)
	c.Request.RemoteAddr = "192.168.1.100:1234"

	middleware := IPFilterMiddleware(blocklist)
	middleware(c)

	if w.Code != 403 {
		t.Errorf("Expected 403 for blocked IP, got %d", w.Code)
	}
}

func TestIPFilterGlobalBlocklistAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	blocklist := []string{"192.168.1.0/24"}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/admin", nil)
	c.Request.RemoteAddr = "10.0.0.1:1234"

	// Set a dummy site in context
	site := &models.Site{Subdomain: "test"}
	c.Set("site", site)

	middleware := IPFilterMiddleware(blocklist)
	middleware(c)

	if w.Code == 403 {
		t.Error("Expected allowed for non-blocked IP")
	}
}

func TestIPFilterPerSiteAllowlist(t *testing.T) {
	gin.SetMode(gin.TestMode)

	blocklist := []string{}

	// Site with allowlist
	site := &models.Site{
		Subdomain:  "test",
		AllowedIPs: `["10.0.0.0/24"]`, // JSON array
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/admin", nil)
	c.Request.RemoteAddr = "192.168.1.1:1234" // Not in allowlist
	c.Set("site", site)

	middleware := IPFilterMiddleware(blocklist)
	middleware(c)

	if w.Code != 403 {
		t.Errorf("Expected 403 for IP not in allowlist, got %d", w.Code)
	}
}

func TestIPFilterPerSiteAllowlistAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	blocklist := []string{}

	site := &models.Site{
		Subdomain:  "test",
		AllowedIPs: `["10.0.0.0/24"]`,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/admin", nil)
	c.Request.RemoteAddr = "10.0.0.100:1234" // In allowlist
	c.Set("site", site)

	middleware := IPFilterMiddleware(blocklist)
	middleware(c)

	if w.Code == 403 {
		t.Error("Expected allowed for IP in allowlist")
	}
}

func TestIPFilterNoAllowlist(t *testing.T) {
	gin.SetMode(gin.TestMode)

	blocklist := []string{}

	// Site without allowlist
	site := &models.Site{
		Subdomain: "test",
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/admin", nil)
	c.Request.RemoteAddr = "192.168.1.1:1234"
	c.Set("site", site)

	middleware := IPFilterMiddleware(blocklist)
	middleware(c)

	if w.Code == 403 {
		t.Error("Expected allowed when no allowlist configured")
	}
}
