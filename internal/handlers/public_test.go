// SPDX-License-Identifier: MIT
package handlers

import (
	"strings"
	"testing"

	"github.com/thatcatcamp/stinkykitty/internal/models"
)

func TestGetGoogleAnalyticsScript(t *testing.T) {
	tests := []struct {
		name     string
		gaID     string
		wantHTML bool
		wantID   string
	}{
		{
			name:     "Valid GA4 ID",
			gaID:     "G-ABC123XYZ",
			wantHTML: true,
			wantID:   "G-ABC123XYZ",
		},
		{
			name:     "Valid UA ID",
			gaID:     "UA-123456-1",
			wantHTML: true,
			wantID:   "UA-123456-1",
		},
		{
			name:     "Empty GA ID",
			gaID:     "",
			wantHTML: false,
		},
		{
			name:     "Whitespace only",
			gaID:     "   ",
			wantHTML: false,
		},
		{
			name:     "XSS attempt with quotes",
			gaID:     "'; alert('XSS'); //",
			wantHTML: false,
		},
		{
			name:     "XSS attempt with script tags",
			gaID:     "<script>alert('XSS')</script>",
			wantHTML: false,
		},
		{
			name:     "Invalid format",
			gaID:     "INVALID-123",
			wantHTML: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			site := &models.Site{
				ID:                1,
				GoogleAnalyticsID: tt.gaID,
			}

			result := getGoogleAnalyticsScript(site)

			if tt.wantHTML {
				if result == "" {
					t.Error("Expected GA script, got empty string")
				}
				if !strings.Contains(result, "gtag") {
					t.Error("Result should contain gtag function")
				}
				if !strings.Contains(result, tt.wantID) {
					t.Errorf("Result should contain GA ID %s", tt.wantID)
				}
				// Verify no obvious XSS vulnerabilities
				if strings.Contains(result, "alert(") {
					t.Error("Result contains potential XSS payload")
				}
			} else {
				if result != "" {
					t.Errorf("Expected empty string for invalid input, got: %s", result)
				}
			}
		})
	}
}

func TestGetGoogleAnalyticsScriptNoID(t *testing.T) {
	site := &models.Site{
		ID:                1,
		GoogleAnalyticsID: "",
	}

	result := getGoogleAnalyticsScript(site)
	if result != "" {
		t.Errorf("Expected empty string when no GA ID configured, got: %s", result)
	}
}
