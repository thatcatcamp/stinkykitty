#!/bin/bash
# Add SPDX license identifiers to all Go source files

set -e

SPDX_HEADER="// SPDX-License-Identifier: MIT"

# Find all .go files
find . -name "*.go" -type f | while read -r file; do
    # Skip if already has SPDX identifier
    if grep -q "SPDX-License-Identifier" "$file"; then
        echo "Skipping $file (already has SPDX header)"
        continue
    fi

    # Add SPDX header at the top
    echo "$SPDX_HEADER" | cat - "$file" > "$file.tmp"
    mv "$file.tmp" "$file"
    echo "Added SPDX header to $file"
done

echo "Done! Added SPDX headers to all Go files."
