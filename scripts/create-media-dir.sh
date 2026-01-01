#!/bin/bash
# SPDX-License-Identifier: MIT
# scripts/create-media-dir.sh

set -e

MEDIA_DIR="${MEDIA_DIR:-/var/lib/stinkykitty/media}"

echo "Creating centralized media directory: $MEDIA_DIR"
mkdir -p "$MEDIA_DIR"
mkdir -p "$MEDIA_DIR/uploads"
mkdir -p "$MEDIA_DIR/uploads/thumbs"

# Set permissions (adjust user as needed)
# Attempt to set stinky user ownership, fall back to current user if stinky doesn't exist
if id "stinky" &>/dev/null; then
    chown -R stinky:stinky "$MEDIA_DIR"
else
    echo "Note: 'stinky' user not found, using $USER for ownership"
    chown -R "$USER:$USER" "$MEDIA_DIR"
fi
chmod -R 755 "$MEDIA_DIR"

echo "Media directory created successfully"
