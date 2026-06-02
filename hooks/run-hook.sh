#!/usr/bin/env bash
# Invoke a hook script from this directory (cwd-independent).
set -euo pipefail
if [ $# -lt 1 ]; then
  echo "run-hook.sh: missing script name" >&2
  exit 1
fi
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT_NAME="$1"
shift
exec bash "${SCRIPT_DIR}/${SCRIPT_NAME}" "$@"