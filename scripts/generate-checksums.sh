#!/usr/bin/env bash
# Generate SHA256 checksums for all build artifacts.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

CHECKSUM_FILE="bin/checksums.sha256"
echo "Generating checksums..."

# Build all binaries first
bash scripts/build-hook.sh
bash scripts/build-mcp.sh

# Generate checksums
: > "$CHECKSUM_FILE"
for f in bin/omg-hook-* bin/omg-mcp-*; do
  if [ -f "$f" ]; then
    sha256sum "$f" >> "$CHECKSUM_FILE"
  fi
done

echo "Checksums written to $CHECKSUM_FILE"
cat "$CHECKSUM_FILE"
