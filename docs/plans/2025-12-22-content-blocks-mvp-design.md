# Content Blocks MVP Design

**Date:** 2025-12-22
**Status:** Approved for Implementation

## Overview

Implement the core content management system for StinkyKitty - structured content blocks that allow camp organizers to build pages without the security nightmares of WordPress WYSIWYG editors.

This MVP focuses on getting the architecture working with one simple block type (Text), then we can iterate and add fancier blocks (Hero, Gallery, etc.) in future sessions.

## Design Decisions

### Page Model: Hybrid Approach
**Choice:** Homepage always exists at `/`, users can create additional pages
- Every site gets a homepage automatically
- Users can create additional pages with custom slugs (`/about`, `/events`, etc.)
- Keeps it simple for camps that just want "one page with info"
- Allows growth for camps that need multiple pages

**Rejected Alternatives:**
- Fixed page structure: Too rigid, camps vary widely
- Full WordPress-style freedom: Too complex for v1

### Admin UI: Server-Rendered HTML
**Choice:** Simple HTML forms with page refreshes
- Zero JavaScript frameworks needed
- Fast to build, works everywhere
- Server-side rendering means Google indexes content
- Can add htmx polish later

**Rejected Alternatives:**
- React/Vue SPA: SEO nightmare, camps won't be discoverable
- Progressive enhancement: Overkill for MVP

### Block Storage: JSON in Text Column
**Choice:** Store block content as JSON blob in `Block.Data` column
- Trivial to add new block types without migrations
- Just define new JSON schemas
- Simple to version and backup

**Rejected Alternatives:**
- Separate tables per block type: Migration hell
- NoSQL document store: Adds complexity, SQLite is fine

## Database Schema

### Page Model
```go
type Page struct {
    ID        uint
    SiteID    uint   // belongs to a site
    Slug      string // e.g., "/" for homepage, "/about", "/events"
    Title     string // e.g., "Camp Asaur - Dinosaur Disco"
    Published bool   // draft vs live
    CreatedAt time.Time
    UpdatedAt time.Time

    Blocks []Block `gorm:"foreignKey:PageID;constraint:OnDelete:CASCADE"`
}
```

**Indexes:**
- Unique index on `(site_id, slug)`
- Homepage always has slug "/"

### Block Model
```go
type Block struct {
    ID       uint
    PageID   uint
    Type     string // "text", "hero", "gallery" (for future)
    Order    int    // 0, 1, 2... for display order
    Data     string // JSON blob with block-specific content

    CreatedAt time.Time
    UpdatedAt time.Time
}
```

**Indexes:**
- Index on `page_id` for fast page loading
- Order determines display sequence (0 = first)

### Text Block JSON Structure
Stored in `Block.Data`:
```json
{
    "content": "Welcome to Camp Asaur! We're a theme camp..."
}
```

For MVP: plain text with line breaks preserved. Future: markdown support.

## Admin Interface Flow

### Dashboard (`/admin/dashboard`)
- Shows list of pages for this site
- "Edit Homepage" button (always available)
- "+ Create New Page" button
- List of additional pages with Edit/Delete actions
- Simple table/list layout

### Page Editor (`/admin/pages/{id}/edit`)
- Page title input field at top
- "Save Draft" and "Publish" buttons
- List of blocks in order, each showing:
  - Block type icon/label ("Text Block")
  - Preview snippet of content (first 100 chars)
  - ↑ Move Up / ↓ Move Down buttons
  - ✎ Edit / 🗑 Delete buttons
- "+ Add Text Block" button at bottom
- All actions use POST requests (no JavaScript required)

### Block Editor (`/admin/pages/{page_id}/blocks/{block_id}/edit`)
- Simple form for the block type
- For Text Block:
  - `<textarea>` with current content
  - Rows=10, full width
- "Save & Return" button goes back to page editor
- "Cancel" button (no save)
- Clean, minimal styling

## Public Rendering

### Homepage (`/`)
Server-rendered page that:
1. Loads page from database (slug="/")
2. Loads all blocks for page, ordered by `Order` ASC
3. Loops through blocks, rendering each via block renderer
4. Wraps in site layout template
5. Returns HTML to browser

### Custom Pages (`/:slug`)
Same as homepage but loads by slug instead.

### Block Renderer Pattern
```go
// internal/blocks/renderer.go
func RenderBlock(blockType string, dataJSON string) (string, error) {
    switch blockType {
    case "text":
        return renderTextBlock(dataJSON)
    default:
        return "", fmt.Errorf("unknown block type: %s", blockType)
    }
}

func renderTextBlock(dataJSON string) (string, error) {
    var data struct {
        Content string `json:"content"`
    }
    json.Unmarshal([]byte(dataJSON), &data)

    // Escape HTML, preserve line breaks
    safe := html.EscapeString(data.Content)
    formatted := strings.ReplaceAll(safe, "\n", "<br>")

    return fmt.Sprintf(`<div class="text-block">%s</div>`, formatted), nil
}
```

## Routes & Handlers

### New Admin Routes
```go
// Inside auth-protected admin routes:
adminGroup.GET("/dashboard", handlers.DashboardHandler)

// Page management
adminGroup.GET("/pages", handlers.ListPagesHandler)
adminGroup.POST("/pages", handlers.CreatePageHandler)
adminGroup.GET("/pages/:id/edit", handlers.EditPageHandler)
adminGroup.POST("/pages/:id", handlers.UpdatePageHandler)
adminGroup.POST("/pages/:id/publish", handlers.PublishPageHandler)
adminGroup.POST("/pages/:id/delete", handlers.DeletePageHandler)

// Block management
adminGroup.POST("/pages/:page_id/blocks", handlers.CreateBlockHandler)
adminGroup.GET("/pages/:page_id/blocks/:id/edit", handlers.EditBlockHandler)
adminGroup.POST("/pages/:page_id/blocks/:id", handlers.UpdateBlockHandler)
adminGroup.POST("/pages/:page_id/blocks/:id/delete", handlers.DeleteBlockHandler)
adminGroup.POST("/pages/:page_id/blocks/:id/move-up", handlers.MoveBlockUpHandler)
adminGroup.POST("/pages/:page_id/blocks/:id/move-down", handlers.MoveBlockDownHandler)
```

### Public Routes
```go
// Update existing homepage handler to render from database
siteGroup.GET("/", handlers.ServeHomepage)

// New route for custom pages
siteGroup.GET("/:slug", handlers.ServePage)
```

### Handler Organization
- `internal/handlers/admin_pages.go` - admin CRUD for pages
- `internal/handlers/admin_blocks.go` - admin CRUD for blocks
- `internal/handlers/public.go` - update ServeHomepage, add ServePage
- `internal/blocks/renderer.go` - block rendering logic

## Site Initialization

When a new site is created, automatically create homepage:
```go
homepage := &models.Page{
    SiteID:    site.ID,
    Slug:      "/",
    Title:     site.Subdomain,
    Published: false,
}
db.Create(homepage)
```

User can then add blocks and publish when ready.

## Error Handling

- **Missing page:** Return 404
- **Unpublished page:** Only show to authenticated site users, 404 for public
- **Invalid block type:** Log error, skip block in rendering (graceful degradation)
- **Malformed block JSON:** Log error, skip block
- **Permission checks:** Verify user has access to site before any mutations

## Security Considerations

- **HTML escaping:** All user content escaped before rendering
- **No inline JavaScript:** Users can't inject scripts
- **CSRF protection:** All POST requests require valid session
- **Authorization:** Check site membership on all admin operations
- **SQL injection:** Using GORM parameterized queries

## Testing Strategy

### Unit Tests
- Block renderer with various content
- Page model validations (unique slug per site)
- Block ordering logic

### Integration Tests
- Create page → add blocks → render public page
- Reorder blocks (move up/down)
- Publish/unpublish pages
- Permission checks (user from site A can't edit site B pages)

### Manual Testing
- Create homepage
- Add text blocks
- Reorder blocks
- Publish page
- View as public user
- Edit and update

## Future Enhancements (Not in MVP)

1. **Additional Block Types**
   - Hero block (title, subtitle, background image, CTA)
   - Image gallery (multiple images, lightbox)
   - Video embed (YouTube/Vimeo)
   - Button block (Google Forms links, etc.)

2. **Rich Text Editing**
   - Markdown support in text blocks
   - Simple formatting toolbar (bold, italic, links)

3. **Page Management**
   - Page templates (About, Events, Contact with pre-filled blocks)
   - Clone page functionality
   - Page reordering in navigation

4. **UX Improvements**
   - htmx for inline editing (no page refresh)
   - Live preview while editing
   - Drag-and-drop block reordering

## Success Criteria

- User can log into admin dashboard
- User can edit homepage title
- User can add text blocks to homepage
- User can reorder blocks (move up/down)
- User can edit block content
- User can delete blocks
- User can publish homepage
- Public can view published homepage with rendered blocks
- Unpublished changes don't show to public
- All content is HTML-escaped (no XSS)

## Implementation Notes

### MVP Scope
This MVP gets the core architecture working:
- Database models for pages and blocks
- Simple admin UI for editing
- Public rendering of pages
- One block type (Text)

Once this works, adding new block types is just:
1. Define JSON schema
2. Create renderer function
3. Build edit form
4. Add to block picker

Start simple, iterate fast.
