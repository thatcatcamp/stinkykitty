package handlers

import (
	"errors"
	"fmt"
	"html"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/auth"
	"github.com/thatcatcamp/stinkykitty/internal/config"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/email"
	"github.com/thatcatcamp/stinkykitty/internal/models"
	"github.com/thatcatcamp/stinkykitty/internal/sites"
	"gorm.io/gorm"
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
	baseDomain := config.GetString("server.base_domain")
	if baseDomain == "" {
		baseDomain = "localhost"
	}

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
					Your camp will be at: <strong id="preview">mycamp.` + baseDomain + `</strong>
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
		const baseDomain = '` + baseDomain + `';
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
			preview.textContent = this.value + '.' + baseDomain;

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
	if err := db.GetDB().Model(&models.User{}).Select("id, email").Find(&users).Error; err != nil {
		c.String(http.StatusInternalServerError, "failed to load users")
		return
	}

	usersHTML := `<option value="">-- Create New User --</option>`
	for _, u := range users {
		usersHTML += `<option value="` + fmt.Sprintf("%d", u.ID) + `">` + html.EscapeString(u.Email) + `</option>`
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
				<label for="user-select">Admin User</label>
				<select id="user-select" name="user_id" onchange="toggleNewUserForm()">
					` + usersHTML + `
				</select>
			</div>

			<div id="new-user-fields" class="toggle-section">
				<div style="background: #f0f4f8; padding: var(--spacing-base); border-radius: var(--radius-sm); margin-bottom: var(--spacing-md);">
					<p style="margin: 0; font-size: 12px; color: var(--color-text-secondary);">A new user will be created with the email:</p>
					<p style="margin: var(--spacing-xs) 0 0 0; font-weight: 600; font-size: 13px;">
						<span id="suggested-email">admin@` + subdomain + `.campasaur.us</span>
					</p>
					<p style="margin: var(--spacing-xs) 0 0 0; font-size: 11px; color: var(--color-text-secondary);">They will receive an email with instructions to set their password.</p>
				</div>

				<div class="form-group">
					<label for="new-email">Email Address</label>
					<input type="email" id="new-email" name="new_email" value="admin@` + subdomain + `.campasaur.us" placeholder="email@example.com" required>
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

			// NEVER pass password in URL - use POST with hidden form instead
			let nextUrl = '/admin/create-camp?step=3&subdomain=' + encodeURIComponent(subdomain);

			if (userId) {
				nextUrl += '&user_id=' + userId;
				window.location.href = nextUrl;
			} else {
				// New user - POST to step 3 with form data
				const form = document.createElement('form');
				form.method = 'POST';
				form.action = '/admin/create-camp?step=3';

				const fields = ['subdomain', 'new_email'];
				fields.forEach(name => {
					const input = document.createElement('input');
					input.type = 'hidden';
					input.name = name;
					const element = this.querySelector('[name="' + name + '"]');
					if (element) {
						input.value = element.value;
						form.appendChild(input);
					}
				});

				document.body.appendChild(form);
				form.submit();
			}
		});

		// Call on page load to show fields if "Create New User" is selected
		toggleNewUserForm();
	</script>
</body>
</html>`

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

func createCampStep3(c *gin.Context) {
	// Get base domain from config
	baseDomain := config.GetString("server.base_domain")
	if baseDomain == "" {
		baseDomain = "localhost"
	}

	// Accept both POST (with password) and GET (existing user) to support step 2 navigation
	subdomain := c.Query("subdomain")
	if subdomain == "" {
		subdomain = c.PostForm("subdomain")
	}

	userID := c.Query("user_id")
	if userID == "" {
		userID = c.PostForm("user_id")
	}

	newEmail := c.PostForm("new_email") // Only from POST

	if subdomain == "" {
		c.Redirect(http.StatusFound, "/admin/create-camp")
		return
	}

	adminDisplay := ""
	if userID != "" {
		// Existing user - fetch their email
		var user models.User
		if err := db.GetDB().First(&user, userID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.String(http.StatusBadRequest, "user not found")
			} else {
				c.String(http.StatusInternalServerError, "database error")
			}
			return
		}
		adminDisplay = html.EscapeString(user.Email)
	} else {
		adminDisplay = html.EscapeString(newEmail) + ` (new user)`
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
				<em>` + html.EscapeString(subdomain) + `.` + baseDomain + `</em>
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
				<input type="hidden" name="subdomain" value="` + html.EscapeString(subdomain) + `">
				<input type="hidden" name="user_id" value="` + html.EscapeString(userID) + `">
				<input type="hidden" name="new_email" value="` + html.EscapeString(newEmail) + `">

				<div class="button-group">
					<a href="/admin/create-camp?step=2&subdomain=` + html.EscapeString(subdomain) + `" class="btn btn-secondary">← Back</a>
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
	newEmail := strings.ToLower(strings.TrimSpace(c.PostForm("new_email")))

	// Issue #2: SERVER-SIDE SUBDOMAIN VALIDATION
	if subdomain == "" {
		c.String(http.StatusBadRequest, "subdomain required")
		return
	}

	// Normalize and validate subdomain
	subdomain = strings.TrimSpace(subdomain)
	if len(subdomain) < 2 || len(subdomain) > 63 {
		c.String(http.StatusBadRequest, "subdomain must be 2-63 characters")
		return
	}

	// Validate format (RFC 1123)
	validSubdomain := regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)
	if !validSubdomain.MatchString(subdomain) {
		c.String(http.StatusBadRequest, "subdomain contains invalid characters")
		return
	}

	// Check reserved subdomains
	reservedSubdomains := map[string]bool{
		"admin": true, "api": true, "www": true, "mail": true,
		"ftp": true, "smtp": true, "pop": true, "imap": true,
		"stinky": true, "status": true,
	}
	if reservedSubdomains[subdomain] {
		c.String(http.StatusBadRequest, "subdomain is reserved")
		return
	}

	var ownerID uint

	if userIDStr != "" {
		// Issue #7: INTEGER PARSING VALIDATION
		var tempID uint
		if _, err := fmt.Sscanf(userIDStr, "%d", &tempID); err != nil || tempID == 0 {
			c.String(http.StatusBadRequest, "invalid user id")
			return
		}
		ownerID = tempID
	} else {
		// Create new user
		if newEmail == "" {
			c.String(http.StatusBadRequest, "email required for new user")
			return
		}

		// Issue #3: EMAIL VALIDATION
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(newEmail) {
			c.String(http.StatusBadRequest, "invalid email format")
			return
		}

		// Check for existing active email
		var existingUser models.User
		result := db.GetDB().Where("email = ?", newEmail).First(&existingUser)
		if result.Error == nil {
			// Active user with this email already exists
			c.String(http.StatusBadRequest, "email already in use")
			return
		}
		if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.String(http.StatusInternalServerError, "database error")
			return
		}

		// Check for soft-deleted user with this email
		var deletedUser models.User
		deletedResult := db.GetDB().Unscoped().Where("email = ? AND deleted_at IS NOT NULL", newEmail).First(&deletedUser)

		var newUser *models.User
		if deletedResult.Error == nil {
			// Found a soft-deleted user - restore them
			log.Printf("INFO: Restoring soft-deleted user with email %s (ID: %d)", newEmail, deletedUser.ID)
			newUser = &deletedUser
			// Clear the soft delete and reset their data
			if err := db.GetDB().Unscoped().Model(newUser).Updates(map[string]interface{}{
				"deleted_at":    nil,
				"password_hash": "",
				"reset_token":   "",
				"reset_expires": time.Time{},
			}).Error; err != nil {
				c.String(http.StatusInternalServerError, "failed to restore user")
				return
			}
		} else {
			// No deleted user found - create new user
			newUser = &models.User{
				Email:        newEmail,
				PasswordHash: "", // No password set initially
			}

			if err := db.GetDB().Create(newUser).Error; err != nil {
				c.String(http.StatusInternalServerError, "failed to create user")
				return
			}
		}

		// Generate reset token and send password setup email
		token, tokenErr := auth.GenerateResetToken()
		if tokenErr != nil {
			log.Printf("ERROR: Failed to generate reset token: %v", tokenErr)
		}

		updateResult := db.GetDB().Model(&newUser).Updates(map[string]interface{}{
			"reset_token":   token,
			"reset_expires": time.Now().Add(24 * time.Hour),
		})
		if updateResult.Error != nil {
			log.Printf("ERROR: Failed to save reset token for user %s: %v", newEmail, updateResult.Error)
		}

		log.Printf("INFO: Attempting to send password reset email to %s", newEmail)
		svc, err := email.NewEmailService()
		if err != nil {
			log.Printf("ERROR: Failed to create email service: %v", err)
			log.Printf("ERROR: Check SMTP environment variables: SMTP, SMTP_PORT, EMAIL, SMTP_SECRET")
		} else {
			baseDomain := config.GetString("server.base_domain")
			if baseDomain == "" {
				baseDomain = "campasaur.us"
			}
			resetURL := fmt.Sprintf("https://%s/admin/reset-confirm?token=%s", baseDomain, token)
			log.Printf("INFO: Reset URL: %s", resetURL)
			if err := svc.SendPasswordReset(newEmail, resetURL); err != nil {
				log.Printf("ERROR: Failed to send password reset email to %s: %v", newEmail, err)
			} else {
				log.Printf("SUCCESS: Password reset email sent to %s", newEmail)
			}
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

	// Add the owner as a site admin in site_users table
	siteUser := &models.SiteUser{
		SiteID: site.ID,
		UserID: ownerID,
		Role:   "owner",
	}
	if err := db.GetDB().Create(siteUser).Error; err != nil {
		log.Printf("WARNING: Failed to create site_users entry for owner: %v", err)
	}

	// Issue #6: DATABASE ERROR HANDLING - Find the homepage (published page with slug "/")
	var homepage models.Page
	if err := db.GetDB().Where("site_id = ? AND slug = ?", site.ID, "/").First(&homepage).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.String(http.StatusInternalServerError, "homepage not found after creation")
		} else {
			c.String(http.StatusInternalServerError, "database error")
		}
		return
	}

	// Redirect to main dashboard (creator might not have access to the new site)
	c.Redirect(http.StatusFound, "/admin/dashboard")
}
