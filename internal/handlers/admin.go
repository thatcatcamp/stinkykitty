package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/auth"
	"github.com/thatcatcamp/stinkykitty/internal/config"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/middleware"
	"github.com/thatcatcamp/stinkykitty/internal/models"
)

// LoginHandler handles admin login requests
func LoginHandler(c *gin.Context) {
	email := strings.ToLower(strings.TrimSpace(c.PostForm("email")))
	password := c.PostForm("password")

	// Get site from context
	siteVal, exists := c.Get("site")
	if !exists {
		c.String(http.StatusInternalServerError, "Site not found")
		return
	}
	site := siteVal.(*models.Site)

	// Find user by email
	var user models.User
	if err := db.GetDB().Where("email = ?", email).First(&user).Error; err != nil {
		c.String(http.StatusUnauthorized, "Invalid email or password")
		return
	}

	// Verify password
	if !auth.CheckPassword(password, user.PasswordHash) {
		c.String(http.StatusUnauthorized, "Invalid email or password")
		return
	}

	// Check if user has access to this site
	hasAccess := false

	// Global admins can access any site
	if user.IsGlobalAdmin {
		hasAccess = true
	}

	// Site owner can access
	if site.OwnerID == user.ID {
		hasAccess = true
	}

	// Check if user is in SiteUsers (member of site)
	if !hasAccess {
		var siteUser models.SiteUser
		err := db.GetDB().Where("site_id = ? AND user_id = ?", site.ID, user.ID).First(&siteUser).Error
		if err == nil {
			hasAccess = true
		}
	}

	if !hasAccess {
		c.String(http.StatusForbidden, "You don't have access to this site")
		return
	}

	// Generate JWT token
	token, err := auth.GenerateToken(&user, site)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to generate token: %v", err)
		return
	}

	// Set SameSite attribute before setting cookie
	c.SetSameSite(http.SameSiteLaxMode)

	// Set HTTP-only cookie (Secure flag enabled when TLS is configured)
	c.SetCookie(
		"stinky_token",                      // name
		token,                               // value
		28800,                               // max age (8 hours in seconds)
		"/",                                 // path
		"",                                  // domain (empty = current domain)
		config.GetBool("server.tls_enabled"), // secure (true in production with HTTPS)
		true,                                // httpOnly
	)

	// Redirect to dashboard
	c.Redirect(http.StatusFound, "/admin/dashboard")
}

// LogoutHandler handles admin logout requests
func LogoutHandler(c *gin.Context) {
	// Clear cookie
	c.SetCookie(
		"stinky_token",
		"",
		-1,                                  // max age -1 deletes the cookie
		"/",
		"",
		config.GetBool("server.tls_enabled"), // secure flag
		true,
	)

	// Redirect to login
	c.Redirect(http.StatusFound, "/admin/login")
}

// LoginFormHandler displays the StinkyKitty login page with warm professional design using the design system
func LoginFormHandler(c *gin.Context) {
	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Sign In - StinkyKitty</title>
    <style>
        ` + GetDesignSystemCSS() + `

        .login-container {
            display: flex;
            align-items: center;
            justify-content: center;
            min-height: 100vh;
            padding: var(--spacing-base);
        }

        .login-card {
            background: var(--color-bg-card);
            border-radius: var(--radius-base);
            padding: calc(var(--spacing-base) * 2.5);
            width: 100%;
            max-width: 400px;
            box-shadow: var(--shadow-sm);
        }

        .login-header {
            text-align: center;
            margin-bottom: calc(var(--spacing-base) * 2);
        }

        .login-logo {
            font-size: 24px;
            font-weight: 700;
            color: var(--color-accent);
            margin-bottom: var(--spacing-base);
        }

        .login-title {
            font-size: 20px;
            font-weight: 600;
            margin-bottom: var(--spacing-base);
        }

        .login-subtitle {
            font-size: 14px;
            color: var(--color-text-secondary);
        }

        .form-group {
            margin-bottom: calc(var(--spacing-base) * 1.5);
        }

        .form-group label {
            display: block;
            margin-bottom: var(--spacing-sm);
            font-weight: 600;
            color: var(--color-text-primary);
            font-size: 14px;
        }

        .form-group input {
            width: 100%;
            font-size: 16px;
        }

        .login-button {
            width: 100%;
            background: var(--color-accent);
            color: white;
            font-size: 16px;
            padding: calc(var(--spacing-base) * 0.75) var(--spacing-base);
            margin-top: var(--spacing-base);
        }

        .login-button:hover {
            background: var(--color-accent-hover);
        }

        .login-footer {
            text-align: center;
            margin-top: var(--spacing-md);
            font-size: 12px;
        }

        @media (max-width: 640px) {
            .login-card {
                padding: var(--spacing-md);
            }
        }
    </style>
</head>
<body>
    <div class="login-container">
        <div class="login-card">
            <div class="login-header">
                <div class="login-logo">🐱 StinkyKitty</div>
                <h1 class="login-title">Sign In</h1>
                <p class="login-subtitle">One account for all your camps</p>
            </div>

            <div id="error-message" style="display:none; color: var(--color-danger); margin-bottom: var(--spacing-md); text-align: center;"></div>

            <form method="POST" action="/admin/login">
                ` + middleware.GetCSRFTokenHTML(c) + `
                <div class="form-group">
                    <label for="email">Email</label>
                    <input type="email" id="email" name="email" placeholder="admin@example.com" autocomplete="email" required>
                </div>

                <div class="form-group">
                    <label for="password">Password</label>
                    <input type="password" id="password" name="password" autocomplete="current-password" required>
                </div>

                <button type="submit" class="login-button">Sign In</button>
            </form>

            <div class="login-footer">
                <p>Secure login • No tracking • Simple & fast</p>
            </div>
        </div>
    </div>
</body>
</html>`

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// DashboardHandler shows the admin dashboard with list of sites
func DashboardHandler(c *gin.Context) {
	// Get user from context
	userVal, exists := c.Get("user")
	if !exists {
		c.Redirect(http.StatusFound, "/admin/login")
		return
	}
	user := userVal.(*models.User)

	// Get base domain from config
	baseDomain := config.GetString("server.base_domain")
	if baseDomain == "" {
		baseDomain = "localhost"
	}

	// Get all sites where user is an admin/owner, OR all sites if global admin
	var userSites []struct {
		ID           uint
		Subdomain    string
		CustomDomain *string
		Role         string
	}

	if user.IsGlobalAdmin {
		// Global admins see all sites
		db.GetDB().Raw(`
			SELECT sites.id, sites.subdomain, sites.custom_domain, COALESCE(site_users.role, 'owner') as role
			FROM sites
			LEFT JOIN site_users ON sites.id = site_users.site_id AND site_users.user_id = ?
			ORDER BY sites.subdomain
		`, user.ID).Scan(&userSites)
	} else {
		// Regular users see only their sites
		db.GetDB().Raw(`
			SELECT sites.id, sites.subdomain, sites.custom_domain, site_users.role
			FROM sites
			JOIN site_users ON sites.id = site_users.site_id
			WHERE site_users.user_id = ? AND (site_users.role = 'owner' OR site_users.role = 'admin')
			ORDER BY sites.subdomain
		`, user.ID).Scan(&userSites)
	}

	// Build sites list HTML
	var sitesHTML string
	if len(userSites) == 0 {
		sitesHTML = `<div class="empty-state">No sites yet. Contact an administrator to create one.</div>`
	} else {
		for _, us := range userSites {
			var domainDisplay string
			if us.CustomDomain != nil && *us.CustomDomain != "" {
				domainDisplay = *us.CustomDomain
			} else {
				domainDisplay = us.Subdomain + "." + baseDomain
			}

			sitesHTML += `
				<div class="site-card">
					<div class="site-info">
						<h3>` + us.Subdomain + `</h3>
						<small>` + domainDisplay + `</small>
					</div>
					<div class="site-actions">
						<a href="/admin/pages?site=` + fmt.Sprintf("%d", us.ID) + `" class="btn-small">Edit</a>
						<a href="https://` + domainDisplay + `" target="_blank" class="btn-small btn-secondary">View</a>
						<button class="btn-small btn-danger" onclick="confirmDelete(` + fmt.Sprintf("%d", us.ID) + `, '` + us.Subdomain + `')">Delete</button>
					</div>
				</div>
			`
		}
	}

	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Dashboard - StinkyKitty</title>
    <style>
        ` + GetDesignSystemCSS() + `

        body { padding: 0; }

        .dashboard-layout {
            min-height: 100vh;
            display: flex;
            flex-direction: column;
        }

        .header {
            background: var(--color-bg-card);
            border-bottom: 1px solid var(--color-border);
            padding: var(--spacing-base) var(--spacing-md);
            box-shadow: var(--shadow-sm);
            position: sticky;
            top: 0;
            z-index: 10;
        }

        .header-content {
            max-width: 1200px;
            margin: 0 auto;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }

        .header-left h1 {
            font-size: 18px;
            color: var(--color-text-primary);
        }

        .header-right {
            display: flex;
            align-items: center;
            gap: var(--spacing-base);
        }

        .header-right small {
            color: var(--color-text-secondary);
            font-size: 14px;
        }

        .logout-btn {
            background: var(--color-accent);
            color: white;
            padding: var(--spacing-sm) var(--spacing-md);
            border-radius: var(--radius-sm);
            border: none;
            cursor: pointer;
            font-size: 14px;
            font-weight: 600;
        }

        .logout-btn:hover {
            background: var(--color-accent-hover);
        }

        .container {
            flex: 1;
            max-width: 1200px;
            margin: 0 auto;
            width: 100%;
            padding: var(--spacing-md);
        }

        .hero {
            background: var(--color-bg-card);
            padding: var(--spacing-lg) var(--spacing-md);
            border-radius: var(--radius-base);
            border: 1px solid var(--color-border);
            margin-bottom: var(--spacing-lg);
            text-align: center;
        }

        .hero h2 {
            margin-bottom: var(--spacing-base);
        }

        .hero-buttons {
            display: flex;
            gap: var(--spacing-base);
            justify-content: center;
            flex-wrap: wrap;
        }

        .btn {
            background: var(--color-accent);
            color: white;
            padding: var(--spacing-sm) var(--spacing-md);
            border-radius: var(--radius-sm);
            border: none;
            cursor: pointer;
            font-size: 14px;
            font-weight: 600;
            text-decoration: none;
            display: inline-block;
            transition: background var(--transition);
        }

        .btn:hover {
            background: var(--color-accent-hover);
        }

        .btn-secondary {
            background: var(--color-text-secondary);
            color: white;
        }

        .btn-secondary:hover {
            background: #5a6268;
        }

        .btn-outline {
            background: transparent;
            border: 1px solid var(--color-accent);
            color: var(--color-accent);
        }

        .btn-outline:hover {
            background: rgba(46, 139, 158, 0.05);
        }

        .section {
            margin-bottom: var(--spacing-lg);
        }

        .section-title {
            font-size: 18px;
            font-weight: 600;
            margin-bottom: var(--spacing-md);
            color: var(--color-text-primary);
        }

        .sites-list {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
            gap: var(--spacing-base);
        }

        .site-card {
            background: var(--color-bg-card);
            border: 1px solid var(--color-border);
            border-radius: var(--radius-base);
            padding: var(--spacing-md);
            display: flex;
            justify-content: space-between;
            align-items: center;
            gap: var(--spacing-md);
            transition: box-shadow var(--transition), background var(--transition);
        }

        .site-card:hover {
            box-shadow: var(--shadow-md);
            background: #fafbfc;
        }

        .site-info h3 {
            font-size: 16px;
            font-weight: 600;
            margin: 0 0 var(--spacing-sm) 0;
            color: var(--color-text-primary);
        }

        .site-info small {
            font-size: 12px;
            color: var(--color-text-secondary);
        }

        .site-actions {
            display: flex;
            gap: var(--spacing-sm);
            flex-shrink: 0;
        }

        .btn-small {
            padding: var(--spacing-sm) var(--spacing-base);
            font-size: 13px;
            background: var(--color-accent);
            color: white;
            text-decoration: none;
            border-radius: var(--radius-sm);
            border: none;
            cursor: pointer;
            transition: background var(--transition);
        }

        .btn-small:hover {
            background: var(--color-accent-hover);
        }

        .btn-small.btn-secondary {
            background: var(--color-text-secondary);
        }

        .btn-small.btn-secondary:hover {
            background: #5a6268;
        }

        .empty-state {
            padding: var(--spacing-lg);
            text-align: center;
            color: var(--color-text-secondary);
            border: 2px dashed var(--color-border);
            border-radius: var(--radius-base);
            background: #fafbfc;
        }

        .footer {
            padding: var(--spacing-md);
            text-align: center;
            border-top: 1px solid var(--color-border);
            margin-top: var(--spacing-lg);
        }

        .footer a {
            color: var(--color-accent);
            text-decoration: none;
            font-size: 14px;
            margin: 0 var(--spacing-base);
        }

        @media (max-width: 640px) {
            .site-card {
                flex-direction: column;
                align-items: flex-start;
            }

            .hero-buttons {
                flex-direction: column;
            }

            .btn, .hero-buttons .btn {
                width: 100%;
            }
        }
    </style>
</head>
<body>
    <div class="dashboard-layout">
        <div class="header">
            <div class="header-content">
                <div class="header-left">
                    <h1>StinkyKitty Admin</h1>
                </div>
                <div class="header-right">
                    <small>` + user.Email + `</small>
                    <form method="POST" action="/admin/logout" style="display:inline;">
                        ` + middleware.GetCSRFTokenHTML(c) + `
                        <button type="submit" class="logout-btn">Sign Out</button>
                    </form>
                </div>
            </div>
        </div>

        <div class="container">
            <div class="hero">
                <h2>Your Camps</h2>
                <p>Select a camp to edit its pages and settings</p>
                <div class="hero-buttons">
                    <a href="/admin/users" class="btn btn-secondary">Manage Users</a>
                    <a href="/admin/create-camp" class="btn">+ Create New Camp</a>
                    <a href="/admin/media" class="btn">
                        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="vertical-align: middle; margin-right: 5px;">
                            <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
                            <circle cx="8.5" cy="8.5" r="1.5"/>
                            <polyline points="21 15 16 10 5 21"/>
                        </svg>
                        Media Library
                    </a>
                </div>
            </div>

            <div class="section">
                <h3 class="section-title">All Camps</h3>
                <div class="sites-list">
                    ` + sitesHTML + `
                </div>
            </div>

            <div class="footer">
                <a href="/">← Back to Home</a>
                <a href="/admin/docs">Documentation</a>
            </div>
        </div>
    </div>

    <!-- Delete confirmation modal -->
    <div id="delete-modal" class="modal" style="display:none;">
        <div class="modal-content">
            <h3>Delete Camp?</h3>
            <p>You are about to delete <strong id="delete-camp-name"></strong>.</p>
            <p style="color: var(--color-text-secondary); font-size: 13px;">
                The camp data and backups will be preserved for manual cleanup if needed later.
            </p>
            <div class="modal-actions">
                <button onclick="cancelDelete()" class="btn-small btn-secondary">Cancel</button>
                <button onclick="confirmDeleteAction()" class="btn-small btn-danger">Delete Camp</button>
            </div>
        </div>
    </div>

    <style>
        .modal {
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: rgba(0,0,0,0.5);
            display: flex;
            align-items: center;
            justify-content: center;
            z-index: 1000;
        }

        .modal-content {
            background: var(--color-bg-card);
            border-radius: var(--radius-base);
            padding: var(--spacing-lg);
            max-width: 400px;
            box-shadow: var(--shadow-lg);
        }

        .modal-content h3 {
            margin-top: 0;
            color: var(--color-text-primary);
        }

        .modal-content p {
            color: var(--color-text-secondary);
            margin: var(--spacing-base) 0;
        }

        .modal-actions {
            display: flex;
            gap: var(--spacing-base);
            margin-top: var(--spacing-lg);
        }

        .modal-actions button {
            flex: 1;
        }

        .btn-danger {
            background: #dc3545;
            color: white;
        }

        .btn-danger:hover {
            background: #c82333;
        }
    </style>

    <script>
        let pendingDeleteSiteId = null;

        function confirmDelete(siteId, subdomain) {
            pendingDeleteSiteId = siteId;
            document.getElementById('delete-camp-name').textContent = subdomain;
            document.getElementById('delete-modal').style.display = 'flex';
        }

        function cancelDelete() {
            pendingDeleteSiteId = null;
            document.getElementById('delete-modal').style.display = 'none';
        }

        function confirmDeleteAction() {
            if (!pendingDeleteSiteId) return;

            const csrfToken = document.cookie
                .split('; ')
                .find(row => row.startsWith('csrf_token='))
                ?.split('=')[1] || '';

            fetch('/admin/sites/' + pendingDeleteSiteId + '/delete', {
                method: 'DELETE',
                headers: {
                    'X-CSRF-Token': csrfToken
                }
            })
                .then(r => r.json())
                .then(data => {
                    if (data.error) {
                        alert('Error: ' + data.error);
                    } else {
                        location.reload();
                    }
                })
                .catch(e => alert('Failed: ' + e));
        }

        // Close modal if user clicks outside
        document.getElementById('delete-modal').onclick = function(e) {
            if (e.target === this) cancelDelete();
        };
    </script>
</body>
</html>`

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// DocsHandler displays documentation page
func DocsHandler(c *gin.Context) {
	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Documentation - Stinky Kitty CMS</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            background: #f5f5f5;
            color: #333;
            line-height: 1.6;
        }

        .container {
            max-width: 900px;
            margin: 0 auto;
            padding: 20px;
            background: white;
            min-height: 100vh;
        }

        h1 {
            margin-bottom: 30px;
            color: #1a1a1a;
            border-bottom: 3px solid #4CAF50;
            padding-bottom: 10px;
        }

        h2 {
            margin-top: 30px;
            margin-bottom: 15px;
            color: #2c3e50;
        }

        p, li {
            margin-bottom: 10px;
            color: #555;
        }

        ul {
            margin-left: 20px;
            margin-bottom: 15px;
        }

        code {
            background: #f4f4f4;
            padding: 2px 6px;
            border-radius: 3px;
            font-family: "Courier New", monospace;
            color: #d63384;
        }

        .section {
            margin-bottom: 30px;
        }

        .footer {
            margin-top: 50px;
            padding-top: 20px;
            border-top: 1px solid #ddd;
            text-align: center;
            color: #999;
            font-size: 14px;
        }

        .back-link {
            display: inline-block;
            margin-bottom: 20px;
            color: #4CAF50;
            text-decoration: none;
        }

        .back-link:hover {
            text-decoration: underline;
        }
    </style>
</head>
<body>
    <div class="container">
        <a href="/admin" class="back-link">← Back to Dashboard</a>

        <h1>Stinky Kitty CMS Documentation</h1>

        <div class="section">
            <h2>Getting Started</h2>
            <p>Stinky Kitty is a multi-tenant CMS platform designed for managing camp websites. Each camp (site) has its own administrators and content.</p>
        </div>

        <div class="section">
            <h2>Creating a New Camp</h2>
            <ol>
                <li>Click "Create New Camp" from the dashboard</li>
                <li>Enter the camp name and choose a subdomain</li>
                <li>Create an administrator account with an email address</li>
                <li>The new administrator will receive a password setup email</li>
            </ol>
        </div>

        <div class="section">
            <h2>Managing Pages</h2>
            <p>Each camp can have multiple pages. Pages are composed of reusable content blocks.</p>
            <ul>
                <li>Click "Pages" to view and manage your site's pages</li>
                <li>Create new pages with custom content blocks</li>
                <li>Use blocks to structure your content (text, images, etc.)</li>
            </ul>
        </div>

        <div class="section">
            <h2>Camp Settings</h2>
            <p>Configure your camp's basic information:</p>
            <ul>
                <li>Camp name and description</li>
                <li>Custom domain (optional)</li>
                <li>Site-level administrators</li>
                <li>Email configuration</li>
            </ul>
        </div>

        <div class="section">
            <h2>Content Blocks</h2>
            <p>Pages are built from reusable content blocks. Supported block types include:</p>
            <ul>
                <li><code>text</code> - Rich text content</li>
                <li><code>heading</code> - Section headings</li>
                <li><code>image</code> - Image blocks</li>
                <li><code>button</code> - Call-to-action buttons</li>
            </ul>
        </div>

        <div class="section">
            <h2>Exporting Content</h2>
            <p>Export your camp's content using the Export feature. Content is exported in a portable format for backup or migration.</p>
        </div>

        <div class="footer">
            <p>For more help, please contact the site administrator.</p>
        </div>
    </div>
</body>
</html>`

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
