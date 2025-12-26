package themes

// Palette defines the base colors for a theme
type Palette struct {
	Name      string // "slate", "indigo", etc.
	Primary   string // hex color #RRGGBB
	Secondary string // hex color #RRGGBB
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

// ListPalettes returns all available palettes in order
func ListPalettes() []*Palette {
	names := []string{
		"slate", "indigo", "rose", "emerald", "navy", "purple",
		"teal", "amber", "rose-mono", "green-mono", "blue-mono", "neutral",
	}
	var palettes []*Palette
	for _, name := range names {
		if p := GetPalette(name); p != nil {
			palettes = append(palettes, p)
		}
	}
	return palettes
}
