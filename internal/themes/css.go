// SPDX-License-Identifier: MIT
package themes

import "fmt"

// GenerateCSS generates CSS with color variables from colors struct
func GenerateCSS(colors *Colors) string {
	return fmt.Sprintf(`:root {
    --color-primary: %s;
    --color-primary-contrast: %s;
    --color-accent: var(--color-primary);
    --color-accent-contrast: var(--color-primary-contrast);
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

/* Base element styles */
body {
  background-color: var(--color-bg);
  color: var(--color-text);
  transition: background-color 0.2s, color 0.2s;
}

a {
  color: var(--color-primary);
  text-decoration: none;
}

a:hover {
  text-decoration: underline;
}

/* Button styles */
button, .btn {
  background-color: var(--color-primary);
  color: var(--color-primary-contrast);
  border: none;
  padding: 8px 16px;
  border-radius: 4px;
  cursor: pointer;
  transition: opacity 0.2s;
}

button a, .btn a {
  color: inherit !important;
  text-decoration: none;
}

button:hover, .btn:hover {
  opacity: 0.9;
}

button:active, .btn:active {
  opacity: 0.8;
}

/* Card/surface styles */
.card, .surface {
  background-color: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  padding: 16px;
}

/* Border and divider styles */
hr, .divider {
  border: none;
  border-top: 1px solid var(--color-border);
}

/* Input styles */
input, textarea, select {
  border: 1px solid var(--color-border);
  background-color: var(--color-surface);
  color: var(--color-text);
  padding: 8px;
  border-radius: 4px;
}

input:focus, textarea:focus, select:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(var(--color-primary), 0.1);
}

/* Heading styles */
h1, h2, h3, h4, h5, h6 {
  color: var(--color-text);
}

/* Muted text */
.text-muted, .muted {
  color: var(--color-text-muted);
}

/* Status colors */
.success { color: var(--color-success); }
.error, .danger { color: var(--color-error); }
.warning { color: var(--color-warning); }
`, colors.Primary, colors.PrimaryContrast, colors.Secondary, colors.Background,
		colors.Surface, colors.Text, colors.TextMuted, colors.Border,
		colors.Success, colors.Error, colors.Warning)
}
