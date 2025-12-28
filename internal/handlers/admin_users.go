package handlers

import (
	"fmt"
	"html"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/auth"
	"github.com/thatcatcamp/stinkykitty/internal/config"
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
		`, html.EscapeString(user.Email), html.EscapeString(user.Sites), html.EscapeString(user.Role), user.CreatedAt.Format("2006-01-02"), user.ID, user.ID)
	}

	htmlContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>User Management - StinkyKitty</title>
	<style>%s
		body { padding: 0; }
		.content-wrapper {
			max-width: 1200px;
			margin: 0 auto;
			padding: var(--spacing-md);
		}
	</style>
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

	<div class="content-wrapper">
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
		baseDomain := config.GetString("server.base_domain")
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
