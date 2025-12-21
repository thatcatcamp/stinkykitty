# Multi-Tenant Routing System Design

**Date:** 2025-12-20
**Status:** Approved for Implementation

## Overview

StinkyKitty's multi-tenant routing system enables a single server instance to host multiple camp websites, each identified by their domain (subdomain or custom). The system uses Gin middleware to resolve sites, enforce security policies, and route requests appropriately.

## Architecture

### Core Components

1. **Site Resolution Middleware** - Examines the `Host` header, queries the database for a matching site, and stores the site in the Gin context. Uses an in-memory cache with 60-second TTL.

2. **IP Filtering Middleware** - Protects `/admin/*` routes with two-tier filtering:
   - Global blocklist (system config) blocks known bad actors
   - Per-site allowlists (optional, in Site model) restrict to specific IPs

3. **Rate Limiting Middleware** - Protects authentication endpoints from brute force with strict limits (5 attempts/minute per IP).

4. **Route Structure**:
   - System routes: `/health` (no site needed)
   - Admin routes: `/admin/*` (requires site context + IP filtering + rate limiting)
   - Public routes: `/*` (serves site content)

### Security Posture

- Unknown sites return 404 (not advertising the CMS)
- IP filtering stops known bad actors
- Rate limiting prevents brute force attacks
- Two-tier security (global + per-site) gives flexibility

## Site Resolution Middleware

### How It Works

1. **Extract Host header** from incoming request (e.g., `thatcatcamp.com` or `mycamp.stinkykitty.org`)

2. **Check cache** - Look in in-memory cache with 60-second TTL
   - Cache hit: Use cached site, continue request
   - Cache miss: Query database

3. **Database lookup**:
   ```go
   // First try custom domain
   site := GetSiteByDomain(host)
   if site == nil {
       // Extract subdomain and try that
       subdomain := extractSubdomain(host, baseDomain)
       site = GetSiteBySubdomain(subdomain)
   }
   ```

4. **Store in context or return 404**:
   - Found: `c.Set("site", site)` and continue to next handler
   - Not found: `c.AbortWithStatus(404)` - silent 404

### Cache Implementation

```go
type CacheEntry struct {
    Site      *models.Site
    ExpiresAt time.Time
}

var siteCache = sync.Map{} // thread-safe map
```

**Benefits:**
- Fast lookups after first request (no DB query)
- New domains work within 60 seconds
- Stale cache auto-expires
- Thread-safe for concurrent requests

## IP Filtering Middleware

### Two-Tier IP Protection

**Tier 1 - Global Blocklist (System Config)**
```yaml
# config.yaml
security:
  blocked_ips:
    - "198.18.0.0/15"      # Example: known bad range
    - "45.142.120.0/22"    # Alibaba Cloud range
  blocked_cidrs:
    - "185.220.100.0/22"   # Tor exit nodes (optional)
```

Checked first - if IP matches any blocked range, immediate 403 Forbidden.

**Tier 2 - Per-Site Allowlist (Site Model)**
```go
// Add to models.Site
AllowedIPs  string  `gorm:"type:text"` // JSON array: ["192.168.1.0/24", "1.2.3.4"]
```

If site has an allowlist configured:
- IP in allowlist: Allow access
- IP not in allowlist: 403 Forbidden

If site has no allowlist: Allow access (only global blocklist applies)

### Implementation Flow

```go
1. Extract client IP (handle X-Forwarded-For if behind proxy)
2. Check global blocklist → 403 if matched
3. Load site from context (set by site resolution middleware)
4. If site.AllowedIPs is set:
   - Parse JSON array
   - Check if client IP in allowlist → 403 if not
5. Continue to next handler
```

### CLI Commands

```bash
stinky site allow-ip mycamp 192.168.1.0/24
stinky site list-allowed-ips mycamp
stinky site remove-allowed-ip mycamp 192.168.1.0/24
```

**Benefits:**
- Centralized protection against known bad actors
- Per-site lockdown for camps wanting extra security
- Easy to script and automate

## Rate Limiting Middleware

### Route-Specific Rate Limiting

Protects authentication endpoints from brute force attacks using in-memory token bucket per IP address.

**Rate limit rules:**
```yaml
# config.yaml
security:
  rate_limits:
    admin_login: "5/minute"    # 5 attempts per minute per IP
    admin_auth: "10/minute"    # 10 API auth calls per minute
```

### Token Bucket Structure

```go
type RateLimiter struct {
    mu      sync.RWMutex
    buckets map[string]*TokenBucket  // key: IP address
}

type TokenBucket struct {
    tokens    int
    capacity  int
    refillAt  time.Time
    interval  time.Duration
}
```

### Middleware Flow

```go
1. Check if route requires rate limiting (/admin/login, etc.)
2. Extract client IP
3. Get or create token bucket for this IP
4. Try to consume 1 token:
   - Token available: Continue request
   - No tokens: Return 429 Too Many Requests with Retry-After header
5. Refill tokens based on time interval
```

**Cleanup:** Background goroutine runs every 5 minutes to remove expired buckets (not accessed in 10 minutes).

**Headers returned on rate limit:**
```
HTTP/1.1 429 Too Many Requests
Retry-After: 45
X-RateLimit-Limit: 5
X-RateLimit-Remaining: 0
```

**Benefits:**
- Stops brute force without slowing legitimate users
- Automatic cleanup prevents memory leaks
- Standard HTTP headers for client feedback

## Middleware Chain & Route Structure

### Middleware Execution Order

```go
1. Gin Logger (request logging)
2. Gin Recovery (panic recovery)
3. Site Resolution (all routes except /health)
4. IP Filtering (only /admin/* routes)
5. Rate Limiting (only /admin/login, /admin/api/auth/*)
6. Authentication (only /admin/* routes - future)
7. Route Handler
```

### Route Structure

```go
// System routes (no site context needed)
r.GET("/health", healthHandler)

// Site-required routes
siteGroup := r.Group("/")
siteGroup.Use(SiteResolutionMiddleware())
{
    // Public content routes
    siteGroup.GET("/", serveHomepage)
    siteGroup.GET("/:slug", servePageBySlug)

    // Admin routes
    adminGroup := siteGroup.Group("/admin")
    adminGroup.Use(IPFilterMiddleware(), RateLimitMiddleware())
    {
        adminGroup.POST("/login", adminLoginHandler)  // Rate limited
        adminGroup.GET("/dashboard", adminDashboard)  // Future
        // ... more admin routes
    }
}
```

### Error Responses

- Site not found: `404` (silent)
- IP blocked: `403 Forbidden`
- Rate limited: `429 Too Many Requests` with Retry-After
- Not authenticated: `401 Unauthorized` (future)

## Package Structure

```
internal/
├── middleware/
│   ├── site.go          # Site resolution + cache
│   ├── ipfilter.go      # IP allow/block lists
│   └── ratelimit.go     # Token bucket rate limiter
├── handlers/
│   ├── public.go        # Public site content handlers
│   └── admin.go         # Admin panel handlers
└── ... (existing packages)
```

## Database Schema Changes

### Site Model Addition

```go
// Add to models.Site
AllowedIPs  string  `gorm:"type:text"` // JSON array of CIDR ranges
```

### Configuration Schema

```yaml
# config.yaml additions
security:
  blocked_ips:
    - "198.18.0.0/15"
  rate_limits:
    admin_login: "5/minute"
    admin_auth: "10/minute"
```

## Testing Strategy

### Unit Tests

1. **Site Resolution**
   - Test cache hit/miss
   - Test subdomain extraction
   - Test custom domain lookup
   - Test 404 for unknown sites

2. **IP Filtering**
   - Test global blocklist matching
   - Test per-site allowlist enforcement
   - Test X-Forwarded-For header parsing
   - Test CIDR range matching

3. **Rate Limiting**
   - Test token consumption
   - Test refill intervals
   - Test 429 responses
   - Test cleanup of old buckets

### Integration Tests

1. Request flow through full middleware chain
2. Multiple concurrent requests to same site
3. Cache expiration and refresh
4. Rate limit enforcement across multiple IPs

### Manual Testing

1. Create test site via CLI
2. Access via subdomain
3. Add custom domain and access
4. Test IP blocking
5. Test rate limiting by hammering login endpoint

## Implementation Phases

### Phase 1: Site Resolution
- Site resolution middleware
- In-memory cache with TTL
- Integration with existing server.go

### Phase 2: IP Filtering
- Global blocklist from config
- Per-site allowlist support
- CLI commands for IP management
- Add AllowedIPs field to Site model

### Phase 3: Rate Limiting
- Token bucket implementation
- Rate limit middleware
- Configuration support
- Background cleanup goroutine

### Phase 4: Integration & Testing
- Wire up all middleware
- Update server.go with route structure
- Comprehensive testing
- Documentation

## Security Considerations

1. **Cache Poisoning:** Cache keys based on lowercased host to prevent case variations
2. **IP Spoofing:** Validate X-Forwarded-For only when behind_proxy config is true
3. **Memory Leaks:** Cleanup goroutines for expired cache entries and rate limit buckets
4. **DoS via Cache:** Limit cache size to prevent memory exhaustion
5. **CIDR Validation:** Validate IP ranges in config and allowlist to prevent malformed entries

## Performance Considerations

1. **Cache Hit Rate:** Expected >95% for established sites (60s TTL)
2. **Database Queries:** Only on cache miss or new sites
3. **Middleware Overhead:** <1ms per request for cache hits
4. **Memory Usage:** ~100 bytes per cached site, ~200 bytes per rate-limited IP
5. **Concurrent Requests:** sync.Map and sync.RWMutex for thread safety

## Future Enhancements

1. **Redis Cache:** Replace in-memory cache with Redis for multi-server deployments
2. **Distributed Rate Limiting:** Use Redis for rate limiting across multiple servers
3. **Geo-IP Blocking:** Block entire countries if needed
4. **Request Logging:** Log blocked requests for security monitoring
5. **Metrics:** Prometheus metrics for cache hit rate, rate limits, IP blocks
6. **Dynamic Config Reload:** Hot-reload security config without restart

## Success Criteria

- Multiple sites accessible via subdomain and custom domain
- <1ms latency for cached site lookups
- Rate limiting prevents brute force (>5 attempts/minute blocked)
- IP filtering blocks known bad actors
- 404 for unknown sites (no information disclosure)
- Easy CLI management of IP allowlists
