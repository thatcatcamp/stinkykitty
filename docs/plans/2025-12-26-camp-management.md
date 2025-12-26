# Camp Management (Delete & Create Flows) Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Implement soft-delete with confirmation and multi-step create camp workflow with subdomain validation and user selection.

**Architecture:** Delete uses soft-delete (GORM's DeletedAt) with client-side confirmation modal. Create is a 4-step workflow: subdomain validation (AJAX), user selection/creation, confirmation, then auto-create site with published "Hello World!" page and redirect to edit.

**Tech Stack:** Go 1.25, Gin, GORM, SQLite, vanilla JavaScript (no deps), HTML forms

---

## 7 Core Tasks

1. Modify CreateSite to publish "Hello World!" page
2. Add delete handler with soft-delete logic
3. Update dashboard: add delete button + confirmation modal
4. Create create-camp form page (Step 1: Subdomain)
5. Implement subdomain validation AJAX endpoint
6. Create user selection/creation step (Step 2) + final create handler
7. Test all flows end-to-end

---

### Task 1: Modify CreateSite to publish "Hello World!" page

**Files:**
- Modify: `internal/sites/sites.go:48-57` (homepage creation)

**Step 1: Read current code**

Current code creates unpublished homepage with title = subdomain. We need to:
- Change title to "Hello World!"
- Set `Published: true`
- Add `Content` block with basic HTML

**Step 2: Implement new homepage creation**

```go
// Auto-create published homepage for new site
homepage := &models.Page{
	SiteID:    site.ID,
	Slug:      "/",
	Title:     "Hello World!",
	Published: true, // Make immediately visible
}
if err := db.Create(homepage).Error; err != nil {
	return nil, fmt.Errorf("failed to create homepage: %w", err)
}

// Add a text block to homepage
helloBlock := &models.Block{
	PageID: homepage.ID,
	Type:   "text",
	Order:  0,
	Data:   `{"content":"<p>Welcome to your new camp! Edit this page to get started.</p>"}`,
}
if err := db.Create(helloBlock).Error; err != nil {
	return nil, fmt.Errorf("failed to create homepage block: %w", err)
}
```

**Step 3: Run tests to verify no regression**

Run: `cd /home/lpreimesberger/projects/mex/stinkycat/.worktrees/camp-management && go test ./internal/sites -v`
Expected: All tests pass, TestCreateSite passes

**Step 4: Commit**

```bash
git add internal/sites/sites.go
git commit -m "feat: auto-create published Hello World! page on site creation"
```

---

### Task 2: Add delete handler with soft-delete logic

**Files:**
- Create: `internal/handlers/admin_delete_site.go`
- Modify: `cmd/stinky/server.go` (add route)

**Step 1: Create delete handler**

```go
// internal/handlers/admin_delete_site.go
package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/auth"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/models"
	"github.com/thatcatcamp/stinkykitty/internal/sites"
)

// DeleteSiteHandler handles soft-delete of a site
func DeleteSiteHandler(c *gin.Context) {
	// Get user from context
	userVal, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}
	user := userVal.(*models.User)

	// Get site ID from query param
	siteIDStr := c.Query("site")
	if siteIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "site parameter required"})
		return
	}

	siteID, err := strconv.ParseUint(siteIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid site ID"})
		return
	}

	// Verify user is owner/admin of site
	site, err := sites.GetSiteByID(db.GetDB(), uint(siteID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "site not found"})
		return
	}

	// Check permissions: owner or global admin only
	if site.OwnerID != user.ID && !user.IsGlobalAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	// Soft-delete the site
	if err := sites.DeleteSite(db.GetDB(), uint(siteID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to delete site: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "site deleted"})
}
```

**Step 2: Add route to server.go**

In `cmd/stinky/server.go`, in the admin routes section (around line 154), add:

```go
// Delete site (owner/admin only)
adminGroup.POST("/sites/:id/delete", handlers.DeleteSiteHandler)
```

**Step 3: Test handler exists and compiles**

Run: `cd /home/lpreimesberger/projects/mex/stinkycat/.worktrees/camp-management && go build ./cmd/stinky`
Expected: Builds without errors

**Step 4: Commit**

```bash
git add internal/handlers/admin_delete_site.go cmd/stinky/server.go
git commit -m "feat: add delete site handler with soft-delete"
```

---

### Task 3: Update dashboard - add delete button + confirmation modal

**Files:**
- Modify: `internal/handlers/admin.go:280-300` (site card HTML in DashboardHandler)

**Step 1: Update site card HTML to include delete button**

In `DashboardHandler`, replace the site card section (lines 288-299) with:

```go
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
```

**Step 2: Add delete modal HTML and JavaScript to dashboard**

Before the closing `</body>` tag (around line 587), add:

```html
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

		fetch(`/admin/sites/${pendingDeleteSiteId}/delete`, { method: 'POST' })
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
```

**Step 3: Test dashboard renders with button**

Build and manually check dashboard loads with delete button visible.

Run: `cd /home/lpreimesberger/projects/mex/stinkycat/.worktrees/camp-management && go build ./cmd/stinky`

**Step 4: Commit**

```bash
git add internal/handlers/admin.go
git commit -m "feat: add delete button and confirmation modal to dashboard"
```

---

### Task 4: Create create-camp form page (Step 1: Subdomain)

**Files:**
- Create: `internal/handlers/admin_create_camp.go`
- Modify: `cmd/stinky/server.go` (add routes for GET and POST)

**Step 1: Create handler for create camp page**

```go
// internal/handlers/admin_create_camp.go
package handlers

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// CreateCampFormHandler displays the multi-step camp creation form
func CreateCampFormHandler(c *gin.Context) {
	step := c.DefaultQuery("step", "1")

	switch step {
	case "1":
		createCampStep1(c)
	case "2":
		createCampStep2(c)
	case "3":
		createCampStep3(c)
	default:
		createCampStep1(c)
	}
}

func createCampStep1(c *gin.Context) {
	html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>Create Camp - Step 1 - StinkyKitty</title>
	<style>
		` + GetDesignSystemCSS() + `

		.create-container {
			max-width: 600px;
			margin: 0 auto;
			padding: var(--spacing-md);
		}

		.create-header {
			margin-bottom: var(--spacing-lg);
		}

		.create-header h1 {
			font-size: 24px;
			margin-bottom: var(--spacing-base);
		}

		.step-indicator {
			display: flex;
			gap: var(--spacing-md);
			margin-bottom: var(--spacing-lg);
		}

		.step {
			flex: 1;
			padding: var(--spacing-base);
			background: var(--color-bg-card);
			border-radius: var(--radius-sm);
			text-align: center;
			font-size: 13px;
		}

		.step.active {
			background: var(--color-accent);
			color: white;
			font-weight: 600;
		}

		.step.completed {
			background: #28a745;
			color: white;
		}

		.form-group {
			margin-bottom: var(--spacing-md);
		}

		.form-group label {
			display: block;
			margin-bottom: var(--spacing-sm);
			font-weight: 600;
		}

		.form-group input {
			width: 100%;
			padding: var(--spacing-sm);
			border: 1px solid var(--color-border);
			border-radius: var(--radius-sm);
			font-size: 14px;
		}

		.form-group input:focus {
			outline: none;
			border-color: var(--color-accent);
			box-shadow: 0 0 0 3px rgba(46, 139, 158, 0.1);
		}

		.help-text {
			font-size: 12px;
			color: var(--color-text-secondary);
			margin-top: var(--spacing-sm);
		}

		.validation-status {
			margin-top: var(--spacing-sm);
			font-size: 13px;
			display: none;
		}

		.validation-status.success {
			color: #28a745;
			display: block;
		}

		.validation-status.error {
			color: #dc3545;
			display: block;
		}

		.button-group {
			display: flex;
			gap: var(--spacing-base);
			margin-top: var(--spacing-lg);
		}

		.btn {
			flex: 1;
			padding: var(--spacing-sm);
			border-radius: var(--radius-sm);
			border: none;
			cursor: pointer;
			font-weight: 600;
			font-size: 14px;
		}

		.btn-primary {
			background: var(--color-accent);
			color: white;
		}

		.btn-primary:hover {
			background: var(--color-accent-hover);
		}

		.btn-primary:disabled {
			background: #ccc;
			cursor: not-allowed;
		}

		.btn-secondary {
			background: var(--color-text-secondary);
			color: white;
		}

		.btn-secondary:hover {
			background: #5a6268;
		}
	</style>
</head>
<body>
	<div class="create-container">
		<div class="create-header">
			<h1>Create New Camp</h1>
			<p>Set up your new camp in a few steps</p>
		</div>

		<div class="step-indicator">
			<div class="step active">1. Subdomain</div>
			<div class="step">2. Admin</div>
			<div class="step">3. Review</div>
		</div>

		<form id="step1-form">
			<div class="form-group">
				<label for="subdomain">Camp Subdomain</label>
				<input
					type="text"
					id="subdomain"
					name="subdomain"
					placeholder="mycamp"
					autocomplete="off"
					required
				>
				<div class="help-text">
					Letters, numbers, and hyphens only. Max 63 characters.
					Your camp will be at: <strong id="preview">mycamp.stinkykitty.org</strong>
				</div>
				<div id="validation-status" class="validation-status"></div>
			</div>

			<div class="button-group">
				<a href="/admin/dashboard" class="btn btn-secondary">Cancel</a>
				<button type="submit" class="btn btn-primary" id="next-btn" disabled>Next: Choose Admin →</button>
			</div>
		</form>
	</div>

	<script>
		const input = document.getElementById('subdomain');
		const preview = document.getElementById('preview');
		const status = document.getElementById('validation-status');
		const nextBtn = document.getElementById('next-btn');

		input.addEventListener('input', function() {
			// Force lowercase
			this.value = this.value.toLowerCase();

			// Remove invalid characters (keep only alphanumeric and hyphens)
			this.value = this.value.replace(/[^a-z0-9-]/g, '');

			// Limit to 63 chars
			if (this.value.length > 63) {
				this.value = this.value.substring(0, 63);
			}

			// Update preview
			preview.textContent = this.value + '.stinkykitty.org';

			// Validate and check availability
			validateSubdomain(this.value);
		});

		function validateSubdomain(subdomain) {
			if (!subdomain) {
				status.textContent = '';
				status.className = 'validation-status';
				nextBtn.disabled = true;
				return;
			}

			if (subdomain.length < 2) {
				status.textContent = '✗ At least 2 characters required';
				status.className = 'validation-status error';
				nextBtn.disabled = true;
				return;
			}

			// Check availability via API
			fetch('/admin/api/subdomain-check?subdomain=' + subdomain)
				.then(r => r.json())
				.then(data => {
					if (data.available) {
						status.textContent = '✓ Subdomain available';
						status.className = 'validation-status success';
						nextBtn.disabled = false;
					} else {
						status.textContent = '✗ Subdomain already taken';
						status.className = 'validation-status error';
						nextBtn.disabled = true;
					}
				})
				.catch(e => {
					status.textContent = '✗ Error checking availability';
					status.className = 'validation-status error';
					nextBtn.disabled = true;
				});
		}

		document.getElementById('step1-form').addEventListener('submit', function(e) {
			e.preventDefault();
			const subdomain = document.getElementById('subdomain').value;
			window.location.href = '/admin/create-camp?step=2&subdomain=' + encodeURIComponent(subdomain);
		});
	</script>
</body>
</html>`

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// Placeholder for steps 2 and 3 (will implement in next tasks)
func createCampStep2(c *gin.Context) {
	c.String(http.StatusOK, "Step 2 coming soon")
}

func createCampStep3(c *gin.Context) {
	c.String(http.StatusOK, "Step 3 coming soon")
}
```

**Step 2: Update dashboard button to point to new route**

In `internal/handlers/admin.go`, change line 570 from:
```go
<a href="/admin/pages/new" class="btn">+ Create New Camp</a>
```
to:
```go
<a href="/admin/create-camp" class="btn">+ Create New Camp</a>
```

**Step 3: Add route to server.go**

In `cmd/stinky/server.go`, in the admin routes section, add:

```go
// Create camp workflow
adminGroup.GET("/create-camp", handlers.CreateCampFormHandler)
```

**Step 4: Test page loads and subdomain input works**

Run: `cd /home/lpreimesberger/projects/mex/stinkycat/.worktrees/camp-management && go build ./cmd/stinky`

**Step 5: Commit**

```bash
git add internal/handlers/admin_create_camp.go cmd/stinky/server.go internal/handlers/admin.go
git commit -m "feat: create camp step 1 form with subdomain validation UI"
```

---

### Task 5: Implement subdomain validation AJAX endpoint

**Files:**
- Create: `internal/handlers/admin_api.go`
- Modify: `cmd/stinky/server.go` (add API route)

**Step 1: Create API handler**

```go
// internal/handlers/admin_api.go
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/models"
)

// SubdomainCheckHandler checks if a subdomain is available
func SubdomainCheckHandler(c *gin.Context) {
	subdomain := c.Query("subdomain")

	if subdomain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subdomain required"})
		return
	}

	// Check if subdomain exists (including soft-deleted)
	var site models.Site
	result := db.GetDB().Unscoped().Where("subdomain = ?", subdomain).First(&site)

	if result.Error == nil {
		// Subdomain exists
		c.JSON(http.StatusOK, gin.H{"available": false})
		return
	}

	// Subdomain is available
	c.JSON(http.StatusOK, gin.H{"available": true})
}
```

**Step 2: Add route to server.go**

In `cmd/stinky/server.go`, add this route in the admin routes section (before protected routes):

```go
// API endpoints (no auth required for these)
adminGroup.GET("/api/subdomain-check", handlers.SubdomainCheckHandler)
```

**Step 3: Test endpoint works**

Run: `cd /home/lpreimesberger/projects/mex/stinkycat/.worktrees/camp-management && go build ./cmd/stinky`

**Step 4: Commit**

```bash
git add internal/handlers/admin_api.go cmd/stinky/server.go
git commit -m "feat: add subdomain availability check API endpoint"
```

---

### Task 6: Create user selection/creation step (Step 2) + final create handler

**Files:**
- Modify: `internal/handlers/admin_create_camp.go` (add step 2 & 3, add create handler)
- Modify: `cmd/stinky/server.go` (add POST route for create)

**Step 1: Implement step 2 form (user selection)**

Replace `createCampStep2` function in `admin_create_camp.go`:

```go
func createCampStep2(c *gin.Context) {
	subdomain := c.Query("subdomain")
	if subdomain == "" {
		c.Redirect(http.StatusFound, "/admin/create-camp")
		return
	}

	// Get list of existing users
	var users []struct {
		ID    uint
		Email string
		Name  string
	}
	db.GetDB().Model(&models.User{}).Select("id, email, name").Find(&users)

	usersHTML := `<option value="">-- Create New User --</option>`
	for _, u := range users {
		usersHTML += `<option value="` + fmt.Sprintf("%d", u.ID) + `">` + u.Email + ` (` + u.Name + `)</option>`
	}

	html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>Create Camp - Step 2 - StinkyKitty</title>
	<style>
		` + GetDesignSystemCSS() + `

		.create-container {
			max-width: 600px;
			margin: 0 auto;
			padding: var(--spacing-md);
		}

		.create-header {
			margin-bottom: var(--spacing-lg);
		}

		.create-header h1 {
			font-size: 24px;
			margin-bottom: var(--spacing-base);
		}

		.step-indicator {
			display: flex;
			gap: var(--spacing-md);
			margin-bottom: var(--spacing-lg);
		}

		.step {
			flex: 1;
			padding: var(--spacing-base);
			background: var(--color-bg-card);
			border-radius: var(--radius-sm);
			text-align: center;
			font-size: 13px;
		}

		.step.active {
			background: var(--color-accent);
			color: white;
			font-weight: 600;
		}

		.step.completed {
			background: #28a745;
			color: white;
		}

		.form-group {
			margin-bottom: var(--spacing-md);
		}

		.form-group label {
			display: block;
			margin-bottom: var(--spacing-sm);
			font-weight: 600;
		}

		.form-group input,
		.form-group select {
			width: 100%;
			padding: var(--spacing-sm);
			border: 1px solid var(--color-border);
			border-radius: var(--radius-sm);
			font-size: 14px;
		}

		.form-group input:focus,
		.form-group select:focus {
			outline: none;
			border-color: var(--color-accent);
			box-shadow: 0 0 0 3px rgba(46, 139, 158, 0.1);
		}

		.toggle-section {
			margin-top: var(--spacing-lg);
			padding-top: var(--spacing-lg);
			border-top: 1px solid var(--color-border);
		}

		.toggle-section h3 {
			font-size: 14px;
			margin-bottom: var(--spacing-base);
		}

		#new-user-fields {
			display: none;
		}

		#new-user-fields.visible {
			display: block;
		}

		.button-group {
			display: flex;
			gap: var(--spacing-base);
			margin-top: var(--spacing-lg);
		}

		.btn {
			flex: 1;
			padding: var(--spacing-sm);
			border-radius: var(--radius-sm);
			border: none;
			cursor: pointer;
			font-weight: 600;
			font-size: 14px;
		}

		.btn-primary {
			background: var(--color-accent);
			color: white;
		}

		.btn-primary:hover {
			background: var(--color-accent-hover);
		}

		.btn-secondary {
			background: var(--color-text-secondary);
			color: white;
		}

		.btn-secondary:hover {
			background: #5a6268;
		}
	</style>
</head>
<body>
	<div class="create-container">
		<div class="create-header">
			<h1>Create New Camp</h1>
			<p>Choose the camp admin</p>
		</div>

		<div class="step-indicator">
			<div class="step completed">1. Subdomain</div>
			<div class="step active">2. Admin</div>
			<div class="step">3. Review</div>
		</div>

		<form id="step2-form">
			<input type="hidden" name="subdomain" value="` + subdomain + `">

			<div class="form-group">
				<label for="user-select">Select Existing User</label>
				<select id="user-select" name="user_id" onchange="toggleNewUserForm()">
					` + usersHTML + `
				</select>
			</div>

			<div id="new-user-fields" class="toggle-section">
				<h3>Or Create New User</h3>

				<div class="form-group">
					<label for="new-name">Name</label>
					<input type="text" id="new-name" name="new_name" placeholder="Jane Doe">
				</div>

				<div class="form-group">
					<label for="new-email">Email</label>
					<input type="email" id="new-email" name="new_email" placeholder="jane@example.com">
				</div>

				<div class="form-group">
					<label for="new-password">Password</label>
					<input type="password" id="new-password" name="new_password" placeholder="••••••••">
				</div>
			</div>

			<div class="button-group">
				<a href="/admin/create-camp?step=1" class="btn btn-secondary">← Back</a>
				<button type="submit" class="btn btn-primary">Next: Review →</button>
			</div>
		</form>
	</div>

	<script>
		function toggleNewUserForm() {
			const select = document.getElementById('user-select');
			const newFields = document.getElementById('new-user-fields');

			if (select.value === '') {
				newFields.classList.add('visible');
			} else {
				newFields.classList.remove('visible');
			}
		}

		document.getElementById('step2-form').addEventListener('submit', function(e) {
			e.preventDefault();

			const subdomain = this.querySelector('input[name="subdomain"]').value;
			const userId = document.getElementById('user-select').value;

			let nextUrl = '/admin/create-camp?step=3&subdomain=' + encodeURIComponent(subdomain);

			if (userId) {
				nextUrl += '&user_id=' + userId;
			} else {
				// New user - pass as form data
				const formData = new FormData(this);
				nextUrl += '&new_name=' + encodeURIComponent(formData.get('new_name'));
				nextUrl += '&new_email=' + encodeURIComponent(formData.get('new_email'));
				nextUrl += '&new_password=' + encodeURIComponent(formData.get('new_password'));
			}

			window.location.href = nextUrl;
		});
	</script>
</body>
</html>`

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
```

**Step 2: Implement step 3 confirmation**

Replace `createCampStep3` function:

```go
func createCampStep3(c *gin.Context) {
	subdomain := c.Query("subdomain")
	userID := c.Query("user_id")
	newName := c.Query("new_name")
	newEmail := c.Query("new_email")
	newPassword := c.Query("new_password")

	if subdomain == "" {
		c.Redirect(http.StatusFound, "/admin/create-camp")
		return
	}

	adminDisplay := ""
	if userID != "" {
		// Existing user - fetch their email
		var user models.User
		db.GetDB().First(&user, userID)
		adminDisplay = user.Email
	} else {
		adminDisplay = newEmail + ` (new user)`
	}

	html := `<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>Create Camp - Step 3 - StinkyKitty</title>
	<style>
		` + GetDesignSystemCSS() + `

		.create-container {
			max-width: 600px;
			margin: 0 auto;
			padding: var(--spacing-md);
		}

		.create-header {
			margin-bottom: var(--spacing-lg);
		}

		.create-header h1 {
			font-size: 24px;
			margin-bottom: var(--spacing-base);
		}

		.step-indicator {
			display: flex;
			gap: var(--spacing-md);
			margin-bottom: var(--spacing-lg);
		}

		.step {
			flex: 1;
			padding: var(--spacing-base);
			background: var(--color-bg-card);
			border-radius: var(--radius-sm);
			text-align: center;
			font-size: 13px;
		}

		.step.completed {
			background: #28a745;
			color: white;
		}

		.step.active {
			background: var(--color-accent);
			color: white;
			font-weight: 600;
		}

		.review-section {
			background: var(--color-bg-card);
			border: 1px solid var(--color-border);
			border-radius: var(--radius-base);
			padding: var(--spacing-md);
			margin-bottom: var(--spacing-md);
		}

		.review-item {
			display: flex;
			justify-content: space-between;
			align-items: center;
			padding: var(--spacing-base) 0;
			border-bottom: 1px solid var(--color-border);
		}

		.review-item:last-child {
			border-bottom: none;
		}

		.review-item strong {
			color: var(--color-text-secondary);
			font-weight: 600;
		}

		.review-item em {
			color: var(--color-text-primary);
			font-style: normal;
			word-break: break-all;
		}

		.button-group {
			display: flex;
			gap: var(--spacing-base);
			margin-top: var(--spacing-lg);
		}

		.btn {
			flex: 1;
			padding: var(--spacing-sm);
			border-radius: var(--radius-sm);
			border: none;
			cursor: pointer;
			font-weight: 600;
			font-size: 14px;
		}

		.btn-primary {
			background: var(--color-accent);
			color: white;
		}

		.btn-primary:hover {
			background: var(--color-accent-hover);
		}

		.btn-secondary {
			background: var(--color-text-secondary);
			color: white;
		}

		.btn-secondary:hover {
			background: #5a6268;
		}

		.loading {
			display: none;
			text-align: center;
			padding: var(--spacing-lg);
		}

		.spinner {
			border: 3px solid var(--color-border);
			border-top: 3px solid var(--color-accent);
			border-radius: 50%;
			width: 30px;
			height: 30px;
			animation: spin 1s linear infinite;
			margin: 0 auto var(--spacing-base);
		}

		@keyframes spin {
			0% { transform: rotate(0deg); }
			100% { transform: rotate(360deg); }
		}
	</style>
</head>
<body>
	<div class="create-container">
		<div class="create-header">
			<h1>Create New Camp</h1>
			<p>Review and confirm</p>
		</div>

		<div class="step-indicator">
			<div class="step completed">1. Subdomain</div>
			<div class="step completed">2. Admin</div>
			<div class="step active">3. Review</div>
		</div>

		<div class="review-section">
			<div class="review-item">
				<strong>Camp Subdomain:</strong>
				<em>` + subdomain + `.stinkykitty.org</em>
			</div>
			<div class="review-item">
				<strong>Admin User:</strong>
				<em>` + adminDisplay + `</em>
			</div>
			<div class="review-item">
				<strong>Initial Page:</strong>
				<em>Hello World! (published)</em>
			</div>
		</div>

		<div id="form-section">
			<form id="create-form" method="POST" action="/admin/create-camp-submit">
				<input type="hidden" name="subdomain" value="` + subdomain + `">
				<input type="hidden" name="user_id" value="` + userID + `">
				<input type="hidden" name="new_name" value="` + newName + `">
				<input type="hidden" name="new_email" value="` + newEmail + `">
				<input type="hidden" name="new_password" value="` + newPassword + `">

				<div class="button-group">
					<a href="/admin/create-camp?step=2&subdomain=` + subdomain + `" class="btn btn-secondary">← Back</a>
					<button type="submit" class="btn btn-primary">Create Camp!</button>
				</div>
			</form>
		</div>

		<div id="loading" class="loading">
			<div class="spinner"></div>
			<p>Creating your camp...</p>
		</div>
	</div>

	<script>
		document.getElementById('create-form').addEventListener('submit', function(e) {
			e.preventDefault();
			document.getElementById('form-section').style.display = 'none';
			document.getElementById('loading').style.display = 'block';
			this.submit();
		});
	</script>
</body>
</html>`

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
```

**Step 3: Add create handler**

Add new function to `admin_create_camp.go`:

```go
// CreateCampSubmitHandler processes the final camp creation
func CreateCampSubmitHandler(c *gin.Context) {
	// Get user from context
	userVal, exists := c.Get("user")
	if !exists {
		c.Redirect(http.StatusFound, "/admin/login")
		return
	}
	user := userVal.(*models.User)

	subdomain := c.PostForm("subdomain")
	userIDStr := c.PostForm("user_id")
	newName := c.PostForm("new_name")
	newEmail := c.PostForm("new_email")
	newPassword := c.PostForm("new_password")

	if subdomain == "" {
		c.String(http.StatusBadRequest, "subdomain required")
		return
	}

	var ownerID uint

	if userIDStr != "" {
		// Use existing user
		var tempID uint
		fmt.Sscanf(userIDStr, "%d", &tempID)
		ownerID = tempID
	} else {
		// Create new user
		if newEmail == "" || newPassword == "" || newName == "" {
			c.String(http.StatusBadRequest, "new user fields required")
			return
		}

		// Hash password
		hash, err := auth.HashPassword(newPassword)
		if err != nil {
			c.String(http.StatusInternalServerError, "failed to process password")
			return
		}

		newUser := &models.User{
			Email:        newEmail,
			Name:         newName,
			PasswordHash: hash,
		}

		if err := db.GetDB().Create(newUser).Error; err != nil {
			c.String(http.StatusInternalServerError, "failed to create user")
			return
		}

		ownerID = newUser.ID
	}

	// Create site with Hello World page
	sitesDir := "/var/lib/stinkykitty/sites"
	site, err := sites.CreateSite(db.GetDB(), subdomain, ownerID, sitesDir)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to create site: "+err.Error())
		return
	}

	// Redirect to edit the Hello World page
	c.Redirect(http.StatusFound, fmt.Sprintf("/admin/pages/%d/edit?site=%d", site.ID, site.ID))
}
```

**Step 4: Add POST route to server.go**

In `cmd/stinky/server.go`, add:

```go
adminGroup.POST("/create-camp-submit", handlers.CreateCampSubmitHandler)
```

**Step 5: Add required imports to admin_create_camp.go**

At top of file, add:
```go
import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/auth"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/models"
	"github.com/thatcatcamp/stinkykitty/internal/sites"
)
```

**Step 6: Test compiles**

Run: `cd /home/lpreimesberger/projects/mex/stinkycat/.worktrees/camp-management && go build ./cmd/stinky`

**Step 7: Commit**

```bash
git add internal/handlers/admin_create_camp.go cmd/stinky/server.go
git commit -m "feat: add create camp steps 2-3 with user selection and submission"
```

---

### Task 7: Test all flows end-to-end

**Files:**
- No code changes (testing only)

**Step 1: Manual test delete flow**

1. Start server: `cd /home/lpreimesberger/projects/mex/stinkycat/.worktrees/camp-management && go run ./cmd/stinky server start`
2. Navigate to dashboard
3. Click delete button on any site
4. Verify modal appears with correct site name
5. Click cancel - verify modal closes
6. Click delete again, then confirm - verify site disappears from list

**Step 2: Manual test create flow**

1. Click "Create New Camp" button
2. Enter subdomain "test-camp-xyz"
3. Verify live validation shows available/taken status
4. Click "Next"
5. Select existing user or create new
6. Click "Next"
7. Verify review shows correct info
8. Click "Create Camp!"
9. Verify redirects to edit page for Hello World!
10. Verify page is published and visible

**Step 3: Run full test suite**

Run: `cd /home/lpreimesberger/projects/mex/stinkycat/.worktrees/camp-management && go test ./... -v 2>&1 | tail -20`

Expected: All tests pass

**Step 4: Commit**

No code changes for testing, but document results:

```bash
git log --oneline | head -1  # Should show your latest commit
```

---

## Implementation Complete!

**Summary of changes:**
- Modified CreateSite to publish "Hello World!" page
- Added delete site handler with soft-delete
- Updated dashboard with delete button and confirmation modal
- Created 3-step create camp workflow with subdomain validation
- Added AJAX subdomain availability check
- Implemented user selection/creation during camp creation
- Auto-redirect to edit new camp's Hello World page

**Ready to merge back to main!**

