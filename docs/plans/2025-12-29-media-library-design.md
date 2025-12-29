# Media Library Design

**Date:** 2025-12-29
**Status:** Approved for Implementation
**Complexity:** 2-3 days focused work

## Overview

A full-featured media library for managing uploaded images across camp sites. Solves the problem of orphaned images and provides organized storage with tagging, search, and easy insertion into content blocks.

## Goals

1. **Organization**: Tag-based system for categorizing camp photos
2. **Safety**: Prevent accidental deletion of in-use images
3. **Discoverability**: Search and filter to find images quickly
4. **Integration**: Easy insertion into image blocks via modal picker
5. **Cleanup**: Identify and remove orphaned/unused images

## User Workflow

### Uploading Images

1. Navigate to `/admin/media`
2. Drag-and-drop images or click to browse (multiple files supported)
3. Images upload with progress indicators
4. After upload, add tags (optional but recommended)
5. Images appear in grid immediately

### Finding & Using Images

1. From image block editor, click "Browse Library" button
2. Modal opens showing all site images
3. Use search box or tag filters to narrow selection
4. Click desired image to select
5. Modal closes, block editor auto-fills with image URL
6. Continue editing alt text and caption

### Managing Images

1. View all images in paginated grid (50 per page)
2. Search by filename or tags
3. Filter by tags (multiple tags = AND logic)
4. Toggle "Show orphaned" to find unused images
5. Click "Edit tags" to add/remove tags inline
6. Click "Delete" to remove (with usage warnings)

### Cleanup Process

1. Toggle "Show orphaned" filter
2. Review images not used in any blocks
3. Delete unwanted images safely
4. Keep images you might reuse later (just tag them)

## Architecture

### Database Schema

**New Table: `media_items`**
```go
type MediaItem struct {
    ID           uint      `gorm:"primaryKey"`
    SiteID       uint      `gorm:"not null;index"`
    Filename     string    `gorm:"not null"` // Random hex filename
    OriginalName string    `gorm:"not null"` // User's original filename
    FileSize     int64     `gorm:"not null"` // Bytes
    MimeType     string    `gorm:"not null"` // image/jpeg, etc.
    UploadedBy   uint      `gorm:"not null"` // User ID
    CreatedAt    time.Time
    UpdatedAt    time.Time
    DeletedAt    gorm.DeletedAt `gorm:"index"`

    // Relationships
    Site Site          `gorm:"foreignKey:SiteID"`
    User User          `gorm:"foreignKey:UploadedBy"`
    Tags []MediaTag    `gorm:"foreignKey:MediaItemID"`
}
```

**New Table: `media_tags`**
```go
type MediaTag struct {
    ID          uint   `gorm:"primaryKey"`
    MediaItemID uint   `gorm:"not null;index:idx_media_tag"`
    TagName     string `gorm:"not null;index:idx_media_tag"`
    CreatedAt   time.Time

    // Relationships
    MediaItem MediaItem `gorm:"foreignKey:MediaItemID"`
}
```

**Indexes:**
- `media_items.site_id` - Fast site-specific queries
- `media_tags(media_item_id, tag_name)` - Fast tag filtering
- `media_tags.tag_name` - Tag autocomplete

### File Storage

**Directory Structure:**
```
{SiteDir}/
  uploads/
    abc123def456.jpg          # Original files (existing)
    xyz789uvw012.png
  uploads/thumbs/              # New: Thumbnails
    abc123def456.jpg           # 200x200 thumbnails
    xyz789uvw012.png
```

**Thumbnail Generation:**
- Create on upload using Go image library
- 200x200px, maintain aspect ratio, center crop
- Save as JPEG quality 85% for all formats
- Fallback: Show original if thumbnail fails

### Components

**Backend (`internal/handlers/admin_media.go`):**
- `MediaLibraryHandler` - Main library page
- `MediaUploadHandler` - Handle file uploads
- `MediaDeleteHandler` - Delete with usage checking
- `MediaTagHandler` - Add/remove tags (AJAX)
- `MediaPickerHandler` - Modal picker for block editors
- `MediaUsageHandler` - Find where image is used (AJAX)

**Models (`internal/models/models.go`):**
- Add MediaItem and MediaTag structs
- Migration functions

**Utilities (`internal/media/`):**
- `FindImageUsage(siteID, imageURL) []UsageLocation` - Scan blocks
- `GenerateThumbnail(srcPath, dstPath)` - Create thumbnails
- `ImportExistingUploads(siteID)` - Migration helper

## User Interface

### Main Library Page (`/admin/media`)

**Layout:**
```
┌─────────────────────────────────────────────┐
│ 📤 Upload Zone (Drag & Drop)                │
│ Click to browse • Multiple files OK         │
└─────────────────────────────────────────────┘

┌─────────────────────────────────────────────┐
│ 🔍 Search: [_________]  Tags: [web] [cats]  │
│ [Show All] [Show Orphaned]  Sort: Newest ▾  │
└─────────────────────────────────────────────┘

┌─────┬─────┬─────┬─────┐
│ img │ img │ img │ img │  ← Responsive grid
│ 📝  │ 📝  │ 📝  │ 📝  │    3-4 cols desktop
├─────┼─────┼─────┼─────┤    2 cols mobile
│ img │ img │ img │ img │
│ 📝  │ 📝  │ 📝  │ 📝  │
└─────┴─────┴─────┴─────┘

        ← 1 2 3 ... 10 →
```

**Image Card:**
```
┌──────────────────┐
│                  │
│   [thumbnail]    │  ← 200x200
│                  │
├──────────────────┤
│ cat-photo.jpg    │  ← Truncated filename
│ [web] [summer]   │  ← Tag badges
│ Dec 29, 2025     │  ← Upload date
│ [Edit] [Delete]  │  ← Actions
└──────────────────┘
```

### Modal Picker (`/admin/media/picker`)

Simplified version for block editors:
- Same grid layout in modal dialog
- Search and tag filtering
- No upload zone (upload from main library)
- No delete buttons (selection only)
- Click image to select and close

### Block Editor Integration

**Current Image Block Edit Form:**
```html
<form>
  <label>Image</label>
  <input type="file" name="image">
  <!-- ADD THIS: -->
  <button type="button" onclick="openMediaPicker()">
    📚 Browse Library
  </button>

  <label>Alt Text</label>
  <input type="text" name="alt">

  <label>Caption</label>
  <input type="text" name="caption">
</form>
```

**JavaScript:**
```javascript
function openMediaPicker() {
  // Open modal with /admin/media/picker
  // On selection: fill image URL into form
  // Close modal
}
```

## Deletion Safety

### Usage Detection

Scan all blocks in current site to find image references:

```go
type UsageLocation struct {
    PageID    uint
    PageTitle string
    BlockID   uint
    BlockType string
}

func FindImageUsage(siteID uint, imageURL string) []UsageLocation {
    // 1. Get all pages for this site
    var pages []Page
    db.Where("site_id = ?", siteID).Find(&pages)

    // 2. For each page, get blocks
    var usages []UsageLocation
    for _, page := range pages {
        var blocks []Block
        db.Where("page_id = ?", page.ID).Find(&blocks)

        // 3. Parse block.Data JSON
        for _, block := range blocks {
            if containsImageURL(block, imageURL) {
                usages = append(usages, UsageLocation{
                    PageID: page.ID,
                    PageTitle: page.Title,
                    BlockID: block.ID,
                    BlockType: block.Type,
                })
            }
        }
    }

    return usages
}

func containsImageURL(block Block, imageURL string) bool {
    switch block.Type {
    case "image":
        var data ImageBlockData
        json.Unmarshal([]byte(block.Data), &data)
        return data.URL == imageURL
    case "button":
        // Check button.image if exists
    case "columns":
        // Recursively check nested content
    }
    return false
}
```

### Deletion Flow

**No Usage (Orphaned):**
```
┌─────────────────────────────────────┐
│ Delete this image?                  │
│                                     │
│ cat-photo.jpg                       │
│ This cannot be undone.              │
│                                     │
│ [Cancel]  [Delete]                  │
└─────────────────────────────────────┘
```

**In Use:**
```
┌─────────────────────────────────────┐
│ ⚠️ This image is used in 3 places:  │
│                                     │
│ • Homepage → Image Block            │
│ • About Page → Button               │
│ • Contact → Image Block             │
│                                     │
│ Delete anyway? Blocks will show     │
│ broken links.                       │
│                                     │
│ [Cancel]  [Delete Anyway]           │
└─────────────────────────────────────┘
```

## Migration & Existing Images

### Auto-Import on First Use

When media library is first accessed:

1. Check if `media_items` table is empty for this site
2. If empty, run import:
   ```go
   func ImportExistingUploads(site Site) {
       uploadsDir := filepath.Join(site.SiteDir, "uploads")
       files := listImageFiles(uploadsDir)

       for _, file := range files {
           // Create media_items record
           mediaItem := MediaItem{
               SiteID: site.ID,
               Filename: file.Name,
               OriginalName: file.Name, // Best guess
               FileSize: file.Size,
               MimeType: detectMimeType(file),
               UploadedBy: site.OwnerID, // Best guess
           }
           db.Create(&mediaItem)
       }
   }
   ```
3. Show notification: "Imported 47 existing images"
4. Admin can now tag and organize

### Backwards Compatibility

- Keep existing upload handler working
- Direct uploads from block editor auto-create `media_items` records
- No tags assigned (user can add later)
- Works seamlessly with old and new workflows

## Performance

### Optimizations

**Thumbnail Generation:**
- Generate on upload (blocking, ~100ms per image)
- Alternative: Background job if too slow

**Usage Scanning:**
- Cache results for 5 minutes
- Invalidate cache on block save
- Only scan current site's blocks

**Pagination:**
- 50 images per page
- Prevents slowdown with 1000+ images
- Database query: `LIMIT 50 OFFSET 100`

**Tag Autocomplete:**
- Query: `SELECT DISTINCT tag_name FROM media_tags WHERE site_id = ?`
- Index makes this fast even with many tags

### Expected Performance

- Upload 10 images: ~5-10 seconds
- Load library page: <500ms (with thumbnails)
- Search/filter: <100ms
- Delete with usage check: <200ms

## Security

### File Upload Validation

Reuse existing security from `internal/uploads/uploader.go`:
- Magic Bytes validation (not just extension)
- 5MB file size limit
- Only allow: JPEG, PNG, GIF, WebP
- Random hex filenames prevent exploits

### Access Control

- Only authenticated site admins can access
- URL: `/admin/media` (protected by auth middleware)
- Users can only see/manage their own site's images
- Site ID checked on every operation

### CSRF Protection

All POST/DELETE operations include CSRF tokens:
- Upload: Form with CSRF token
- Delete: AJAX with `X-CSRF-Token` header
- Tag operations: AJAX with token

## Dashboard Integration

Add to admin dashboard navigation:

```html
<!-- After Settings link -->
<a href="/admin/media" class="nav-link">
  <svg>...</svg> Media Library
</a>
```

Icon: 🖼️ or image/photo icon from existing design system

## Implementation Phases

### Phase 1: Core Infrastructure (Day 1)
- Database models and migrations
- File upload handler with thumbnail generation
- Main library page with upload zone
- Basic grid display (no tags yet)
- Delete with simple confirm

### Phase 2: Tagging & Search (Day 2)
- Add/remove tags (AJAX)
- Tag filtering
- Search by filename
- Show orphaned filter
- Usage detection and deletion warnings

### Phase 3: Block Integration (Day 2-3)
- Modal picker interface
- "Browse Library" button in image block editor
- JavaScript for image selection
- Backwards compatibility testing

### Phase 4: Polish & Testing (Day 3)
- Import existing uploads
- Thumbnail generation optimization
- Error handling and edge cases
- Mobile responsive tweaks
- Documentation

## Testing Checklist

- [ ] Upload single image
- [ ] Upload multiple images (5+ at once)
- [ ] Upload fails for non-images
- [ ] Upload fails for files >5MB
- [ ] Thumbnails generate correctly
- [ ] Tags can be added/removed
- [ ] Search finds images by filename
- [ ] Search finds images by tags
- [ ] Filter by multiple tags (AND logic)
- [ ] "Show orphaned" filter works
- [ ] Delete orphaned image (simple confirm)
- [ ] Delete in-use image (shows warning with locations)
- [ ] "Delete anyway" removes file and database record
- [ ] Modal picker opens from block editor
- [ ] Selecting image fills block editor form
- [ ] Direct upload from block editor creates media_items record
- [ ] Pagination works (test with 100+ images)
- [ ] Mobile responsive layout works
- [ ] Existing images imported on first use
- [ ] Usage detection finds all block types
- [ ] CSRF tokens on all operations

## Future Enhancements (Not in V1)

- **Bulk operations**: Select multiple images, add tags, delete
- **Image editing**: Crop, resize, filters
- **Alt text editor**: Edit alt text without opening block
- **Storage stats**: Show total storage used per site
- **Image replace**: Replace image in all usages at once
- **Folders**: Optional folder organization (in addition to tags)
- **External storage**: S3/CDN support (mentioned in Site model)
- **Smart tags**: AI auto-tagging based on image content

## Success Metrics

After deployment:
- Orphaned images identified and cleaned up on campasaur.us
- Users can find images faster (measure by support requests)
- Fewer broken image links (track 404s for /uploads/*)
- Image reuse increases (same photo used in multiple blocks)
- Storage usage visible and manageable

## Open Questions

None - design approved for implementation.
