package handlers

import (
	"fmt"
	"html"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/blocks"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/models"
	"gorm.io/gorm"
)

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
</head>
<body>
	<div class="placeholder">
		<h1>%s</h1>
		<p>This site hasn't been set up yet.</p>
		<p><a href="/admin/login">Admin Login</a></p>
	</div>
</body>
</html>
`, site.Subdomain, themeCSSStr, site.Subdomain)))
		return
	}

	// Render navigation
	navigation := renderNavigation(site.ID)

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

		/* Navigation styles */
		.site-nav { border-bottom: 2px solid var(--color-border); margin: 0 -20px 30px; padding: 0 20px; }
		.site-nav ul { list-style: none; margin: 0; padding: 0; display: flex; flex-wrap: wrap; }
		.site-nav li { margin: 0; }
		.site-nav a { display: block; padding: 15px 20px; text-decoration: none; transition: background-color 0.2s; }
		.site-nav a:hover { opacity: 0.8; }

		/* Mobile responsive */
		@media (max-width: 600px) {
			.site-nav ul { flex-direction: column; }
			.site-nav a { padding: 12px 15px; border-bottom: 1px solid var(--color-border); }
		}
	</style>
</head>
<body>
	%s
	<h1>%s</h1>
	%s
	<footer style="margin-top: 3em; padding-top: 1em; border-top: 1px solid var(--color-border); font-size: 0.9em;">
		<a href="/admin/login">Admin Login</a>
	</footer>
</body>
</html>
`, page.Title, themeCSSStr, navigation, page.Title, content.String())

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
</html>`, site.Subdomain, themeCSSStr)
		c.Data(http.StatusNotFound, "text/html; charset=utf-8", []byte(html))
		return
	}

	// Render navigation
	navigation := renderNavigation(site.ID)

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

		/* Navigation styles */
		.site-nav { border-bottom: 2px solid var(--color-border); margin: 0 -20px 30px; padding: 0 20px; }
		.site-nav ul { list-style: none; margin: 0; padding: 0; display: flex; flex-wrap: wrap; }
		.site-nav li { margin: 0; }
		.site-nav a { display: block; padding: 15px 20px; text-decoration: none; transition: background-color 0.2s; }
		.site-nav a:hover { opacity: 0.8; }

		/* Mobile responsive */
		@media (max-width: 600px) {
			.site-nav ul { flex-direction: column; }
			.site-nav a { padding: 12px 15px; border-bottom: 1px solid var(--color-border); }
		}
	</style>
</head>
<body>
	%s
	<h1>%s</h1>
	%s
	<footer style="margin-top: 3em; padding-top: 1em; border-top: 1px solid var(--color-border); font-size: 0.9em;">
		<a href="/">Home</a> | <a href="/admin/login">Admin Login</a>
	</footer>
</body>
</html>
`, page.Title, themeCSSStr, navigation, page.Title, content.String())

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
