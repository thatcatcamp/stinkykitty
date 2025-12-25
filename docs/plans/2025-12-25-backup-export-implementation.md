# Backup and Export System Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Implement automatic backup system with retention, site export for data portability, and CLI management commands.

**Architecture:** Three-layer system - backup package handles all backup/restore logic, admin handler provides site export UI endpoint, scheduler runs daily backups, CLI commands manage backups. Full data exports (database + media) stored as tarballs in `/var/lib/stinkykitty/backups/`.

**Tech Stack:** Go 1.24+, tar/gzip, cron-like scheduler (ticker), existing GORM/Gin

---

## 10 Core Tasks

1. Create backup package structure (types, manager)
2. Implement system backup creation (tar.gz compression)
3. Implement backup restoration from tarball
4. Implement site export (user-facing)
5. Implement backup scheduler (daily automated)
6. Add CLI backup commands (list, restore, delete, status)
7. Create admin export handler + UI button
8. Initialize scheduler in server startup
9. Add backup configuration defaults
10. Run full test suite

## Key Implementation Notes

- Backup paths: `/var/lib/stinkykitty/backups/system/` for system backups, `/site-exports/` for user exports
- Keep last 10 backups, delete older ones automatically
- Export includes database + media as single tarball
- Scheduler runs daily at 2 AM (configurable)
- CLI requires manual confirmation for restore (safety)
- All tests use temporary directories
- Error handling: cleanup on failure, detailed error messages

---

Ready to implement! Which approach?

**Option A:** Subagent-driven (this session) - Fresh subagent per task, code review after each task

**Option B:** Execute-plans (parallel session) - Batch execution with checkpoints in separate session

Which would you prefer?
