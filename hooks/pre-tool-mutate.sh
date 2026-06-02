#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

stdin_tmp="$(mktemp)"
trap 'rm -f "$stdin_tmp"' EXIT
cat >"$stdin_tmp" || true
apply_hook_env_from_stdin "$stdin_tmp"

ensure_skill_catalog

count="$(catalog_count)"
loaded_count=0
if [ -f "$(skills_loaded_file)" ]; then
  loaded_count="$(loaded_ids | wc -l | tr -d ' ')"
fi

if [ "$loaded_count" -gt 0 ]; then
  emit_allow
fi

if [ "$count" -eq 0 ]; then
  emit_allow
fi

unloaded="$(list_unloaded_ids | head -15 | tr '\n' ', ' | sed 's/,$//')"
emit_deny "Read at least one applicable skill from the grok inspect catalog before mutating files. Rules: ${RULES_PATH}. Unloaded examples: ${unloaded}"