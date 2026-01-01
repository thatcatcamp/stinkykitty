#!/bin/bash
# scripts/create-media-dir.sh

MEDIA_DIR="${MEDIA_DIR:-/var/lib/stinkykitty/media}"

echo "Creating centralized media directory: $MEDIA_DIR"
mkdir -p "$MEDIA_DIR"
mkdir -p "$MEDIA_DIR/uploads"
mkdir -p "$MEDIA_DIR/uploads/thumbs"

# Set permissions (adjust user as needed)
chown -R stinky:stinky "$MEDIA_DIR" || chown -R $USER:$USER "$MEDIA_DIR"
chmod -R 755 "$MEDIA_DIR"

echo "Media directory created successfully"
