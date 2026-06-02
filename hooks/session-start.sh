#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"
# shellcheck source=lib/using-superpowers.sh
source "${SCRIPT_DIR}/lib/using-superpowers.sh"

stdin_tmp="$(mktemp)"
trap 'rm -f "$stdin_tmp"' EXIT
cat >"$stdin_tmp" || true
apply_hook_env_from_stdin "$stdin_tmp"

reset_session_state
reset_using_superpowers_first_prompt
ensure_skill_catalog

message="$(build_session_context_message 20)"
emit_additional_context "$message"