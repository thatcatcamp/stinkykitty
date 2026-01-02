// SPDX-License-Identifier: MIT
package handlers

import (
	"fmt"
	"html"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/blocks"
	"github.com/thatcatcamp/stinkykitty/internal/config"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/email"
	"github.com/thatcatcamp/stinkykitty/internal/middleware"
	"github.com/thatcatcamp/stinkykitty/internal/models"
	"gorm.io/gorm"
)

// renderNavigationLinks generates just the navigation links (for header)
func renderNavigationLinks(siteID uint) string {
	var menuItems []models.MenuItem
	db.GetDB().Where("site_id = ?", siteID).
		Order("`order` ASC").
		Find(&menuItems)

	if len(menuItems) == 0 {
		return ""
	}

	var nav strings.Builder
	for _, item := range menuItems {
		nav.WriteString(fmt.Sprintf(
			`<a href="%s">%s</a>`,
			html.EscapeString(item.URL),
			html.EscapeString(item.Label),
		))
		nav.WriteString("\n\t\t\t")
	}

	return nav.String()
}

// renderNavigation generates the navigation menu HTML for a site
func renderNavigation(siteID uint) string {
	var menuItems []models.MenuItem
	db.GetDB().Where("site_id = ?", siteID).
		Order("`order` ASC").
		Find(&menuItems)

	if len(menuItems) == 0 {
		return ""
	}

	var nav strings.Builder
	nav.WriteString(`<nav class="site-nav">`)
	nav.WriteString(`<ul>`)

	for _, item := range menuItems {
		nav.WriteString(fmt.Sprintf(
			`<li><a href="%s">%s</a></li>`,
			html.EscapeString(item.URL),
			html.EscapeString(item.Label),
		))
	}

	nav.WriteString(`</ul>`)
	nav.WriteString(`</nav>`)

	return nav.String()
}

// getCopyrightText returns formatted copyright text with replacements
func getCopyrightText(site *models.Site) string {
	copyright := site.CopyrightText
	if copyright == "" {
		// Default copyright
		copyright = "© {year} {site}. All rights reserved."
	}

	// Escape the template first
	copyright = html.EscapeString(copyright)

	// Replace placeholders with escaped values
	currentYear := time.Now().Format("2006")
	escapedSiteTitle := html.EscapeString(site.SiteTitle)

	copyright = strings.ReplaceAll(copyright, "{year}", currentYear)
	copyright = strings.ReplaceAll(copyright, "{site}", escapedSiteTitle)

	return copyright
}

// renderHeader generates header HTML for pages
func renderHeader(site *models.Site, navigationLinks string) string {
	return fmt.Sprintf(`
<header class="site-header">
	<div class="site-header-content">
		<a href="/" class="site-header-logo">%s</a>
		<nav class="site-header-nav">
			%s
			<a href="/admin/login" class="site-header-login">Login</a>
		</nav>
	</div>
</header>`, html.EscapeString(site.SiteTitle), navigationLinks)
}

// renderFooter generates footer HTML for pages
func renderFooter(site *models.Site, includeHomeLink bool) string {
	links := ""
	if includeHomeLink {
		links = `<p style="margin: 0.5em 0 0 0;"><a href="/">Home</a></p>`
	}

	return fmt.Sprintf(`
<footer style="margin-top: 3em; padding-top: 1em; border-top: 1px solid var(--color-border); font-size: 0.9em;">
	<p style="margin: 0; font-size: 14px; color: var(--color-text-secondary);">%s</p>
	%s
</footer>`, getCopyrightText(site), links)
}

// ServeHomepage renders the site's homepage
func ServeHomepage(c *gin.Context) {
	// Get site from context (set by middleware)
	siteVal, exists := c.Get("site")
	if !exists {
		c.String(http.StatusInternalServerError, "Site not found")
		return
	}
	site := siteVal.(*models.Site)

	// Load homepage (slug = "/")
	var page models.Page
	result := db.GetDB().Where("site_id = ? AND slug = ?", site.ID, "/").
		Preload("Blocks", func(db *gorm.DB) *gorm.DB {
			return db.Order("`order` ASC")
		}).
		First(&page)

	if result.Error != nil {
		// No homepage exists yet - show placeholder
		themeCSS, _ := c.Get("themeCSS")
		themeCSSStr, _ := themeCSS.(string)

		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>%s</title>
	<style>
		%s
		body { font-family: system-ui, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
		.placeholder { text-align: center; }
	</style>
	%s
</head>
<body>
	<div class="placeholder">
		<h1>%s</h1>
		<p>This site hasn't been set up yet.</p>
		<p><a href="/admin/login">Admin Login</a></p>
	</div>
</body>
</html>
`, site.Subdomain, GetDesignSystemCSS()+"\n"+themeCSSStr, getGoogleAnalyticsScript(site), site.Subdomain)))
		return
	}

	// Render navigation links for header
	navigationLinks := renderNavigationLinks(site.ID)

	// Render all blocks
	var content strings.Builder
	for _, block := range page.Blocks {
		blockHTML, err := blocks.RenderBlock(block.Type, block.Data)
		if err != nil {
			// Log error but continue rendering other blocks
			log.Printf("Error rendering block %d: %v", block.ID, err)
			continue
		}
		content.WriteString(blockHTML)
		content.WriteString("\n")
	}

	// Get theme CSS from context
	themeCSS, _ := c.Get("themeCSS")
	themeCSSStr, _ := themeCSS.(string)

	// Wrap in HTML template
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>%s</title>
	<style>
		%s
		body { font-family: system-ui, sans-serif; max-width: 800px; margin: 0 auto; padding: 0 20px 20px; line-height: 1.6; }
		.text-block { margin-bottom: 1.5em; }

		/* Search bar styles */
		.search-bar { margin: 30px 0; }
		.search-bar form { display: flex; gap: 10px; }
		.search-bar input[type="text"] { flex: 1; padding: 10px; border: 1px solid var(--color-border); border-radius: 4px; font-size: 16px; }
		.search-bar button { padding: 10px 20px; background: var(--color-primary); color: var(--color-primary-contrast); border: none; border-radius: 4px; cursor: pointer; font-size: 16px; }
		.search-bar button:hover { opacity: 0.9; }

		/* Mobile responsive */
		@media (max-width: 600px) {
			.search-bar form { flex-direction: column; }
			.search-bar button { width: 100%%; }
		}
	</style>
	%s
</head>
<body>
	%s
	<div class="search-bar">
		<form action="/search" method="GET">
			<input type="text" name="q" placeholder="Search pages..." required>
			<button type="submit">Search</button>
		</form>
	</div>
	<h1>%s</h1>
	%s
	%s
</body>
</html>
`, page.Title, GetDesignSystemCSS()+"\n"+themeCSSStr, getGoogleAnalyticsScript(site), renderHeader(site, navigationLinks), page.Title, content.String(), renderFooter(site, false))

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// ServePage renders a page by its slug
func ServePage(c *gin.Context) {
	// Get site from context (set by middleware)
	siteVal, exists := c.Get("site")
	if !exists {
		c.String(http.StatusInternalServerError, "Site not found")
		return
	}
	site := siteVal.(*models.Site)

	// Get slug from URL path (e.g., "/about")
	slug := c.Request.URL.Path

	// Load page by slug
	var page models.Page
	result := db.GetDB().Where("site_id = ? AND slug = ? AND published = ?", site.ID, slug, true).
		Preload("Blocks", func(db *gorm.DB) *gorm.DB {
			return db.Order("`order` ASC")
		}).
		First(&page)

	if result.Error != nil {
		// Render nice 404 page
		themeCSS, _ := c.Get("themeCSS")
		themeCSSStr, _ := themeCSS.(string)

		html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>Page Not Found - %s</title>
	<style>
		%s
		body { font-family: system-ui, sans-serif; max-width: 600px; margin: 100px auto; padding: 20px; text-align: center; }
		h1 { font-size: 72px; margin: 0; color: var(--color-error); }
		h2 { font-size: 24px; margin: 20px 0; }
		p { line-height: 1.6; }
		.links { margin-top: 30px; }
		.links a { margin: 0 10px; }
	</style>
	%s
</head>
<body>
	<h1>404</h1>
	<h2>Page Not Found</h2>
	<p>The page you're looking for doesn't exist or hasn't been published yet.</p>
	<div class="links">
		<a href="/">← Home</a>
		<a href="/admin/login">Admin Login</a>
	</div>
</body>
</html>`, site.Subdomain, GetDesignSystemCSS()+"\n"+themeCSSStr, getGoogleAnalyticsScript(site))
		c.Data(http.StatusNotFound, "text/html; charset=utf-8", []byte(html))
		return
	}

	// Render navigation links for header
	navigationLinks := renderNavigationLinks(site.ID)

	// Render all blocks
	var content strings.Builder
	for _, block := range page.Blocks {
		blockHTML, err := blocks.RenderBlock(block.Type, block.Data)
		if err != nil {
			// Log error but continue rendering other blocks
			log.Printf("Error rendering block %d: %v", block.ID, err)
			continue
		}
		content.WriteString(blockHTML)
		content.WriteString("\n")
	}

	// Get theme CSS from context
	themeCSS, _ := c.Get("themeCSS")
	themeCSSStr, _ := themeCSS.(string)

	// Wrap in HTML template
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>%s</title>
	<style>
		%s
		body { font-family: system-ui, sans-serif; max-width: 800px; margin: 0 auto; padding: 0 20px 20px; line-height: 1.6; }
		.text-block { margin-bottom: 1.5em; }

		/* Search bar styles */
		.search-bar { margin: 30px 0; }
		.search-bar form { display: flex; gap: 10px; }
		.search-bar input[type="text"] { flex: 1; padding: 10px; border: 1px solid var(--color-border); border-radius: 4px; font-size: 16px; }
		.search-bar button { padding: 10px 20px; background: var(--color-primary); color: var(--color-primary-contrast); border: none; border-radius: 4px; cursor: pointer; font-size: 16px; }
		.search-bar button:hover { opacity: 0.9; }

		/* Mobile responsive */
		@media (max-width: 600px) {
			.search-bar form { flex-direction: column; }
			.search-bar button { width: 100%%; }
		}
	</style>
	%s
</head>
<body>
	%s
	<div class="search-bar">
		<form action="/search" method="GET">
			<input type="text" name="q" placeholder="Search pages..." required>
			<button type="submit">Search</button>
		</form>
	</div>
	<h1>%s</h1>
	%s
	%s
</body>
</html>
`, page.Title, GetDesignSystemCSS()+"\n"+themeCSSStr, getGoogleAnalyticsScript(site), renderHeader(site, navigationLinks), page.Title, content.String(), renderFooter(site, true))

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// ContactFormHandler displays the contact form or processes submissions
func ContactFormHandler(c *gin.Context) {
	// Get site from context
	siteVal, exists := c.Get("site")
	if !exists {
		c.String(http.StatusInternalServerError, "Site not found")
		return
	}
	site := siteVal.(*models.Site)

	// Get CSRF token
	csrfToken := middleware.GetCSRFTokenHTML(c)

	// Get theme CSS from context
	themeCSSStr := ""
	themeVal, exists := c.Get("theme_css")
	if exists {
		themeCSSStr = themeVal.(string)
	}

	// Get navigation links for header
	navigationLinks := renderNavigationLinks(site.ID)

	// Handle POST requests (form submission)
	if c.Request.Method == "POST" {
		name := strings.TrimSpace(c.PostForm("name"))
		senderEmail := strings.TrimSpace(c.PostForm("email"))
		subject := strings.TrimSpace(c.PostForm("subject"))
		message := strings.TrimSpace(c.PostForm("message"))

		// Validate form fields
		if name == "" || senderEmail == "" || subject == "" || message == "" {
			c.String(http.StatusBadRequest, "All fields are required")
			return
		}

		// Validate email format
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(senderEmail) {
			c.String(http.StatusBadRequest, "Invalid email format")
			return
		}

		// Sanitize inputs
		name = html.EscapeString(name)
		senderEmail = html.EscapeString(senderEmail)
		subject = html.EscapeString(subject)
		message = html.EscapeString(message)

		// Load site owner to get admin email
		var owner models.User
		if err := db.GetDB().First(&owner, site.OwnerID).Error; err != nil {
			log.Printf("Error loading site owner: %v", err)
			c.String(http.StatusInternalServerError, "Error processing contact form")
			return
		}

		// Send email to site owner
		svc, err := email.NewEmailService()
		if err != nil {
			log.Printf("Error creating email service: %v", err)
			// Don't expose error to user, just log it
		} else {
			emailSubject := fmt.Sprintf("Contact Form Submission: %s", subject)
			emailBody := fmt.Sprintf(`New contact form submission:

From: %s (%s)
Subject: %s

Message:
%s

---
Do not reply to this email. To respond, contact the sender at: %s`, name, senderEmail, subject, message, senderEmail)

			if err := svc.SendEmail(owner.Email, emailSubject, emailBody); err != nil {
				log.Printf("Error sending contact email: %v", err)
				// Don't expose error to user
			}
		}

		// Show success message
		successHTML := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>Thank You - %s</title>
	<style>
		%s
		body { font-family: system-ui; margin: 0; padding: 20px; }
		.container { max-width: 800px; margin: 50px auto; }
		.success-message { background: #d4edda; border: 1px solid #c3e6cb; color: #155724; padding: 15px; border-radius: 4px; margin-bottom: 20px; }
		a { color: var(--color-primary, #2563eb); text-decoration: none; }
		a:hover { text-decoration: underline; }
	</style>
</head>
<body>
	%s
	<div class="container">
		<div class="success-message">
			<strong>Thank you!</strong> Your message has been sent. We'll get back to you soon.
		</div>
		<p><a href="/">← Back to Home</a></p>
		%s
	</div>
</body>
</html>`, site.SiteTitle, themeCSSStr, renderHeader(site, navigationLinks), renderFooter(site, true))

		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(successHTML))
		return
	}

	// Display the contact form (GET request)
	formHTML := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>Contact Us - %s</title>
	<style>
		%s
		body { font-family: system-ui; margin: 0; padding: 20px; }
		.container { max-width: 600px; margin: 0 auto; }
		.form-group { margin-bottom: 20px; }
		label { display: block; margin-bottom: 5px; font-weight: 500; }
		input[type="text"],
		input[type="email"],
		textarea {
			width: 100%%;
			padding: 10px;
			border: 1px solid var(--color-border, #ddd);
			border-radius: 4px;
			font-family: inherit;
			font-size: 1em;
			box-sizing: border-box;
		}
		input[type="text"]:focus,
		input[type="email"]:focus,
		textarea:focus {
			outline: none;
			border-color: var(--color-primary, #2563eb);
			box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.1);
		}
		textarea { resize: vertical; min-height: 150px; }
		button { background: var(--color-primary, #2563eb); color: white; padding: 10px 20px; border: none; border-radius: 4px; cursor: pointer; font-size: 1em; }
		button:hover { opacity: 0.9; }
		a { color: var(--color-primary, #2563eb); text-decoration: none; }
		a:hover { text-decoration: underline; }
	</style>
</head>
<body>
	%s
	<div class="container">
		<h1>Contact Us</h1>
		<form method="POST" action="/contact">
			%s
			<div class="form-group">
				<label for="name">Name</label>
				<input type="text" id="name" name="name" required>
			</div>
			<div class="form-group">
				<label for="email">Email Address</label>
				<input type="email" id="email" name="email" required>
			</div>
			<div class="form-group">
				<label for="subject">Subject</label>
				<input type="text" id="subject" name="subject" required>
			</div>
			<div class="form-group">
				<label for="message">Message</label>
				<textarea id="message" name="message" required></textarea>
			</div>
			<div class="form-group">
				<button type="submit">Send Message</button>
			</div>
		</form>
		<p style="margin-top: 30px;"><a href="/">← Back to Home</a></p>
		%s
	</div>
</body>
</html>`, site.SiteTitle, themeCSSStr, renderHeader(site, navigationLinks), csrfToken, renderFooter(site, true))

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(formHTML))
}

// RobotsTxtHandler serves robots.txt for the site
func RobotsTxtHandler(c *gin.Context) {
	site, exists := c.Get("site")
	if !exists {
		c.String(http.StatusNotFound, "Site not found")
		return
	}
	s := site.(*models.Site)

	// Get the domain for the sitemap URL
	var domain string
	if s.CustomDomain != nil && *s.CustomDomain != "" {
		domain = *s.CustomDomain
	} else {
		baseDomain := config.GetString("server.base_domain")
		if baseDomain == "" {
			baseDomain = "localhost"
		}
		domain = s.Subdomain + "." + baseDomain
	}

	robotsTxt := fmt.Sprintf(`User-agent: *
Allow: /

Sitemap: https://%s/sitemap.xml
`, domain)

	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(robotsTxt))
}

// SitemapXMLHandler generates sitemap.xml for the site
func SitemapXMLHandler(c *gin.Context) {
	site, exists := c.Get("site")
	if !exists {
		c.String(http.StatusNotFound, "Site not found")
		return
	}
	s := site.(*models.Site)

	// Get the domain
	var domain string
	if s.CustomDomain != nil && *s.CustomDomain != "" {
		domain = *s.CustomDomain
	} else {
		baseDomain := config.GetString("server.base_domain")
		if baseDomain == "" {
			baseDomain = "localhost"
		}
		domain = s.Subdomain + "." + baseDomain
	}

	// Get all pages for this site
	var pages []models.Page
	db.GetDB().Where("site_id = ?", s.ID).Order("updated_at DESC").Find(&pages)

	// Build sitemap XML
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
`

	// Add each page to the sitemap
	for _, page := range pages {
		var url string
		if page.Slug == "home" || page.Slug == "" {
			url = fmt.Sprintf("https://%s/", domain)
		} else {
			url = fmt.Sprintf("https://%s/%s", domain, page.Slug)
		}

		// Format the last modified date in W3C format
		lastmod := page.UpdatedAt.Format("2006-01-02T15:04:05-07:00")

		// Set priority based on whether it's the homepage
		priority := "0.8"
		if page.Slug == "home" || page.Slug == "" {
			priority = "1.0"
		}

		xml += fmt.Sprintf(`  <url>
    <loc>%s</loc>
    <lastmod>%s</lastmod>
    <changefreq>weekly</changefreq>
    <priority>%s</priority>
  </url>
`, url, lastmod, priority)
	}

	xml += `</urlset>`

	c.Data(http.StatusOK, "application/xml; charset=utf-8", []byte(xml))
}

// getGoogleAnalyticsScript returns GA tracking script if configured
func getGoogleAnalyticsScript(site *models.Site) string {
	if site.GoogleAnalyticsID == "" {
		return ""
	}

	// Sanitize and validate the GA ID
	gaID := strings.TrimSpace(site.GoogleAnalyticsID)
	if gaID == "" {
		log.Printf("Site %d: Empty Google Analytics ID after trimming", site.ID)
		return ""
	}

	// Validate GA ID format: G-XXXXXXXXXX (GA4) or UA-XXXXXXXXX-X (Universal)
	// This prevents XSS attacks and ensures only valid GA IDs are used
	validFormat := regexp.MustCompile(`^(G|UA)-[A-Z0-9\-]+$`)
	if !validFormat.MatchString(gaID) {
		log.Printf("Site %d: Invalid GA ID format: %s", site.ID, gaID)
		return ""
	}

	// Escape for JavaScript context as defense-in-depth
	escapedGAID := template.JSEscapeString(gaID)

	return fmt.Sprintf(`
<!-- Google Analytics -->
<script async src="https://www.googletagmanager.com/gtag/js?id=%s"></script>
<script>
  window.dataLayer = window.dataLayer || [];
  function gtag(){dataLayer.push(arguments);}
  gtag('js', new Date());
  gtag('config', '%s');
</script>
`, escapedGAID, escapedGAID)
}
