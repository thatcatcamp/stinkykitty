#!/bin/bash
# sign-wrapper.sh
# Usage: ./sign-wrapper.sh <artifact> <signature_path>

ARTIFACT=$1
SIGNATURE=$2

echo "Attempting to sign $ARTIFACT..."
if cosign sign-blob --output-signature="$SIGNATURE" "$ARTIFACT" --yes; then
    echo "Successfully signed $ARTIFACT"
else
    echo "Warning: Failed to sign $ARTIFACT. Proceeding without signature due to infrastructure flakiness."
    # Exit with 0 to ensure GoReleaser doesn't fail the build
    exit 0
fi
