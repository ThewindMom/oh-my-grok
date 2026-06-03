#!/usr/bin/env bash
# Vendor obra/superpowers skills into vendor/superpowers/ (skills only; no hooks).
# Maintainers: run before release or when bumping the pin in vendor/superpowers/VERSION.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DEST="$ROOT/vendor/superpowers"
REF="${SUPERPOWERS_REF:-v5.1.0}"
REPO="${SUPERPOWERS_REPO:-https://github.com/obra/superpowers.git}"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

git clone --depth 1 --branch "$REF" "$REPO" "$TMP/repo" 2>/dev/null || {
  echo "vendor-superpowers: clone failed for ref=$REF; try SUPERPOWERS_REF=main" >&2
  git clone --depth 1 "$REPO" "$TMP/repo"
  (cd "$TMP/repo" && git checkout "$REF" 2>/dev/null || true)
}

test -d "$TMP/repo/skills" || {
  echo "vendor-superpowers: missing skills/ in $REPO" >&2
  exit 1
}

rm -rf "$DEST"
mkdir -p "$DEST"
cp -a "$TMP/repo/skills" "$DEST/skills"
cp -a "$TMP/repo/LICENSE" "$DEST/LICENSE"
printf '%s\n' "$REF" >"$DEST/VERSION"
printf '%s\n' "$REPO" >"$DEST/SOURCE"

count="$(find "$DEST/skills" -name SKILL.md | wc -l | tr -d ' ')"
echo "vendor/superpowers: $count skills @ $REF"