#!/bin/bash
set -e

# Reload systemd to pick up the new service file
systemctl daemon-reload

# Ensure data directories exist
mkdir -p /var/lib/stinkykitty/sites
mkdir -p /var/lib/stinkykitty/backups
mkdir -p /etc/stinkykitty

# Permissions (running as root, so these are owned by root)
chmod 755 /var/lib/stinkykitty
chmod 755 /etc/stinkykitty

# Note: We don't start the service here to avoid surprises during install,
# unless it's an upgrade and it was already running.
# Most distros have their own ways of handling this (e.g. deb-systemd-helper).
