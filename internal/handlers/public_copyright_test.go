package handlers

import (
	"strings"
	"testing"
	"time"

	"github.com/thatcatcamp/stinkykitty/internal/models"
)

func TestGetCopyrightText(t *testing.T) {
	currentYear := time.Now().Format("2006")

	tests := []struct {
		name               string
		copyrightText      string
		siteTitle          string
		expectedContains   []string
		expectedNotContain []string
	}{
		{
			name:          "Default copyright (empty string)",
			copyrightText: "",
			siteTitle:     "My Awesome Site",
			expectedContains: []string{
				"©",
				currentYear,
				"My Awesome Site",
				"All rights reserved",
			},
			expectedNotContain: []string{
				"{year}",
				"{site}",
			},
		},
		{
			name:          "Custom copyright with placeholders",
			copyrightText: "© {year} {site} - Custom Text",
			siteTitle:     "Test Site",
			expectedContains: []string{
				"©",
				currentYear,
				"Test Site",
				"Custom Text",
			},
			expectedNotContain: []string{
				"{year}",
				"{site}",
			},
		},
		{
			name:          "Copyright with only year placeholder",
			copyrightText: "Copyright {year}",
			siteTitle:     "Example",
			expectedContains: []string{
				"Copyright",
				currentYear,
			},
			expectedNotContain: []string{
				"{year}",
				"{site}",
			},
		},
		{
			name:          "Copyright with only site placeholder",
			copyrightText: "{site} Website",
			siteTitle:     "Cool Camp",
			expectedContains: []string{
				"Cool Camp",
				"Website",
			},
			expectedNotContain: []string{
				"{site}",
			},
		},
		{
			name:          "Copyright with no placeholders",
			copyrightText: "All Rights Reserved 2025",
			siteTitle:     "Some Site",
			expectedContains: []string{
				"All Rights Reserved 2025",
			},
			expectedNotContain: []string{},
		},
		{
			name:          "XSS attempt in copyright text should be escaped",
			copyrightText: "© {year} <script>alert('xss')</script>",
			siteTitle:     "Normal Site",
			expectedContains: []string{
				"&lt;script&gt;",
				"&lt;/script&gt;",
			},
			expectedNotContain: []string{
				"<script>",
				"</script>",
			},
		},
		{
			name:          "XSS attempt in site title should be escaped",
			copyrightText: "© {year} {site}",
			siteTitle:     "<b>Bold Site</b>",
			expectedContains: []string{
				"&lt;b&gt;",
				"&lt;/b&gt;",
				"Bold Site",
			},
			expectedNotContain: []string{
				"<b>",
				"</b>",
			},
		},
		{
			name:          "Copyright symbol renders correctly",
			copyrightText: "© {year} {site}",
			siteTitle:     "Test Site",
			expectedContains: []string{
				"©", // Verify symbol is not escaped to &copy;
				currentYear,
				"Test Site",
			},
			expectedNotContain: []string{
				"&copy;", // Should NOT be HTML entity encoded
			},
		},
		{
			name:          "Multiple placeholder instances",
			copyrightText: "© {year}-{year} {site} by {site}",
			siteTitle:     "Camp Site",
			expectedContains: []string{
				"©",
				currentYear + "-" + currentYear,
				"Camp Site by Camp Site",
			},
			expectedNotContain: []string{
				"{year}",
				"{site}",
				"&copy;",
			},
		},
		{
			name:          "Unicode characters",
			copyrightText: "🏕️ {year} Camp {site} ™",
			siteTitle:     "Adventure",
			expectedContains: []string{
				"🏕️",
				currentYear,
				"Camp Adventure",
				"™",
			},
			expectedNotContain: []string{
				"{year}",
				"{site}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			site := &models.Site{
				CopyrightText: tt.copyrightText,
				SiteTitle:     tt.siteTitle,
			}

			result := getCopyrightText(site)

			// Check expected strings are present
			for _, expected := range tt.expectedContains {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected result to contain %q, got: %s", expected, result)
				}
			}

			// Check strings that should not be present
			for _, notExpected := range tt.expectedNotContain {
				if strings.Contains(result, notExpected) {
					t.Errorf("Expected result NOT to contain %q, got: %s", notExpected, result)
				}
			}
		})
	}
}
