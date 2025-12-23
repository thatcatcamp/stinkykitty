# Automatic SSL/TLS with Let's Encrypt - Design Document

**Date:** 2025-12-23
**Status:** Approved
**Goal:** Add automatic HTTPS support with Let's Encrypt certificate provisioning

## Overview

Enable StinkyKitty to automatically obtain and renew SSL/TLS certificates from Let's Encrypt for all hosted domains (base domain subdomains and custom domains). The system will run as a self-contained appliance handling both HTTP (port 80) and HTTPS (port 443) directly.

## Architecture

### Components

**1. TLS Manager (`internal/tls/`)**
- Wraps `certmagic` library for ACME protocol automation
- Manages domain allowlist from database
- Handles certificate storage and retrieval
- Provides TLS config for HTTPS server

**2. Dual Server Architecture**
- **HTTP Server (port 80):** ACME HTTP-01 challenges + redirect to HTTPS
- **HTTPS Server (port 443):** Main application traffic with TLS
- Both servers share same Gin router and handlers

**3. Certificate Storage**
- Certificates cached to `/var/lib/stinkykitty/certs/` (configurable)
- Certmagic handles disk format and locking
- Automatic renewal 30 days before expiry

## Configuration

### New Config Values

```yaml
server:
  tls_enabled: true                    # Enable HTTPS server
  http_port: 80                        # HTTP port (existing)
  https_port: 443                      # HTTPS port (existing)
  base_domain: "campasaur.us"         # For ACME registration (existing)

tls:
  email: "admin@campasaur.us"         # Required by Let's Encrypt
  cert_dir: "/var/lib/stinkykitty/certs"
  staging: false                       # Use Let's Encrypt staging for testing
```

### Environment Variables

- `STINKY_TLS_EMAIL` - Override TLS email address
- `STINKY_TLS_STAGING` - Enable staging mode (true/false)

## Domain Management

### Certificate Provisioning

**Domains that get certificates:**

1. **Base domain** (`campasaur.us`) - Always included when TLS enabled
2. **Site subdomains** (`mycamp.campasaur.us`) - Auto-provisioned on first HTTPS request
3. **Custom domains** (`mycamp.com`) - Auto-provisioned when added via CLI

**Provisioning Flow:**

1. Server startup: Load all domains from database into allowlist
2. First HTTPS request to `mycamp.campasaur.us`:
   - Certmagic checks for cached certificate
   - If none exists, triggers ACME HTTP-01 challenge via port 80
   - Let's Encrypt verifies ownership by fetching challenge from `http://mycamp.campasaur.us/.well-known/acme-challenge/TOKEN`
   - Certificate issued and cached to disk
   - Subsequent requests use cached certificate
3. Automatic renewal happens in background before expiry

### Domain Allowlist

**Purpose:** Prevent certificate provisioning for random domains pointing at server

**Implementation:**
- Query database on startup for all valid domains:
  - Base domain from config
  - `{subdomain}.{base_domain}` for all sites
  - All `custom_domain` values from sites table
- Pass allowlist to certmagic
- Only allowed domains can trigger certificate provisioning

## Error Handling

### Challenge Failures

**Scenario:** DNS not pointed correctly, port 80 blocked, firewall issues

**Behavior:**
- Certmagic retries with exponential backoff
- Falls back to self-signed certificate (browser warning, site still loads)
- Logs clear error: `Failed to obtain certificate for mycamp.campasaur.us: ACME challenge failed`
- Admin can check logs and verify DNS/firewall configuration

### Rate Limit Protection

**Let's Encrypt Limits:**
- 50 certificates per registered domain per week
- 5 failed validations per account per hostname per hour

**Protection:**
- Use `staging: true` for development/testing (unlimited rate limits)
- Certmagic caches valid certificates, won't re-request unnecessarily
- If rate limit hit, certmagic backs off automatically

### Startup Safety

**Scenario:** TLS misconfiguration or Let's Encrypt unavailable

**Behavior:**
- HTTP server (port 80) always starts successfully
- HTTPS server logs error but doesn't crash process
- Sites remain accessible via HTTP while configuration is fixed
- Clear error message with remediation steps

## Implementation Structure

### New Package: `internal/tls/`

```
internal/tls/
├── manager.go       # Certmagic setup, domain management
├── config.go        # TLS configuration helpers
└── status.go        # Certificate status inspection
```

### Key Functions

**manager.go:**
```go
type Manager struct {
    certmagic *certmagic.Config
    db        *gorm.DB
    email     string
    certDir   string
    staging   bool
}

// Initialize certmagic with storage, email, staging mode
func NewManager(db *gorm.DB, cfg *Config) (*Manager, error)

// Query database for all domains to serve
// Returns: [base_domain, site1.base_domain, site2.base_domain, custom.com]
func (m *Manager) GetAllowedDomains() ([]string, error)

// Returns TLS config for HTTPS server with certmagic GetCertificate callback
func (m *Manager) GetTLSConfig() *tls.Config

// Refresh allowlist when sites are created/domains added
func (m *Manager) RefreshDomains() error
```

**config.go:**
```go
type Config struct {
    Email     string
    CertDir   string
    Staging   bool
    BaseDomain string
}

// Load TLS config from viper
func LoadConfig() (*Config, error)
```

**status.go:**
```go
type CertificateStatus struct {
    Domain    string
    Issuer    string
    NotBefore time.Time
    NotAfter  time.Time
    DaysUntilExpiry int
}

// Get status of all managed certificates
func (m *Manager) GetCertificateStatus() ([]CertificateStatus, error)
```

### Server Changes: `cmd/stinky/server.go`

```go
func startServer() error {
    // Initialize database and config (existing code)

    // Create Gin router (existing code)
    r := setupRouter()

    // Check if TLS is enabled
    if config.GetBool("server.tls_enabled") {
        // Initialize TLS manager
        tlsCfg, err := tls.LoadConfig()
        if err != nil {
            return fmt.Errorf("failed to load TLS config: %w", err)
        }

        tlsManager, err := tls.NewManager(db.GetDB(), tlsCfg)
        if err != nil {
            return fmt.Errorf("failed to initialize TLS manager: %w", err)
        }

        // Start HTTP server (port 80) for ACME challenges + redirects
        httpAddr := fmt.Sprintf(":%s", config.GetString("server.http_port"))
        go func() {
            log.Printf("Starting HTTP server on %s (ACME challenges + redirects)\n", httpAddr)
            if err := http.ListenAndServe(httpAddr, r); err != nil {
                log.Fatalf("HTTP server failed: %v", err)
            }
        }()

        // Start HTTPS server (port 443) for main traffic
        httpsAddr := fmt.Sprintf(":%s", config.GetString("server.https_port"))
        log.Printf("Starting HTTPS server on %s\n", httpsAddr)

        server := &http.Server{
            Addr:      httpsAddr,
            Handler:   r,
            TLSConfig: tlsManager.GetTLSConfig(),
        }

        return server.ListenAndServeTLS("", "") // Certs handled by TLSConfig
    } else {
        // Dev mode - HTTP only (existing behavior)
        httpAddr := fmt.Sprintf(":%s", config.GetString("server.http_port"))
        log.Printf("Starting HTTP server on %s (TLS disabled)\n", httpAddr)
        return r.Run(httpAddr)
    }
}
```

### HTTP to HTTPS Redirect Middleware

```go
// middleware/https_redirect.go
func HTTPSRedirectMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Skip if already HTTPS
        if c.Request.TLS != nil {
            c.Next()
            return
        }

        // Skip for ACME challenges
        if strings.HasPrefix(c.Request.URL.Path, "/.well-known/acme-challenge/") {
            c.Next()
            return
        }

        // Redirect to HTTPS
        httpsURL := "https://" + c.Request.Host + c.Request.RequestURI
        c.Redirect(http.StatusMovedPermanently, httpsURL)
        c.Abort()
    }
}
```

## CLI Commands

### Certificate Status

```bash
stinky tls status
```

**Output:**
```
Domain                      Issuer              Expires          Days Left
campasaur.us               Let's Encrypt        2025-03-15       82
mycamp.campasaur.us        Let's Encrypt        2025-03-20       87
customdomain.com           Let's Encrypt        2025-02-28       67
```

### Force Certificate Renewal (Optional)

```bash
stinky tls renew <domain>
```

Forces immediate renewal of certificate for specified domain.

## Dependencies

### New Go Modules

```go
require (
    github.com/caddyserver/certmagic v0.20.0
    github.com/libdns/libdns v0.2.1  // Required by certmagic
)
```

## Testing Strategy

### Development Testing

1. **Local Testing:** Use `staging: true` to avoid rate limits
2. **Test Domains:** Create test subdomains pointing to dev server
3. **Validation:** Verify certificates issued by Let's Encrypt staging

### Staging Environment

1. Enable staging mode in config
2. Test certificate provisioning for:
   - Base domain
   - Site subdomains
   - Custom domains
3. Verify automatic renewal (certmagic has test mode for this)

### Production Rollout

1. Deploy with `tls_enabled: false` initially
2. Verify HTTP server working correctly
3. Enable TLS with `staging: true` to test without rate limits
4. Switch to `staging: false` for production certificates
5. Monitor logs for certificate provisioning success

## Security Considerations

### Private Key Storage

- Private keys stored in `/var/lib/stinkykitty/certs/`
- Directory permissions: `0700` (owner only)
- Certmagic handles key generation and storage securely

### ACME Account Security

- Account private key stored by certmagic
- Email address registered with Let's Encrypt for expiry notifications
- No credentials stored in config (account key automatically generated)

### Domain Validation

- Allowlist prevents unauthorized certificate provisioning
- Only domains in database can get certificates
- Prevents abuse if random DNS points at server

## Monitoring & Operations

### Logging

**Certificate Events:**
- `Certificate obtained for domain: mycamp.campasaur.us`
- `Certificate renewed for domain: mycamp.campasaur.us (30 days before expiry)`
- `Certificate provisioning failed for domain: mycamp.com - DNS not configured`

**Error Logging:**
- ACME challenge failures with remediation steps
- Rate limit warnings
- Storage errors

### Metrics (Future)

Potential metrics to expose:
- Number of active certificates
- Days until next renewal
- Failed provisioning attempts
- Certificate age

## Future Enhancements

### Phase 2 (Post-MVP)

1. **Wildcard Certificates:** Add DNS-01 challenge support for `*.campasaur.us`
2. **Multi-Provider Support:** Support DNS APIs (Cloudflare, Route53, etc.)
3. **Certificate Dashboard:** Web UI showing certificate status
4. **Alerts:** Email notifications for expiring or failed certificates
5. **OCSP Stapling:** Improve TLS handshake performance

## Success Criteria

1. ✅ Server starts with both HTTP (80) and HTTPS (443) listeners
2. ✅ First HTTPS request to campasaur.us obtains valid certificate
3. ✅ Site subdomains automatically get certificates on first request
4. ✅ Custom domains get certificates when added via CLI
5. ✅ Certificates automatically renew before expiry
6. ✅ HTTP requests redirect to HTTPS (except ACME challenges)
7. ✅ Clear error messages when certificate provisioning fails
8. ✅ `stinky tls status` shows certificate information
9. ✅ Staging mode works for testing without rate limits
10. ✅ HTTP server remains functional if TLS fails

## Migration Path

### Existing Deployments

For servers currently running HTTP-only:

1. Update StinkyKitty binary
2. Add TLS configuration to config file
3. Restart server (will start in HTTP-only mode if `tls_enabled: false`)
4. Enable TLS when ready: `stinky config set server.tls_enabled true`
5. Restart server to activate HTTPS

### Zero-Downtime Approach

1. Keep existing HTTP server running
2. Start new instance with TLS enabled on different port temporarily
3. Test HTTPS works correctly
4. Switch traffic to new instance
5. Shutdown old instance

## References

- [Let's Encrypt Documentation](https://letsencrypt.org/docs/)
- [Certmagic Library](https://github.com/caddyserver/certmagic)
- [ACME Protocol (RFC 8555)](https://datatracker.ietf.org/doc/html/rfc8555)
- [Let's Encrypt Rate Limits](https://letsencrypt.org/docs/rate-limits/)
