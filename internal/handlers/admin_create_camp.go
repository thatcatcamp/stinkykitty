package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/auth"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/models"
	"github.com/thatcatcamp/stinkykitty/internal/sites"
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
			box-sizing: border-box;
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
	}
	db.GetDB().Model(&models.User{}).Select("id, email").Find(&users)

	usersHTML := `<option value="">-- Create New User --</option>`
	for _, u := range users {
		usersHTML += `<option value="` + fmt.Sprintf("%d", u.ID) + `">` + u.Email + `</option>`
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

func createCampStep3(c *gin.Context) {
	subdomain := c.Query("subdomain")
	userID := c.Query("user_id")
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

// CreateCampSubmitHandler processes the final camp creation
func CreateCampSubmitHandler(c *gin.Context) {
	// Get user from context (for auth check)
	_, exists := c.Get("user")
	if !exists {
		c.Redirect(http.StatusFound, "/admin/login")
		return
	}

	subdomain := c.PostForm("subdomain")
	userIDStr := c.PostForm("user_id")
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
		if newEmail == "" || newPassword == "" {
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
