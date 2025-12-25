package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/auth"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/models"
)

// LoginHandler handles admin login requests
func LoginHandler(c *gin.Context) {
	email := c.PostForm("email")
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

	// Set HTTP-only cookie
	c.SetCookie(
		"stinky_token",        // name
		token,                 // value
		28800,                 // max age (8 hours in seconds)
		"/",                   // path
		"",                    // domain (empty = current domain)
		false,                 // secure (set to true in production with HTTPS)
		true,                  // httpOnly
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
		-1,     // max age -1 deletes the cookie
		"/",
		"",
		false,
		true,
	)

	// Redirect to login
	c.Redirect(http.StatusFound, "/admin/login")
}

// LoginFormHandler shows the login form
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

            <form method="POST" action="/admin/login">
                <div class="form-group">
                    <label for="email">Email</label>
                    <input type="email" id="email" name="email" placeholder="admin@example.com" required>
                </div>

                <div class="form-group">
                    <label for="password">Password</label>
                    <input type="password" id="password" name="password" required>
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

// DashboardHandler renders the admin dashboard
func DashboardHandler(c *gin.Context) {
	// Get user and site from context (set by auth middleware)
	userVal, exists := c.Get("user")
	if !exists {
		c.String(http.StatusUnauthorized, "Not authenticated")
		return
	}
	user := userVal.(*models.User)

	siteVal, exists := c.Get("site")
	if !exists {
		c.String(http.StatusInternalServerError, "Site not found")
		return
	}
	site := siteVal.(*models.Site)

	// Load all pages for this site
	var pages []models.Page
	db.GetDB().Where("site_id = ?", site.ID).Order("slug ASC").Find(&pages)

	// Build pages list HTML
	var pagesList strings.Builder
	homepageExists := false

	for _, page := range pages {
		if page.Slug == "/" {
			homepageExists = true
			status := "Draft"
			if page.Published {
				status = "Published"
			}
			pagesList.WriteString(fmt.Sprintf(`
				<div class="page-item">
					<strong>Homepage</strong> <span class="status">%s</span>
					<div class="actions">
						<a href="/admin/pages/%d/edit" class="btn-small">Edit</a>
					</div>
				</div>
			`, status, page.ID))
		} else {
			status := "Draft"
			if page.Published {
				status = "Published"
			}
			pagesList.WriteString(fmt.Sprintf(`
				<div class="page-item">
					<strong>%s</strong> <code>%s</code> <span class="status">%s</span>
					<div class="actions">
						<a href="/admin/pages/%d/edit" class="btn-small">Edit</a>
						<form method="POST" action="/admin/pages/%d/delete" style="display:inline;" onsubmit="return confirm('Delete this page?')">
							<button type="submit" class="btn-small btn-danger">Delete</button>
						</form>
					</div>
				</div>
			`, page.Title, page.Slug, status, page.ID, page.ID))
		}
	}

	if !homepageExists {
		pagesList.WriteString(`
			<div class="page-item placeholder">
				<em>No homepage yet</em>
				<form method="POST" action="/admin/pages" style="display:inline;">
					<input type="hidden" name="slug" value="/">
					<input type="hidden" name="title" value="` + site.Subdomain + `">
					<button type="submit" class="btn-small">Create Homepage</button>
				</form>
			</div>
		`)
	}

	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Admin Dashboard - ` + site.Subdomain + `</title>
    <style>
        body { font-family: system-ui, -apple-system, sans-serif; background: #f5f5f5; margin: 0; padding: 20px; }
        .container { max-width: 900px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { margin: 0 0 10px 0; font-size: 28px; color: #333; }
        .user-info { color: #666; font-size: 14px; margin-bottom: 30px; }
        .section { margin-bottom: 30px; }
        .section h2 { font-size: 18px; margin-bottom: 15px; color: #444; }
        .page-item { padding: 15px; border: 1px solid #e0e0e0; border-radius: 4px; margin-bottom: 10px; display: flex; justify-content: space-between; align-items: center; }
        .page-item.placeholder { border-style: dashed; color: #999; }
        .status { font-size: 12px; padding: 2px 8px; background: #e0e0e0; border-radius: 3px; margin-left: 10px; }
        .actions { display: flex; gap: 8px; }
        .btn { padding: 10px 20px; background: #007bff; color: white; text-decoration: none; border-radius: 4px; border: none; cursor: pointer; font-size: 14px; }
        .btn:hover { background: #0056b3; }
        .btn-small { padding: 6px 12px; font-size: 13px; background: #007bff; color: white; text-decoration: none; border-radius: 4px; border: none; cursor: pointer; }
        .btn-small:hover { background: #0056b3; }
        .btn-danger { background: #dc3545; }
        .btn-danger:hover { background: #c82333; }
        code { background: #f0f0f0; padding: 2px 6px; border-radius: 3px; font-size: 13px; }
        .logout { float: right; font-size: 14px; }
    </style>
</head>
<body>
    <div class="container">
        <form method="POST" action="/admin/logout" class="logout">
            <button type="submit" class="btn-small">Logout</button>
        </form>
        <h1>Admin Dashboard</h1>
        <div class="user-info">
            ` + user.Email + ` • ` + site.Subdomain + `
        </div>

        <div class="section">
            <h2>Pages</h2>
            ` + pagesList.String() + `
            <div style="margin-top: 15px;">
                <a href="/admin/pages/new" class="btn">+ Create New Page</a>
                <a href="/admin/menu" class="btn" style="background: #17a2b8; margin-left: 10px;">Navigation Menu</a>
            </div>
        </div>

        <div class="section">
            <a href="/" target="_blank" style="color: #007bff; text-decoration: none;">→ View Public Site</a>
        </div>
    </div>
</body>
</html>`

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
