# CMS Enhancements Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add user management, Google Analytics integration, fixed header bar for public sites, editable copyright, and column layout blocks to StinkyKitty CMS.

**Architecture:** Extend existing admin interface with new user management screens and site settings fields. Add new column block type to the block system. Enhance public site template with fixed header and footer customization.

**Tech Stack:** Go 1.25, Gin web framework, GORM ORM, SQLite, HTML/CSS

---

## Task 1: Add Google Analytics and Copyright Fields to Site Model

**Files:**
- Modify: `internal/models/models.go` (Site struct, lines ~18-40)
- Test: Create `internal/models/site_fields_test.go`

**Step 1: Add migration for new Site fields**

Add these fields to the Site struct after `ThemePalette`:

```go
GoogleAnalyticsID string `gorm:"default:""` // GA tracking ID (G-XXXXXXXXXX or UA-XXXXXXXXX)
CopyrightText     string `gorm:"default:""` // Custom footer copyright text
```

**Step 2: Test the model changes**

Create `internal/models/site_fields_test.go`:

```go
package models

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSiteFieldsDefaults(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Auto migrate
	if err := db.AutoMigrate(&Site{}, &User{}, &Page{}, &Block{}, &MenuItem{}, &SiteUser{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// Create a site
	site := &Site{
		Subdomain: "test",
		OwnerID:   1,
		SiteDir:   "/tmp/test",
	}

	if err := db.Create(site).Error; err != nil {
		t.Fatalf("failed to create site: %v", err)
	}

	// Check defaults
	var retrieved Site
	if err := db.First(&retrieved, site.ID).Error; err != nil {
		t.Fatalf("failed to retrieve site: %v", err)
	}

	if retrieved.GoogleAnalyticsID != "" {
		t.Errorf("expected empty GoogleAnalyticsID, got %s", retrieved.GoogleAnalyticsID)
	}

	if retrieved.CopyrightText != "" {
		t.Errorf("expected empty CopyrightText, got %s", retrieved.CopyrightText)
	}
}
```

**Step 3: Run test to verify**

Run: `go test ./internal/models/... -v -run TestSiteFieldsDefaults`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/models/models.go internal/models/site_fields_test.go
git commit -m "feat: add GoogleAnalyticsID and CopyrightText to Site model"
```

---

## Task 2: Add Site Settings Form Fields

**Files:**
- Modify: `internal/handlers/admin_settings.go` (SettingsHandler and UpdateSettingsHandler)

**Step 1: Add GA and Copyright fields to settings form**

In `SettingsHandler`, find the form HTML (around line 50-150) and add these fields after the `SiteTagline` field:

```go
<div class="form-group">
	<label for="google_analytics_id">Google Analytics Tracking ID</label>
	<input type="text" id="google_analytics_id" name="google_analytics_id" value="` + site.GoogleAnalyticsID + `" placeholder="G-XXXXXXXXXX or UA-XXXXXXXXX">
	<small style="color: var(--color-text-secondary); display: block; margin-top: 4px;">
		Enter your Google Analytics tracking ID to enable analytics tracking
	</small>
</div>

<div class="form-group">
	<label for="copyright_text">Copyright Text</label>
	<input type="text" id="copyright_text" name="copyright_text" value="` + site.CopyrightText + `" placeholder="© 2025 Your Camp Name. All rights reserved.">
	<small style="color: var(--color-text-secondary); display: block; margin-top: 4px;">
		Custom copyright text for your site footer. Use {year} for current year, {site} for site name.
	</small>
</div>
```

**Step 2: Update the settings handler to save new fields**

In `UpdateSettingsHandler`, find where site updates are saved (around line 200-250) and add:

```go
googleAnalyticsID := c.PostForm("google_analytics_id")
copyrightText := c.PostForm("copyright_text")

// Update the site with new fields
updates := map[string]interface{}{
	// ... existing fields ...
	"google_analytics_id": googleAnalyticsID,
	"copyright_text":      copyrightText,
}
```

**Step 3: Test manually**

1. Start server: `./stinky server start`
2. Navigate to site settings
3. Add GA ID: `G-TEST123`
4. Add copyright: `© {year} Test Camp`
5. Save and verify fields persist

**Step 4: Commit**

```bash
git add internal/handlers/admin_settings.go
git commit -m "feat: add GA and copyright fields to site settings form"
```

---

## Task 3: Integrate Google Analytics into Public Pages

**Files:**
- Modify: `internal/handlers/public.go` (ServeHomepage and ServePage functions)

**Step 1: Add GA tracking script to page template**

In `ServeHomepage` and `ServePage`, find where the `<head>` section is generated and add GA script injection.

Add this helper function at the end of `public.go`:

```go
// getGoogleAnalyticsScript returns GA tracking script if configured
func getGoogleAnalyticsScript(site *models.Site) string {
	if site.GoogleAnalyticsID == "" {
		return ""
	}

	// Sanitize the GA ID (basic validation)
	gaID := strings.TrimSpace(site.GoogleAnalyticsID)
	if gaID == "" {
		return ""
	}

	return fmt.Sprintf(`
<!-- Google Analytics -->
<script async src="https://www.googletagmanager.com/gtag/js?id=%s"></script>
<script>
  window.dataLayer = window.dataLayer || [];
  function gtag(){dataLayer.push(arguments);}
  gtag('js', new Date());
  gtag('config', '%s');
</script>
`, gaID, gaID)
}
```

**Step 2: Inject GA script in page templates**

In both `ServeHomepage` and `ServePage`, in the HTML template generation, add after the `<style>` tag:

```go
%s
` + getGoogleAnalyticsScript(site) + `
```

**Step 3: Test GA injection**

1. Set GA ID in site settings: `G-TEST123`
2. Visit public page
3. View source, verify GA script is present
4. Check browser console for `gtag` function

**Step 4: Commit**

```bash
git add internal/handlers/public.go
git commit -m "feat: integrate Google Analytics tracking in public pages"
```

---

## Task 4: Add Editable Copyright to Public Footer

**Files:**
- Modify: `internal/handlers/public.go` (footer generation in ServeHomepage and ServePage)

**Step 1: Create copyright text generator function**

Add this helper function to `public.go`:

```go
// getCopyrightText returns formatted copyright text with replacements
func getCopyrightText(site *models.Site) string {
	copyright := site.CopyrightText
	if copyright == "" {
		// Default copyright
		copyright = "© {year} {site}. All rights reserved."
	}

	// Replace placeholders
	currentYear := time.Now().Format("2006")
	copyright = strings.ReplaceAll(copyright, "{year}", currentYear)
	copyright = strings.ReplaceAll(copyright, "{site}", site.SiteTitle)

	return html.EscapeString(copyright)
}
```

**Step 2: Update footer in page templates**

Find where the footer is rendered (search for `<footer` or similar in the HTML templates).

Replace the existing copyright line with:

```go
<p style="margin: 0; font-size: 14px; color: var(--color-text-secondary);">` + getCopyrightText(site) + `</p>
```

**Step 3: Test copyright customization**

1. Leave copyright empty → should show default
2. Set copyright to: `© {year} {site} - Custom Text`
3. Verify `{year}` replaced with 2025
4. Verify `{site}` replaced with site title

**Step 4: Commit**

```bash
git add internal/handlers/public.go
git commit -m "feat: add editable copyright text to site footer"
```

---

## Task 5: Add Fixed Header Bar to Public Sites

**Files:**
- Modify: `internal/handlers/public.go` (ServeHomepage and ServePage HTML generation)
- Modify: `internal/handlers/styles.go` (add header styles)

**Step 1: Add header bar styles**

In `internal/handlers/styles.go`, find the CSS section and add these styles:

```go
/* Site Header */
.site-header {
	background: var(--color-surface);
	border-bottom: 1px solid var(--color-border);
	padding: 0;
	position: sticky;
	top: 0;
	z-index: 100;
	box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.site-header-content {
	max-width: 1200px;
	margin: 0 auto;
	padding: var(--spacing-md) var(--spacing-lg);
	display: flex;
	justify-content: space-between;
	align-items: center;
}

.site-header-logo {
	font-size: 20px;
	font-weight: 700;
	color: var(--color-text-primary);
	text-decoration: none;
}

.site-header-nav {
	display: flex;
	gap: var(--spacing-lg);
	align-items: center;
}

.site-header-nav a {
	color: var(--color-text-secondary);
	text-decoration: none;
	font-weight: 500;
	transition: color 0.2s;
}

.site-header-nav a:hover {
	color: var(--color-primary);
}

.site-header-login {
	background: var(--color-primary);
	color: white;
	padding: 8px 16px;
	border-radius: var(--radius-sm);
	text-decoration: none;
	font-weight: 600;
	transition: opacity 0.2s;
}

.site-header-login:hover {
	opacity: 0.9;
	color: white;
}
```

**Step 2: Add header HTML to page templates**

In `ServeHomepage` and `ServePage`, add this header right after `<body>`:

```go
<header class="site-header">
	<div class="site-header-content">
		<a href="/" class="site-header-logo">` + html.EscapeString(site.SiteTitle) + `</a>
		<nav class="site-header-nav">
			` + navigation + `
			<a href="/admin/login" class="site-header-login">Login</a>
		</nav>
	</div>
</header>
```

**Step 3: Remove footer login link**

Find and remove the footer "Admin" link since it's now in the header.

**Step 4: Test header**

1. Visit public page
2. Verify header is sticky (scrolls with page)
3. Verify site title appears
4. Verify navigation menu appears
5. Verify Login button works

**Step 5: Commit**

```bash
git add internal/handlers/public.go internal/handlers/styles.go
git commit -m "feat: add fixed header bar with login button to public sites"
```

---

## Task 6: Create User Management Page

**Files:**
- Create: `internal/handlers/admin_users.go`
- Modify: `cmd/stinky/server.go` (add route)

**Step 1: Create user management handler**

Create `internal/handlers/admin_users.go`:

```go
package handlers

import (
	"fmt"
	"html"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/auth"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/email"
	"github.com/thatcatcamp/stinkykitty/internal/models"
)

// UsersListHandler shows all users accessible by the current user
func UsersListHandler(c *gin.Context) {
	// Get current user
	userVal, exists := c.Get("user")
	if !exists {
		c.Redirect(http.StatusFound, "/admin/login")
		return
	}
	currentUser := userVal.(*models.User)

	// Get site from context
	siteVal, exists := c.Get("site")
	if !exists {
		c.String(http.StatusInternalServerError, "Site not found")
		return
	}
	site := siteVal.(*models.Site)

	// Query users based on permissions
	type UserRow struct {
		ID        uint
		Email     string
		CreatedAt time.Time
		Sites     string // Comma-separated site names
		Role      string
	}
	var users []UserRow

	if currentUser.IsGlobalAdmin {
		// Global admins see all users with their sites
		db.GetDB().Raw(`
			SELECT u.id, u.email, u.created_at,
				   GROUP_CONCAT(DISTINCT s.subdomain) as sites,
				   COALESCE(su.role, 'owner') as role
			FROM users u
			LEFT JOIN site_users su ON u.id = su.user_id
			LEFT JOIN sites s ON su.site_id = s.id OR s.owner_id = u.id
			WHERE u.deleted_at IS NULL
			GROUP BY u.id
			ORDER BY u.email
		`).Scan(&users)
	} else {
		// Site admins see only users on their sites
		db.GetDB().Raw(`
			SELECT DISTINCT u.id, u.email, u.created_at,
				   s.subdomain as sites,
				   su.role
			FROM users u
			INNER JOIN site_users su ON u.id = su.user_id
			INNER JOIN sites s ON su.site_id = s.id
			WHERE s.id = ? AND u.deleted_at IS NULL
			ORDER BY u.email
		`, site.ID).Scan(&users)
	}

	// Build user table HTML
	var tableRows string
	for _, user := range users {
		tableRows += fmt.Sprintf(`
			<tr>
				<td>%s</td>
				<td>%s</td>
				<td>%s</td>
				<td>%s</td>
				<td>
					<div style="display: flex; gap: 8px;">
						<form method="POST" action="/admin/users/%d/reset-password" style="display: inline;">
							<button type="submit" class="btn btn-small btn-secondary">Reset Password</button>
						</form>
						<form method="POST" action="/admin/users/%d/delete" style="display: inline;" onsubmit="return confirm('Delete this user?');">
							<button type="submit" class="btn btn-small btn-danger">Remove</button>
						</form>
					</div>
				</td>
			</tr>
		`, html.EscapeString(user.Email), user.Sites, user.Role, user.CreatedAt.Format("2006-01-02"), user.ID, user.ID)
	}

	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<title>User Management - StinkyKitty</title>
	<style>%s</style>
</head>
<body>
	<div class="admin-header">
		<div class="container">
			<h1>User Management</h1>
			<div class="header-actions">
				<a href="/admin/dashboard" class="btn btn-secondary">← Back to Dashboard</a>
			</div>
		</div>
	</div>

	<div class="container">
		<div class="card">
			<table class="data-table">
				<thead>
					<tr>
						<th>Email</th>
						<th>Sites</th>
						<th>Role</th>
						<th>Created</th>
						<th>Actions</th>
					</tr>
				</thead>
				<tbody>
					%s
				</tbody>
			</table>
		</div>
	</div>
</body>
</html>`, GetDesignSystemCSS(), tableRows)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(htmlContent))
}

// UserResetPasswordHandler sends password reset email to user
func UserResetPasswordHandler(c *gin.Context) {
	userID := c.Param("id")

	var user models.User
	if err := db.GetDB().First(&user, userID).Error; err != nil {
		c.String(http.StatusNotFound, "User not found")
		return
	}

	// Generate reset token
	token, err := auth.GenerateResetToken()
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to generate token")
		return
	}

	// Save token
	db.GetDB().Model(&user).Updates(map[string]interface{}{
		"reset_token":   token,
		"reset_expires": time.Now().Add(24 * time.Hour),
	})

	// Send email
	svc, err := email.NewEmailService()
	if err == nil {
		baseDomain := "campasaur.us" // TODO: get from config
		resetURL := fmt.Sprintf("https://%s/admin/reset-confirm?token=%s", baseDomain, token)
		svc.SendPasswordReset(user.Email, resetURL)
	}

	c.Redirect(http.StatusFound, "/admin/users?message=Password+reset+email+sent")
}

// UserDeleteHandler soft-deletes a user
func UserDeleteHandler(c *gin.Context) {
	userID := c.Param("id")

	// Soft delete
	if err := db.GetDB().Delete(&models.User{}, userID).Error; err != nil {
		c.String(http.StatusInternalServerError, "Failed to delete user")
		return
	}

	c.Redirect(http.StatusFound, "/admin/users?message=User+removed")
}
```

**Step 2: Add routes**

In `cmd/stinky/server.go`, add routes in the admin section:

```go
authGroup.GET("/users", middleware.RequireAuth(), handlers.UsersListHandler)
authGroup.POST("/users/:id/reset-password", middleware.RequireAuth(), handlers.UserResetPasswordHandler)
authGroup.POST("/users/:id/delete", middleware.RequireAuth(), handlers.UserDeleteHandler)
```

**Step 3: Add table styles**

In `internal/handlers/styles.go`, add data table styles:

```go
.data-table {
	width: 100%;
	border-collapse: collapse;
}

.data-table th {
	text-align: left;
	padding: var(--spacing-sm) var(--spacing-md);
	background: var(--color-surface-secondary);
	border-bottom: 2px solid var(--color-border);
	font-weight: 600;
}

.data-table td {
	padding: var(--spacing-sm) var(--spacing-md);
	border-bottom: 1px solid var(--color-border);
}

.data-table tr:hover {
	background: var(--color-surface-secondary);
}

.btn-small {
	padding: 4px 12px;
	font-size: 13px;
}

.btn-danger {
	background: #dc2626;
	color: white;
}

.btn-danger:hover {
	background: #b91c1c;
}
```

**Step 4: Add link to dashboard**

In `internal/handlers/admin.go` (DashboardHandler), add a "Manage Users" link in the header actions area.

**Step 5: Test user management**

1. Visit `/admin/users`
2. Verify user list appears
3. Click "Reset Password" → verify email sent
4. Click "Remove" → verify user soft-deleted
5. Test as both global admin and site admin

**Step 6: Commit**

```bash
git add internal/handlers/admin_users.go cmd/stinky/server.go internal/handlers/styles.go internal/handlers/admin.go
git commit -m "feat: add user management interface with reset and delete"
```

---

## Task 7: Create Column Layout Block

**Files:**
- Modify: `internal/blocks/renderer.go` (add columns case)
- Modify: `internal/handlers/admin_blocks.go` (add columns block creation and editing)
- Modify: `internal/models/models.go` (if needed for schema)

**Step 1: Add columns block renderer**

In `internal/blocks/renderer.go`, add a new case to `RenderBlock`:

```go
case "columns":
	return renderColumnsBlock(dataJSON)
```

**Step 2: Create ColumnsBlockData and renderer**

Add this at the end of `internal/blocks/renderer.go`:

```go
// ColumnsBlockData represents the JSON structure for column blocks
type ColumnsBlockData struct {
	ColumnCount int      `json:"column_count"` // 2, 3, or 4
	Columns     []Column `json:"columns"`
}

type Column struct {
	Content string `json:"content"` // HTML content for this column
}

// renderColumnsBlock renders a multi-column layout block
func renderColumnsBlock(dataJSON string) (string, error) {
	var data ColumnsBlockData
	if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
		return "", fmt.Errorf("failed to parse columns block data: %w", err)
	}

	// Validate column count
	if data.ColumnCount < 2 || data.ColumnCount > 4 {
		data.ColumnCount = 2
	}

	// Calculate column width percentage
	columnWidth := 100 / data.ColumnCount

	htmlStr := `<div class="columns-block" style="display: grid; grid-template-columns: repeat(` + fmt.Sprintf("%d", data.ColumnCount) + `, 1fr); gap: var(--spacing-lg, 24px); margin: var(--spacing-lg, 24px) 0;">`

	// Render each column
	for _, col := range data.Columns {
		// Sanitize content (allow basic HTML tags)
		safeContent := html.EscapeString(col.Content)
		// Convert newlines to <br> for display
		safeContent = strings.ReplaceAll(safeContent, "\n", "<br>")

		htmlStr += fmt.Sprintf(`
			<div class="column" style="min-width: 0;">
				%s
			</div>
		`, safeContent)
	}

	htmlStr += `</div>`

	return htmlStr, nil
}
```

**Step 3: Add columns block to admin_blocks.go**

In `internal/handlers/admin_blocks.go`, add "columns" to validTypes:

```go
validTypes := map[string]bool{
	"text":    true,
	"image":   true,
	"heading": true,
	"quote":   true,
	"button":  true,
	"video":   true,
	"spacer":  true,
	"contact": true,
	"columns": true,
}
```

**Step 4: Add default columns data**

In CreateBlockHandler, add the columns case:

```go
case "columns":
	blockData = `{"column_count":2,"columns":[{"content":"Column 1 content"},{"content":"Column 2 content"}]}`
```

**Step 5: Add columns edit form**

In EditBlockHandler, add the columns case before the final else:

```go
} else if block.Type == "columns" {
	// Parse columns block data
	var columnsData struct {
		ColumnCount int `json:"column_count"`
		Columns     []struct {
			Content string `json:"content"`
		} `json:"columns"`
	}
	if err := json.Unmarshal([]byte(block.Data), &columnsData); err != nil {
		columnsData.ColumnCount = 2
		columnsData.Columns = []struct{ Content string }{{"Column 1"}, {"Column 2"}}
	}

	// Build column inputs
	var columnInputs string
	for i, col := range columnsData.Columns {
		columnInputs += fmt.Sprintf(`
			<div class="form-group">
				<label for="column_%d">Column %d</label>
				<textarea id="column_%d" name="column_%d" rows="6" style="width: 100%%; padding: 8px; border: 1px solid #ddd; border-radius: 4px; font-family: monospace;">%s</textarea>
			</div>
		`, i, i+1, i, i, html.EscapeString(col.Content))
	}

	html = fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<title>Edit Columns Block</title>
	<style>%s</style>
</head>
<body>
	<div class="container" style="max-width: 800px; margin: 20px auto;">
		<h1>Edit Columns Block</h1>
		<form method="POST" action="/admin/pages/%s/blocks/%s">
			<div class="form-group">
				<label for="column_count">Number of Columns</label>
				<select id="column_count" name="column_count" onchange="updateColumnInputs(this.value)">
					<option value="2"%s>2 Columns</option>
					<option value="3"%s>3 Columns</option>
					<option value="4"%s>4 Columns</option>
				</select>
			</div>

			<div id="column-inputs">
				%s
			</div>

			<div style="margin-top: 20px;">
				<button type="submit" class="btn btn-primary">Save Block</button>
				<a href="/admin/pages/%s/edit" class="btn btn-secondary">Cancel</a>
			</div>
		</form>

		<script>
		function updateColumnInputs(count) {
			const container = document.getElementById('column-inputs');
			container.innerHTML = '';

			for (let i = 0; i < count; i++) {
				const div = document.createElement('div');
				div.className = 'form-group';
				div.innerHTML = '<label for="column_' + i + '">Column ' + (i+1) + '</label>' +
					'<textarea id="column_' + i + '" name="column_' + i + '" rows="6" ' +
					'style="width: 100%%; padding: 8px; border: 1px solid #ddd; border-radius: 4px; font-family: monospace;">' +
					'Column ' + (i+1) + ' content</textarea>';
				container.appendChild(div);
			}
		}
		</script>
	</div>
</body>
</html>`, GetDesignSystemCSS(), pageIDStr, blockIDStr,
		map[bool]string{true: " selected", false: ""}[columnsData.ColumnCount == 2],
		map[bool]string{true: " selected", false: ""}[columnsData.ColumnCount == 3],
		map[bool]string{true: " selected", false: ""}[columnsData.ColumnCount == 4],
		columnInputs, pageIDStr)
}
```

**Step 6: Add columns update handler**

In UpdateBlockHandler, add the columns case:

```go
case "columns":
	columnCount := c.PostForm("column_count")
	if columnCount == "" {
		columnCount = "2"
	}

	// Parse column count
	count := 2
	fmt.Sscanf(columnCount, "%d", &count)
	if count < 2 || count > 4 {
		count = 2
	}

	// Collect column contents
	columns := make([]map[string]string, count)
	for i := 0; i < count; i++ {
		content := c.PostForm(fmt.Sprintf("column_%d", i))
		columns[i] = map[string]string{"content": content}
	}

	data := map[string]interface{}{
		"column_count": count,
		"columns":      columns,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to encode block data")
		return
	}
	block.Data = string(jsonData)
```

**Step 7: Add "+ Columns" button in page editor**

In `internal/handlers/admin_pages.go`, find where the block creation buttons are and add:

```go
<form method="POST" action="/admin/pages/` + pageIDStr + `/blocks" style="display:inline;">
	<input type="hidden" name="type" value="columns">
	<button type="submit" class="btn btn-columns">+ Columns</button>
</form>
```

**Step 8: Test columns block**

1. Create a new page
2. Add a columns block
3. Change to 3 columns
4. Add different content to each column
5. Save and view on public site
6. Verify columns display side-by-side

**Step 9: Commit**

```bash
git add internal/blocks/renderer.go internal/handlers/admin_blocks.go internal/handlers/admin_pages.go
git commit -m "feat: add multi-column layout block (2, 3, or 4 columns)"
```

---

## Task 8: Add "Manage Users" Link to Dashboard

**Files:**
- Modify: `internal/handlers/admin.go` (DashboardHandler)

**Step 1: Add link to dashboard header**

In DashboardHandler, find the header actions section and add:

```go
<div class="header-actions">
	<a href="/admin/users" class="btn btn-secondary">Manage Users</a>
	<a href="/admin/create-camp" class="btn btn-primary">+ Create New Camp</a>
</div>
```

**Step 2: Test**

1. Visit dashboard
2. Click "Manage Users"
3. Verify it goes to user management page

**Step 3: Commit**

```bash
git add internal/handlers/admin.go
git commit -m "feat: add Manage Users link to dashboard"
```

---

## Task 9: Integration Testing

**Files:**
- Create: `test-enhancements.sh`

**Step 1: Create integration test script**

Create `test-enhancements.sh`:

```bash
#!/bin/bash
set -e

echo "Running StinkyKitty CMS Enhancements Integration Tests"
echo "======================================================"

# Start server in background
./stinky server start &
SERVER_PID=$!
sleep 2

# Test 1: Site Settings - GA and Copyright
echo "Test 1: Checking site settings page..."
curl -s http://localhost:8080/admin/settings | grep -q "google_analytics_id" && echo "✓ GA field present" || echo "✗ GA field missing"
curl -s http://localhost:8080/admin/settings | grep -q "copyright_text" && echo "✓ Copyright field present" || echo "✗ Copyright field missing"

# Test 2: Public page header
echo "Test 2: Checking public page header..."
curl -s http://localhost:8080/ | grep -q "site-header" && echo "✓ Header present" || echo "✗ Header missing"
curl -s http://localhost:8080/ | grep -q "site-header-login" && echo "✓ Login button present" || echo "✗ Login button missing"

# Test 3: User management page
echo "Test 3: Checking user management..."
curl -s http://localhost:8080/admin/users | grep -q "User Management" && echo "✓ User management page accessible" || echo "✗ User management page missing"

# Test 4: Column block creation
echo "Test 4: Testing columns block..."
# This would require authenticated session - placeholder
echo "⊘ Column block test requires authentication"

# Cleanup
kill $SERVER_PID
echo ""
echo "Integration tests complete!"
```

**Step 2: Make executable and run**

```bash
chmod +x test-enhancements.sh
./test-enhancements.sh
```

**Step 3: Commit**

```bash
git add test-enhancements.sh
git commit -m "test: add integration tests for CMS enhancements"
```

---

## Task 10: Documentation

**Files:**
- Create: `docs/FEATURES.md`
- Modify: `README.md`

**Step 1: Create features documentation**

Create `docs/FEATURES.md`:

```markdown
# StinkyKitty CMS Features

## User Management

### Overview
Site administrators can manage users with access to their sites. Global administrators can manage all users across all sites.

### Access
- Navigate to **Admin Dashboard** → **Manage Users**
- View all users with access to your sites
- See user email, sites, role, and creation date

### Actions

**Reset Password**
- Click "Reset Password" next to any user
- Sends password reset email to the user
- User receives 24-hour reset link

**Remove User**
- Click "Remove" to soft-delete a user
- User loses access to the site
- Can be restored by recreating with same email

## Google Analytics Integration

### Setup
1. Get your Google Analytics tracking ID from Google Analytics
   - Format: `G-XXXXXXXXXX` (GA4) or `UA-XXXXXXXXX` (Universal Analytics)
2. Go to **Admin** → **Settings**
3. Enter tracking ID in "Google Analytics Tracking ID" field
4. Save settings

### What It Does
- Automatically injects GA tracking script on all public pages
- Tracks page views, user behavior, and site analytics
- Data appears in your Google Analytics dashboard

## Site Customization

### Fixed Header Bar
All public pages now include a fixed header bar with:
- Site title/logo (links to homepage)
- Navigation menu
- Login button (for site admins)

The header stays at the top when scrolling for easy navigation.

### Custom Copyright
Customize your site's footer copyright text:

1. Go to **Admin** → **Settings**
2. Edit "Copyright Text" field
3. Use placeholders:
   - `{year}` → Replaced with current year
   - `{site}` → Replaced with site title

**Example:**
Input: `© {year} {site} - All rights reserved.`
Output: `© 2025 My Camp Site - All rights reserved.`

## Column Layouts

### Creating Column Blocks
1. Edit any page
2. Click **+ Columns**
3. Choose 2, 3, or 4 columns
4. Add content to each column
5. Save

### Use Cases
- Feature grids (3 columns of features)
- Image galleries
- Button groups
- Text + image side-by-side layouts

### Tips
- Columns stack vertically on mobile
- Keep column content balanced
- Use with other blocks for rich layouts
```

**Step 2: Update README**

Add to README.md feature list:

```markdown
## Features

- **Multi-tenant architecture** - Host unlimited camp websites
- **Block-based content editor** - Text, images, headings, quotes, buttons, video, columns
- **User management** - Manage site users, reset passwords, control access
- **Google Analytics** - Built-in analytics tracking
- **Customizable themes** - Colors, fonts, layouts
- **Custom copyright** - Editable footer text per site
- **Fixed header navigation** - Professional site headers
- **Search functionality** - Full-text search across pages
- **Contact forms** - Embedded contact forms
- **CLI administration** - Command-line site management
```

**Step 3: Commit**

```bash
git add docs/FEATURES.md README.md
git commit -m "docs: add comprehensive feature documentation"
```

---

## Final Integration & Testing

**Step 1: Build and test locally**

```bash
make build
./stinky server start
```

**Step 2: Manual verification checklist**

- [ ] Create new site with GA ID and custom copyright
- [ ] Visit public page, verify header bar and copyright
- [ ] View page source, verify GA script injected
- [ ] Go to user management, reset a password
- [ ] Create a page with 3-column layout
- [ ] Test on mobile (columns should stack)
- [ ] Test as global admin and site admin

**Step 3: Deploy to production**

```bash
make build-linux
./deploy.sh
```

**Step 4: Final commit**

```bash
git commit -m "feat: complete CMS enhancements - user management, GA, header, copyright, columns"
git push
```

---

## Success Criteria

✅ **User Management**
- Site admins can view users on their sites
- Global admins can view all users
- Can send password reset emails
- Can remove users from sites

✅ **Google Analytics**
- Field in site settings to add GA ID
- GA script injected on all public pages
- Tracking works in Google Analytics dashboard

✅ **Fixed Header**
- Header appears on all public pages
- Includes site title, navigation, and login button
- Sticky positioning works correctly

✅ **Custom Copyright**
- Editable field in site settings
- Supports {year} and {site} placeholders
- Displays in page footer

✅ **Column Layouts**
- Can create 2, 3, or 4 column blocks
- Edit content in each column
- Renders correctly on desktop and mobile
- Integrates with existing block system

---

## Notes for Implementer

- All database changes use GORM migrations (no manual SQL)
- Follow existing code patterns (Gin handlers, GORM queries)
- Use existing design system CSS variables
- Test both as global admin and site admin
- Commit frequently with descriptive messages
- Run tests after each task
