# Search & Theming System Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Implement full-text search + color palette theming system for site discovery and visual customization.

**Architecture:** Two-layer system - search uses SQLite FTS5 for indexing pages/metadata; theming uses CSS variables injected per-request from palette definitions.

**Tech Stack:** Go 1.24+, SQLite FTS5, CSS variables, existing GORM/Gin

---

## 8 Core Tasks

1. Create palette definitions and color generation
2. Implement CSS variable injection in templates
3. Add theme fields to site model (database migration)
4. Create theme settings UI (palette selector)
5. Implement FTS search index and handlers
6. Build search UI (bar + results page)
7. Add search result styling with theme
8. Run full test suite and verification

## Key Implementation Notes

- Palettes: 12-16 predefined colors (primary + secondary only)
- Color generation: Use color math to ensure accessibility
- CSS variables injected on every public page request
- Search index: FTS5 on page content, title, description, menu labels
- Search scoped: All queries filtered by site_id
- Admin theme preview: Toggle button (defaults to neutral/gray)
- All colors tested for WCAG AA contrast (4.5:1 text, 3:1 graphics)
- Test coverage: Search isolation, theme generation, contrast ratios

---

## Tasks

### Task 1: Create palette definitions and color generation

**Files:**
- Create: `internal/themes/palettes.go`
- Create: `internal/themes/colors.go`
- Create: `internal/themes/palettes_test.go`

**Step 1: Write failing test**

```go
// palettes_test.go
package themes

import (
	"testing"
)

func TestPaletteExists(t *testing.T) {
	palette := GetPalette("slate")
	if palette == nil {
		t.Fatal("slate palette not found")
	}
}

func TestGenerateLightModeColors(t *testing.T) {
	palette := GetPalette("slate")
	colors := GenerateColors(palette, false) // false = light mode

	if colors.Primary == "" {
		t.Fatal("Primary color not generated")
	}
	if colors.Background == "" {
		t.Fatal("Background color not generated")
	}
}

func TestGenerateDarkModeColors(t *testing.T) {
	palette := GetPalette("slate")
	colors := GenerateColors(palette, true) // true = dark mode

	if colors.Primary == "" {
		t.Fatal("Primary color not generated")
	}
	if colors.Background == "" {
		t.Fatal("Background color not generated")
	}
}

func TestContrastRatioValid(t *testing.T) {
	palette := GetPalette("slate")
	colors := GenerateColors(palette, false)

	// Text on background should have 4.5:1 contrast (WCAG AA)
	ratio := CalculateContrast(colors.Text, colors.Background)
	if ratio < 4.5 {
		t.Errorf("contrast ratio too low: %.2f (need 4.5+)", ratio)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/themes -v -run TestPalette`
Expected: FAIL (packages don't exist)

**Step 3: Create palette definitions**

Create `internal/themes/palettes.go`:

```go
package themes

// Palette defines the base colors for a theme
type Palette struct {
	Name      string // "slate", "indigo", etc.
	Primary   string // hex color
	Secondary string // hex color
}

// GetPalette returns a palette by name
func GetPalette(name string) *Palette {
	palettes := map[string]*Palette{
		"slate": {
			Name:      "slate",
			Primary:   "#64748b",
			Secondary: "#0f172a",
		},
		"indigo": {
			Name:      "indigo",
			Primary:   "#4f46e5",
			Secondary: "#f97316",
		},
		"rose": {
			Name:      "rose",
			Primary:   "#e11d48",
			Secondary: "#64748b",
		},
		"emerald": {
			Name:      "emerald",
			Primary:   "#059669",
			Secondary: "#f59e0b",
		},
		"navy": {
			Name:      "navy",
			Primary:   "#000080",
			Secondary: "#fbbf24",
		},
		"purple": {
			Name:      "purple",
			Primary:   "#a855f7",
			Secondary: "#ec4899",
		},
		"teal": {
			Name:      "teal",
			Primary:   "#14b8a6",
			Secondary: "#f87171",
		},
		"amber": {
			Name:      "amber",
			Primary:   "#f59e0b",
			Secondary: "#6366f1",
		},
		"rose-mono": {
			Name:      "rose-mono",
			Primary:   "#e11d48",
			Secondary: "#c41e3a",
		},
		"green-mono": {
			Name:      "green-mono",
			Primary:   "#22c55e",
			Secondary: "#16a34a",
		},
		"blue-mono": {
			Name:      "blue-mono",
			Primary:   "#3b82f6",
			Secondary: "#1e40af",
		},
		"neutral": {
			Name:      "neutral",
			Primary:   "#6b7280",
			Secondary: "#4b5563",
		},
	}

	return palettes[name]
}

// ListPalettes returns all available palettes
func ListPalettes() []*Palette {
	names := []string{"slate", "indigo", "rose", "emerald", "navy", "purple", "teal", "amber", "rose-mono", "green-mono", "blue-mono", "neutral"}
	var palettes []*Palette
	for _, name := range names {
		palettes = append(palettes, GetPalette(name))
	}
	return palettes
}
```

**Step 4: Create color generation**

Create `internal/themes/colors.go`:

```go
package themes

import (
	"fmt"
	"math"
)

// Colors represents all generated colors for a theme
type Colors struct {
	Primary      string
	Secondary    string
	Background   string
	Surface      string
	Text         string
	TextMuted    string
	Border       string
	Success      string
	Error        string
	Warning      string
}

// GenerateColors generates full color set from palette
func GenerateColors(palette *Palette, darkMode bool) *Colors {
	if darkMode {
		return generateDarkColors(palette)
	}
	return generateLightColors(palette)
}

func generateLightColors(palette *Palette) *Colors {
	return &Colors{
		Primary:    palette.Primary,
		Secondary:  palette.Secondary,
		Background: "#ffffff",
		Surface:    "#f9fafb",
		Text:       "#000000",
		TextMuted:  "#6b7280",
		Border:     lighten(palette.Primary, 0.7),
		Success:    "#22c55e",
		Error:      "#ef4444",
		Warning:    "#f59e0b",
	}
}

func generateDarkColors(palette *Palette) *Colors {
	return &Colors{
		Primary:    lighten(palette.Primary, 0.2),
		Secondary:  lighten(palette.Secondary, 0.2),
		Background: "#0f172a",
		Surface:    "#1e293b",
		Text:       "#f1f5f9",
		TextMuted:  "#94a3b8",
		Border:     darken(palette.Primary, 0.5),
		Success:    "#22c55e",
		Error:      "#ef4444",
		Warning:    "#f59e0b",
	}
}

// CalculateContrast calculates WCAG contrast ratio between two colors
func CalculateContrast(color1, color2 string) float64 {
	lum1 := getLuminance(color1)
	lum2 := getLuminance(color2)

	max := math.Max(lum1, lum2)
	min := math.Min(lum1, lum2)

	return (max + 0.05) / (min + 0.05)
}

// Helper functions for color manipulation

func lighten(hex string, amount float64) string {
	// Simplified: in production, use proper hex parsing
	return hex // TODO: implement proper color math
}

func darken(hex string, amount float64) string {
	// Simplified: in production, use proper hex parsing
	return hex // TODO: implement proper color math
}

func getLuminance(hex string) float64 {
	// Simplified WCAG luminance calculation
	// In production, parse hex properly
	return 0.5 // TODO: implement proper calculation
}
```

**Step 5: Run tests**

Run: `go test ./internal/themes -v`
Expected: PASS (basic tests pass, color math is stubbed)

**Step 6: Commit**

```bash
git add internal/themes/palettes.go internal/themes/colors.go internal/themes/palettes_test.go
git commit -m "feat: add palette definitions and color generation foundation"
```

---

### Task 2: Implement CSS variable injection in templates

**Files:**
- Create: `internal/themes/css.go`
- Modify: `cmd/stinky/server.go` (add theme middleware)
- Modify: templates to use CSS variables

**Step 1: Create CSS generation**

```go
// css.go
package themes

import "fmt"

// GenerateCSS generates CSS variables from colors
func GenerateCSS(colors *Colors) string {
	return fmt.Sprintf(`
:root {
  --color-primary: %s;
  --color-secondary: %s;
  --color-bg: %s;
  --color-surface: %s;
  --color-text: %s;
  --color-text-muted: %s;
  --color-border: %s;
  --color-success: %s;
  --color-error: %s;
  --color-warning: %s;
}

/* Utility classes */
body { background: var(--color-bg); color: var(--color-text); }
a { color: var(--color-primary); }
button { background: var(--color-primary); color: white; }
button:hover { opacity: 0.9; }
`, colors.Primary, colors.Secondary, colors.Background, colors.Surface,
		colors.Text, colors.TextMuted, colors.Border,
		colors.Success, colors.Error, colors.Warning)
}
```

**Step 2: Add theme middleware to inject CSS**

Modify `cmd/stinky/server.go`:

```go
// In server setup, after routes:
public.Use(themeMiddleware(db))

func themeMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get site from context (already set by site middleware)
		site := c.MustGet("site").(*models.Site)

		// Get palette and generate CSS
		palette := themes.GetPalette(site.ThemePalette)
		colors := themes.GenerateColors(palette, site.DarkMode)
		css := themes.GenerateCSS(colors)

		// Inject into context for template access
		c.Set("themeCSS", css)
	}
}
```

**Step 3: Update templates to use theme CSS**

In public page templates (add to `<head>`):

```html
<style>{{ .themeCSS }}</style>
```

**Step 4: Run and verify**

Build: `go build ./cmd/stinky`
Expected: No errors

**Step 5: Commit**

```bash
git add internal/themes/css.go cmd/stinky/server.go
git commit -m "feat: implement CSS variable injection and theme middleware"
```

---

### Task 3: Add theme fields to site model

**Files:**
- Modify: `internal/models/models.go`
- Create: migration for theme fields

**Step 1: Add fields to Site model**

```go
// In models.go, Site struct:
type Site struct {
	// ... existing fields ...
	ThemePalette string `gorm:"default:slate" json:"theme_palette"`
	DarkMode     bool   `gorm:"default:false" json:"dark_mode"`
}
```

**Step 2: Create migration**

```bash
# In cmd/stinky or db migration setup:
# Add SQL:
ALTER TABLE sites ADD COLUMN theme_palette VARCHAR(50) DEFAULT 'slate';
ALTER TABLE sites ADD COLUMN dark_mode BOOLEAN DEFAULT FALSE;
```

Or via GORM AutoMigrate in db initialization.

**Step 3: Commit**

```bash
git add internal/models/models.go
git commit -m "feat: add theme_palette and dark_mode fields to sites"
```

---

### Task 4: Create theme settings UI

**Files:**
- Modify: `internal/handlers/admin_settings.go` (create if needed)
- Modify: admin settings template

**Similar structure to other tasks - UI form to select palette + dark mode toggle**

---

### Task 5: Implement FTS search index

**Files:**
- Create: `internal/search/index.go`
- Create: `internal/search/index_test.go`
- Modify: `internal/models/models.go` (add search index table)

**Similar TDD approach - create failing tests, implement FTS indexing**

---

### Task 6: Build search handler and UI

**Files:**
- Create: `internal/handlers/search.go`
- Modify: public templates (add search bar)
- Create: search results template

---

### Task 7: Add search result styling

**Files:**
- Modify: search results template to use CSS variables
- Add search-specific styling

---

### Task 8: Run full test suite and verification

- Run all tests
- Verify contrast ratios
- Test search isolation
- Test theme switching
- Final build and verification

---

Ready to set up for implementation?
