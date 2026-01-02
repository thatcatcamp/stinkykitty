#!/bin/bash
# SPDX-License-Identifier: MIT
# scripts/migrate-media.sh
# Migrate existing site-specific media to centralized storage

set -e

SITES_DIR="${SITES_DIR:-/var/lib/stinkykitty/sites}"
MEDIA_DIR="${MEDIA_DIR:-/var/lib/stinkykitty/media}"

echo "Migrating media from $SITES_DIR to $MEDIA_DIR"

# Create centralized media directory if it doesn't exist
mkdir -p "$MEDIA_DIR/uploads"
mkdir -p "$MEDIA_DIR/uploads/thumbs"

# Find all site directories
for site_dir in "$SITES_DIR"/site-*/; do
    if [ ! -d "$site_dir" ]; then
        continue
    fi

    site_name=$(basename "$site_dir")
    echo "Processing $site_name..."

    # Check if uploads directory exists
    if [ ! -d "$site_dir/uploads" ]; then
        echo "  No uploads directory, skipping"
        continue
    fi

    # Copy all files from site uploads to centralized uploads
    if [ -d "$site_dir/uploads" ]; then
        echo "  Copying uploads..."
        cp -n "$site_dir/uploads"/*.{jpg,jpeg,png,gif,webp} "$MEDIA_DIR/uploads/" 2>/dev/null || true
    fi

    # Copy thumbnails
    if [ -d "$site_dir/uploads/thumbs" ]; then
        echo "  Copying thumbnails..."
        cp -n "$site_dir/uploads/thumbs"/*.{jpg,jpeg,png,gif,webp} "$MEDIA_DIR/uploads/thumbs/" 2>/dev/null || true
    fi
done

# Set permissions
chown -R stinky:stinky "$MEDIA_DIR" || chown -R $USER:$USER "$MEDIA_DIR"
chmod -R 755 "$MEDIA_DIR"

echo "Migration complete!"
echo "Note: Old files in site directories are NOT deleted. Clean up manually after verifying."
