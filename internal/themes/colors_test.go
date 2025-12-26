package themes

import (
	"testing"
	"strings"
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
	if colors.Text == "" {
		t.Fatal("Text color not generated")
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

func TestListPalettes(t *testing.T) {
	palettes := ListPalettes()
	if len(palettes) < 12 {
		t.Errorf("expected at least 12 palettes, got %d", len(palettes))
	}
}

func TestPaletteNamesUnique(t *testing.T) {
	palettes := ListPalettes()
	names := make(map[string]bool)
	for _, p := range palettes {
		if names[p.Name] {
			t.Errorf("duplicate palette name: %s", p.Name)
		}
		names[p.Name] = true
	}
}

func TestGeneratedColorsAreHex(t *testing.T) {
	palette := GetPalette("indigo")
	colors := GenerateColors(palette, false)

	colorMap := map[string]string{
		"Primary": colors.Primary,
		"Secondary": colors.Secondary,
		"Background": colors.Background,
		"Surface": colors.Surface,
		"Text": colors.Text,
	}

	for name, color := range colorMap {
		if !strings.HasPrefix(color, "#") {
			t.Errorf("%s should be hex format, got: %s", name, color)
		}
		if len(color) != 7 && len(color) != 4 { // #RRGGBB or #RGB
			t.Errorf("%s invalid hex length: %s", name, color)
		}
	}
}
