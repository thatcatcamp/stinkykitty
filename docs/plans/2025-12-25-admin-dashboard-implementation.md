# Admin Dashboard UI Redesign Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement professional, warm Figma-inspired design for login and dashboard pages with proper CSS variables, color palette, and component styling.

**Architecture:** Create a shared CSS file with design system (variables, utilities, base styles), then update login and dashboard handler templates to use new styles. Keep all styling consistent via CSS variables for easy maintenance.

**Tech Stack:** Go (Gin), inline CSS with CSS custom properties, no external CSS framework

---

## Task 1: Create CSS Design System File

**Files:**
- Create: `internal/handlers/styles.go` (shared CSS variables and base styles)

**Step 1: Write constants for design tokens**

```go
package handlers

const (
	// Color Palette
	ColorBgPrimary   = "#FAFAF8"   // Cream background
	ColorBgCard      = "#FFFFFF"   // White card
	ColorTextPrimary = "#2D2D2D"   // Dark charcoal
	ColorTextSecond  = "#6B7280"   // Light gray
	ColorAccent      = "#2E8B9E"   // Teal accent
	ColorAccentHover = "#1E6F7F"   // Darker teal
	ColorSuccess     = "#10B981"   // Green (published)
	ColorWarning     = "#F59E0B"   // Amber (draft)
	ColorDanger      = "#EF4444"   // Red (delete)
	ColorBorder      = "#E5E5E3"   // Subtle border
)

// Returns full stylesheet with CSS variables and base styles
func GetDesignSystemCSS() string {
	return `
:root {
	--color-bg-primary: ` + ColorBgPrimary + `;
	--color-bg-card: ` + ColorBgCard + `;
	--color-text-primary: ` + ColorTextPrimary + `;
	--color-text-secondary: ` + ColorTextSecond + `;
	--color-accent: ` + ColorAccent + `;
	--color-accent-hover: ` + ColorAccentHover + `;
	--color-success: ` + ColorSuccess + `;
	--color-warning: ` + ColorWarning + `;
	--color-danger: ` + ColorDanger + `;
	--color-border: ` + ColorBorder + `;
	--font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
	--spacing-xs: 4px;
	--spacing-sm: 8px;
	--spacing-base: 16px;
	--spacing-md: 24px;
	--spacing-lg: 40px;
	--radius-sm: 4px;
	--radius-base: 6px;
	--shadow-sm: 0 1px 3px rgba(0, 0, 0, 0.1);
	--shadow-md: 0 2px 8px rgba(0, 0, 0, 0.12);
	--transition: 200ms ease;
}

* { box-sizing: border-box; }

body {
	font-family: var(--font-family);
	background: var(--color-bg-primary);
	color: var(--color-text-primary);
	margin: 0;
	padding: 0;
	line-height: 1.5;
}

h1 { font-size: 28px; font-weight: 700; margin: 0; }
h2 { font-size: 20px; font-weight: 600; margin: 0; }
p { font-size: 16px; margin: 0; }
small { font-size: 12px; color: var(--color-text-secondary); }

a {
	color: var(--color-accent);
	text-decoration: none;
	transition: color var(--transition);
}

a:hover { color: var(--color-accent-hover); }

button {
	font-family: inherit;
	font-size: 14px;
	font-weight: 600;
	border: none;
	border-radius: var(--radius-base);
	padding: var(--spacing-base) calc(var(--spacing-base) * 1.5);
	cursor: pointer;
	transition: background var(--transition), box-shadow var(--transition);
}

button:hover { opacity: 0.9; }

input, textarea {
	font-family: inherit;
	font-size: 16px;
	padding: var(--spacing-base);
	border: 1px solid var(--color-border);
	border-radius: var(--radius-sm);
	transition: border-color var(--transition);
}

input:focus, textarea:focus {
	outline: none;
	border-color: var(--color-accent);
	box-shadow: 0 0 0 2px rgba(46, 139, 158, 0.1);
}

input::placeholder { color: var(--color-text-secondary); }
`
}
```

**Step 2: Add function to handlers package**

Verify the function exists in `internal/handlers/styles.go` by checking it compiles.

**Step 3: Run Go build to verify it compiles**

Run: `go build -o /tmp/test-build ./cmd/stinky`
Expected: Builds without errors

**Step 4: Commit**

```bash
git add internal/handlers/styles.go
git commit -m "feat: add design system CSS variables and base styles"
```

---

## Task 2: Update Login Page Handler

**Files:**
- Modify: `internal/handlers/admin.go` (LoginFormHandler function)

**Step 1: Get current login handler code**

Read the LoginFormHandler in `internal/handlers/admin.go` to understand current structure.

**Step 2: Replace LoginFormHandler with new design**

Replace the entire LoginFormHandler function:

```go
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
```

**Step 3: Verify syntax and build**

Run: `go build -o /tmp/test-build ./cmd/stinky`
Expected: Builds without errors

**Step 4: Commit**

```bash
git add internal/handlers/admin.go
git commit -m "style: redesign login page with warm professional aesthetic"
```

---

## Task 3: Update Dashboard Handler (Part 1 - Header & Layout)

**Files:**
- Modify: `internal/handlers/admin_pages.go` (EditPageHandler function - lines 200-330)

**Step 1: Get current EditPageHandler structure**

Read lines 200-330 of `internal/handlers/admin_pages.go` to see current dashboard structure.

**Step 2: Replace the CSS section in EditPageHandler**

Replace the `<style>` block (around line 296-326) with:

```go
    <style>
        ` + GetDesignSystemCSS() + `

        body {
            padding: 0;
        }

        .page-layout {
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
            font-size: 14px;
        }

        .header-right small {
            color: var(--color-text-secondary);
        }

        .logout-btn {
            background: transparent;
            color: var(--color-accent);
            padding: var(--spacing-sm) var(--spacing-base);
            font-size: 14px;
        }

        .logout-btn:hover {
            color: var(--color-accent-hover);
        }

        .container {
            flex: 1;
            max-width: 1200px;
            margin: 0 auto;
            width: 100%;
            padding: var(--spacing-md);
        }

        .back-link {
            display: inline-block;
            margin-bottom: var(--spacing-md);
            color: var(--color-accent);
            font-size: 14px;
        }

        .page-header {
            margin-bottom: var(--spacing-lg);
        }

        .page-title-section {
            background: var(--color-bg-card);
            padding: var(--spacing-md);
            border-radius: var(--radius-base);
            border: 1px solid var(--color-border);
            margin-bottom: var(--spacing-md);
        }

        .page-title-section input {
            width: 100%;
            font-size: 20px;
            font-weight: 600;
            padding: var(--spacing-base);
            margin-bottom: var(--spacing-base);
        }

        .page-actions {
            display: flex;
            gap: var(--spacing-base);
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
            color: white;
        }

        .btn-secondary:hover {
            background: #5a6268;
        }

        .btn-success {
            background: var(--color-success);
        }

        .btn-success:hover {
            background: #059669;
        }

        .section {
            margin-bottom: var(--spacing-lg);
        }

        .section h2 {
            font-size: 18px;
            margin-bottom: var(--spacing-md);
            color: var(--color-text-primary);
        }
    </style>`
```

**Step 3: Verify no syntax errors**

Run: `go build -o /tmp/test-build ./cmd/stinky`
Expected: Builds without errors

**Step 4: Commit**

```bash
git add internal/handlers/admin_pages.go
git commit -m "style: update dashboard header and base layout styles"
```

---

## Task 4: Update Dashboard Handler (Part 2 - Pages List & Blocks)

**Files:**
- Modify: `internal/handlers/admin_pages.go` (EditPageHandler function - continues CSS)

**Step 1: Add block card and button styles to CSS**

Add to the `<style>` block in EditPageHandler (after previous section):

```go
        .blocks-list {
            display: flex;
            flex-direction: column;
            gap: var(--spacing-base);
        }

        .block-item {
            background: var(--color-bg-card);
            border: 1px solid var(--color-border);
            border-radius: var(--radius-base);
            padding: var(--spacing-base);
            display: flex;
            justify-content: space-between;
            align-items: flex-start;
            gap: var(--spacing-md);
            transition: box-shadow var(--transition), background var(--transition);
        }

        .block-item:hover {
            box-shadow: var(--shadow-md);
            background: #fafbfc;
        }

        .block-info {
            flex: 1;
            min-width: 0;
        }

        .block-type {
            font-weight: 600;
            margin-bottom: var(--spacing-sm);
            font-size: 14px;
            color: var(--color-text-primary);
        }

        .block-preview {
            font-size: 13px;
            color: var(--color-text-secondary);
            font-family: "Monaco", "Courier New", monospace;
            background: #f8f9fa;
            padding: var(--spacing-sm);
            border-radius: var(--radius-sm);
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
            max-width: 100%;
        }

        .block-actions {
            display: flex;
            gap: var(--spacing-sm);
            align-items: center;
            flex-shrink: 0;
        }

        .btn-icon {
            padding: var(--spacing-sm) calc(var(--spacing-sm) * 1.25);
            font-size: 14px;
            background: var(--color-text-secondary);
            color: white;
            border: none;
            border-radius: var(--radius-sm);
            cursor: pointer;
            transition: background var(--transition);
        }

        .btn-icon:hover {
            background: #4b5563;
        }

        .btn-icon:disabled {
            opacity: 0.3;
            cursor: not-allowed;
        }

        .btn-small {
            padding: var(--spacing-sm) calc(var(--spacing-base));
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

        .btn-danger {
            background: var(--color-danger);
        }

        .btn-danger:hover {
            background: #dc2626;
        }

        .empty-state {
            padding: var(--spacing-lg);
            text-align: center;
            color: var(--color-text-secondary);
            border: 2px dashed var(--color-border);
            border-radius: var(--radius-base);
            background: #fafbfc;
        }

        .add-block {
            display: flex;
            flex-wrap: wrap;
            gap: var(--spacing-base);
            padding: var(--spacing-md);
            background: var(--color-bg-card);
            border-radius: var(--radius-base);
            border: 1px solid var(--color-border);
        }

        .add-block .btn {
            padding: var(--spacing-sm) var(--spacing-md);
            font-size: 13px;
        }

        .btn-text { background: var(--color-accent); }
        .btn-heading { background: #6c757d; }
        .btn-image { background: #17a2b8; }
        .btn-quote { background: #6f42c1; }
        .btn-button { background: var(--color-success); }
        .btn-video { background: var(--color-danger); }
        .btn-spacer { background: #e0e0e0; color: var(--color-text-primary); }
```

**Step 2: Verify build**

Run: `go build -o /tmp/test-build ./cmd/stinky`
Expected: Builds without errors

**Step 3: Commit**

```bash
git add internal/handlers/admin_pages.go
git commit -m "style: add block list and button styling"
```

---

## Task 5: Update Dashboard Handler HTML Structure

**Files:**
- Modify: `internal/handlers/admin_pages.go` (EditPageHandler function - HTML structure)

**Step 1: Wrap entire page in new layout structure**

Find the line: `<div class="container">` and replace the entire body section (from `<body>` to `</body>`) with:

```html
<body>
    <div class="page-layout">
        <div class="header">
            <div class="header-content">
                <div class="header-left">
                    <h1>Your Site</h1>
                </div>
                <div class="header-right">
                    <small>` + siteVal.(*models.Site).Subdomain + `</small>
                    <small>` + user.Email + `</small>
                    <form method="POST" action="/admin/logout" style="display:inline;">
                        <button type="submit" class="logout-btn">Sign Out</button>
                    </form>
                </div>
            </div>
        </div>

        <div class="container">
            <a href="/admin/dashboard" class="back-link">← Back to Dashboard</a>

            <div class="page-header">
                <div class="page-title-section">
                    <form method="POST" action="/admin/pages/` + pageIDStr + `">
                        <input type="text" name="title" value="` + page.Title + `" placeholder="Page Title" required>
                        <div class="page-actions">
                            <button type="submit" class="btn">Save Draft</button>`
```

**Step 2: Replace publish/unpublish buttons**

Replace the publish section with:

```html
                            ` + publishButton + `
                        </div>
                    </form>
                </div>
            </div>

            <div class="section">
                <h2>Content Blocks</h2>
                ` + blocksHTML + `
                <div class="add-block">
                    <form method="POST" action="/admin/pages/` + pageIDStr + `/blocks" style="display:inline;">
                        <input type="hidden" name="type" value="text">
                        <button type="submit" class="btn btn-text">+ Text</button>
                    </form>
                    <form method="POST" action="/admin/pages/` + pageIDStr + `/blocks" style="display:inline;">
                        <input type="hidden" name="type" value="heading">
                        <button type="submit" class="btn btn-heading">+ Heading</button>
                    </form>
                    <a href="/admin/pages/` + pageIDStr + `/blocks/new-image" class="btn btn-image">+ Image</a>
                    <form method="POST" action="/admin/pages/` + pageIDStr + `/blocks" style="display:inline;">
                        <input type="hidden" name="type" value="quote">
                        <button type="submit" class="btn btn-quote">+ Quote</button>
                    </form>
                    <form method="POST" action="/admin/pages/` + pageIDStr + `/blocks" style="display:inline;">
                        <input type="hidden" name="type" value="button">
                        <button type="submit" class="btn btn-button">+ Button</button>
                    </form>
                    <form method="POST" action="/admin/pages/` + pageIDStr + `/blocks" style="display:inline;">
                        <input type="hidden" name="type" value="video">
                        <button type="submit" class="btn btn-video">+ Video</button>
                    </form>
                    <form method="POST" action="/admin/pages/` + pageIDStr + `/blocks" style="display:inline;">
                        <input type="hidden" name="type" value="spacer">
                        <button type="submit" class="btn btn-spacer">+ Spacer</button>
                    </form>
                </div>
            </div>
        </div>
    </div>
</body>
```

**Step 3: Update blocks rendering to use new classes**

In the blocks loop (where `blocksHTML` is built), replace the block item HTML with:

```go
		blocksHTML += `
			<div class="block-item">
				<div class="block-info">
					<div class="block-type">` + blockTypeLabel + `</div>
					<div class="block-preview">` + preview + `</div>
				</div>
				<div class="block-actions">
					` + moveUpBtn + `
					` + moveDownBtn + `
					<a href="/admin/pages/` + pageIDStr + `/blocks/` + strconv.Itoa(int(block.ID)) + `/edit" class="btn-small">Edit</a>
					<form method="POST" action="/admin/pages/` + pageIDStr + `/blocks/` + strconv.Itoa(int(block.ID)) + `/delete" style="display:inline;" onsubmit="return confirm('Delete this block?')">
						<button type="submit" class="btn-small btn-danger">Delete</button>
					</form>
				</div>
			</div>
		`
```

**Step 4: Verify build**

Run: `go build -o /tmp/test-build ./cmd/stinky`
Expected: Builds without errors

**Step 5: Commit**

```bash
git add internal/handlers/admin_pages.go
git commit -m "style: redesign dashboard page layout with new structure"
```

---

## Task 6: Update Dashboard Handler Function

**Files:**
- Modify: `internal/handlers/admin_pages.go` (Dashboard handler changes)

**Step 1: Remove old inline styles from block type label logic**

Find where `blockTypeLabel` is set (around line 230) and update to handle new block types:

```go
		blockTypeLabel := "Text Block"
		if block.Type == "image" {
			blockTypeLabel = "Image Block"
		} else if block.Type == "heading" {
			blockTypeLabel = "Heading Block"
		} else if block.Type == "quote" {
			blockTypeLabel = "Quote Block"
		} else if block.Type == "button" {
			blockTypeLabel = "Button Block"
		} else if block.Type == "video" {
			blockTypeLabel = "Video Block"
		} else if block.Type == "spacer" {
			blockTypeLabel = "Spacer Block"
		}
```

**Step 2: Extract publish button to variable**

Before the HTML string, add:

```go
	var publishButton string
	if page.Published {
		publishButton = `
                            <form method="POST" action="/admin/pages/` + pageIDStr + `/unpublish" style="display:inline;">
                                <button type="submit" class="btn btn-secondary">Unpublish</button>
                            </form>`
	} else {
		publishButton = `
                            <form method="POST" action="/admin/pages/` + pageIDStr + `/publish" style="display:inline;">
                                <button type="submit" class="btn btn-success">Publish</button>
                            </form>`
	}
```

**Step 3: Verify build**

Run: `go build -o /tmp/test-build ./cmd/stinky`
Expected: Builds without errors

**Step 4: Commit**

```bash
git add internal/handlers/admin_pages.go
git commit -m "refactor: clean up dashboard handler code structure"
```

---

## Task 7: Update Admin Dashboard List Handler

**Files:**
- Modify: `internal/handlers/admin.go` (DashboardHandler function)

**Step 1: Get current DashboardHandler**

Read the DashboardHandler function in `internal/handlers/admin.go`.

**Step 2: Replace entire DashboardHandler with new design**

```go
// DashboardHandler shows the admin dashboard with list of sites
func DashboardHandler(c *gin.Context) {
	// Get user from context
	userVal, exists := c.Get("user")
	if !exists {
		c.Redirect(http.StatusFound, "/admin/login")
		return
	}
	user := userVal.(*models.User)

	// Get all sites where user is an admin/owner
	var userSites []struct {
		Site models.Site
		Role string
	}

	db.GetDB().Raw(`
		SELECT sites.*, site_users.role
		FROM sites
		JOIN site_users ON sites.id = site_users.site_id
		WHERE site_users.user_id = ? AND (site_users.role = 'owner' OR site_users.role = 'admin')
		ORDER BY sites.subdomain
	`, user.ID).Scan(&userSites)

	// Build sites list HTML
	var sitesHTML string
	if len(userSites) == 0 {
		sitesHTML = `<div class="empty-state">No sites yet. Contact an administrator to create one.</div>`
	} else {
		for _, us := range userSites {
			var domainDisplay string
			if us.Site.CustomDomain != "" {
				domainDisplay = us.Site.CustomDomain
			} else {
				domainDisplay = us.Site.Subdomain + ".stinkykitty.org"
			}

			sitesHTML += `
				<div class="site-card">
					<div class="site-info">
						<h3>` + us.Site.Subdomain + `</h3>
						<small>` + domainDisplay + `</small>
					</div>
					<div class="site-actions">
						<a href="/admin/pages?site=` + strconv.Itoa(int(us.Site.ID)) + `" class="btn-small">Edit</a>
						<a href="https://` + domainDisplay + `" target="_blank" class="btn-small btn-secondary">View</a>
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

        .btn-secondary {
            background: var(--color-text-secondary);
        }

        .btn-secondary:hover {
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
                    <a href="/admin/pages/new" class="btn">+ Create New Camp</a>
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
</body>
</html>`

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
```

**Step 3: Verify build**

Run: `go build -o /tmp/test-build ./cmd/stinky`
Expected: Builds without errors

**Step 4: Commit**

```bash
git add internal/handlers/admin.go
git commit -m "style: redesign dashboard handler with warm professional aesthetic"
```

---

## Task 8: Test Login & Dashboard Pages

**Files:**
- Test: Manual browser testing

**Step 1: Build the application**

Run: `go build -o stinky ./cmd/stinky`
Expected: Builds successfully

**Step 2: Start the server**

Run: `./stinky server start` (or appropriate command for your setup)
Expected: Server starts on port 8080 (or configured port)

**Step 3: Test login page**

Navigate to: `http://localhost:8080/admin/login`
Expected:
- Page loads with new professional design
- Warm cream background
- Centered white card with teal accents
- Clean typography
- Input fields focus with teal border

**Step 4: Test dashboard (after login)**

Navigate to: `http://localhost:8080/admin/dashboard` (or login first)
Expected:
- Header with site name and user email
- Hero section with "Create New Camp" button
- Sites list displayed as cards
- Cards show hover effect (slight lift)
- Buttons are properly styled

**Step 5: Test page editor**

Edit a page
Expected:
- Header at top with site name
- Page title input at top
- Save Draft and Publish buttons
- Content Blocks section with new block cards
- Block cards show up/down, Edit, Delete buttons
- Block addition buttons at bottom

**Step 6: Commit test results**

```bash
git add -A
git commit -m "test: verify login and dashboard redesign in browser"
```

---

## Task 9: Update EditPageHandler Test

**Files:**
- Modify: `internal/handlers/admin_pages_test.go` (TestEditPageHandler_Success)

**Step 1: Read the failing test**

Look at `TestEditPageHandler_Success` to see what it's checking for.

**Step 2: Update test to look for new button text**

Replace the assertion that looks for "Add Text Block" with assertions for new structure. Update to check for:

```go
	// Check for hero section
	if !strings.Contains(body, "Content Blocks") {
		t.Error("'Content Blocks' section not found")
	}

	// Check for new button structure
	if !strings.Contains(body, "+ Text") {
		t.Error("'+ Text' button not found")
	}

	if !strings.Contains(body, "Save Draft") {
		t.Error("'Save Draft' button not found")
	}
```

**Step 3: Run tests**

Run: `go test ./internal/handlers -v`
Expected: TestEditPageHandler_Success passes

**Step 4: Commit**

```bash
git add internal/handlers/admin_pages_test.go
git commit -m "test: update dashboard test for new HTML structure"
```

---

## Summary

All tasks complete when:
- ✅ Design system CSS created with variables
- ✅ Login page redesigned with warm professional aesthetic
- ✅ Dashboard page redesigned with header, hero, sites list
- ✅ Page editor redesigned with sticky header and improved layouts
- ✅ All styling uses CSS variables for maintainability
- ✅ All code builds without errors
- ✅ Tests pass
- ✅ Browser testing confirms visual design

**Next Steps:** After implementation, collect feedback on the aesthetic before moving to block editors.
