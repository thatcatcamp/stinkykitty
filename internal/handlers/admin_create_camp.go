package handlers

import (
	"net/http"

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

// Placeholder for steps 2 and 3 (will implement in next tasks)
func createCampStep2(c *gin.Context) {
	c.String(http.StatusOK, "Step 2 coming soon")
}

func createCampStep3(c *gin.Context) {
	c.String(http.StatusOK, "Step 3 coming soon")
}
