# Automatic SSL/TLS Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add automatic HTTPS support with Let's Encrypt certificate provisioning using certmagic library.

**Architecture:** Dual HTTP/HTTPS server architecture where HTTP (port 80) handles ACME challenges and redirects, while HTTPS (port 443) serves main traffic. Certmagic library manages certificate provisioning, storage, and renewal automatically.

**Tech Stack:** Go 1.21+, certmagic v0.20.0, existing Gin framework

---

## Task 1: Add TLS Configuration Defaults

**Files:**
- Modify: `internal/config/config.go`

**Step 1: Add TLS config defaults to setDefaults function**

In `internal/config/config.go`, add TLS defaults after the auth defaults:

```go
// TLS defaults
v.SetDefault("server.tls_enabled", false)
v.SetDefault("tls.email", "")
v.SetDefault("tls.cert_dir", "/var/lib/stinkykitty/certs")
v.SetDefault("tls.staging", false)
```

**Step 2: Verify config loads correctly**

Run:
```bash
go run cmd/stinky/main.go config get server.tls_enabled
```

Expected: `false`

**Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "feat: add TLS configuration defaults"
```

---

## Task 2: Add Certmagic Dependency

**Files:**
- Modify: `go.mod`

**Step 1: Add certmagic dependency**

Run:
```bash
go get github.com/caddyserver/certmagic@v0.20.0
```

**Step 2: Verify dependency added**

Run:
```bash
go mod tidy
```

Expected: No errors, `go.mod` and `go.sum` updated

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add certmagic v0.20.0 for SSL support"
```

---

## Task 3: Create TLS Config Package

**Files:**
- Create: `internal/tls/config.go`

**Step 1: Create internal/tls directory**

Run:
```bash
mkdir -p internal/tls
```

**Step 2: Write TLS config structure**

Create `internal/tls/config.go`:

```go
package tls

import (
	"fmt"
	"os"

	"github.com/thatcatcamp/stinkykitty/internal/config"
)

// Config holds TLS configuration
type Config struct {
	Email      string
	CertDir    string
	Staging    bool
	BaseDomain string
	Enabled    bool
}

// LoadConfig loads TLS configuration from config system
func LoadConfig() (*Config, error) {
	cfg := &Config{
		Email:      config.GetString("tls.email"),
		CertDir:    config.GetString("tls.cert_dir"),
		Staging:    config.GetBool("tls.staging"),
		BaseDomain: config.GetString("server.base_domain"),
		Enabled:    config.GetBool("server.tls_enabled"),
	}

	// Validate required fields if TLS is enabled
	if cfg.Enabled {
		if cfg.Email == "" {
			return nil, fmt.Errorf("tls.email is required when TLS is enabled")
		}
		if cfg.BaseDomain == "" {
			return nil, fmt.Errorf("server.base_domain is required when TLS is enabled")
		}
	}

	// Create cert directory if it doesn't exist
	if cfg.CertDir != "" {
		if err := os.MkdirAll(cfg.CertDir, 0700); err != nil {
			return nil, fmt.Errorf("failed to create cert directory: %w", err)
		}
	}

	return cfg, nil
}
```

**Step 3: Build to verify no errors**

Run:
```bash
go build ./internal/tls
```

Expected: No errors

**Step 4: Commit**

```bash
git add internal/tls/config.go
git commit -m "feat: add TLS configuration loader"
```

---

## Task 4: Create TLS Manager

**Files:**
- Create: `internal/tls/manager.go`

**Step 1: Write TLS manager structure**

Create `internal/tls/manager.go`:

```go
package tls

import (
	"crypto/tls"
	"fmt"
	"log"

	"github.com/caddyserver/certmagic"
	"gorm.io/gorm"
	"github.com/thatcatcamp/stinkykitty/internal/models"
)

// Manager handles certificate provisioning and management
type Manager struct {
	cfg         *Config
	db          *gorm.DB
	certmagic   *certmagic.Config
}

// NewManager creates a new TLS manager
func NewManager(db *gorm.DB, cfg *Config) (*Manager, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if db == nil {
		return nil, fmt.Errorf("database is required")
	}

	// Create certmagic config
	cache := certmagic.NewCache(certmagic.CacheOptions{
		GetConfigForCert: func(certmagic.Certificate) (*certmagic.Config, error) {
			return certmagic.Default, nil
		},
	})

	magicCfg := certmagic.New(cache, certmagic.Config{
		Storage: &certmagic.FileStorage{Path: cfg.CertDir},
	})

	// Configure ACME issuer
	if cfg.Staging {
		magicCfg.Issuers = []certmagic.Issuer{
			certmagic.NewACMEIssuer(magicCfg, certmagic.ACMEIssuer{
				CA:     certmagic.LetsEncryptStagingCA,
				Email:  cfg.Email,
				Agreed: true,
			}),
		}
	} else {
		magicCfg.Issuers = []certmagic.Issuer{
			certmagic.NewACMEIssuer(magicCfg, certmagic.ACMEIssuer{
				CA:     certmagic.LetsEncryptProductionCA,
				Email:  cfg.Email,
				Agreed: true,
			}),
		}
	}

	m := &Manager{
		cfg:       cfg,
		db:        db,
		certmagic: magicCfg,
	}

	// Load and manage allowed domains
	if err := m.RefreshDomains(); err != nil {
		return nil, fmt.Errorf("failed to load domains: %w", err)
	}

	return m, nil
}

// GetAllowedDomains queries database for all domains that should have certificates
func (m *Manager) GetAllowedDomains() ([]string, error) {
	domains := []string{
		m.cfg.BaseDomain,
	}

	// Get all sites for subdomains
	var sites []models.Site
	if err := m.db.Find(&sites).Error; err != nil {
		return nil, fmt.Errorf("failed to query sites: %w", err)
	}

	for _, site := range sites {
		// Add subdomain
		subdomain := fmt.Sprintf("%s.%s", site.Subdomain, m.cfg.BaseDomain)
		domains = append(domains, subdomain)

		// Add custom domain if set
		if site.CustomDomain != nil && *site.CustomDomain != "" {
			domains = append(domains, *site.CustomDomain)
		}
	}

	return domains, nil
}

// RefreshDomains reloads the allowed domains list from database
func (m *Manager) RefreshDomains() error {
	domains, err := m.GetAllowedDomains()
	if err != nil {
		return err
	}

	log.Printf("TLS: Managing certificates for %d domains", len(domains))
	for _, domain := range domains {
		log.Printf("TLS: - %s", domain)
	}

	// Tell certmagic to manage these domains
	if err := m.certmagic.ManageAsync(m.db.Statement.Context, domains); err != nil {
		return fmt.Errorf("failed to manage domains: %w", err)
	}

	return nil
}

// GetTLSConfig returns TLS config for HTTPS server
func (m *Manager) GetTLSConfig() *tls.Config {
	return m.certmagic.TLSConfig()
}
```

**Step 2: Build to verify no errors**

Run:
```bash
go build ./internal/tls
```

Expected: No errors

**Step 3: Commit**

```bash
git add internal/tls/manager.go
git commit -m "feat: add TLS manager with certmagic integration"
```

---

## Task 5: Add HTTPS Redirect Middleware

**Files:**
- Create: `internal/middleware/https_redirect.go`

**Step 1: Write HTTPS redirect middleware**

Create `internal/middleware/https_redirect.go`:

```go
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// HTTPSRedirectMiddleware redirects HTTP requests to HTTPS
// Exceptions: ACME challenges (/.well-known/acme-challenge/)
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

**Step 2: Build to verify no errors**

Run:
```bash
go build ./internal/middleware
```

Expected: No errors

**Step 3: Commit**

```bash
git add internal/middleware/https_redirect.go
git commit -m "feat: add HTTPS redirect middleware with ACME exception"
```

---

## Task 6: Update Server for Dual HTTP/HTTPS

**Files:**
- Modify: `cmd/stinky/server.go`

**Step 1: Add TLS imports**

At the top of `cmd/stinky/server.go`, add imports:

```go
import (
	// ... existing imports ...
	"crypto/tls"
	"net/http"
	tlspkg "github.com/thatcatcamp/stinkykitty/internal/tls"
)
```

**Step 2: Replace server startup logic**

In `serverStartCmd` Run function, replace the existing `r.Run(addr)` section with:

```go
// Check if TLS is enabled
if config.GetBool("server.tls_enabled") {
	// Initialize TLS manager
	tlsCfg, err := tlspkg.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading TLS config: %v\n", err)
		os.Exit(1)
	}

	tlsManager, err := tlspkg.NewManager(db.GetDB(), tlsCfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing TLS manager: %v\n", err)
		os.Exit(1)
	}

	// Add HTTPS redirect middleware to router
	r.Use(middleware.HTTPSRedirectMiddleware())

	// Start HTTP server (port 80) for ACME challenges + redirects
	httpAddr := fmt.Sprintf(":%s", config.GetString("server.http_port"))
	go func() {
		fmt.Printf("Starting HTTP server on %s (ACME challenges + redirects)\n", httpAddr)
		if err := http.ListenAndServe(httpAddr, r); err != nil {
			fmt.Fprintf(os.Stderr, "HTTP server error: %v\n", err)
			os.Exit(1)
		}
	}()

	// Start HTTPS server (port 443) for main traffic
	httpsAddr := fmt.Sprintf(":%s", config.GetString("server.https_port"))
	fmt.Printf("Starting HTTPS server on %s\n", httpsAddr)
	fmt.Printf("Base domain: %s\n", baseDomain)
	fmt.Printf("TLS email: %s\n", tlsCfg.Email)
	if tlsCfg.Staging {
		fmt.Printf("WARNING: Using Let's Encrypt STAGING mode\n")
	}

	server := &http.Server{
		Addr:      httpsAddr,
		Handler:   r,
		TLSConfig: tlsManager.GetTLSConfig(),
	}

	if err := server.ListenAndServeTLS("", ""); err != nil {
		fmt.Fprintf(os.Stderr, "HTTPS server error: %v\n", err)
		os.Exit(1)
	}
} else {
	// TLS disabled - HTTP only (existing behavior)
	httpPort := config.GetString("server.http_port")
	addr := fmt.Sprintf(":%s", httpPort)

	fmt.Printf("Starting StinkyKitty server on %s\n", addr)
	fmt.Printf("Base domain: %s\n", baseDomain)
	fmt.Printf("TLS: disabled\n")
	if err := r.Run(addr); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
```

**Step 3: Build to verify no compilation errors**

Run:
```bash
go build ./cmd/stinky
```

Expected: No errors

**Step 4: Test HTTP-only mode still works**

Run:
```bash
./stinky server start
```

Expected: Server starts on port 80, "TLS: disabled" message

Stop server with Ctrl+C

**Step 5: Commit**

```bash
git add cmd/stinky/server.go
git commit -m "feat: add dual HTTP/HTTPS server support with TLS toggle"
```

---

## Task 7: Add TLS Status Command

**Files:**
- Create: `internal/tls/status.go`
- Create: `cmd/stinky/tls.go`

**Step 1: Write certificate status inspection code**

Create `internal/tls/status.go`:

```go
package tls

import (
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CertificateStatus holds information about a certificate
type CertificateStatus struct {
	Domain          string
	Issuer          string
	NotBefore       time.Time
	NotAfter        time.Time
	DaysUntilExpiry int
}

// GetCertificateStatus returns status for all managed certificates
func (m *Manager) GetCertificateStatus() ([]CertificateStatus, error) {
	var statuses []CertificateStatus

	domains, err := m.GetAllowedDomains()
	if err != nil {
		return nil, fmt.Errorf("failed to get domains: %w", err)
	}

	for _, domain := range domains {
		// Check if certificate exists
		certPath := filepath.Join(m.cfg.CertDir, "certificates", domain, domain+".crt")
		if _, err := os.Stat(certPath); os.IsNotExist(err) {
			// Certificate not yet provisioned
			statuses = append(statuses, CertificateStatus{
				Domain: domain,
				Issuer: "Not provisioned",
			})
			continue
		}

		// Read certificate
		certData, err := os.ReadFile(certPath)
		if err != nil {
			continue
		}

		// Parse certificate
		cert, err := x509.ParseCertificate(certData)
		if err != nil {
			continue
		}

		daysUntilExpiry := int(time.Until(cert.NotAfter).Hours() / 24)

		statuses = append(statuses, CertificateStatus{
			Domain:          domain,
			Issuer:          cert.Issuer.CommonName,
			NotBefore:       cert.NotBefore,
			NotAfter:        cert.NotAfter,
			DaysUntilExpiry: daysUntilExpiry,
		})
	}

	return statuses, nil
}
```

**Step 2: Write TLS CLI command**

Create `cmd/stinky/tls.go`:

```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/thatcatcamp/stinkykitty/internal/tls"
)

var tlsCmd = &cobra.Command{
	Use:   "tls",
	Short: "TLS certificate management",
	Long:  "Manage TLS certificates and view certificate status",
}

var tlsStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show certificate status",
	Run: func(cmd *cobra.Command, args []string) {
		if err := initConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if err := initSystemDB(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		tlsCfg, err := tls.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading TLS config: %v\n", err)
			os.Exit(1)
		}

		if !tlsCfg.Enabled {
			fmt.Println("TLS is not enabled")
			os.Exit(0)
		}

		tlsManager, err := tls.NewManager(db.GetDB(), tlsCfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing TLS manager: %v\n", err)
			os.Exit(1)
		}

		statuses, err := tlsManager.GetCertificateStatus()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting certificate status: %v\n", err)
			os.Exit(1)
		}

		// Print table header
		fmt.Printf("%-30s %-20s %-12s %s\n", "Domain", "Issuer", "Expires", "Days Left")
		fmt.Println("--------------------------------------------------------------------------------")

		// Print each certificate
		for _, status := range statuses {
			if status.Issuer == "Not provisioned" {
				fmt.Printf("%-30s %-20s %-12s %s\n", status.Domain, status.Issuer, "-", "-")
			} else {
				expiresStr := status.NotAfter.Format("2006-01-02")
				fmt.Printf("%-30s %-20s %-12s %d\n",
					status.Domain,
					status.Issuer,
					expiresStr,
					status.DaysUntilExpiry)
			}
		}
	},
}

func init() {
	tlsCmd.AddCommand(tlsStatusCmd)
	rootCmd.AddCommand(tlsCmd)
}
```

**Step 3: Build to verify no errors**

Run:
```bash
go build ./cmd/stinky
```

Expected: No errors

**Step 4: Test TLS status command**

Run:
```bash
./stinky tls status
```

Expected: "TLS is not enabled" (since TLS is disabled by default)

**Step 5: Commit**

```bash
git add internal/tls/status.go cmd/stinky/tls.go
git commit -m "feat: add TLS status CLI command"
```

---

## Task 8: Manual Testing with Staging Mode

**Files:**
- Modify: Config file

**Step 1: Enable TLS in staging mode**

Run:
```bash
./stinky config set server.tls_enabled true
./stinky config set tls.email "test@example.com"
./stinky config set tls.staging true
```

**Step 2: Verify config**

Run:
```bash
./stinky config get server.tls_enabled
./stinky config get tls.email
./stinky config get tls.staging
```

Expected: true, test@example.com, true

**Step 3: Start server and verify output**

Run:
```bash
sudo ./stinky server start
```

Expected output should include:
- "Starting HTTP server on :80"
- "Starting HTTPS server on :443"
- "WARNING: Using Let's Encrypt STAGING mode"

**Step 4: Test HTTP redirect (in another terminal)**

Run:
```bash
curl -I http://localhost/
```

Expected: 301 redirect to https://

**Step 5: Stop server**

Press Ctrl+C

**Step 6: Disable TLS for now**

Run:
```bash
./stinky config set server.tls_enabled false
```

---

## Task 9: Update README with TLS Documentation

**Files:**
- Modify: `README.md`

**Step 1: Add TLS section to README**

Add to README.md after the configuration section:

```markdown
## TLS/HTTPS Configuration

StinkyKitty supports automatic HTTPS with Let's Encrypt:

### Enable HTTPS

```bash
stinky config set server.tls_enabled true
stinky config set tls.email "admin@yourdomain.com"
```

### Testing with Staging

For testing, use Let's Encrypt staging to avoid rate limits:

```bash
stinky config set tls.staging true
```

### Check Certificate Status

```bash
stinky tls status
```

### Requirements

- Server must be publicly accessible on ports 80 and 443
- DNS must point to your server
- Must run as root (or with CAP_NET_BIND_SERVICE capability)

### Configuration Options

- `server.tls_enabled` - Enable HTTPS (default: false)
- `tls.email` - Email for Let's Encrypt registration (required)
- `tls.cert_dir` - Certificate storage directory (default: /var/lib/stinkykitty/certs)
- `tls.staging` - Use Let's Encrypt staging for testing (default: false)
```

**Step 2: Commit**

```bash
git add README.md
git commit -m "docs: add TLS/HTTPS configuration documentation"
```

---

## Task 10: Final Build and Verification

**Files:**
- All modified files

**Step 1: Run full build**

Run:
```bash
make build
```

Expected: Build succeeds, binary created

**Step 2: Verify help text**

Run:
```bash
./stinky tls --help
```

Expected: Shows TLS commands

**Step 3: Run go vet**

Run:
```bash
go vet ./...
```

Expected: No issues

**Step 4: Final commit if any cleanup needed**

```bash
git add .
git commit -m "chore: final cleanup for SSL implementation"
```

---

## Production Deployment Checklist

After implementation, before deploying to production:

1. **DNS Configuration**
   - Ensure A record points to server IP
   - Wait for DNS propagation (can take minutes to hours)

2. **Firewall Rules**
   - Open port 80 (HTTP - ACME challenges)
   - Open port 443 (HTTPS - main traffic)

3. **Test with Staging First**
   ```bash
   stinky config set tls.staging true
   stinky config set tls.email "your-email@example.com"
   stinky config set server.tls_enabled true
   sudo stinky server start
   ```

4. **Verify Staging Certificate**
   ```bash
   curl -I https://campasaur.us
   ```
   Should show "Fake LE Intermediate" in certificate chain

5. **Switch to Production**
   ```bash
   stinky config set tls.staging false
   sudo systemctl restart stinkykitty  # or however you restart
   ```

6. **Monitor Logs**
   - Watch for certificate provisioning messages
   - Check for any ACME challenge failures

7. **Verify Production Certificate**
   ```bash
   curl -I https://campasaur.us
   stinky tls status
   ```

---

## Troubleshooting

**Certificate provisioning fails:**
- Check DNS is pointing to server: `dig campasaur.us`
- Verify port 80 is accessible: `curl http://campasaur.us/.well-known/acme-challenge/test`
- Check firewall rules
- Review logs for ACME error details

**"Permission denied" on ports 80/443:**
- Run as root: `sudo stinky server start`
- Or grant capabilities: `sudo setcap CAP_NET_BIND_SERVICE=+eip /path/to/stinky`

**Rate limit errors:**
- Use staging mode for testing
- Let's Encrypt allows 50 certs per domain per week
- Wait for rate limit to reset (one week)

**Certificates not renewing:**
- Certmagic handles renewal automatically
- Renewal happens 30 days before expiry
- Check logs for renewal errors
- Verify server has been running continuously
