#!/usr/bin/env bash
# First-prompt injection for superpowers:using-superpowers (Grok /using-superpowers).

using_superpowers_state_dir() {
  local session_id="${GROK_SESSION_ID:-unknown}"
  printf '%s/state/using-superpowers/%s' "$GROK_HOME" "$session_id"
}

using_superpowers_done_file() {
  printf '%s/first_prompt_done' "$(using_superpowers_state_dir)"
}

resolve_using_superpowers_skill_path() {
  local plugin_root="${GROK_HOME}/installed-plugins"
  local candidate
  if [ -n "${USING_SUPERPOWERS_SKILL:-}" ] && [ -f "${USING_SUPERPOWERS_SKILL}" ]; then
    printf '%s' "${USING_SUPERPOWERS_SKILL}"
    return 0
  fi
  for candidate in "${plugin_root}"/*/skills/using-superpowers/SKILL.md; do
    if [ -f "$candidate" ]; then
      printf '%s' "$candidate"
      return 0
    fi
  done
  return 1
}

first_user_prompt_not_yet_handled() {
  local done
  done="$(using_superpowers_done_file)"
  [ ! -f "$done" ]
}

mark_first_user_prompt_handled() {
  local dir done
  dir="$(using_superpowers_state_dir)"
  done="$(using_superpowers_done_file)"
  mkdir -p "$dir"
  : >"$done"
}

reset_using_superpowers_first_prompt() {
  local dir
  dir="$(using_superpowers_state_dir)"
  if [ -d "$dir" ]; then
    rm -rf "$dir"
  fi
}

cleanup_using_superpowers_state() {
  reset_using_superpowers_first_prompt
}

build_using_superpowers_context() {
  local skill_path="$1"
  python3 - "$skill_path" <<'PY'
import sys

path = sys.argv[1]
try:
    with open(path, encoding="utf-8") as f:
        body = f.read()
except OSError as e:
    body = f"(using-superpowers skill unavailable: {e})"

print(
    "<USING_SUPERPOWERS_FIRST_PROMPT>\n"
    "MANDATORY: You are starting this session's first user turn. "
    "Treat this as invoking `/using-superpowers` — follow the skill below before "
    "any response, tool use, or clarifying question (subagents: skip per skill).\n\n"
    "**Full content of superpowers:using-superpowers:**\n\n"
    f"{body}\n"
    "</USING_SUPERPOWERS_FIRST_PROMPT>"
)
PY
}

collect_using_superpowers_on_first_prompt() {
  if ! first_user_prompt_not_yet_handled; then
    return 0
  fi
  local skill_path
  if ! skill_path="$(resolve_using_superpowers_skill_path)"; then
    return 0
  fi
  mark_first_user_prompt_handled
  mark_skill_loaded "using-superpowers" 2>/dev/null || true
  build_using_superpowers_context "$skill_path"
}

maybe_emit_using_superpowers_on_first_prompt() {
  local message
  message="$(collect_using_superpowers_on_first_prompt)" || return 0
  [ -n "$message" ] || return 0
  emit_additional_context "$message" "UserPromptSubmit"
}