// SPDX-License-Identifier: MIT
package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thatcatcamp/stinkykitty/internal/db"
	"github.com/thatcatcamp/stinkykitty/internal/models"
)

// MediaPickerHandler shows modal picker for block editors
func MediaPickerHandler(c *gin.Context) {
	// Get site from context
	siteVal, exists := c.Get("site")
	if !exists {
		c.String(http.StatusInternalServerError, "Site not found")
		return
	}
	site := siteVal.(*models.Site)

	// Get all media items for this site
	var mediaItems []models.MediaItem
	db.GetDB().Where("site_id = ?", site.ID).
		Preload("Tags").
		Order("created_at DESC").
		Find(&mediaItems)

	// Build image grid
	var imageGrid string
	for _, item := range mediaItems {
		thumbURL := fmt.Sprintf("/uploads/thumbs/%s", item.Filename)
		imageURL := fmt.Sprintf("/uploads/%s", item.Filename)

		imageGrid += fmt.Sprintf(`
		<div class="picker-card" onclick="selectImage('%s', '%s')">
			<img src="%s" alt="%s">
			<div class="picker-filename">%s</div>
		</div>
		`, imageURL, item.OriginalName, thumbURL, item.OriginalName, item.OriginalName)
	}

	if len(mediaItems) == 0 {
		imageGrid = `<div class="empty-state">No images in library. Upload images from the Media Library page.</div>`
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>Select Image</title>
	<style>
		%s

		body {
			margin: 0;
			padding: var(--spacing-base);
		}

		.picker-header {
			margin-bottom: var(--spacing-base);
			display: flex;
			justify-content: space-between;
			align-items: center;
		}

		.picker-grid {
			display: grid;
			grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
			gap: var(--spacing-base);
		}

		.picker-card {
			cursor: pointer;
			border: 2px solid transparent;
			border-radius: var(--radius-base);
			overflow: hidden;
			transition: border-color 0.2s;
		}

		.picker-card:hover {
			border-color: var(--color-accent);
		}

		.picker-card img {
			width: 100%%;
			height: 150px;
			object-fit: cover;
			display: block;
		}

		.picker-filename {
			padding: var(--spacing-sm);
			font-size: 12px;
			text-align: center;
			white-space: nowrap;
			overflow: hidden;
			text-overflow: ellipsis;
		}

		.empty-state {
			text-align: center;
			padding: calc(var(--spacing-base) * 4);
			color: var(--color-text-secondary);
		}
	</style>
</head>
<body>
	<div class="picker-header">
		<h2>Select Image</h2>
		<div style="display: flex; gap: 10px;">
			<button onclick="document.getElementById('upload-input').click()" class="btn btn-primary">Upload New</button>
			<button onclick="window.close()" class="btn btn-secondary">Cancel</button>
		</div>
	</div>

	<input type="file" id="upload-input" accept="image/*" style="display: none;">

	<div class="picker-grid">
		%s
	</div>

	<script>
		function selectImage(url, filename) {
			// Send message to parent window
			if (window.opener) {
				window.opener.postMessage({
					type: 'image-selected',
					url: url,
					filename: filename
				}, window.location.origin);
				window.close();
			}
		}

		// Get CSRF token from cookie
		function getCsrfToken() {
			const value = document.cookie
				.split('; ')
				.find(row => row.startsWith('csrf_token='));
			if (!value) return '';
			return decodeURIComponent(value.split('=')[1]);
		}

		// Handle file upload
		document.addEventListener('DOMContentLoaded', function() {
			const uploadInput = document.getElementById('upload-input');
			if (uploadInput) {
				uploadInput.addEventListener('change', async function(e) {
					const file = e.target.files[0];
					if (!file) return;

					// Show loading state
					const header = document.querySelector('.picker-header h2');
					const originalText = header.textContent;
					header.textContent = 'Uploading...';

					try {
						const formData = new FormData();
						formData.append('images', file);

						const response = await fetch('/admin/media/upload', {
							method: 'POST',
							headers: {
								'X-CSRF-Token': getCsrfToken()
							},
							body: formData
						});

						const result = await response.json();

						if (result.success && result.items && result.items.length > 0) {
							const uploadedItem = result.items[0];
							const url = '/uploads/' + uploadedItem.Filename;
							selectImage(url, uploadedItem.OriginalName);
						} else {
							alert('Upload failed: ' + (result.error || 'Unknown error'));
							header.textContent = originalText;
						}
					} catch (error) {
						alert('Upload failed: ' + error.message);
						header.textContent = originalText;
					}
				});
			}
		});
	</script>
</body>
</html>`, GetDesignSystemCSS(), imageGrid)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
