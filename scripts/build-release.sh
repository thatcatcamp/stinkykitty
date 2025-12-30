#!/bin/bash
# Build release with SBOM generation and signing
# Requirements: syft, cosign or gpg

set -e

VERSION="${1:-dev}"
OUTPUT_DIR="dist"
BINARY_NAME="stinky"

echo "Building StinkyKitty CMS v${VERSION}..."

# Clean and create output directory
rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"

# Build the binary
echo "Building binary..."
go build -ldflags "-X main.Version=${VERSION}" -o "${OUTPUT_DIR}/${BINARY_NAME}" ./cmd/stinky

# Generate SBOM with syft
echo "Generating SBOM..."
if ! command -v syft &> /dev/null; then
    echo "Error: syft is not installed. Install it with:"
    echo "  curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin"
    exit 1
fi

# Generate SBOM in multiple formats
syft "${OUTPUT_DIR}/${BINARY_NAME}" -o spdx-json="${OUTPUT_DIR}/${BINARY_NAME}.sbom.spdx.json"
syft "${OUTPUT_DIR}/${BINARY_NAME}" -o cyclonedx-json="${OUTPUT_DIR}/${BINARY_NAME}.sbom.cyclonedx.json"

# Generate checksums
echo "Generating checksums..."
cd "$OUTPUT_DIR"
sha256sum "${BINARY_NAME}" > "${BINARY_NAME}.sha256"
sha256sum "${BINARY_NAME}.sbom.spdx.json" > "${BINARY_NAME}.sbom.spdx.json.sha256"
sha256sum "${BINARY_NAME}.sbom.cyclonedx.json" > "${BINARY_NAME}.sbom.cyclonedx.json.sha256"
cd ..

# Sign the artifacts
echo "Signing artifacts..."

# Try cosign first (recommended for modern workflows)
if command -v cosign &> /dev/null; then
    echo "Using cosign for signing..."
    echo "Note: Ensure COSIGN_PASSWORD or COSIGN_KEY is set, or use --key flag"

    # Sign binary
    if [ -n "$COSIGN_KEY" ]; then
        cosign sign-blob --key "$COSIGN_KEY" --output-signature "${OUTPUT_DIR}/${BINARY_NAME}.sig" "${OUTPUT_DIR}/${BINARY_NAME}"
        cosign sign-blob --key "$COSIGN_KEY" --output-signature "${OUTPUT_DIR}/${BINARY_NAME}.sbom.spdx.json.sig" "${OUTPUT_DIR}/${BINARY_NAME}.sbom.spdx.json"
    else
        echo "Skipping signing: COSIGN_KEY not set"
        echo "To sign, set COSIGN_KEY environment variable or use GPG"
    fi
# Fall back to GPG
elif command -v gpg &> /dev/null; then
    echo "Using GPG for signing..."
    gpg --detach-sign --armor "${OUTPUT_DIR}/${BINARY_NAME}"
    gpg --detach-sign --armor "${OUTPUT_DIR}/${BINARY_NAME}.sbom.spdx.json"
else
    echo "Warning: Neither cosign nor gpg found. Artifacts will not be signed."
    echo "Install cosign: https://docs.sigstore.dev/cosign/installation/"
    echo "Or use GPG: apt-get install gnupg"
fi

echo ""
echo "✅ Build complete!"
echo ""
echo "Artifacts in ${OUTPUT_DIR}/:"
ls -lh "${OUTPUT_DIR}/"
echo ""
echo "To verify checksums:"
echo "  cd ${OUTPUT_DIR} && sha256sum -c ${BINARY_NAME}.sha256"
echo ""
if [ -f "${OUTPUT_DIR}/${BINARY_NAME}.sig" ]; then
    echo "To verify cosign signature:"
    echo "  cosign verify-blob --key <public-key> --signature ${OUTPUT_DIR}/${BINARY_NAME}.sig ${OUTPUT_DIR}/${BINARY_NAME}"
elif [ -f "${OUTPUT_DIR}/${BINARY_NAME}.asc" ]; then
    echo "To verify GPG signature:"
    echo "  gpg --verify ${OUTPUT_DIR}/${BINARY_NAME}.asc ${OUTPUT_DIR}/${BINARY_NAME}"
fi
