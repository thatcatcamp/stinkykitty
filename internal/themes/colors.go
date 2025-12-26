package themes

// Colors represents all generated colors for a theme
type Colors struct {
	Primary      string // Main brand color
	Secondary    string // Accent/highlight color
	Background   string // Page background
	Surface      string // Card/container background
	Text         string // Main text color
	TextMuted    string // Secondary/muted text
	Border       string // Border/divider color
	Success      string // Success state color
	Error        string // Error state color
	Warning      string // Warning state color
}

// GenerateColors generates full color set from palette for light or dark mode
func GenerateColors(palette *Palette, darkMode bool) *Colors {
	if darkMode {
		return generateDarkColors(palette)
	}
	return generateLightColors(palette)
}

// generateLightColors creates colors for light mode
func generateLightColors(palette *Palette) *Colors {
	return &Colors{
		Primary:    palette.Primary,
		Secondary:  palette.Secondary,
		Background: "#ffffff",
		Surface:    "#f9fafb",
		Text:       "#000000",
		TextMuted:  "#6b7280",
		Border:     "#e5e7eb",
		Success:    "#22c55e",
		Error:      "#ef4444",
		Warning:    "#f59e0b",
	}
}

// generateDarkColors creates colors for dark mode
func generateDarkColors(palette *Palette) *Colors {
	return &Colors{
		Primary:    "#f1f5f9", // Light version of primary
		Secondary:  "#e2e8f0", // Light version of secondary
		Background: "#0f172a",
		Surface:    "#1e293b",
		Text:       "#f1f5f9",
		TextMuted:  "#94a3b8",
		Border:     "#334155",
		Success:    "#22c55e",
		Error:      "#ef4444",
		Warning:    "#f59e0b",
	}
}
