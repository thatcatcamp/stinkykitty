# Media Library Picker Integration Design

**Goal:** Replace prompt-based image insertion with integrated media library picker that supports both selecting existing images and uploading new ones.

**Architecture:** Enhanced modal picker with upload capability, using postMessage for parent-child communication.

**Tech Stack:** Existing media library system, vanilla JavaScript, AJAX file upload

---

## Problem Statement

Currently, the block editor uses `prompt()` dialogs for image URLs, which:
- Requires manual URL typing
- Doesn't show available images
- Doesn't encourage image reuse
- Creates friction in the editing workflow

Users want a WordPress-like media picker, but with better UX:
- One modal that handles both browse and upload
- No need to navigate back to media library after upload
- Auto-selection after upload (fewer clicks)

## Solution Architecture

### Components

1. **Enhanced Media Picker Modal** (`admin_media_picker.go`)
   - Current: Grid of existing images with click-to-select
   - New: Add "Upload New" button in header
   - New: AJAX upload handler with auto-select

2. **Block Editor Integration** (`admin_blocks.go`)
   - Current: `insertImage()` uses `prompt()` for URL
   - New: Opens picker modal via `window.open()`
   - New: `postMessage` listener receives selected image URL
   - New: Auto-inserts `<img>` tag with received URL

3. **Upload Endpoint** (already exists)
   - `/admin/media/upload` already creates MediaItem records
   - Already generates thumbnails
   - Already prevents orphaned uploads
   - No changes needed

### Communication Flow

```
User clicks "Add Image" in block editor
  ↓
JavaScript: window.open('/admin/media/picker')
  ↓
Picker modal displays existing images + "Upload New" button
  ↓
User action branches:

  Option A: Click existing image
    → selectImage() sends postMessage
    → Modal closes
    → Parent receives URL and inserts <img>

  Option B: Click "Upload New"
    → File picker opens
    → File selected
    → AJAX POST to /admin/media/upload
    → Server creates MediaItem + thumbnail
    → Returns {url, filename}
    → Auto-call selectImage() with new URL
    → Modal closes
    → Parent receives URL and inserts <img>
```

### Key Design Decisions

**Decision 1: Upload in Modal vs Redirect**
- **Chosen:** Upload within modal via AJAX
- **Alternative:** Redirect to media library page
- **Rationale:** Fewer clicks, better UX than WordPress

**Decision 2: Auto-select After Upload**
- **Chosen:** Automatically select and close after upload
- **Alternative:** Upload, add to grid, user clicks to select
- **Rationale:** If user is uploading, they clearly want to use that image

**Decision 3: Upload Button vs Drag-Drop Zone**
- **Chosen:** Simple "Upload New" button
- **Alternative:** Prominent drag-and-drop zone
- **Rationale:** Simpler UI, drag-drop rarely used, keeps focus on existing images

## Implementation Tasks

### Task 1: Enhance Media Picker Modal
**File:** `internal/handlers/admin_media_picker.go`

Add to header:
- "Upload New" button next to Cancel
- Hidden file input (`<input type="file" accept="image/*">`)

Add JavaScript:
- Click handler for Upload button → trigger file input
- File input change handler → upload via AJAX
- AJAX upload to `/admin/media/upload` with CSRF token
- On success: call `selectImage(uploadedURL, filename)`

### Task 2: Update Block Editor
**File:** `internal/handlers/admin_blocks.go`

Replace `insertImage()` function:
- Remove `prompt()` call
- Open picker modal: `window.open('/admin/media/picker', '_blank', 'width=800,height=600')`
- Add `window.addEventListener('message', ...)` listener
- On message receive: validate origin, extract URL, insert `<img>` tag

Add cleanup:
- Remove message listener when block editor unloads

### Task 3: Verify Upload Endpoint
**File:** `internal/handlers/admin_media.go`

Verify `/admin/media/upload` returns proper JSON:
```json
{
  "success": true,
  "url": "/uploads/filename.jpg",
  "filename": "original-name.jpg"
}
```

Already creates MediaItem, already generates thumbnails - no changes needed.

## User Experience Flow

### Before (Current):
1. Click "Add Image" button
2. See JavaScript prompt: "Enter image URL"
3. Type URL manually (error-prone)
4. Click OK
5. Image inserted

### After (New):
1. Click "Add Image" button
2. Picker modal opens showing all images
3. Either:
   - Click existing image → Done!
   - Click "Upload New" → Select file → Done!
4. Image inserted automatically

**Improvement:** Reduces steps, shows available images, prevents orphaned uploads.

## Security Considerations

- CSRF token already enforced on upload endpoint
- File type validation already in place (images only)
- Size limit already enforced (5MB)
- postMessage origin validation in receiver
- Modal opened to same origin (no CORS issues)

## Success Metrics

- All images uploaded through block editor create MediaItem records
- No more orphaned images in `/uploads/` directory
- Users can easily reuse existing images
- Fewer support requests about "where are my images"

## Future Enhancements (Not in This Phase)

- Search/filter in picker modal
- Pagination for large media libraries
- Image editing (crop, resize) before insertion
- Multi-select for galleries
- Drag-and-drop upload zone (if users request it)
