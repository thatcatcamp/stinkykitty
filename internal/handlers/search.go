package handlers

import (
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/models"
	"github.com/thatcatcamp/stinkykitty/internal/search"
)

// SearchHandler handles search requests for a site
func SearchHandler(c *gin.Context) {
	// Get site from context
	siteVal, exists := c.Get("site")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Site not found"})
		return
	}
	site := siteVal.(*models.Site)

	// Get query parameter
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter 'q' is required"})
		return
	}

	// Perform search
	results, err := search.Search(db.GetDB(), site.ID, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}

	// Render navigation
	navigation := renderNavigation(site.ID)

	// Get theme CSS from context
	themeCSS, _ := c.Get("themeCSS")
	themeCSSStr, _ := themeCSS.(string)

	// Build results HTML
	var resultsHTML strings.Builder
	if len(results) == 0 {
		resultsHTML.WriteString(`<div class="no-results">
			<p>No results found for your search.</p>
			<p><a href="/">← Back to home</a></p>
		</div>`)
	} else {
		resultsHTML.WriteString(`<div class="search-results">`)
		for _, result := range results {
			resultsHTML.WriteString(fmt.Sprintf(`
				<div class="search-result">
					<h2><a href="%s">%s</a></h2>
					<p class="snippet">%s</p>
					<p class="url">%s</p>
				</div>
			`, html.EscapeString(result.URL), html.EscapeString(result.Title), result.Snippet, html.EscapeString(result.URL)))
		}
		resultsHTML.WriteString(`</div>`)
	}

	// Render search results page
	htmlOutput := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>Search: %s - %s</title>
	<style>
		%s
		body { font-family: system-ui, sans-serif; max-width: 800px; margin: 0 auto; padding: 0 20px 20px; line-height: 1.6; }

		/* Navigation styles */
		.site-nav { border-bottom: 2px solid var(--color-border); margin: 0 -20px 30px; padding: 0 20px; }
		.site-nav ul { list-style: none; margin: 0; padding: 0; display: flex; flex-wrap: wrap; }
		.site-nav li { margin: 0; }
		.site-nav a { display: block; padding: 15px 20px; text-decoration: none; transition: background-color 0.2s; }
		.site-nav a:hover { opacity: 0.8; }

		/* Search bar styles - primary button styling */
		.search-bar { margin: 30px 0; }
		.search-bar form { display: flex; gap: 10px; }
		.search-bar input[type="text"] {
			flex: 1;
			padding: 10px;
			border: 1px solid var(--color-border);
			background-color: var(--color-surface);
			color: var(--color-text);
			border-radius: 4px;
			font-size: 16px;
		}
		.search-bar input[type="text"]:focus {
			outline: none;
			border-color: var(--color-primary);
			box-shadow: 0 0 0 3px rgba(0, 0, 0, 0.05);
		}
		.search-bar button {
			padding: 10px 20px;
			background: var(--color-primary);
			color: white;
			border: none;
			border-radius: 4px;
			cursor: pointer;
			font-size: 16px;
			transition: opacity 0.2s;
		}
		.search-bar button:hover { opacity: 0.9; }

		/* Search results styles */
		.search-header { margin-bottom: 30px; }
		.search-header h1 { margin-bottom: 10px; color: var(--color-text); }
		.search-header .count { color: var(--color-text-muted); font-size: 0.9em; }

		/* Result item - surface background, border */
		.search-result {
			margin-bottom: 30px;
			padding: 20px;
			background-color: var(--color-surface);
			border: 1px solid var(--color-border);
			border-radius: 8px;
		}
		.search-result:last-child { margin-bottom: 0; }
		.search-result h2 { margin: 0 0 10px 0; font-size: 1.3em; }
		.search-result h2 a { color: var(--color-primary); text-decoration: none; }
		.search-result h2 a:hover { text-decoration: underline; }
		.search-result .snippet { margin: 10px 0; color: var(--color-text); line-height: 1.5; }

		/* Highlighted search terms - secondary color */
		.search-result .snippet mark {
			background: var(--color-secondary);
			color: var(--color-bg);
			padding: 2px 4px;
			border-radius: 2px;
			font-weight: 600;
		}
		.search-result .url { font-size: 0.85em; color: var(--color-text-muted); margin: 5px 0 0 0; }

		.no-results { text-align: center; margin: 50px 0; padding: 40px; background-color: var(--color-surface); border: 1px solid var(--color-border); border-radius: 8px; }
		.no-results p { margin: 20px 0; color: var(--color-text); }
		.no-results a { color: var(--color-primary); }

		/* Footer links use primary color */
		footer { margin-top: 3em; padding-top: 1em; border-top: 1px solid var(--color-border); font-size: 0.9em; }
		footer a { color: var(--color-primary); text-decoration: none; }
		footer a:hover { text-decoration: underline; }

		/* Mobile responsive */
		@media (max-width: 600px) {
			.site-nav ul { flex-direction: column; }
			.site-nav a { padding: 12px 15px; border-bottom: 1px solid var(--color-border); }
			.search-bar form { flex-direction: column; }
			.search-bar button { width: 100%%; }
		}
	</style>
</head>
<body>
	%s
	<div class="search-bar">
		<form action="/search" method="GET">
			<input type="text" name="q" placeholder="Search pages..." value="%s" required>
			<button type="submit">Search</button>
		</form>
	</div>
	<div class="search-header">
		<h1>Search Results</h1>
		<p class="count">Found %d result(s) for "%s"</p>
	</div>
	%s
	<footer>
		<a href="/">← Home</a> | <a href="/admin/login">Admin Login</a>
	</footer>
</body>
</html>
`, html.EscapeString(query), html.EscapeString(site.Subdomain), themeCSSStr, navigation, html.EscapeString(query), len(results), html.EscapeString(query), resultsHTML.String())

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(htmlOutput))
}
