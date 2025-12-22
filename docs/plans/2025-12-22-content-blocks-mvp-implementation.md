# Content Blocks MVP Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build the core CMS with Pages and Text blocks - allow camp organizers to create and edit content through a simple admin interface.

**Architecture:** Server-rendered HTML forms for editing, JSON storage for block data, extensible block renderer pattern. Homepage auto-created for each site, users can add pages as needed.

**Tech Stack:** Go 1.25+, Gin, GORM, SQLite, server-rendered HTML templates

---

## Task 1: Database Models for Pages and Blocks

**Files:**
- Modify: `internal/models/models.go`
- Create: `internal/models/models_test.go` (if doesn't exist, or add to existing)

### Step 1: Write failing test for Page model

Add to `internal/models/models_test.go`:

```go
func TestPageModel(t *testing.T) {
	database := setupTestDB(t)

	// Create a site first
	site := Site{Subdomain: "testcamp"}
	database.Create(&site)

	// Create a page
	page := Page{
		SiteID:    site.ID,
		Slug:      "/",
		Title:     "Homepage",
		Published: false,
	}

	result := database.Create(&page)
	if result.Error != nil {
		t.Fatalf("Failed to create page: %v", result.Error)
	}

	// Verify it was saved
	var fetched Page
	database.First(&fetched, page.ID)

	if fetched.Slug != "/" {
		t.Errorf("Expected slug '/', got '%s'", fetched.Slug)
	}
	if fetched.Title != "Homepage" {
		t.Errorf("Expected title 'Homepage', got '%s'", fetched.Title)
	}
}

func TestBlockModel(t *testing.T) {
	database := setupTestDB(t)

	// Create site and page
	site := Site{Subdomain: "testcamp"}
	database.Create(&site)

	page := Page{SiteID: site.ID, Slug: "/", Title: "Home"}
	database.Create(&page)

	// Create a block
	block := Block{
		PageID: page.ID,
		Type:   "text",
		Order:  0,
		Data:   `{"content":"Hello world"}`,
	}

	result := database.Create(&block)
	if result.Error != nil {
		t.Fatalf("Failed to create block: %v", result.Error)
	}

	// Verify it was saved
	var fetched Block
	database.First(&fetched, block.ID)

	if fetched.Type != "text" {
		t.Errorf("Expected type 'text', got '%s'", fetched.Type)
	}
	if fetched.Order != 0 {
		t.Errorf("Expected order 0, got %d", fetched.Order)
	}
}

func TestPageBlockRelationship(t *testing.T) {
	database := setupTestDB(t)

	site := Site{Subdomain: "testcamp"}
	database.Create(&site)

	page := Page{SiteID: site.ID, Slug: "/", Title: "Home"}
	database.Create(&page)

	// Create multiple blocks
	block1 := Block{PageID: page.ID, Type: "text", Order: 0, Data: `{"content":"First"}`}
	block2 := Block{PageID: page.ID, Type: "text", Order: 1, Data: `{"content":"Second"}`}
	database.Create(&block1)
	database.Create(&block2)

	// Load page with blocks
	var fetchedPage Page
	database.Preload("Blocks").First(&fetchedPage, page.ID)

	if len(fetchedPage.Blocks) != 2 {
		t.Errorf("Expected 2 blocks, got %d", len(fetchedPage.Blocks))
	}
}
```

### Step 2: Run test to verify it fails

Run:
```bash
go test ./internal/models -v -run TestPage
```

Expected output: Compilation errors - `Page` and `Block` types not defined

### Step 3: Add Page and Block models

Add to `internal/models/models.go`:

```go
// Page represents a content page on a site
type Page struct {
	ID        uint           `gorm:"primaryKey"`
	SiteID    uint           `gorm:"not null;index:idx_site_slug,unique"`
	Slug      string         `gorm:"not null;index:idx_site_slug,unique"` // "/" for homepage, "/about", etc
	Title     string         `gorm:"not null"`
	Published bool           `gorm:"default:false"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Site   Site    `gorm:"foreignKey:SiteID"`
	Blocks []Block `gorm:"foreignKey:PageID;constraint:OnDelete:CASCADE"`
}

// Block represents a content block on a page
type Block struct {
	ID        uint           `gorm:"primaryKey"`
	PageID    uint           `gorm:"not null;index"`
	Type      string         `gorm:"not null"` // "text", "hero", "gallery", etc
	Order     int            `gorm:"not null;default:0"`
	Data      string         `gorm:"type:text"` // JSON blob with block-specific content
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Page Page `gorm:"foreignKey:PageID"`
}
```

### Step 4: Update AutoMigrate to include new models

Find the AutoMigrate call in the codebase and add Page and Block:

In files that call AutoMigrate (like test setup, initSystemDB, etc), update to:
```go
database.AutoMigrate(&models.User{}, &models.Site{}, &models.SiteUser{}, &models.Page{}, &models.Block{})
```

### Step 5: Run test to verify it passes

Run:
```bash
go test ./internal/models -v -run TestPage
go test ./internal/models -v -run TestBlock
```

Expected output: All tests PASS

### Step 6: Commit

```bash
git add internal/models/models.go internal/models/models_test.go
git commit -m "feat: add Page and Block models for content management

Pages belong to sites, have unique slugs per site.
Blocks belong to pages, ordered by Order field, store content as JSON."
```

---

## Task 2: Block Renderer Package

**Files:**
- Create: `internal/blocks/renderer.go`
- Create: `internal/blocks/renderer_test.go`

### Step 1: Write failing test for block renderer

Create `internal/blocks/renderer_test.go`:

```go
package blocks

import (
	"strings"
	"testing"
)

func TestRenderTextBlock(t *testing.T) {
	dataJSON := `{"content":"Hello world"}`
	html, err := RenderBlock("text", dataJSON)

	if err != nil {
		t.Fatalf("RenderBlock failed: %v", err)
	}

	if !strings.Contains(html, "Hello world") {
		t.Errorf("Expected HTML to contain 'Hello world', got: %s", html)
	}

	if !strings.Contains(html, `class="text-block"`) {
		t.Errorf("Expected HTML to have text-block class, got: %s", html)
	}
}

func TestRenderTextBlockWithLineBreaks(t *testing.T) {
	dataJSON := `{"content":"Line 1\nLine 2"}`
	html, err := RenderBlock("text", dataJSON)

	if err != nil {
		t.Fatalf("RenderBlock failed: %v", err)
	}

	if !strings.Contains(html, "<br>") {
		t.Errorf("Expected HTML to contain <br> for line breaks, got: %s", html)
	}
}

func TestRenderTextBlockEscapesHTML(t *testing.T) {
	dataJSON := `{"content":"<script>alert('xss')</script>"}`
	html, err := RenderBlock("text", dataJSON)

	if err != nil {
		t.Fatalf("RenderBlock failed: %v", err)
	}

	if strings.Contains(html, "<script>") {
		t.Errorf("HTML should be escaped, got: %s", html)
	}

	if !strings.Contains(html, "&lt;script&gt;") {
		t.Errorf("Expected escaped HTML, got: %s", html)
	}
}

func TestRenderUnknownBlockType(t *testing.T) {
	_, err := RenderBlock("unknown", `{}`)

	if err == nil {
		t.Error("Expected error for unknown block type")
	}
}
```

### Step 2: Run test to verify it fails

Run:
```bash
go test ./internal/blocks -v
```

Expected output: Compilation error - `RenderBlock` not defined

### Step 3: Implement block renderer

Create `internal/blocks/renderer.go`:

```go
package blocks

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"
)

// RenderBlock renders a block to HTML based on its type and data
func RenderBlock(blockType string, dataJSON string) (string, error) {
	switch blockType {
	case "text":
		return renderTextBlock(dataJSON)
	default:
		return "", fmt.Errorf("unknown block type: %s", blockType)
	}
}

// TextBlockData represents the JSON structure for text blocks
type TextBlockData struct {
	Content string `json:"content"`
}

// renderTextBlock renders a text block with HTML escaping and line break preservation
func renderTextBlock(dataJSON string) (string, error) {
	var data TextBlockData
	if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
		return "", fmt.Errorf("failed to parse text block data: %w", err)
	}

	// Escape HTML to prevent XSS
	safe := html.EscapeString(data.Content)

	// Preserve line breaks
	formatted := strings.ReplaceAll(safe, "\n", "<br>")

	return fmt.Sprintf(`<div class="text-block">%s</div>`, formatted), nil
}
```

### Step 4: Run test to verify it passes

Run:
```bash
go test ./internal/blocks -v
```

Expected output: All tests PASS

### Step 5: Commit

```bash
git add internal/blocks/
git commit -m "feat: add block renderer with text block support

Renders blocks to HTML with proper escaping and formatting.
Text blocks preserve line breaks and escape HTML to prevent XSS."
```

---

## Task 3: Update Homepage to Render from Database

**Files:**
- Modify: `internal/handlers/public.go`

### Step 1: Update ServeHomepage to render blocks

In `internal/handlers/public.go`, replace the existing ServeHomepage function:

```go
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
```

### Step 2: Add necessary imports

Add to imports in `internal/handlers/public.go`:

```go
import (
	"fmt"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/blocks"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/models"
	"gorm.io/gorm"
)
```

### Step 3: Test manually

Run:
```bash
make build
./stinky server start
```

Visit http://localhost:17890 - should see placeholder page (no homepage created yet)

### Step 4: Commit

```bash
git add internal/handlers/public.go
git commit -m "feat: update homepage to render from database

Loads homepage from database and renders blocks.
Shows placeholder if no homepage exists yet."
```

---

## Task 4: Admin Dashboard Handler

**Files:**
- Modify: `internal/handlers/admin.go`

### Step 1: Update DashboardHandler to show pages list

Replace the existing DashboardHandler in `internal/handlers/admin.go`:

```go
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
```

### Step 2: Add imports

Make sure these imports are in `internal/handlers/admin.go`:

```go
import (
	"fmt"
	"strings"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/auth"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/models"
)
```

### Step 3: Test manually

Run server, log in, visit /admin/dashboard - should see pages list with "Create Homepage" button

### Step 4: Commit

```bash
git add internal/handlers/admin.go
git commit -m "feat: add pages list to admin dashboard

Shows homepage and additional pages with edit/delete actions.
Allows creating homepage if it doesn't exist."
```

---

## Task 5: Create Page Handler

**Files:**
- Create: `internal/handlers/admin_pages.go`

### Step 1: Create page creation handler

Create `internal/handlers/admin_pages.go`:

```go
package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/models"
)

// CreatePageHandler creates a new page
func CreatePageHandler(c *gin.Context) {
	siteVal, _ := c.Get("site")
	site := siteVal.(*models.Site)

	slug := c.PostForm("slug")
	title := c.PostForm("title")

	// Validate
	if slug == "" {
		c.String(http.StatusBadRequest, "Slug is required")
		return
	}
	if title == "" {
		c.String(http.StatusBadRequest, "Title is required")
		return
	}

	// Check if page with this slug already exists
	var existing models.Page
	result := db.GetDB().Where("site_id = ? AND slug = ?", site.ID, slug).First(&existing)
	if result.Error == nil {
		c.String(http.StatusBadRequest, "Page with this slug already exists")
		return
	}

	// Create page
	page := models.Page{
		SiteID:    site.ID,
		Slug:      slug,
		Title:     title,
		Published: false,
	}

	if err := db.GetDB().Create(&page).Error; err != nil {
		c.String(http.StatusInternalServerError, "Failed to create page")
		return
	}

	// Redirect to edit page
	c.Redirect(http.StatusFound, "/admin/pages/"+strconv.Itoa(int(page.ID))+"/edit")
}
```

### Step 2: Add route to server.go

In `cmd/stinky/server.go`, add to admin routes:

```go
adminGroup.POST("/pages", handlers.CreatePageHandler)
```

### Step 3: Test manually

From dashboard, click "Create Homepage" - should create page and redirect to editor

### Step 4: Commit

```bash
git add internal/handlers/admin_pages.go cmd/stinky/server.go
git commit -m "feat: add page creation handler

Allows creating new pages via POST /admin/pages.
Validates slug uniqueness per site."
```

---

*Due to token limits, I'll create the rest of the implementation plan in a follow-up. The remaining tasks will cover:*

- Task 6: Edit Page Handler (show page editor with blocks list)
- Task 7: Create Block Handler
- Task 8: Edit Block Handler
- Task 9: Update Block Handler
- Task 10: Delete Block Handler
- Task 11: Move Block Up/Down Handlers
- Task 12: Publish Page Handler
- Task 13: Integration testing

Let me save this partial plan and continue...
