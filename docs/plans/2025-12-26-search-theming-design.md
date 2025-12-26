# Search & Theming System Design

**Date:** 2025-12-26
**Status:** Design Approved, Ready for Implementation
**Scope:** Full-text search + customizable color palettes for site differentiation

## Overview

StinkyKitty will gain two major features enabling site discovery and visual customization:

1. **Full-text search** - Visitors can search page content, titles, descriptions, and menu items
2. **Color palettes** - Site admins choose from 12-16 curated palettes; each with light/dark variants

This enables camp sites to be visually distinct and easily discoverable while maintaining accessibility guarantees.

## Key Requirements

- **Search:** Index pages + metadata, deliver instant results, scope to current site
- **Theming:** 12-16 prebuilt palettes with light/dark modes, guaranteed accessibility
- **Admin multi-site:** Admin panel stays neutral by default for easy navigation between sites
- **Public differentiation:** Each site looks visually unique when choosing different palettes
- **No custom colors:** Palettes are curated; admins pick, not design

## Architecture

### Search System

```
User searches → Public search bar
               ↓
          FTS5 index query (site-scoped)
               ↓
          Return matching pages
               ↓
          Display results (styled with palette)
               ↓
          Click → Navigate to page
```

**Components:**
- **FTS Index:** SQLite FTS5 table (`page_search_index`)
  - Content: page title, description, body, menu labels
  - Updated on page create/update/delete via triggers
  - Site-scoped (queries filtered by site_id)

- **Search Handler:** HTTP endpoint `/search?q=...`
  - Accepts query string
  - Scopes to current site (via middleware)
  - Returns JSON: `[{title, snippet, url, relevance}, ...]`

- **Search UI:** Public search bar in site header
  - Real-time dropdown results (as user types)
  - Full results page for larger query sets
  - Result cards show title + content snippet + link

### Theming System

```
Site admin chooses palette + dark mode setting
               ↓
          Store in database (theme_palette, dark_mode)
               ↓
          On page load, generate CSS variables
               ↓
          Inject into HTML (in <style> tag or link)
               ↓
          Components use var(--color-primary), etc.
               ↓
          Site renders with chosen palette
```

**Color Generation:**
- Primary color → main brand, buttons, links, accents
- Secondary color → highlights, special actions
- Generated colors:
  - `--color-primary` - primary brand color
  - `--color-secondary` - accent/highlight color
  - `--color-bg` - page background
  - `--color-surface` - card/container background
  - `--color-text` - main text
  - `--color-text-muted` - secondary text
  - `--color-border` - borders, dividers
  - `--color-success`, `--color-error`, `--color-warning` - system colors

**Light Mode:** Light backgrounds, dark text, primary/secondary prominent
**Dark Mode:** Dark backgrounds, light text, primary/secondary muted appropriately

**Accessibility:** All color combinations tested for WCAG AA contrast ratios (4.5:1 text, 3:1 graphics)

### Predefined Palettes (12-16 options)

Example palettes:
1. Indigo + Orange
2. Rose + Slate
3. Emerald + Amber
4. Navy + Gold
5. Purple + Pink
6. Teal + Coral
7. Slate + Blue
8. Amber + Indigo
9. Rose + Rose (monochromatic)
10. Green + Green (monochromatic)
11. Blue + Blue (monochromatic)
12. Neutral + Neutral (grayscale)
13. [+ 3 more based on feedback]

Each tested in both light and dark modes.

## Components

### Database Changes
- **sites table:** Add `theme_palette` (varchar, default "slate"), `dark_mode` (bool, default false)

### Code Structure
- **internal/themes/** - New package
  - `palettes.go` - Predefined palette definitions
  - `colors.go` - Color generation logic
  - `css.go` - CSS variable generation

- **internal/handlers/**
  - `admin_settings.go` - Updated to include palette selector
  - `search.go` - Search handler
  - `public.go` - Inject theme CSS on page render

- **cmd/stinky/server.go** - Register search route, theme middleware

### Data Flow

**Theme Injection:**
1. Request arrives for public site
2. Middleware loads site + settings
3. Handler renders page template
4. Template includes CSS variables based on palette
5. Browser applies theme to all components

**Search Flow:**
1. User types in search bar (JavaScript event)
2. Fetch `/search?q=term&site=...`
3. Backend queries FTS index
4. Return results as JSON
5. JavaScript renders dropdown
6. User clicks result → navigate

## Error Handling

- **Search:** No results → show "No pages found" message
- **Theme:** Invalid palette → default to "slate"
- **Dark mode:** Respect system preference if not explicitly set
- **FTS index:** Auto-rebuild on startup if corrupted

## Testing Strategy

- **Search:**
  - Unit tests for FTS query logic
  - Integration test: index page → search → verify results
  - Cross-site isolation: search site A doesn't find site B content
  - Edge cases: special characters, empty queries, pagination

- **Theming:**
  - Unit tests for color generation
  - Visual regression: render each palette in light/dark
  - Contrast ratio validation (WCAG AA)
  - CSS variable injection test

## Success Criteria

- Search finds pages by content, title, description, menu items within <100ms
- 12-16 palettes ship with all light/dark variants tested
- All colors meet WCAG AA contrast requirements (4.5:1 text)
- Admin can toggle between site themes without seeing theme change in admin UI (default neutral)
- Public site visibly changes when palette is switched
- No performance regression from theme injection or search

## Future Enhancements

- Admin-created custom palettes (with validation)
- Per-page theme override
- Theme preview before committing
- Search analytics (what users search for)
- Scheduled theme changes (seasonal palettes)
