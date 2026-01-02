// SPDX-License-Identifier: MIT
package handlers

import (
	"html"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/middleware"
	"github.com/thatcatcamp/stinkykitty/internal/models"
	"github.com/thatcatcamp/stinkykitty/internal/themes"
)

// AdminSettingsHandler shows the settings form for site information and theme settings
func AdminSettingsHandler(c *gin.Context) {
	// Get site from context
	siteVal, exists := c.Get("site")
	if !exists {
		c.String(http.StatusInternalServerError, "Site not found")
		return
	}
	site := siteVal.(*models.Site)

	// Get CSRF token
	csrfToken := middleware.GetCSRFTokenHTML(c)

	// Get all available palettes
	palettes := themes.ListPalettes()

	// Build palette options HTML
	var paletteOptions string
	for _, p := range palettes {
		selected := ""
		if p.Name == site.ThemePalette {
			selected = " selected"
		}
		paletteOptions += `<option value="` + p.Name + `"` + selected + `>` + p.Name + `</option>`
	}

	// Dark mode checkbox
	darkModeChecked := ""
	if site.DarkMode {
		darkModeChecked = " checked"
	}

	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Site Settings - StinkyKitty</title>
    <style>
        ` + GetDesignSystemCSS() + `

        body { padding: 0; }

        .settings-layout {
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

        .header-left {
            display: flex;
            align-items: center;
            gap: var(--spacing-base);
        }

        .header-left h1 {
            font-size: 18px;
            color: var(--color-text-primary);
            margin: 0;
        }

        .back-link {
            color: var(--color-accent);
            text-decoration: none;
            font-size: 14px;
        }

        .back-link:hover {
            text-decoration: underline;
        }

        .container {
            flex: 1;
            max-width: 800px;
            margin: 0 auto;
            width: 100%;
            padding: var(--spacing-md);
        }

        .card {
            background: var(--color-bg-card);
            border: 1px solid var(--color-border);
            border-radius: var(--radius-base);
            padding: var(--spacing-lg);
            margin-bottom: var(--spacing-md);
        }

        .card-title {
            font-size: 20px;
            font-weight: 600;
            margin-bottom: var(--spacing-base);
            color: var(--color-text-primary);
        }

        .card-description {
            color: var(--color-text-secondary);
            margin-bottom: var(--spacing-md);
            font-size: 14px;
        }

        .form-group {
            margin-bottom: var(--spacing-md);
        }

        .form-group label {
            display: block;
            margin-bottom: var(--spacing-sm);
            font-weight: 600;
            color: var(--color-text-primary);
            font-size: 14px;
        }

        .form-group select {
            width: 100%;
            padding: var(--spacing-sm) var(--spacing-base);
            border: 1px solid var(--color-border);
            border-radius: var(--radius-sm);
            font-size: 14px;
            background: var(--color-bg-base);
            color: var(--color-text-primary);
        }

        .form-group select:focus {
            outline: none;
            border-color: var(--color-accent);
        }

        .form-group input[type="text"] {
            width: 100%;
            padding: var(--spacing-sm) var(--spacing-base);
            border: 1px solid var(--color-border);
            border-radius: var(--radius-sm);
            font-size: 14px;
            background: var(--color-bg-base);
            color: var(--color-text-primary);
        }

        .form-group input[type="text"]:focus {
            outline: none;
            border-color: var(--color-accent);
        }

        .checkbox-group {
            display: flex;
            align-items: center;
            gap: var(--spacing-sm);
        }

        .checkbox-group input[type="checkbox"] {
            width: 18px;
            height: 18px;
            cursor: pointer;
        }

        .checkbox-group label {
            margin: 0;
            cursor: pointer;
            font-weight: 600;
            color: var(--color-text-primary);
        }

        .help-text {
            font-size: 12px;
            color: var(--color-text-secondary);
            margin-top: var(--spacing-sm);
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
            transition: background var(--transition);
        }

        .btn:hover {
            background: var(--color-accent-hover);
        }

        .btn-secondary {
            background: var(--color-text-secondary);
            margin-left: var(--spacing-sm);
        }

        .btn-secondary:hover {
            background: #5a6268;
        }

        .info-box {
            background: #e3f2fd;
            border: 1px solid #90caf9;
            border-radius: var(--radius-sm);
            padding: var(--spacing-base);
            margin-top: var(--spacing-md);
            font-size: 13px;
            color: #1565c0;
        }

        .button-group {
            display: flex;
            gap: var(--spacing-sm);
            margin-top: var(--spacing-md);
        }

        @media (max-width: 640px) {
            .container {
                padding: var(--spacing-sm);
            }

            .card {
                padding: var(--spacing-base);
            }
        }
    </style>
</head>
<body>
    <div class="settings-layout">
        <div class="header">
            <div class="header-content">
                <div class="header-left">
                    <a href="/admin/pages" class="back-link">← Pages</a>
                    <h1>Site Settings</h1>
                </div>
            </div>
        </div>

        <div class="container">
            <form method="POST" action="/admin/settings">
                ` + csrfToken + `
                <div class="card">
                    <h2 class="card-title">Site Settings</h2>
                    <p class="card-description">Configure your site's information and theme</p>

                    <div class="form-group">
                        <label for="site_title">Site Title</label>
                        <input type="text" id="site_title" name="site_title" value="` + html.EscapeString(site.SiteTitle) + `" placeholder="My Camp Name">
                        <small style="color: var(--color-text-secondary); display: block; margin-top: 4px;">
                            The main title of your website
                        </small>
                    </div>

                    <div class="form-group">
                        <label for="site_tagline">Site Tagline</label>
                        <input type="text" id="site_tagline" name="site_tagline" value="` + html.EscapeString(site.SiteTagline) + `" placeholder="Where adventure begins">
                        <small style="color: var(--color-text-secondary); display: block; margin-top: 4px;">
                            A brief description or slogan for your site
                        </small>
                    </div>

                    <div class="form-group">
                        <label for="google_analytics_id">Google Analytics Tracking ID</label>
                        <input type="text" id="google_analytics_id" name="google_analytics_id" value="` + html.EscapeString(site.GoogleAnalyticsID) + `" placeholder="G-XXXXXXXXXX or UA-XXXXXXXXX">
                        <small style="color: var(--color-text-secondary); display: block; margin-top: 4px;">
                            Enter your Google Analytics tracking ID to enable analytics tracking
                        </small>
                    </div>

                    <div class="form-group">
                        <label for="copyright_text">Copyright Text</label>
                        <input type="text" id="copyright_text" name="copyright_text" value="` + html.EscapeString(site.CopyrightText) + `" placeholder="© 2025-2026 Your Camp Name. All rights reserved.">
                        <small style="color: var(--color-text-secondary); display: block; margin-top: 4px;">
                            Custom copyright text for your site footer. Use {year} for current year, {site} for site name.
                        </small>
                    </div>

                    <div class="form-group">
                        <label for="palette">Color Palette</label>
                        <select id="palette" name="palette">
                            ` + paletteOptions + `
                        </select>
                        <div class="help-text">Select a color scheme for your site. Changes will be visible on the public site.</div>
                    </div>

                    <div class="form-group">
                        <div class="checkbox-group">
                            <input type="checkbox" id="dark_mode" name="dark_mode" value="true"` + darkModeChecked + `>
                            <label for="dark_mode">Enable Dark Mode</label>
                        </div>
                        <div class="help-text">Switch to a dark color scheme for better viewing in low light.</div>
                    </div>

                    <div class="info-box">
                        💡 Theme changes apply immediately to your public site after saving.
                    </div>
                </div>

                <div class="button-group">
                    <button type="submit" class="btn">Save Settings</button>
                    <a href="/admin/pages" class="btn btn-secondary" style="text-decoration: none; display: inline-block; padding: var(--spacing-sm) var(--spacing-md);">Cancel</a>
                </div>
            </form>
        </div>
    </div>
</body>
</html>`

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// AdminSettingsSaveHandler saves the site information and theme settings
func AdminSettingsSaveHandler(c *gin.Context) {
	// Get site from context
	siteVal, exists := c.Get("site")
	if !exists {
		c.String(http.StatusInternalServerError, "Site not found")
		return
	}
	site := siteVal.(*models.Site)

	// Get site information form values and trim whitespace
	siteTitle := strings.TrimSpace(c.PostForm("site_title"))
	siteTagline := strings.TrimSpace(c.PostForm("site_tagline"))
	googleAnalyticsID := strings.TrimSpace(c.PostForm("google_analytics_id"))
	copyrightText := strings.TrimSpace(c.PostForm("copyright_text"))

	// Validate GA ID format if provided
	if googleAnalyticsID != "" {
		// GA4: G-XXXXXXXXXX or Universal Analytics: UA-XXXXXXXXX-X
		matched, _ := regexp.MatchString(`^(G-[A-Z0-9]+|UA-[0-9]+-[0-9]+)$`, googleAnalyticsID)
		if !matched {
			c.String(http.StatusBadRequest, "Invalid Google Analytics tracking ID format. Expected G-XXXXXXXXXX or UA-XXXXXXXXX-X")
			return
		}
	}

	// Get theme form values
	palette := c.PostForm("palette")
	darkMode := c.PostForm("dark_mode") == "true"

	// Validate palette
	validPalette := false
	palettes := themes.ListPalettes()
	for _, p := range palettes {
		if p.Name == palette {
			validPalette = true
			break
		}
	}

	// Default to slate if invalid
	if !validPalette {
		palette = "slate"
	}

	// Update site record with all fields
	site.SiteTitle = siteTitle
	site.SiteTagline = siteTagline
	site.GoogleAnalyticsID = googleAnalyticsID
	site.CopyrightText = copyrightText
	site.ThemePalette = palette
	site.DarkMode = darkMode

	if err := db.GetDB().Save(site).Error; err != nil {
		c.String(http.StatusInternalServerError, "Failed to save settings: %v", err)
		return
	}

	// Redirect back to settings page
	c.Redirect(http.StatusFound, "/admin/settings")
}
