# Backup and Export System Design

**Date:** 2025-12-25
**Status:** Design Approved, Ready for Implementation
**Scope:** Automated backups + user-accessible exports for data portability

## Overview

Complete backup and export system preventing vendor lock-in. Site owners can export their complete site data on demand, while global admins manage automatic system backups via CLI.

## Key Requirements

- **Full data exports** - Database dumps + all uploaded media as single tarball
- **Automatic backups** - Daily scheduled backups, keep last 10, delete older
- **Site owner self-service** - One-click export from admin UI (their site only)
- **Global admin control** - CLI commands to list, restore, manage backups
- **Seasonal scaling** - Retention policy works for peak activity mid-year and dead season after August
- **Simple storage** - Filesystem-based (`/var/lib/stinkykitty/backups/`), user handles S3 mirroring

## Architecture

### Backup Storage Structure

```
/var/lib/stinkykitty/backups/
├── system/
│   ├── stinkykitty-2025-12-25-143022.tar.gz
│   ├── stinkykitty-2025-12-24-120000.tar.gz
│   └── [last 10 backups]
└── site-exports/
    ├── site-1-2025-12-25-143022.tar.gz
    └── [temporary exports before download]
```

### Backup Contents

**System backups** include:
- Full database dump (SQLite or MariaDB depending on config)
- All uploaded media from `var/lib/stinkykitty/uploads/`
- Metadata: backup timestamp, database type, version

**Site exports** include:
- That site's pages (with all blocks)
- Menu items for that site
- All uploaded media for that site
- Metadata: export timestamp, site name, site ID

### Components

1. **Backup Package** (`internal/backup/`)
   - `backup.go` - Core backup logic (create, list, delete, restore)
   - `export.go` - Site export logic (create site-specific export)
   - `scheduler.go` - Background job for automatic backups

2. **CLI Commands** (`cmd/stinky/backup.go`)
   - `stinky backup list` - Show all backups
   - `stinky backup restore <timestamp>` - Restore system from backup
   - `stinky backup delete <timestamp>` - Manually delete backup
   - `stinky backup status` - Show backup usage stats

3. **Admin Handler** (`internal/handlers/admin_export.go`)
   - `ExportSiteHandler` - Site owner downloads their site export
   - HTTP endpoint: `POST /admin/export` (triggers download in browser)

4. **Server Integration** (`cmd/stinky/server.go`)
   - Initialize backup scheduler on server start
   - Daily backup job runs at configured time (default: 2 AM)
   - Add export route to admin routes

### Data Flow

**Automatic Backup Flow:**
1. Server starts → scheduler initialized
2. Daily at 2 AM: Create database dump + tar media
3. Save as `stinkykitty-YYYY-MM-DD-HHMMSS.tar.gz`
4. Check backup count → if > 10, delete oldest
5. Log backup status

**Site Export Flow:**
1. Site owner clicks "Download My Site" button in admin
2. POST to `/admin/export` with site context
3. Handler creates temporary tarball with that site's data
4. Return as downloadable file (Content-Disposition: attachment)
5. Cleanup temporary file after download

**Restore Flow:**
1. Admin runs `stinky backup restore <timestamp>`
2. Verify timestamp exists in backups/system/
3. Backup interactive prompt: "Restore will overwrite database. Continue? (y/n)"
4. Extract database dump, restore via GORM migrations
5. Extract media files to uploads/ directory
6. Verify integrity (file counts match)
7. Log restore operation

## Error Handling

- **Backup fails** - Log error, alert via stderr, don't delete previous backup
- **Restore fails** - Rollback database (GORM transaction), restore previous media, error message to user
- **Disk full** - Check available space before backup, fail gracefully with clear message
- **Corrupted backup** - Verify tar integrity on restore, abort if corrupted
- **Export fails** - Return 500 error with clear message to site owner

## Testing Strategy

- Unit tests for backup/restore logic
- Integration test: create backup → restore → verify data matches
- Export test: create site export → verify tar contents
- Scheduler test: verify daily job runs and cleans up
- CLI tests: backup list, restore, delete commands

## Success Criteria

- Site owners can download complete export of their site in under 5 seconds
- Automatic backups run daily with zero manual intervention
- Last 10 backups retained automatically
- Restore operation recovers all data perfectly
- CLI commands work for global admin backup management
- No data loss in any scenario (backup failure, restore failure, etc.)
