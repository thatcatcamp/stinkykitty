// SPDX-License-Identifier: MIT
package handlers

const (
	// Color Palette
	ColorBgPrimary   = "#FAFAF8" // Cream background
	ColorBgSecondary = "#F3F4F6" // Light gray background
	ColorBgCard      = "#FFFFFF" // White card
	ColorTextPrimary = "#2D2D2D" // Dark charcoal
	ColorTextSecond  = "#6B7280" // Light gray
	ColorAccent      = "#2E8B9E" // Teal accent
	ColorAccentHover = "#1E6F7F" // Darker teal
	ColorSuccess     = "#10B981" // Green (published)
	ColorWarning     = "#F59E0B" // Amber (draft)
	ColorDanger      = "#EF4444" // Red (delete)
	ColorBorder      = "#E5E5E3" // Subtle border
)

// Returns full stylesheet with CSS variables and base styles
func GetDesignSystemCSS() string {
	return `
:root {
	--color-bg-primary: ` + ColorBgPrimary + `;
	--color-bg-secondary: ` + ColorBgSecondary + `;
	--color-bg-card: ` + ColorBgCard + `;
	--color-text-primary: ` + ColorTextPrimary + `;
	--color-text-secondary: ` + ColorTextSecond + `;
	--color-accent: ` + ColorAccent + `;
	--color-accent-hover: ` + ColorAccentHover + `;
	--color-success: ` + ColorSuccess + `;
	--color-warning: ` + ColorWarning + `;
	--color-danger: ` + ColorDanger + `;
	--color-border: ` + ColorBorder + `;
	--font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
	--spacing-xs: 4px;
	--spacing-sm: 8px;
	--spacing-base: 16px;
	--spacing-md: 24px;
	--spacing-lg: 40px;
	--radius-sm: 4px;
	--radius-base: 6px;
	--shadow-sm: 0 1px 3px rgba(0, 0, 0, 0.1);
	--shadow-md: 0 2px 8px rgba(0, 0, 0, 0.12);
	--transition: 200ms ease;
	--focus-ring-color: rgba(46, 139, 158, 0.1);
}

* { box-sizing: border-box; }

body {
	font-family: var(--font-family);
	background: var(--color-bg-primary);
	color: var(--color-text-primary);
	margin: 0;
	padding: 0;
	line-height: 1.5;
}

h1 { font-size: 28px; font-weight: 700; margin: 0; }
h2 { font-size: 20px; font-weight: 600; margin: 0; }
p { font-size: 16px; margin: 0; }
small { font-size: 12px; color: var(--color-text-secondary); }

a {
	color: var(--color-accent);
	text-decoration: none;
	transition: color var(--transition);
}

a:hover { color: var(--color-accent-hover); }

button {
	font-family: inherit;
	font-size: 14px;
	font-weight: 600;
	border: none;
	border-radius: var(--radius-base);
	padding: var(--spacing-base) calc(var(--spacing-base) * 1.5);
	cursor: pointer;
	transition: background var(--transition), box-shadow var(--transition);
}

button:hover { opacity: 0.9; }

input, textarea {
	font-family: inherit;
	font-size: 16px;
	padding: var(--spacing-base);
	border: 1px solid var(--color-border);
	border-radius: var(--radius-sm);
	transition: border-color var(--transition);
}

input:focus, textarea:focus {
	outline: none;
	border-color: var(--color-accent);
	box-shadow: 0 0 0 2px var(--focus-ring-color);
}

input::placeholder { color: var(--color-text-secondary); }

/* Site Header */
.site-header {
	background: var(--color-bg-card);
	border-bottom: 1px solid var(--color-border);
	padding: 0;
	position: sticky;
	top: 0;
	z-index: 100;
	box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.site-header-content {
	max-width: 1200px;
	margin: 0 auto;
	padding: var(--spacing-md) var(--spacing-lg);
	display: flex;
	justify-content: space-between;
	align-items: center;
}

.site-header-logo {
	font-size: 20px;
	font-weight: 700;
	color: var(--color-text-primary);
	text-decoration: none;
}

.site-header-nav {
	display: flex;
	gap: var(--spacing-lg);
	align-items: center;
}

.site-header-nav a {
	color: var(--color-text-secondary);
	text-decoration: none;
	font-weight: 500;
	transition: color 0.2s;
}

.site-header-nav a:hover {
	color: var(--color-accent);
}

.site-header-nav a.site-header-login {
	background: var(--color-primary, var(--color-accent));
	color: var(--color-primary-contrast, white) !important;
	padding: 8px 16px;
	border-radius: var(--radius-sm);
	text-decoration: none;
	font-weight: 600;
	transition: opacity 0.2s;
}

.site-header-nav a.site-header-login:hover {
	opacity: 0.9;
	color: var(--color-primary-contrast, white) !important;
}

/* Mobile Responsive */
@media (max-width: 600px) {
	.site-header-content {
		flex-direction: column;
		gap: var(--spacing-base);
	}

	.site-header-nav {
		flex-wrap: wrap;
		justify-content: center;
	}
}

/* Admin Layout */
.admin-header {
	background: var(--color-bg-card);
	border-bottom: 1px solid var(--color-border);
	padding: var(--spacing-base) 0;
	box-shadow: var(--shadow-sm);
	position: sticky;
	top: 0;
	z-index: 10;
}

.container {
	max-width: 1200px;
	margin: 0 auto;
	padding: var(--spacing-md);
	display: flex;
	justify-content: space-between;
	align-items: center;
}

.header-actions {
	display: flex;
	gap: var(--spacing-base);
}

.card {
	background: var(--color-bg-card);
	border: 1px solid var(--color-border);
	border-radius: var(--radius-base);
	padding: var(--spacing-md);
	box-shadow: var(--shadow-sm);
}

.btn-secondary {
	background: var(--color-text-secondary);
	color: white;
}

.btn-secondary:hover {
	background: #4B5563;
}

/* Data Tables */
.data-table {
	width: 100%;
	border-collapse: collapse;
}

.data-table th {
	text-align: left;
	padding: var(--spacing-sm) var(--spacing-md);
	background: var(--color-bg-primary);
	border-bottom: 2px solid var(--color-border);
	font-weight: 600;
}

.data-table td {
	padding: var(--spacing-sm) var(--spacing-md);
	border-bottom: 1px solid var(--color-border);
}

.data-table tr:hover {
	background: var(--color-bg-primary);
}

.btn-small {
	padding: 4px 12px;
	font-size: 13px;
}

.btn-danger {
	background: #dc2626;
	color: white;
}

.btn-danger:hover {
	background: #b91c1c;
}

.btn-contact {
	background: #8b5cf6;
	color: white;
}

.btn-contact:hover {
	background: #7c3aed;
}

/* Admin Container */
.admin-container {
	max-width: 1400px;
	margin: 0 auto;
	padding: var(--spacing-lg);
}

/* Button Styles */
.btn {
	background: var(--color-accent);
	color: white;
	padding: var(--spacing-base) calc(var(--spacing-base) * 1.5);
	border: none;
	border-radius: var(--radius-base);
	font-size: 14px;
	font-weight: 600;
	cursor: pointer;
	transition: opacity var(--transition);
	text-decoration: none;
	display: inline-block;
}

.btn:hover {
	opacity: 0.9;
	color: white;
}

/* Columns Block Styling */
.columns-block {
	margin: var(--spacing-lg) 0;
	align-items: start;
}

.columns-block .column {
	display: flex;
	flex-direction: column;
}

.columns-block .column img {
	width: 100%;
	height: auto;
	aspect-ratio: 16 / 9;
	object-fit: contain;
	background-color: #fdfdfd;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-sm);
	margin-bottom: var(--spacing-sm);
}
`
}
