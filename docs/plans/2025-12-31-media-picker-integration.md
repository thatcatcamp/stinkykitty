# Media Picker Integration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Replace prompt-based image insertion with integrated media library picker supporting both browse and upload.

**Architecture:** Enhanced modal picker with AJAX upload, using postMessage for parent-child communication. All uploads create MediaItem records to prevent orphaned files.

**Tech Stack:** Go (Gin framework), vanilla JavaScript, existing media library system

---

## Task 1: Add Upload Button to Media Picker Modal

**Files:**
- Modify: `internal/handlers/admin_media_picker.go:110-114`

**Step 1: Add upload button and file input to picker header**

In `admin_media_picker.go`, modify the picker header section (around line 111):

```go
<div class="picker-header">
    <h2>Select Image</h2>
    <div style="display: flex; gap: 10px;">
        <button onclick="document.getElementById('upload-input').click()" class="btn btn-primary">Upload New</button>
        <button onclick="window.close()" class="btn btn-secondary">Cancel</button>
    </div>
</div>

<input type="file" id="upload-input" accept="image/*" style="display: none;">
```

**Step 2: Verify markup is correct**

Run: `go build -o /tmp/stinky ./cmd/stinky`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add internal/handlers/admin_media_picker.go
git commit -m "feat: add upload button to media picker modal"
```

---

## Task 2: Implement AJAX Upload Handler in Picker Modal

**Files:**
- Modify: `internal/handlers/admin_media_picker.go:120-132`

**Step 1: Add JavaScript upload handler**

In `admin_media_picker.go`, add to the `<script>` section (after the `selectImage` function):

```javascript
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
                formData.append('image', file);

                const response = await fetch('/admin/media/upload', {
                    method: 'POST',
                    headers: {
                        'X-CSRF-Token': getCsrfToken()
                    },
                    body: formData
                });

                const result = await response.json();

                if (result.success && result.url) {
                    // Auto-select the uploaded image
                    selectImage(result.url, file.name);
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
```

**Step 2: Verify JavaScript is valid**

Run: `go build -o /tmp/stinky ./cmd/stinky`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add internal/handlers/admin_media_picker.go
git commit -m "feat: add AJAX upload handler to media picker"
```

---

## Task 3: Update Block Editor to Open Picker Modal

**Files:**
- Modify: `internal/handlers/admin_blocks.go:875-881`

**Step 1: Replace prompt with modal opener**

In `admin_blocks.go`, replace the `insertImage` function (around line 875):

```javascript
function insertImage(colIndex) {
    // Open media picker in popup window
    const picker = window.open(
        '/admin/media/picker',
        'mediaPicker',
        'width=800,height=600,scrollbars=yes'
    );

    // Store which column we're inserting into
    if (picker) {
        window.currentColumnIndex = colIndex;
    }
}
```

**Step 2: Verify JavaScript is valid**

Run: `go build -o /tmp/stinky ./cmd/stinky`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add internal/handlers/admin_blocks.go
git commit -m "feat: replace image prompt with media picker modal"
```

---

## Task 4: Add postMessage Listener in Block Editor

**Files:**
- Modify: `internal/handlers/admin_blocks.go:864` (in `<script>` section)

**Step 1: Add message listener for image selection**

In `admin_blocks.go`, add to the `<script>` section (after the existing functions):

```javascript
// Listen for image selection from picker modal
window.addEventListener('message', function(event) {
    // Validate origin (same-origin only)
    if (event.origin !== window.location.origin) {
        return;
    }

    // Check message type
    if (event.data && event.data.type === 'image-selected') {
        const url = event.data.url;
        const colIndex = window.currentColumnIndex;

        if (url && colIndex !== undefined) {
            // Insert image tag
            const html = '<img src="' + url + '" style="width: 100%; height: auto;">\n';
            insertAtCursor(colIndex, html);
        }
    }
});
```

**Step 2: Verify JavaScript is valid**

Run: `go build -o /tmp/stinky ./cmd/stinky`
Expected: Build succeeds

**Step 3: Test manually**

1. Start server: `go run ./cmd/stinky server start`
2. Navigate to a page with column block editor
3. Click "Add Image" button
4. Verify: Picker modal opens
5. Verify: Clicking an image inserts it into the column
6. Verify: Upload button triggers file picker
7. Verify: Uploading an image auto-selects and inserts it

**Step 4: Commit**

```bash
git add internal/handlers/admin_blocks.go
git commit -m "feat: add postMessage listener for image selection"
```

---

## Task 5: Verify Upload Creates MediaItem Records

**Files:**
- Read: `internal/handlers/admin_media.go:267-330` (MediaUploadHandler)
- No changes needed - just verification

**Step 1: Review upload handler**

Verify `MediaUploadHandler` (line 267) already:
- Creates MediaItem record (line 309)
- Generates thumbnail (line 324)
- Returns JSON with url and filename (line 327)

**Step 2: Test upload creates MediaItem**

1. Start server: `go run ./cmd/stinky server start`
2. Open block editor
3. Click "Add Image" → "Upload New"
4. Select a test image
5. Navigate to `/admin/media`
6. Verify: Uploaded image appears in media library

**Step 3: Document verification**

No code changes needed - upload handler already correct.

---

## Task 6: Add Route for Media Picker (if not exists)

**Files:**
- Check: `cmd/stinky/server.go` (admin routes section)
- Modify: If route missing

**Step 1: Check if route exists**

Run: `grep -n "media/picker" cmd/stinky/server.go`
Expected: Should find route like `adminGroup.GET("/media/picker", ...)`

**Step 2: If route missing, add it**

In `cmd/stinky/server.go`, in the admin routes section (after authentication middleware):

```go
adminGroup.GET("/media/picker", handlers.MediaPickerHandler)
```

**Step 3: Verify route works**

Run: `go build -o /tmp/stinky ./cmd/stinky`
Expected: Build succeeds

**Step 4: Commit if changed**

```bash
git add cmd/stinky/server.go
git commit -m "feat: add media picker route to admin section"
```

---

## Task 7: Manual Integration Testing

**Files:**
- None (testing only)

**Step 1: Test complete flow**

1. Start fresh: `go run ./cmd/stinky server start`
2. Navigate to page editor
3. Add a column block
4. Click "Add Image" button in column editor
5. Verify: Modal opens at `/admin/media/picker`
6. Test Case A: Select existing image
   - Click an image in the grid
   - Verify: Image URL inserted into column textarea
   - Verify: Modal closes
7. Test Case B: Upload new image
   - Click "Upload New" button
   - Select a file
   - Verify: Upload completes
   - Verify: Image URL inserted into column textarea
   - Verify: Modal closes
8. Save the page
9. View the page on frontend
10. Verify: Images display correctly

**Step 2: Test error handling**

1. Click "Upload New" with no file selected
   - Verify: Nothing happens (graceful)
2. Try uploading a non-image file
   - Verify: Server returns error, alert shown
3. Try uploading a file > 5MB
   - Verify: Server returns error, alert shown

**Step 3: Check no orphaned uploads**

1. Upload an image via block editor
2. Navigate to media library page (`/admin/media`)
3. Verify: Uploaded image appears in library
4. Check database: `SELECT * FROM media_items WHERE filename = '<uploaded-file>'`
5. Verify: MediaItem record exists

**Step 4: Document results**

Create test results summary in commit message.

---

## Task 8: Run Full Test Suite

**Files:**
- All test files

**Step 1: Run all tests**

Run: `go test ./...`
Expected: Same baseline as before (1 pre-existing failure in blocks test)

**Step 2: If new failures, fix them**

If new test failures appear, investigate and fix before proceeding.

**Step 3: Commit test fixes if needed**

```bash
git add <modified-files>
git commit -m "test: fix test failures from picker integration"
```

---

## Task 9: Final Cleanup and Documentation

**Files:**
- Modify: `docs/FEATURES.md` (if exists)
- Create: None

**Step 1: Update features documentation**

If `docs/FEATURES.md` exists, add to media library section:

```markdown
### Media Picker Integration

- Click "Add Image" in block editor to open media library picker
- Browse existing images or upload new ones
- Uploaded images automatically added to media library
- No more orphaned images in uploads directory
```

**Step 2: Commit documentation**

```bash
git add docs/FEATURES.md
git commit -m "docs: document media picker integration"
```

---

## Verification Checklist

After implementation, verify:

- [ ] "Add Image" button opens media picker modal
- [ ] Clicking existing image inserts URL and closes modal
- [ ] "Upload New" button triggers file picker
- [ ] Uploading image creates MediaItem record
- [ ] Uploaded image thumbnail appears in media library
- [ ] Uploaded image auto-inserts into block editor
- [ ] Modal closes after selection or upload
- [ ] CSRF token properly included in upload request
- [ ] Error messages shown for failed uploads
- [ ] All tests pass (same baseline as before)
- [ ] No orphaned images created

## Success Criteria

1. Block editor no longer uses `prompt()` for image URLs
2. All image uploads create MediaItem records
3. Media library shows all uploaded images
4. User can browse and select existing images
5. User can upload new images without leaving editor
6. Integration is seamless and fast

## Notes for Implementation

- The upload handler (`MediaUploadHandler`) already exists and works correctly
- The picker modal (`MediaPickerHandler`) already exists and works correctly
- Main work is adding upload button to modal and connecting block editor
- Use CSRF token from cookie (URL-decoded) for AJAX requests
- Origin validation prevents XSS attacks via postMessage
- All JavaScript is vanilla - no frameworks needed
