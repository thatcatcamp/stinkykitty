// SPDX-License-Identifier: MIT
package themes

import (
	"strings"
	"testing"
)

func TestGenerateCSS(t *testing.T) {
	palette := GetPalette("slate")
	colors := GenerateColors(palette, false)
	css := GenerateCSS(colors)

	if css == "" {
		t.Fatal("GenerateCSS returned empty string")
	}
}

func TestGeneratedCSSContainsVariables(t *testing.T) {
	palette := GetPalette("indigo")
	colors := GenerateColors(palette, false)
	css := GenerateCSS(colors)

	expectedVars := []string{
		"--color-primary",
		"--color-primary-contrast",
		"--color-secondary",
		"--color-bg",
		"--color-surface",
		"--color-text",
		"--color-text-muted",
		"--color-border",
	}

	for _, variable := range expectedVars {
		if !strings.Contains(css, variable) {
			t.Errorf("CSS missing variable: %s", variable)
		}
	}
}

func TestGeneratedCSSContainsHexValues(t *testing.T) {
	palette := GetPalette("rose")
	colors := GenerateColors(palette, false)
	css := GenerateCSS(colors)

	if !strings.Contains(css, colors.Primary) {
		t.Errorf("CSS does not contain primary color: %s", colors.Primary)
	}
	if !strings.Contains(css, colors.Background) {
		t.Errorf("CSS does not contain background color: %s", colors.Background)
	}
}

func TestGeneratedCSSIsValid(t *testing.T) {
	palette := GetPalette("emerald")
	colors := GenerateColors(palette, true) // dark mode
	css := GenerateCSS(colors)

	if !strings.Contains(css, ":root {") {
		t.Fatal("CSS missing :root selector")
	}
	if !strings.Contains(css, "}") {
		t.Fatal("CSS missing closing brace")
	}
}

func TestCSSGenerationLightVsDark(t *testing.T) {
	palette := GetPalette("navy")
	colorsLight := GenerateColors(palette, false)
	colorsDark := GenerateColors(palette, true)

	cssLight := GenerateCSS(colorsLight)
	cssDark := GenerateCSS(colorsDark)

	if cssLight == cssDark {
		t.Fatal("Light and dark CSS should be different")
	}
}
