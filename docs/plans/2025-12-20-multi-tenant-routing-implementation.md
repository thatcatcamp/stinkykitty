# Multi-Tenant Routing Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build multi-tenant routing middleware that identifies sites by Host header, with IP filtering and rate limiting for security.

**Architecture:** Gin middleware chain with site resolution (cached lookups), IP filtering (global blocklist + per-site allowlist), and rate limiting (token bucket). Each middleware modular and testable.

**Tech Stack:** Go 1.25+, Gin (HTTP), GORM (ORM), sync.Map (thread-safe cache)

---

## Task 1: Site Resolution Middleware

**Files:**
- Create: `internal/middleware/site.go`
- Create: `internal/middleware/site_test.go`

### Step 1: Write failing test for site resolution

Create `internal/middleware/site_test.go`:

```go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"github.com/thatcatcamp/stinkykitty/internal/models"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	db.AutoMigrate(&models.User{}, &models.Site{})
	return db
}

func TestSiteResolutionBySubdomain(t *testing.T) {
	db := setupTestDB(t)

	// Create test site
	user := models.User{Email: "owner@test.com", PasswordHash: "hash"}
	db.Create(&user)

	site := models.Site{
		Subdomain: "testcamp",
		OwnerID:   user.ID,
		SiteDir:   "/tmp/test",
	}
	db.Create(&site)

	// Create Gin context
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Host = "testcamp.stinkykitty.org"

	// Create middleware
	middleware := SiteResolutionMiddleware(db, "stinkykitty.org")

	// Execute middleware
	middleware(c)

	// Check site was set in context
	siteFromCtx, exists := c.Get("site")
	if !exists {
		t.Fatal("Site not set in context")
	}

	resolvedSite := siteFromCtx.(*models.Site)
	if resolvedSite.Subdomain != "testcamp" {
		t.Errorf("Expected subdomain 'testcamp', got '%s'", resolvedSite.Subdomain)
	}
}

func TestSiteResolutionByCustomDomain(t *testing.T) {
	db := setupTestDB(t)

	user := models.User{Email: "owner@test.com", PasswordHash: "hash"}
	db.Create(&user)

	customDomain := "thatcatcamp.com"
	site := models.Site{
		Subdomain:    "testcamp",
		CustomDomain: &customDomain,
		OwnerID:      user.ID,
		SiteDir:      "/tmp/test",
	}
	db.Create(&site)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Host = "thatcatcamp.com"

	middleware := SiteResolutionMiddleware(db, "stinkykitty.org")
	middleware(c)

	siteFromCtx, exists := c.Get("site")
	if !exists {
		t.Fatal("Site not set in context")
	}

	resolvedSite := siteFromCtx.(*models.Site)
	if resolvedSite.Subdomain != "testcamp" {
		t.Errorf("Expected subdomain 'testcamp', got '%s'", resolvedSite.Subdomain)
	}
}

func TestSiteResolutionNotFound(t *testing.T) {
	db := setupTestDB(t)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Host = "nonexistent.stinkykitty.org"

	middleware := SiteResolutionMiddleware(db, "stinkykitty.org")
	middleware(c)

	// Should return 404
	if w.Code != 404 {
		t.Errorf("Expected 404, got %d", w.Code)
	}

	// Should not set site in context
	_, exists := c.Get("site")
	if exists {
		t.Error("Site should not be set for nonexistent subdomain")
	}
}

func TestSiteResolutionCache(t *testing.T) {
	db := setupTestDB(t)

	user := models.User{Email: "owner@test.com", PasswordHash: "hash"}
	db.Create(&user)

	site := models.Site{
		Subdomain: "testcamp",
		OwnerID:   user.ID,
		SiteDir:   "/tmp/test",
	}
	db.Create(&site)

	// First request - should cache
	gin.SetMode(gin.TestMode)
	w1 := httptest.NewRecorder()
	c1, _ := gin.CreateTestContext(w1)
	c1.Request = httptest.NewRequest("GET", "/", nil)
	c1.Request.Host = "testcamp.stinkykitty.org"

	middleware := SiteResolutionMiddleware(db, "stinkykitty.org")
	middleware(c1)

	// Second request - should use cache
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest("GET", "/", nil)
	c2.Request.Host = "testcamp.stinkykitty.org"

	middleware(c2)

	siteFromCtx, exists := c2.Get("site")
	if !exists {
		t.Fatal("Site not set in context")
	}

	resolvedSite := siteFromCtx.(*models.Site)
	if resolvedSite.Subdomain != "testcamp" {
		t.Errorf("Expected subdomain 'testcamp', got '%s'", resolvedSite.Subdomain)
	}
}
```

### Step 2: Run test to verify it fails

Run: `go test ./internal/middleware -v`

Expected: FAIL with "undefined: SiteResolutionMiddleware"

### Step 3: Implement site resolution middleware

Create `internal/middleware/site.go`:

```go
package middleware

import (
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"github.com/thatcatcamp/stinkykitty/internal/models"
	"github.com/thatcatcamp/stinkykitty/internal/sites"
)

// CacheEntry represents a cached site with expiration
type CacheEntry struct {
	Site      *models.Site
	ExpiresAt time.Time
}

var (
	siteCache   sync.Map
	cacheTTL    = 60 * time.Second
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
```

### Step 4: Run test to verify it passes

Run: `go test ./internal/middleware -v`

Expected: PASS (all 4 tests)

### Step 5: Commit

```bash
git add internal/middleware/
git commit -m "feat: add site resolution middleware with caching

- Resolves sites by subdomain or custom domain
- 60-second TTL cache for performance
- Returns 404 for unknown sites
- Thread-safe using sync.Map
- Comprehensive tests for all lookup paths"
```

---

## Task 2: Add AllowedIPs Field to Site Model

**Files:**
- Modify: `internal/models/models.go`
- Create: `internal/models/migrations.go` (if using migrations)

### Step 1: Add AllowedIPs field to Site model

Modify `internal/models/models.go`, add to Site struct (around line 40):

```go
// Site represents a camp website
type Site struct {
	ID            uint           `gorm:"primaryKey"`
	Subdomain     string         `gorm:"uniqueIndex"`
	CustomDomain  *string        `gorm:"uniqueIndex"`
	OwnerID       uint           `gorm:"not null"`
	SiteDir       string         `gorm:"not null"`
	DatabaseType  string         `gorm:"default:sqlite"`
	DatabasePath  string
	DatabaseHost  string
	DatabaseName  string
	StorageType   string         `gorm:"default:local"`
	S3Bucket      string
	PrimaryColor  string         `gorm:"default:#2563eb"`
	SecondaryColor string        `gorm:"default:#64748b"`
	SiteTitle     string
	SiteTagline   string
	LogoPath      string
	FontPair      string         `gorm:"default:system"`
	AllowedIPs    string         `gorm:"type:text"` // JSON array of CIDR ranges
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`

	// Relationships
	Owner     User       `gorm:"foreignKey:OwnerID"`
	SiteUsers []SiteUser `gorm:"foreignKey:SiteID"`
	Pages     []Page     `gorm:"foreignKey:SiteID"`
}
```

### Step 2: Test database migration

Run: `go run cmd/stinky/main.go config set database.path /tmp/test-migration.db`

Run: `go run cmd/stinky/main.go user create test@example.com` (enter password)

Run: `go run cmd/stinky/main.go site create testsite --owner test@example.com`

Expected: Site created successfully (migration auto-runs)

### Step 3: Verify with sqlite

Run: `sqlite3 ~/.stinkykitty/stinkykitty.db "PRAGMA table_info(sites);" | grep allowed_ips`

Expected: Shows the allowed_ips column

### Step 4: Commit

```bash
git add internal/models/models.go
git commit -m "feat: add AllowedIPs field to Site model

- Add AllowedIPs text field for storing JSON CIDR ranges
- Enables per-site IP allowlist configuration
- Auto-migrates on next server start"
```

---

## Task 3: IP Filtering Middleware

**Files:**
- Create: `internal/middleware/ipfilter.go`
- Create: `internal/middleware/ipfilter_test.go`

### Step 1: Write failing test for IP filtering

Create `internal/middleware/ipfilter_test.go`:

```go
package middleware

import (
	"net/http"
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
```

### Step 2: Run test to verify it fails

Run: `go test ./internal/middleware -v -run TestIPFilter`

Expected: FAIL with "undefined: IPFilterMiddleware"

### Step 3: Implement IP filter middleware

Create `internal/middleware/ipfilter.go`:

```go
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
	remoteAddr := c.Request.RemoteAddr
	if idx := strings.LastIndex(remoteAddr, ":"); idx != -1 {
		remoteAddr = remoteAddr[:idx]
	}

	return net.ParseIP(remoteAddr)
}
```

### Step 4: Run test to verify it passes

Run: `go test ./internal/middleware -v -run TestIPFilter`

Expected: PASS (all 5 IP filter tests)

### Step 5: Commit

```bash
git add internal/middleware/ipfilter.go internal/middleware/ipfilter_test.go
git commit -m "feat: add IP filtering middleware

- Global blocklist from config
- Per-site allowlist from database
- CIDR range matching
- X-Forwarded-For header support
- Returns 403 for blocked IPs
- Comprehensive tests for all scenarios"
```

---

## Task 4: Rate Limiting Middleware

**Files:**
- Create: `internal/middleware/ratelimit.go`
- Create: `internal/middleware/ratelimit_test.go`

### Step 1: Write failing test for rate limiting

Create `internal/middleware/ratelimit_test.go`:

```go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimitAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewRateLimiter(5, time.Minute)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/admin/login", nil)
	c.Request.RemoteAddr = "10.0.0.1:1234"

	middleware := RateLimitMiddleware(limiter, "/admin/login")
	middleware(c)

	if w.Code == 429 {
		t.Error("Expected request to be allowed")
	}
}

func TestRateLimitExceeded(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewRateLimiter(2, time.Minute)

	clientIP := "10.0.0.1:1234"

	// First request - allowed
	w1 := httptest.NewRecorder()
	c1, _ := gin.CreateTestContext(w1)
	c1.Request = httptest.NewRequest("POST", "/admin/login", nil)
	c1.Request.RemoteAddr = clientIP

	middleware := RateLimitMiddleware(limiter, "/admin/login")
	middleware(c1)

	if w1.Code == 429 {
		t.Error("First request should be allowed")
	}

	// Second request - allowed
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest("POST", "/admin/login", nil)
	c2.Request.RemoteAddr = clientIP

	middleware(c2)

	if w2.Code == 429 {
		t.Error("Second request should be allowed")
	}

	// Third request - should be rate limited
	w3 := httptest.NewRecorder()
	c3, _ := gin.CreateTestContext(w3)
	c3.Request = httptest.NewRequest("POST", "/admin/login", nil)
	c3.Request.RemoteAddr = clientIP

	middleware(c3)

	if w3.Code != 429 {
		t.Errorf("Third request should be rate limited, got %d", w3.Code)
	}

	// Check headers
	if w3.Header().Get("X-RateLimit-Limit") != "2" {
		t.Errorf("Expected X-RateLimit-Limit: 2, got %s", w3.Header().Get("X-RateLimit-Limit"))
	}
}

func TestRateLimitDifferentPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewRateLimiter(5, time.Minute)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/admin/dashboard", nil)
	c.Request.RemoteAddr = "10.0.0.1:1234"

	// Middleware only applies to /admin/login
	middleware := RateLimitMiddleware(limiter, "/admin/login")
	middleware(c)

	// Should not be rate limited (different path)
	if w.Code == 429 {
		t.Error("Different path should not be rate limited")
	}
}
```

### Step 2: Run test to verify it fails

Run: `go test ./internal/middleware -v -run TestRateLimit`

Expected: FAIL with "undefined: NewRateLimiter"

### Step 3: Implement rate limiting middleware

Create `internal/middleware/ratelimit.go`:

```go
package middleware

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// TokenBucket implements a token bucket rate limiter
type TokenBucket struct {
	tokens    int
	capacity  int
	refillAt  time.Time
	interval  time.Duration
	mu        sync.Mutex
}

// RateLimiter manages token buckets per IP
type RateLimiter struct {
	mu       sync.RWMutex
	buckets  map[string]*TokenBucket
	capacity int
	interval time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(capacity int, interval time.Duration) *RateLimiter {
	limiter := &RateLimiter{
		buckets:  make(map[string]*TokenBucket),
		capacity: capacity,
		interval: interval,
	}

	// Start cleanup goroutine
	go limiter.cleanup()

	return limiter
}

// cleanup removes old buckets every 5 minutes
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, bucket := range rl.buckets {
			bucket.mu.Lock()
			// Remove buckets not accessed in 10 minutes
			if now.Sub(bucket.refillAt) > 10*time.Minute {
				delete(rl.buckets, ip)
			}
			bucket.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow(ip string) (bool, int) {
	rl.mu.RLock()
	bucket, exists := rl.buckets[ip]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		bucket = &TokenBucket{
			tokens:   rl.capacity,
			capacity: rl.capacity,
			refillAt: time.Now().Add(rl.interval),
			interval: rl.interval,
		}
		rl.buckets[ip] = bucket
		rl.mu.Unlock()
	}

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	// Refill tokens if interval has passed
	now := time.Now()
	if now.After(bucket.refillAt) {
		bucket.tokens = bucket.capacity
		bucket.refillAt = now.Add(bucket.interval)
	}

	// Try to consume a token
	if bucket.tokens > 0 {
		bucket.tokens--
		return true, bucket.tokens
	}

	return false, 0
}

// RateLimitMiddleware creates a rate limiting middleware for specific paths
func RateLimitMiddleware(limiter *RateLimiter, paths ...string) gin.HandlerFunc {
	pathMap := make(map[string]bool)
	for _, path := range paths {
		pathMap[path] = true
	}

	return func(c *gin.Context) {
		// Check if this path requires rate limiting
		if !pathMap[c.Request.URL.Path] {
			c.Next()
			return
		}

		// Extract client IP
		clientIP := getClientIP(c)

		// Check rate limit
		allowed, remaining := limiter.Allow(clientIP)

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.capacity))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

		if !allowed {
			// Calculate retry-after (seconds until next refill)
			retryAfter := int(limiter.interval.Seconds())
			c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
			c.AbortWithStatus(429)
			return
		}

		c.Next()
	}
}

// getClientIP extracts the client IP address
func getClientIP(c *gin.Context) string {
	// Check X-Forwarded-For header
	forwarded := c.GetHeader("X-Forwarded-For")
	if forwarded != "" {
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return host
}
```

### Step 4: Run test to verify it passes

Run: `go test ./internal/middleware -v -run TestRateLimit`

Expected: PASS (all 3 rate limit tests)

### Step 5: Commit

```bash
git add internal/middleware/ratelimit.go internal/middleware/ratelimit_test.go
git commit -m "feat: add rate limiting middleware

- Token bucket algorithm per IP
- Configurable capacity and interval
- Path-specific rate limiting
- Automatic cleanup of old buckets
- Standard rate limit headers (X-RateLimit-*, Retry-After)
- Tests for allowed, exceeded, and path filtering"
```

---

## Task 5: CLI Commands for IP Management

**Files:**
- Modify: `cmd/stinky/site.go`

### Step 1: Add IP management commands to site.go

Add these commands to `cmd/stinky/site.go` (after existing site commands):

```go
var siteAllowIPCmd = &cobra.Command{
	Use:   "allow-ip <subdomain> <cidr>",
	Short: "Add an IP or CIDR range to site's allowlist",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := initSystemDB(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		subdomain := args[0]
		cidr := args[1]

		// Validate CIDR format
		_, _, err := net.ParseCIDR(cidr)
		if err != nil {
			// Try as single IP
			ip := net.ParseIP(cidr)
			if ip == nil {
				fmt.Fprintf(os.Stderr, "Error: invalid CIDR or IP address: %s\n", cidr)
				os.Exit(1)
			}
			// Convert single IP to CIDR
			if strings.Contains(cidr, ":") {
				cidr = cidr + "/128" // IPv6
			} else {
				cidr = cidr + "/32" // IPv4
			}
		}

		site, err := sites.GetSiteBySubdomain(db.GetDB(), subdomain)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Parse existing allowlist
		var allowlist []string
		if site.AllowedIPs != "" {
			if err := json.Unmarshal([]byte(site.AllowedIPs), &allowlist); err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing allowlist: %v\n", err)
				os.Exit(1)
			}
		}

		// Add new CIDR if not already present
		for _, existing := range allowlist {
			if existing == cidr {
				fmt.Printf("IP range %s already in allowlist for %s\n", cidr, subdomain)
				return
			}
		}

		allowlist = append(allowlist, cidr)

		// Marshal back to JSON
		allowedIPsJSON, err := json.Marshal(allowlist)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		site.AllowedIPs = string(allowedIPsJSON)
		if err := db.GetDB().Save(site).Error; err != nil {
			fmt.Fprintf(os.Stderr, "Error saving site: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Added %s to allowlist for %s\n", cidr, subdomain)
	},
}

var siteRemoveAllowedIPCmd = &cobra.Command{
	Use:   "remove-allowed-ip <subdomain> <cidr>",
	Short: "Remove an IP or CIDR range from site's allowlist",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := initSystemDB(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		subdomain := args[0]
		cidr := args[1]

		site, err := sites.GetSiteBySubdomain(db.GetDB(), subdomain)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if site.AllowedIPs == "" {
			fmt.Printf("No allowlist configured for %s\n", subdomain)
			return
		}

		var allowlist []string
		if err := json.Unmarshal([]byte(site.AllowedIPs), &allowlist); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing allowlist: %v\n", err)
			os.Exit(1)
		}

		// Remove CIDR
		newAllowlist := []string{}
		found := false
		for _, existing := range allowlist {
			if existing == cidr {
				found = true
				continue
			}
			newAllowlist = append(newAllowlist, existing)
		}

		if !found {
			fmt.Printf("IP range %s not in allowlist for %s\n", cidr, subdomain)
			return
		}

		// Update site
		if len(newAllowlist) == 0 {
			site.AllowedIPs = ""
		} else {
			allowedIPsJSON, err := json.Marshal(newAllowlist)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			site.AllowedIPs = string(allowedIPsJSON)
		}

		if err := db.GetDB().Save(site).Error; err != nil {
			fmt.Fprintf(os.Stderr, "Error saving site: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Removed %s from allowlist for %s\n", cidr, subdomain)
	},
}

var siteListAllowedIPsCmd = &cobra.Command{
	Use:   "list-allowed-ips <subdomain>",
	Short: "List all allowed IPs for a site",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := initSystemDB(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		subdomain := args[0]

		site, err := sites.GetSiteBySubdomain(db.GetDB(), subdomain)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if site.AllowedIPs == "" {
			fmt.Printf("No IP allowlist configured for %s\n", subdomain)
			fmt.Println("All IPs are allowed (only global blocklist applies)")
			return
		}

		var allowlist []string
		if err := json.Unmarshal([]byte(site.AllowedIPs), &allowlist); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing allowlist: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Allowed IPs for %s:\n", subdomain)
		for _, cidr := range allowlist {
			fmt.Printf("  - %s\n", cidr)
		}
	},
}

// In init(), add these commands:
func init() {
	// ... existing code ...

	siteCmd.AddCommand(siteAllowIPCmd)
	siteCmd.AddCommand(siteRemoveAllowedIPCmd)
	siteCmd.AddCommand(siteListAllowedIPsCmd)

	// ... rest of init ...
}
```

Add imports to `cmd/stinky/site.go`:

```go
import (
	"encoding/json"
	"net"
	// ... existing imports ...
)
```

### Step 2: Test CLI commands manually

Run: `go run cmd/stinky/main.go site allow-ip testcamp 192.168.1.0/24`

Expected: "Added 192.168.1.0/24 to allowlist for testcamp"

Run: `go run cmd/stinky/main.go site list-allowed-ips testcamp`

Expected: Shows the allowed IP range

Run: `go run cmd/stinky/main.go site remove-allowed-ip testcamp 192.168.1.0/24`

Expected: "Removed 192.168.1.0/24 from allowlist for testcamp"

### Step 3: Commit

```bash
git add cmd/stinky/site.go
git commit -m "feat: add CLI commands for IP allowlist management

- site allow-ip: Add IP/CIDR to allowlist
- site remove-allowed-ip: Remove IP/CIDR from allowlist
- site list-allowed-ips: Show all allowed IPs
- Validates CIDR format
- Auto-converts single IPs to CIDR notation
- Stores as JSON array in database"
```

---

## Task 6: Update Server with Middleware Chain

**Files:**
- Modify: `cmd/stinky/server.go`
- Create: `internal/handlers/public.go`

### Step 1: Create public handler for testing

Create `internal/handlers/public.go`:

```go
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
```

### Step 2: Update server.go with middleware chain

Modify `cmd/stinky/server.go`:

```go
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/thatcatcamp/stinkykitty/internal/config"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/handlers"
	"github.com/thatcatcamp/stinkykitty/internal/middleware"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Server operations",
	Long:  "Start, stop, and manage the StinkyKitty HTTP server",
}

var serverStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the HTTP server",
	Run: func(cmd *cobra.Command, args []string) {
		if err := initConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := initSystemDB(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Create Gin router
		r := gin.Default()

		// System routes (no site context needed)
		r.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":  "ok",
				"service": "stinkykitty",
			})
		})

		// Get base domain from config (default to localhost for development)
		baseDomain := config.GetString("server.base_domain")
		if baseDomain == "" {
			baseDomain = "localhost"
		}

		// Get global IP blocklist from config
		var blocklist []string
		// TODO: Load from config when we add security.blocked_ips to config schema

		// Create rate limiter for admin routes
		loginRateLimiter := middleware.NewRateLimiter(5, time.Minute)

		// Site-required routes
		siteGroup := r.Group("/")
		siteGroup.Use(middleware.SiteResolutionMiddleware(db.GetDB(), baseDomain))
		{
			// Public content routes
			siteGroup.GET("/", handlers.ServeHomepage)

			// Admin routes
			adminGroup := siteGroup.Group("/admin")
			adminGroup.Use(
				middleware.IPFilterMiddleware(blocklist),
				middleware.RateLimitMiddleware(loginRateLimiter, "/admin/login"),
			)
			{
				adminGroup.POST("/login", func(c *gin.Context) {
					c.String(200, "Admin login placeholder")
				})
				adminGroup.GET("/dashboard", func(c *gin.Context) {
					c.String(200, "Admin dashboard placeholder")
				})
			}
		}

		httpPort := config.GetString("server.http_port")
		addr := fmt.Sprintf(":%s", httpPort)

		fmt.Printf("Starting StinkyKitty server on %s\n", addr)
		fmt.Printf("Base domain: %s\n", baseDomain)
		if err := r.Run(addr); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	serverCmd.AddCommand(serverStartCmd)
	rootCmd.AddCommand(serverCmd)
}
```

### Step 3: Add base_domain to config defaults

Modify `internal/config/config.go`, add to `setDefaults()`:

```go
// Server defaults
v.SetDefault("server.http_port", "80")
v.SetDefault("server.https_port", "443")
v.SetDefault("server.behind_proxy", false)
v.SetDefault("server.base_domain", "localhost")  // Add this line
```

### Step 4: Test end-to-end

Run: `go run cmd/stinky/main.go server start`

In another terminal:
```bash
# Test health check (no site needed)
curl http://localhost:8080/health

# Test site resolution with subdomain
# First create a test site
go run cmd/stinky/main.go site create testcamp --owner <your-test-user-email>

# Test with subdomain (requires /etc/hosts entry or DNS)
# Add to /etc/hosts: 127.0.0.1 testcamp.localhost
curl http://testcamp.localhost:8080/

# Test 404 for unknown site
curl http://unknown.localhost:8080/

# Test admin route (placeholder)
curl http://testcamp.localhost:8080/admin/dashboard
```

Expected:
- Health check returns JSON
- Homepage shows site info
- Unknown site returns 404
- Admin route returns placeholder text

### Step 5: Commit

```bash
git add cmd/stinky/server.go internal/handlers/ internal/config/config.go
git commit -m "feat: integrate multi-tenant routing into server

- Wire up site resolution, IP filtering, and rate limiting middleware
- Add public homepage handler (placeholder)
- Add admin route placeholders
- Configure base domain from config
- Full middleware chain: logger → recovery → site → IP filter → rate limit
- Health check route bypasses site resolution
- Ready for content block system"
```

---

## Summary

This plan implements the complete multi-tenant routing system:

1. **Site Resolution** - 60s cached lookups by subdomain or custom domain
2. **IP Filtering** - Global blocklist + per-site allowlists
3. **Rate Limiting** - Token bucket per IP for admin endpoints
4. **CLI Management** - Commands to configure IP allowlists
5. **Server Integration** - Full middleware chain with route structure

**Next phases** will build:
- Content block system (hero, text, gallery, video, button)
- Page editor API
- Authentication with JWT
- Admin web UI

**Testing approach:**
- Unit tests for each middleware component
- Integration tests for middleware chain
- Manual CLI testing for IP management
- End-to-end server testing

**Development principles:**
- TDD: Tests before implementation
- DRY: Reusable middleware functions
- YAGNI: No premature optimization
- Small commits: One feature at a time
