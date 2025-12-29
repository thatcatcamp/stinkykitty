# CSRF Protection Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Enable CSRF protection by adding tokens to all forms and AJAX requests

**Architecture:** Custom CSRF middleware already implemented in `internal/middleware/csrf.go`. Token stored in cookie and validated on POST/PUT/PATCH/DELETE requests. Forms must include token as hidden field; AJAX must include token in X-CSRF-Token header.

**Tech Stack:** Go, Gin framework, custom CSRF middleware

**Scope:**
- 42 forms across 9 handler files
- 4 fetch() API calls
- CSRF middleware activation in server.go

---

## Task 1: Create CSRF Token Helper Function

**Files:**
- Modify: `internal/middleware/csrf.go:79-86`

**Step 1: Add HTML helper function**

Add this function after `GetCSRFToken`:

```go
// GetCSRFTokenHTML returns an HTML hidden input field with the CSRF token
func GetCSRFTokenHTML(c *gin.Context) string {
	token := GetCSRFToken(c)
	if token == "" {
		return ""
	}
	return `<input type="hidden" name="csrf_token" value="` + token + `">`
}
```

**Step 2: Verify the function compiles**

Run: `go build ./internal/middleware`
Expected: SUCCESS (no errors)

**Step 3: Commit**

```bash
git add internal/middleware/csrf.go
git commit -m "feat: add CSRF token HTML helper function"
```

---

## Task 2: Update Login Form (admin.go)

**Files:**
- Modify: `internal/handlers/admin.go:215-227`

**Step 1: Import middleware package**

Add to imports at top of file (around line 10):

```go
"github.com/thatcatcamp/stinkykitty/internal/middleware"
```

**Step 2: Update LoginFormHandler**

Replace lines 215-227 with:

```go
            <form method="POST" action="/admin/login">
                ` + middleware.GetCSRFTokenHTML(c) + `
                <div class="form-group">
                    <label for="email">Email</label>
                    <input type="email" id="email" name="email" placeholder="admin@example.com" autocomplete="email" required>
                </div>

                <div class="form-group">
                    <label for="password">Password</label>
                    <input type="password" id="password" name="password" autocomplete="current-password" required>
                </div>

                <button type="submit" class="login-button">Sign In</button>
            </form>
```

**Step 3: Update logout form**

Find line ~567 and replace the logout form:

```go
                    <form method="POST" action="/admin/logout" style="display:inline;">
                        ` + middleware.GetCSRFTokenHTML(c) + `
                        <button type="submit" class="btn btn-sm btn-danger">Logout</button>
                    </form>
```

**Step 4: Verify it compiles**

Run: `go build ./internal/handlers`
Expected: SUCCESS

**Step 5: Commit**

```bash
git add internal/handlers/admin.go
git commit -m "feat: add CSRF tokens to login and logout forms"
```

---

## Task 3: Update Block Edit Forms (admin_blocks.go)

**Files:**
- Modify: `internal/handlers/admin_blocks.go` (9 forms at lines: 274, 395, 445, 528, 574, 637, 681, 739, 940)

**Step 1: Import middleware package**

Add to imports:

```go
"github.com/thatcatcamp/stinkykitty/internal/middleware"
```

**Step 2: Update each form tag**

For each form, change from:
```go
<form method="POST" action="...">
```

To:
```go
<form method="POST" action="...">
    ` + middleware.GetCSRFTokenHTML(c) + `
```

Apply this pattern to all 9 forms in the file.

**Step 3: Verify it compiles**

Run: `go build ./internal/handlers`
Expected: SUCCESS

**Step 4: Commit**

```bash
git add internal/handlers/admin_blocks.go
git commit -m "feat: add CSRF tokens to block edit forms"
```

---

## Task 4: Update Camp Creation Form (admin_create_camp.go)

**Files:**
- Modify: `internal/handlers/admin_create_camp.go:755`

**Step 1: Import middleware package**

Add to imports:

```go
"github.com/thatcatcamp/stinkykitty/internal/middleware"
```

**Step 2: Update create-camp-submit form**

Replace line 755:

```go
			<form id="create-form" method="POST" action="/admin/create-camp-submit">
				` + middleware.GetCSRFTokenHTML(c) + `
```

**Step 3: Verify it compiles**

Run: `go build ./internal/handlers`
Expected: SUCCESS

**Step 4: Commit**

```bash
git add internal/handlers/admin_create_camp.go
git commit -m "feat: add CSRF token to camp creation form"
```

---

## Task 5: Update Menu Forms (admin_menu.go)

**Files:**
- Modify: `internal/handlers/admin_menu.go` (4 forms at lines: 44, 53, 69, 159)

**Step 1: Import middleware package**

Add to imports:

```go
"github.com/thatcatcamp/stinkykitty/internal/middleware"
```

**Step 2: Update inline forms with token variable**

The forms in admin_menu.go are generated dynamically. Update the MenuHandler function to create a csrf token variable first:

```go
func MenuHandler(c *gin.Context) {
	csrfToken := middleware.GetCSRFTokenHTML(c)
	// ... rest of function
```

Then update each inline form to include `+ csrfToken +`:

Line ~44:
```go
moveUpBtn = `<form method="POST" action="/admin/menu/` + strconv.Itoa(int(item.ID)) + `/move-up" style="display:inline;">
	` + csrfToken + `
	<button...`
```

Line ~53:
```go
moveDownBtn = `<form method="POST" action="/admin/menu/` + strconv.Itoa(int(item.ID)) + `/move-down" style="display:inline;">
	` + csrfToken + `
	<button...`
```

Line ~69:
```go
<form method="POST" action="/admin/menu/%d/delete" style="display:inline;" onsubmit="return confirm('Delete this menu item?')">
	` + csrfToken + `
	<button...`
```

Line ~159 (Add Menu Item form):
```go
            <form method="POST" action="/admin/menu">
                ` + csrfToken + `
```

**Step 3: Verify it compiles**

Run: `go build ./internal/handlers`
Expected: SUCCESS

**Step 4: Commit**

```bash
git add internal/handlers/admin_menu.go
git commit -m "feat: add CSRF tokens to menu forms"
```

---

## Task 6: Update Page Forms (admin_pages.go)

**Files:**
- Modify: `internal/handlers/admin_pages.go` (4 forms at lines: 122, 299, 308, 325)

**Step 1: Import middleware package**

Add to imports:

```go
"github.com/thatcatcamp/stinkykitty/internal/middleware"
```

**Step 2: Update NewPageFormHandler**

Line ~122:

```go
        <form method="POST" action="/admin/pages">
            ` + middleware.GetCSRFTokenHTML(c) + `
```

**Step 3: Update EditPageHandler inline forms**

Add csrf token variable at the start of EditPageHandler:

```go
func EditPageHandler(c *gin.Context) {
	csrfToken := middleware.GetCSRFTokenHTML(c)
	// ... rest of function
```

Then update the inline button forms (~lines 299, 308, 325):

```go
moveUpBtn = `<form method="POST" action="/admin/pages/` + pageIDStr + `/blocks/` + strconv.Itoa(int(block.ID)) + `/move-up" style="display:inline;">
	` + csrfToken + `
	<button...`
```

```go
moveDownBtn = `<form method="POST" action="/admin/pages/` + pageIDStr + `/blocks/` + strconv.Itoa(int(block.ID)) + `/move-down" style="display:inline;">
	` + csrfToken + `
	<button...`
```

```go
<form method="POST" action="/admin/pages/` + pageIDStr + `/blocks/` + strconv.Itoa(int(block.ID)) + `/delete" style="display:inline;" onsubmit="return confirm('Delete this block?')">
	` + csrfToken + `
	<button...`
```

**Step 4: Verify it compiles**

Run: `go build ./internal/handlers`
Expected: SUCCESS

**Step 5: Commit**

```bash
git add internal/handlers/admin_pages.go
git commit -m "feat: add CSRF tokens to page forms"
```

---

## Task 7: Update Settings Form (admin_settings.go)

**Files:**
- Modify: `internal/handlers/admin_settings.go`

**Step 1: Import middleware package**

Add to imports:

```go
"github.com/thatcatcamp/stinkykitty/internal/middleware"
```

**Step 2: Find and update settings form**

Find the settings form (search for `<form method="POST"` in the file) and add CSRF token:

```go
<form method="POST" action="/admin/settings">
    ` + middleware.GetCSRFTokenHTML(c) + `
```

**Step 3: Verify it compiles**

Run: `go build ./internal/handlers`
Expected: SUCCESS

**Step 4: Commit**

```bash
git add internal/handlers/admin_settings.go
git commit -m "feat: add CSRF token to settings form"
```

---

## Task 8: Update User Management Forms (admin_users.go)

**Files:**
- Modify: `internal/handlers/admin_users.go`

**Step 1: Import middleware package**

Add to imports:

```go
"github.com/thatcatcamp/stinkykitty/internal/middleware"
```

**Step 2: Find and update user management forms**

Find all POST forms (reset password, delete user) and add CSRF token variable:

```go
func UsersListHandler(c *gin.Context) {
	csrfToken := middleware.GetCSRFTokenHTML(c)
	// ... rest of function
```

Then update each form to include `+ csrfToken +`

**Step 3: Verify it compiles**

Run: `go build ./internal/handlers`
Expected: SUCCESS

**Step 4: Commit**

```bash
git add internal/handlers/admin_users.go
git commit -m "feat: add CSRF tokens to user management forms"
```

---

## Task 9: Update Password Reset Forms (password_reset.go)

**Files:**
- Modify: `internal/handlers/password_reset.go`

**Step 1: Import middleware package**

Add to imports:

```go
"github.com/thatcatcamp/stinkykitty/internal/middleware"
```

**Step 2: Find and update password reset forms**

Find all POST forms and add CSRF token:

```go
<form method="POST" action="...">
    ` + middleware.GetCSRFTokenHTML(c) + `
```

**Step 3: Verify it compiles**

Run: `go build ./internal/handlers`
Expected: SUCCESS

**Step 4: Commit**

```bash
git add internal/handlers/password_reset.go
git commit -m "feat: add CSRF tokens to password reset forms"
```

---

## Task 10: Update Contact Form (public.go)

**Files:**
- Modify: `internal/handlers/public.go`

**Step 1: Import middleware package**

Add to imports:

```go
"github.com/thatcatcamp/stinkykitty/internal/middleware"
```

**Step 2: Find and update contact form**

Find the contact form (search for `<form method="POST"`) and add CSRF token:

```go
<form method="POST" action="/contact">
    ` + middleware.GetCSRFTokenHTML(c) + `
```

**Step 3: Verify it compiles**

Run: `go build ./internal/handlers`
Expected: SUCCESS

**Step 4: Commit**

```bash
git add internal/handlers/public.go
git commit -m "feat: add CSRF token to contact form"
```

---

## Task 11: Update AJAX/Fetch Calls with CSRF Tokens

**Files:**
- Modify: `internal/handlers/admin.go:682` (DELETE site fetch)
- Modify: `internal/handlers/admin_pages.go:1009` (image upload fetch)

**Step 1: Update DELETE site fetch in admin.go**

Find line ~682 and update the fetch call to include CSRF token header:

```javascript
            const csrfToken = document.cookie
                .split('; ')
                .find(row => row.startsWith('csrf_token='))
                ?.split('=')[1] || '';

            fetch('/admin/sites/' + pendingDeleteSiteId + '/delete', {
                method: 'DELETE',
                headers: {
                    'X-CSRF-Token': csrfToken
                }
            })
```

**Step 2: Update image upload fetch in admin_pages.go**

Find line ~1009 and update the fetch call:

```javascript
                const csrfToken = document.cookie
                    .split('; ')
                    .find(row => row.startsWith('csrf_token='))
                    ?.split('=')[1] || '';

                const response = await fetch('/admin/upload/image', {
                    method: 'POST',
                    headers: {
                        'X-CSRF-Token': csrfToken
                    },
                    body: formData
                });
```

**Step 3: Verify it compiles**

Run: `go build ./internal/handlers`
Expected: SUCCESS

**Step 4: Commit**

```bash
git add internal/handlers/admin.go internal/handlers/admin_pages.go
git commit -m "feat: add CSRF tokens to AJAX fetch calls"
```

---

## Task 12: Enable CSRF Middleware

**Files:**
- Modify: `cmd/stinky/server.go:168-169`

**Step 1: Uncomment CSRF middleware**

Replace lines 168-169:

```go
				// Protected admin routes (auth required + CSRF protection)
				adminGroup.Use(auth.RequireAuth())
				adminGroup.Use(middleware.CSRFMiddleware())
```

(Remove the TODO comment and uncomment the middleware)

**Step 2: Verify it compiles**

Run: `go build ./cmd/stinky`
Expected: SUCCESS

**Step 3: Commit**

```bash
git add cmd/stinky/server.go
git commit -m "feat: enable CSRF middleware for all admin routes"
```

---

## Task 13: Manual Testing

**Step 1: Start the server**

Run: `go run cmd/stinky/main.go server start`
Expected: Server starts without errors

**Step 2: Test login flow**

1. Navigate to `/admin/login`
2. Inspect the form - verify hidden `csrf_token` field is present
3. Submit login with valid credentials
4. Expected: Successful login, redirect to dashboard

**Step 3: Test form submission**

1. Create a new page
2. Verify the form works without "Invalid CSRF token" error
3. Test editing blocks
4. Test menu items
5. Test settings

**Step 4: Test logout**

1. Click logout button
2. Expected: Successfully logged out

**Step 5: Test AJAX operations**

1. Try uploading an image (block editor)
2. Expected: Upload succeeds
3. Try deleting a site (if available)
4. Expected: Delete succeeds

**Step 6: Test CSRF protection is working**

1. Use browser dev tools to modify the csrf_token value in a form
2. Submit the form
3. Expected: "Invalid CSRF token" error (403 status)

**Step 7: Document results**

Create test report in: `docs/csrf-testing-results.md`

---

## Task 14: Clean Up Uncommitted Changes

**Files:**
- Check: `go.mod`, `go.sum`, `internal/handlers/styles.go`, `bugs.md`

**Step 1: Review uncommitted changes**

Run: `git diff`
Expected: Shows changes to go.mod, go.sum, styles.go, bugs.md

**Step 2: Determine if changes are needed**

- If CSRF dependencies in go.mod/go.sum are unused, remove them
- If btn-contact styles are needed, commit them
- If bugs.md updates are valid, commit them

**Step 3: Clean up or commit**

Either:
```bash
git checkout go.mod go.sum  # if dependencies are unused
```

Or:
```bash
git add go.mod go.sum internal/handlers/styles.go bugs.md
git commit -m "chore: clean up uncommitted changes"
```

**Step 4: Verify clean working tree**

Run: `git status`
Expected: Clean working tree or only intentional changes

---

## Completion Checklist

- [ ] CSRF helper function created
- [ ] All 42 forms updated with CSRF tokens
- [ ] All 4 fetch() calls updated with CSRF header
- [ ] CSRF middleware enabled in server.go
- [ ] Manual testing completed successfully
- [ ] CSRF protection verified (rejected invalid tokens)
- [ ] Uncommitted changes cleaned up
- [ ] All commits have clear messages

**Security Verification:**
- Forms include hidden csrf_token field
- AJAX calls include X-CSRF-Token header
- Invalid tokens are rejected with 403
- Token is regenerated on new sessions
