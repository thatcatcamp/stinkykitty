# StinkyKitty CMS Platform Design

**Date:** 2025-12-20
**Status:** Approved for Implementation

## Overview

**StinkyKitty** is a multi-tenant CMS platform designed to replace WordPress for Burning Man camps and similar community groups. The core philosophy is "WordPress without the chaos" - providing rich, professional-looking websites while avoiding the security and maintenance nightmares of plugin ecosystems.

The platform will be built in Go 1.24+ (for hybrid PQC support), using Gin for HTTP routing, Cobra for CLI commands, and Viper for configuration. Each camp site gets its own isolated database (SQLite by default, MariaDB for high-traffic sites) and media storage, making backups and migrations trivial.

### Multi-Tenancy Model

- Free subdomain hosting (e.g., `yourcamp.stinkykitty.org`) with wildcard SSL
- Optional custom domain support (e.g., `thatcatcamp.com`) with individual Let's Encrypt certs
- Single IP can host dozens of camps with minimal resource usage
- Small sites stay on cheap shared hosting, busy sites can scale independently

### Target Users

Non-technical camp organizers who need simple, secure, good-looking websites without becoming WordPress experts.

## Content Block System

At the heart of StinkyKitty is a structured content block system that gives camps rich-looking pages without security risks.

### Initial Content Blocks (v1)

1. **Hero Block** - Title, subtitle, background image, optional CTA button
2. **Text Block** - Rich formatted text (headings, bold, italic, links, lists)
3. **Image Gallery** - Multiple images with captions, lightbox viewing
4. **Video Embed** - YouTube/Vimeo URLs, automatically embedded
5. **Button Block** - Configurable button with text, link (Google Forms, etc), and style

### Page Editing Model

- Pages are composed of an ordered list of blocks
- Simple up/down reordering (no drag-and-drop complexity)
- Click a block to edit its fields in a form
- Add/remove blocks easily
- Mobile-friendly admin interface

### Block Storage

Each block stored as structured JSON in the database with type, order, and configuration. Easy to version, backup, and migrate.

### Extensibility

Block system designed so adding new block types (forms, maps, FAQs) is straightforward - define schema, create renderer, add to block picker.

## Multi-Tenancy Architecture

### Site Identification and Routing

When an HTTP request arrives, Gin middleware inspects the `Host` header to determine which camp site to serve:

- Subdomain requests (`thatcatcamp.stinkykitty.org`) → lookup site by subdomain
- Custom domain requests (`thatcatcamp.com`) → lookup site by custom domain
- Route to appropriate site database and content

### Site Data Isolation

Each site gets its own directory structure:

```
/var/lib/stinkykitty/sites/
  ├── site-abc123/
  │   ├── site.db (SQLite database)
  │   ├── media/ (uploaded images, files)
  │   └── config.json (site-specific settings)
  └── site-def456/
      ├── mariadb.conf (MariaDB connection for busy site)
      ├── media/ (or S3 config for scaled site)
      └── config.json
```

### Database Strategy

- **New sites:** SQLite database in site directory
- **Growing sites:** CLI migration command to MariaDB (`stinky site migrate-db site-abc123 --to mariadb`)
- **Application code:** Database-agnostic (use GORM or similar ORM)

### Media Storage

- **Default:** Local filesystem in `media/` directory
- **Threshold trigger:** When site hits configurable size/traffic, admin gets notification to migrate
- **Migration:** `stinky site migrate-storage site-abc123 --to s3 --bucket camp-media`
- **Serving:** App serves media through consistent URL scheme regardless of backend

**Hybrid strategy:** Most sites have zero money, so cheap local storage is fine. Larger sites can scale up with S3.

## Authentication & User Management

### Global User Accounts

- Users create one account (email + password) stored in a central users database
- One login gives access to all sites they're authorized for
- Dashboard shows "Your Sites" - jump between camp sites easily

### Site Permissions Model

- **Owner:** Creates the site, ultimate control (delete site, transfer ownership)
- **Admin:** Manage site settings, domains, SSL, backups, user permissions
- **Editor:** Edit pages and content only, no access to settings

### User-to-Site Relationships

Stored in central database:

```
users Table: id, email, password_hash, created_at
site_users Table: user_id, site_id, role (owner/admin/editor)
sites Table: id, subdomain, custom_domain, owner_id, created_at
```

### Authentication Flow

1. User logs in → JWT token issued
2. Token contains user_id
3. When accessing site admin panel, check `site_users` table for permission
4. Editor sees content editor only, Admin sees full settings

### CLI User Management

```bash
stinky user create admin@thatcatcamp.com
stinky site add-user thatcatcamp admin@thatcatcamp.com --role admin
stinky site add-user thatcatcamp editor@thatcatcamp.com --role editor
```

## SSL & Let's Encrypt Integration

### Automatic SSL for Subdomains

- One wildcard certificate (`*.stinkykitty.org`) obtained at system setup
- All subdomain-based camps automatically get HTTPS
- Auto-renewal handled by system cron job
- Zero per-site configuration needed

### Custom Domain SSL

- When admin adds custom domain: `stinky site add-domain thatcatcamp thatcatcamp.com`
- System immediately initiates ACME challenge (HTTP-01 or DNS-01)
- Certificate stored in site directory: `site-abc123/ssl/`
- Auto-renewal tracked per-site (certificate ownership tied to camp)
- Failed renewals send email notification to site owner/admins

### Certificate Storage

```
/var/lib/stinkykitty/
  ├── ssl/
  │   └── wildcard-stinkykitty.org/ (wildcard cert)
  └── sites/
      └── site-abc123/
          └── ssl/
              └── thatcatcamp.com/ (custom domain cert)
```

### Renewal Strategy

- Daily cron checks all certificates
- Renews anything expiring in < 30 days
- Uses Go ACME library (golang.org/x/crypto/acme/autocert or similar)

### DNS Requirements

- **Subdomain:** Must point A record `*.stinkykitty.org` → server IP
- **Custom domain:** Camp must point A record `thatcatcamp.com` → server IP before adding domain

## Backup & Restore System

### Automatic Scheduled Backups

- Configurable schedule (default: daily at 3am)
- Each site backed up independently to `/var/lib/stinkykitty/backups/`
- Backup includes: database dump, media files, SSL certs, config, user permissions
- Retention policy: Keep last 7 daily, 4 weekly, 12 monthly (configurable)
- Old backups automatically pruned

### Backup Storage

```
/var/lib/stinkykitty/backups/
  └── site-abc123/
      ├── 2025-12-20_030000.tar.gz
      ├── 2025-12-19_030000.tar.gz
      └── ...
```

### Manual Export

```bash
stinky site export thatcatcamp --output ~/thatcatcamp-export.tar.gz
```

Creates portable archive with:
- Site database (SQLite file or MariaDB dump)
- All media files
- SSL certificates
- Site config
- `users.json` - all users with access + their roles
- `README.txt` - instructions for import

### Restore Operations

```bash
# Restore from automatic backup
stinky site restore thatcatcamp --from 2025-12-20_030000

# Import exported site
stinky site import ~/thatcatcamp-export.tar.gz --subdomain newcamp
```

### Backup Destination Options

- Local filesystem (default, included in server backups)
- Optional S3 sync: `stinky config set backups.s3_bucket my-backup-bucket`
- Rsync to remote server: `stinky config set backups.rsync_target user@backup.server:/path`

## Theming & Appearance

### Base Theme System

- Single, clean, responsive base theme (mobile-first design)
- Professional look that works for most camps
- Semantic HTML for accessibility

### Per-Site Customization

Admins can configure through web UI or CLI:

- **Primary color** - Used for buttons, links, accents
- **Secondary color** - Backgrounds, hover states
- **Logo image** - Displayed in header
- **Font choice** - 3-4 pre-selected Google Font pairings (readable, fast loading)
- **Site title & tagline**

### CSS Generation

- System generates custom CSS file per site based on their color/font choices
- Cached and served efficiently
- All customization within safe parameters (no custom CSS injection)

### Example

```bash
stinky site config thatcatcamp --primary-color "#FF6B35" --secondary-color "#004E89"
stinky site config thatcatcamp --logo media/cat-logo.png
stinky site config thatcatcamp --font-pair "Montserrat/OpenSans"
```

### Theme Updates

- Base theme improvements/fixes deployed system-wide
- Site customizations preserved
- No per-site theme version hell

### Future Extensibility

- Architecture supports multiple base themes later
- For now, one great theme > three mediocre ones

## CLI Architecture & Commands

### Cobra Command Structure

The `stinky` CLI will be the primary administrative interface, organized into logical command groups.

**Configuration commands:**
```bash
stinky config set key value          # Set config value
stinky config get key                # Get config value
stinky config list                   # Show all config
```

**Site management:**
```bash
stinky site create <name>            # Create new site
stinky site list                     # List all sites
stinky site delete <site>            # Delete site
stinky site add-domain <site> <domain>   # Add custom domain
stinky site migrate-db <site> --to mariadb
stinky site migrate-storage <site> --to s3
stinky site export <site>            # Export site
stinky site import <file>            # Import site
stinky site restore <site> --from <backup>
```

**User management:**
```bash
stinky user create <email>           # Create user account
stinky user list                     # List all users
stinky user delete <email>           # Delete user
stinky site add-user <site> <email> --role admin|editor
stinky site remove-user <site> <email>
stinky site list-users <site>        # Show site users
```

**Server operations:**
```bash
stinky server start                  # Start HTTP server
stinky server stop                   # Stop server
stinky server status                 # Show status
stinky backup run                    # Manual backup all sites
stinky ssl renew                     # Force cert renewal check
```

### Viper Configuration

- Config stored in `/etc/stinkykitty/config.yaml` (or `~/.stinkykitty/config.yaml`)
- All CLI commands modify config programmatically (no manual YAML editing needed)
- Environment variable override support: `STINKY_DATABASE_PATH`
- Configurable ports for reverse proxy support:
  - `stinky config set server.http_port 8080`
  - `stinky config set server.https_port 8443`
  - `stinky config set server.behind_proxy true` (trust X-Forwarded-* headers)

## Technology Stack & Dependencies

### Core Framework

- **Go 1.24+** - Hybrid PQC support, modern tooling
- **Gin** - HTTP router and middleware (fast, mature, good docs)
- **Cobra** - CLI command framework (industry standard)
- **Viper** - Configuration management (YAML, env vars, command-line)

### Database

- **GORM** - ORM supporting both SQLite and MariaDB seamlessly
- **SQLite** - Default database (single file, zero config)
- **MariaDB driver** - For sites that migrate to dedicated database

### Additional Libraries

- **golang.org/x/crypto/acme/autocert** - Let's Encrypt automation
- **AWS SDK for Go** - S3 storage support (when sites migrate)
- **golang-jwt/jwt** - User authentication tokens
- **bluemonday** - HTML sanitization for rich text blocks
- **go-yaml/yaml** - Viper config parsing

### Frontend (Admin Panel)

- Server-rendered HTML templates (Go templates)
- Minimal JavaScript for block editor interactivity
- **HTMX** or **Alpine.js** - Progressive enhancement without SPA complexity
- **Tailwind CSS** - Utility-first styling, consistent UI

### File Structure

```
stinkykitty/
├── cmd/
│   └── stinky/          # Cobra CLI entry point
├── internal/
│   ├── server/          # Gin HTTP server
│   ├── models/          # GORM models
│   ├── blocks/          # Content block system
│   ├── auth/            # Authentication
│   ├── ssl/             # Let's Encrypt handling
│   └── backup/          # Backup/restore logic
├── web/
│   ├── templates/       # HTML templates
│   └── static/          # CSS, JS, images
└── go.mod               # github.com/thatcatcamp/stinkykitty
```

## Deployment & Installation

### Installation

```bash
# Download binary
curl -L https://github.com/thatcatcamp/stinkykitty/releases/latest/download/stinky-linux-amd64 -o stinky
chmod +x stinky
sudo mv stinky /usr/local/bin/

# Initialize system
sudo stinky init --domain stinkykitty.org --email admin@yourhost.com
```

### Init Process

- Creates directory structure (`/var/lib/stinkykitty/`, `/etc/stinkykitty/`)
- Generates initial config
- Sets up wildcard SSL for subdomains
- Creates first admin user
- Installs systemd service

### System Requirements

- Linux server (Ubuntu/Debian preferred)
- 1GB RAM minimum (2GB+ recommended for multiple sites)
- Port 80/443 access for HTTP/HTTPS (or configurable ports for reverse proxy)
- DNS control for wildcard subdomain setup

### Running as Service

```bash
sudo systemctl enable stinkykitty
sudo systemctl start stinkykitty
sudo systemctl status stinkykitty
```

### Update Process

```bash
sudo stinky update       # Downloads latest binary, restarts service
```

### Houston Camps Shared Hosting Scenario

- One beefy VPS ($20-40/month)
- Wildcard DNS `*.camps.houston.org` → server IP
- Each camp gets free subdomain: `thatcat.camps.houston.org`
- Camps with custom domains add A record, then `stinky site add-domain`
- All camps share resources, isolated data

## Implementation Phases

### Phase 1: Core Foundation
- Go module setup with dependencies
- Basic CLI structure (Cobra + Viper)
- Configuration management
- Directory structure setup

### Phase 2: Database & Models
- GORM setup with SQLite
- User, Site, Page, Block models
- Basic CRUD operations

### Phase 3: HTTP Server & Routing
- Gin server setup
- Multi-tenant routing middleware
- Basic site serving

### Phase 4: Content Block System
- Block type definitions (hero, text, gallery, video, button)
- Block renderer system
- Page composition engine

### Phase 5: Authentication
- User registration/login
- JWT token management
- Permission checking middleware

### Phase 6: Admin Interface
- User dashboard (list of sites)
- Page editor UI
- Block add/edit/remove
- Site settings UI

### Phase 7: SSL Integration
- Wildcard cert setup
- ACME client for custom domains
- Auto-renewal system

### Phase 8: Backup & Restore
- Scheduled backup system
- Manual export/import
- Site portability with user data

### Phase 9: Scaling Features
- MariaDB migration support
- S3 storage migration
- Performance optimization

### Phase 10: Polish & Production
- Error handling and logging
- Documentation
- Deployment tooling
- Security hardening

## Success Criteria

- Camp organizers can set up a professional-looking site in < 15 minutes
- Zero WordPress security vulnerabilities
- Sites can be backed up and restored trivially
- One person can manage hosting for dozens of camps
- Lower total cost than WordPress hosting for multiple sites
