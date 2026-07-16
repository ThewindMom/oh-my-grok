#!/usr/bin/env bash
# Build the omg-mcp binary for all supported platforms.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

PLATFORMS=(
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
  "windows/amd64"
)

for platform in "${PLATFORMS[@]}"; do
  os="${platform%/*}"
  arch="${platform#*/}"
  output="bin/omg-mcp-${os}-${arch}"
  if [ "$os" = "windows" ]; then
    output="${output}.exe"
  fi
  echo "Building $output..."
  GOOS="$os" GOARCH="$arch" go build -o "$output" ./cmd/omg-mcp/
done

echo "Built bin/omg-mcp-*"
