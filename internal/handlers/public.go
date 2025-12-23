package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/blocks"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/models"
	"gorm.io/gorm"
)

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
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>%s</title>
	<style>
		body { font-family: system-ui, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
		.placeholder { text-align: center; color: #666; }
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
`, site.Subdomain, site.Subdomain)))
		return
	}

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

	// Wrap in HTML template
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>%s</title>
	<style>
		body { font-family: system-ui, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; line-height: 1.6; }
		.text-block { margin-bottom: 1.5em; }
	</style>
</head>
<body>
	<h1>%s</h1>
	%s
	<footer style="margin-top: 3em; padding-top: 1em; border-top: 1px solid #ddd; font-size: 0.9em; color: #666;">
		<a href="/admin/login">Admin Login</a>
	</footer>
</body>
</html>
`, page.Title, page.Title, content.String())

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
		c.String(http.StatusNotFound, "Page not found")
		return
	}

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

	// Wrap in HTML template
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>%s</title>
	<style>
		body { font-family: system-ui, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; line-height: 1.6; }
		.text-block { margin-bottom: 1.5em; }
	</style>
</head>
<body>
	<h1>%s</h1>
	%s
	<footer style="margin-top: 3em; padding-top: 1em; border-top: 1px solid #ddd; font-size: 0.9em; color: #666;">
		<a href="/">Home</a> | <a href="/admin/login">Admin Login</a>
	</footer>
</body>
</html>
`, page.Title, page.Title, content.String())

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
