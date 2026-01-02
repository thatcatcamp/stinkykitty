#!/bin/bash
# SPDX-License-Identifier: MIT
# scripts/update-block-urls.sh
# Update image block URLs from /uploads/ to /assets/

set -e

DB_PATH="${DB_PATH:-/var/lib/stinkykitty/stinkykitty.db}"

echo "Updating image block URLs in $DB_PATH"

# Check if database exists
if [ ! -f "$DB_PATH" ]; then
    echo "Error: Database not found at $DB_PATH"
    exit 1
fi

# Backup database first
BACKUP_PATH="$DB_PATH.backup-$(date +%Y%m%d-%H%M%S)"
echo "Creating backup at $BACKUP_PATH..."
cp "$DB_PATH" "$BACKUP_PATH"

# Update block data - replace /uploads/ with /assets/ in all blocks
echo "Updating blocks..."
sqlite3 "$DB_PATH" "
UPDATE blocks
SET data = REPLACE(data, '/uploads/', '/assets/')
WHERE data LIKE '%/uploads/%';
"

# Count blocks that now reference /assets/
COUNT=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM blocks WHERE data LIKE '%/assets/%';")
echo "Blocks now referencing /assets/: $COUNT"

# Count blocks still referencing /uploads/ (should be 0 if successful)
REMAINING=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM blocks WHERE data LIKE '%/uploads/%';")
echo "Blocks still referencing /uploads/: $REMAINING"

if [ "$REMAINING" -eq 0 ]; then
    echo "✓ Successfully updated all block URLs!"
else
    echo "⚠ Warning: Some blocks still contain /uploads/ references"
fi

echo "Complete! Database backed up to $BACKUP_PATH"
