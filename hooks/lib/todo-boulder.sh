#!/usr/bin/env bash
# Todo continuation + boulder.json (omo-compatible) for Grok hooks.

_omo_lib_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OMO_STATE_PY="${_omo_lib_dir}/omo_state.py"

omo_state() {
  python3 "$OMO_STATE_PY" "$@"
}

evaluate_todo_continuation_stop() {
  local stdin_file="${1:-}"
  OMO_ACTION=evaluate_todo_stop omo_state "$stdin_file"
}

evaluate_boulder_stop() {
  local stdin_file="${1:-}"
  OMO_ACTION=evaluate_boulder_stop omo_state "$stdin_file"
}

build_boulder_context() {
  OMO_ACTION=build_boulder_context omo_state 2>/dev/null || true
}

set_continuation_stopped() {
  OMO_ACTION=set_continuation_stopped omo_state
}

clear_continuation_stopped() {
  OMO_ACTION=clear_continuation_stopped omo_state
}

clear_boulder_state() {
  OMO_ACTION=clear_boulder omo_state
}

mirror_session_todos() {
  OMO_ACTION=mirror_todos omo_state
}

auto_continue_paused() {
  local workspace="${GROK_WORKSPACE_ROOT:-}"
  local session_id="${GROK_SESSION_ID:-}"
  [ -n "$workspace" ] && [ -n "$session_id" ] || return 1
  OMO_ACTION=is_continuation_stopped python3 "$OMO_STATE_PY" 2>/dev/null | rg -q '^1$'
}

collect_stop_continuation_prompt() {
  local stdin_file="$1"
  local workspace="${GROK_WORKSPACE_ROOT:-}"
  local state_path=""
  local prompt
  prompt="$(
    python3 - "$stdin_file" <<'PY'
import json, sys
try:
    with open(sys.argv[1], encoding="utf-8") as f:
        data = json.load(f)
except (OSError, json.JSONDecodeError):
    raise SystemExit(0)
for k in ("prompt", "userPrompt", "user_prompt", "message"):
    v = data.get(k)
    if isinstance(v, str) and v.strip():
        print(v.strip())
        break
PY
  )"
  [ -n "$prompt" ] || return 0
  if printf '%s' "$prompt" | rg -qi '^/?stop-continuation\b'; then
    set_continuation_stopped
    if [ -n "$workspace" ] && state_path="$(ralph_state_path "$workspace" 2>/dev/null)"; then
      ralph_clear_state "$state_path" >/dev/null 2>&1 || true
    fi
    clear_boulder_state
    printf '%s\n' "<STOP_CONTINUATION>Stopped: todo continuation, Ralph/ultrawork loop, and boulder.json cleared. Auto-continue resumes on SessionEnd or /resume-continuation.</STOP_CONTINUATION>"
    return 0
  fi
  if printf '%s' "$prompt" | rg -qi '^/?resume-continuation\b'; then
    clear_continuation_stopped
    printf '%s\n' "<STOP_CONTINUATION>Auto-continuation resumed for this session.</STOP_CONTINUATION>"
  fi
}

collect_boulder_prompt_context() {
  build_boulder_context
}

handle_user_prompt_stop_continuation() {
  local ctx
  ctx="$(collect_stop_continuation_prompt "$1")"
  [ -n "$ctx" ] || return 0
  emit_additional_context "$ctx" "UserPromptSubmit"
}

handle_user_prompt_boulder_context() {
  local ctx
  ctx="$(collect_boulder_prompt_context)"
  [ -n "$ctx" ] || return 0
  emit_additional_context "$ctx" "UserPromptSubmit"
}

mirror_todos_after_write() {
  mirror_session_todos
}

cleanup_omo_session_state() {
  local session_id="${GROK_SESSION_ID:-}"
  local workspace="${GROK_WORKSPACE_ROOT:-}"
  [ -n "$session_id" ] || return 0
  clear_continuation_stopped
  rm -rf "${GROK_HOME}/state/boulder-nudge/${session_id}" 2>/dev/null || true
}