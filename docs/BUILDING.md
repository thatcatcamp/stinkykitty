# Building and Releasing StinkyKitty CMS

## License

StinkyKitty CMS is licensed under the MIT License. All source files include SPDX license identifiers for compliance.

## Quick Build

```bash
go build -o stinky cmd/stinky/main.go
```

## Release Build with SBOM

For production releases with Software Bill of Materials (SBOM) and signing:

### Prerequisites

Install required tools:

```bash
# Install syft (SBOM generation)
curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin

# Install cosign (artifact signing) - optional but recommended
go install github.com/sigstore/cosign/v2/cmd/cosign@latest
```

### Build Release

```bash
./scripts/build-release.sh v1.0.0
```

This will:
1. Build the binary
2. Generate SBOM in SPDX and CycloneDX formats
3. Generate SHA256 checksums
4. Sign artifacts (if cosign or gpg is configured)

### Output Artifacts

The `dist/` directory will contain:

```
stinky                          # Binary
stinky.sbom.spdx.json          # SBOM in SPDX format
stinky.sbom.cyclonedx.json     # SBOM in CycloneDX format
stinky.sha256                   # Binary checksum
stinky.sbom.spdx.json.sha256   # SBOM checksum
stinky.sig                      # Signature (if signed)
```

## Automated GitHub Releases

When you push a git tag starting with `v`, GitHub Actions will automatically:

1. Build binaries for multiple platforms (Linux/macOS, AMD64/ARM64)
2. Generate SBOMs for each binary
3. Sign all artifacts with cosign
4. Create a GitHub release with all artifacts
5. Upload SBOMs as artifacts

### Creating a Release

```bash
# Tag the release
git tag -a v1.0.0 -m "Release v1.0.0"

# Push the tag
git push origin v1.0.0
```

GitHub Actions will handle the rest!

## SBOM Formats

We generate SBOMs in two formats:

- **SPDX**: Industry standard, ISO/IEC 5962:2021
- **CycloneDX**: OWASP standard, popular in security tools

Both formats include:
- Complete dependency tree
- License information
- Package versions
- CVE information (where available)

## Signature Verification

### With cosign

```bash
# Verify binary signature
cosign verify-blob --key <public-key> --signature dist/stinky.sig dist/stinky

# Verify SBOM signature
cosign verify-blob --key <public-key> --signature dist/stinky.sbom.spdx.json.sig dist/stinky.sbom.spdx.json
```

### With GPG

```bash
# Verify binary signature
gpg --verify dist/stinky.asc dist/stinky
```

## Checksum Verification

```bash
cd dist
sha256sum -c stinky.sha256
```

## SPDX Compliance

All Go source files include SPDX license identifiers:

```go
// SPDX-License-Identifier: MIT
package main
```

This enables automated license scanning and compliance checking.

## Cross-Platform Builds

Build for specific platforms:

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o stinky-linux-amd64 ./cmd/stinky

# macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o stinky-darwin-arm64 ./cmd/stinky
```

## Security

- All releases are signed with cosign (keyless signing via Sigstore)
- SBOMs allow vulnerability tracking
- Checksums prevent tampering
- SPDX identifiers enable license compliance
