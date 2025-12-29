# StinkyKitty CMS Features

## User Management

### Overview
Site administrators can manage users with access to their sites. Global administrators can manage all users across all sites.

### Access
- Navigate to **Admin Dashboard** → **Manage Users**
- View all users with access to your sites
- See user email, sites, role, and creation date

### Actions

**Reset Password**
- Click "Reset Password" next to any user
- Sends password reset email to the user
- User receives 24-hour reset link

**Remove User**
- Click "Remove" to soft-delete a user
- User loses access to the site
- Can be restored by recreating with same email

## Google Analytics Integration

### Setup
1. Get your Google Analytics tracking ID from Google Analytics
   - Format: `G-XXXXXXXXXX` (GA4) or `UA-XXXXXXXXX` (Universal Analytics)
2. Go to **Admin** → **Settings**
3. Enter tracking ID in "Google Analytics Tracking ID" field
4. Save settings

### What It Does
- Automatically injects GA tracking script on all public pages
- Tracks page views, user behavior, and site analytics
- Data appears in your Google Analytics dashboard

## Site Customization

### Fixed Header Bar
All public pages now include a fixed header bar with:
- Site title/logo (links to homepage)
- Navigation menu
- Login button (for site admins)

The header stays at the top when scrolling for easy navigation.

### Custom Copyright
Customize your site's footer copyright text:

1. Go to **Admin** → **Settings**
2. Edit "Copyright Text" field
3. Use placeholders:
   - `{year}` → Replaced with current year
   - `{site}` → Replaced with site title

**Example:**
Input: `© {year} {site} - All rights reserved.`
Output: `© 2025 My Camp Site - All rights reserved.`

## Column Layouts

### Creating Column Blocks
1. Edit any page
2. Click **+ Columns**
3. Choose 2, 3, or 4 columns
4. Add content to each column
5. Save

### Use Cases
- Feature grids (3 columns of features)
- Image galleries
- Button groups
- Text + image side-by-side layouts

### Tips
- Columns stack vertically on mobile
- Keep column content balanced
- Use with other blocks for rich layouts
