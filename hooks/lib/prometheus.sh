#!/usr/bin/env bash
# Prometheus plan mode: /plan interview, md-only guard, /start-work boulder bootstrap.

_prometheus_lib_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=intent-gate.sh
source "${_prometheus_lib_dir}/intent-gate.sh"

plan_mode_flag() {
  printf '%s/state/plan-mode/%s/enabled' "$GROK_HOME" "${GROK_SESSION_ID:-unknown}"
}

prometheus_plan_mode_on() {
  local f
  f="$(plan_mode_flag)"
  mkdir -p "$(dirname "$f")"
  printf '%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" >"$f"
}

prometheus_plan_mode_off() {
  rm -f "$(plan_mode_flag)" 2>/dev/null || true
}

prometheus_plan_mode_active() {
  case "${OMG_PLAN_MODE:-}" in
    1|true|yes|on) return 0 ;;
  esac
  [ -f "$(plan_mode_flag)" ]
}

collect_user_prompt_prometheus() {
  local stdin_file="${1:-}"
  local prompt
  prompt="$(intent_gate_extract_prompt "$stdin_file" 2>/dev/null || true)"
  [ -n "$prompt" ] || return 0

  if printf '%s' "$prompt" | rg -qi '^/?plan\b|^/?prometheus\b'; then
    prometheus_plan_mode_on
    cat <<'EOF'
<PROMETHEUS_PLAN_MODE>
You are in planning mode. ONLY create or edit files under `.omg/` (plans, drafts).
Interview the user, then Task(subagent_type="metis-consultant") for gaps, write plan to `.omg/plans/<name>.md`, optional Task(subagent_type="momus-reviewer").
Implementation starts only after `/start-work <plan-file>`.
</PROMETHEUS_PLAN_MODE>
EOF
    return 0
  fi

  if printf '%s' "$prompt" | rg -qi '^/?start-work\b'; then
    prometheus_handle_start_work "$prompt"
    return 0
  fi

  if printf '%s' "$prompt" | rg -qi '^/?cancel-plan\b'; then
    prometheus_plan_mode_off
    printf '%s\n' '<PROMETHEUS_PLAN_MODE>Plan mode cancelled.</PROMETHEUS_PLAN_MODE>'
  fi
}

prometheus_handle_start_work() {
  local prompt="$1"
  local workspace="${GROK_WORKSPACE_ROOT:-}"
  local session_id="${GROK_SESSION_ID:-}"
  prometheus_plan_mode_off

  if [ -z "$workspace" ] || [ -z "$session_id" ]; then
    printf '%s\n' '<PROMETHEUS_PLAN_MODE>Start-work failed: missing workspace or session.</PROMETHEUS_PLAN_MODE>'
    return 0
  fi

  python3 - "$workspace" "$session_id" "$prompt" <<'PY' || true
import json
import os
import re
import sys
from datetime import datetime, timezone
from pathlib import Path

workspace, session_id, prompt = sys.argv[1], sys.argv[2], sys.argv[3]
now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%S+00:00")

m = re.search(r"^/?start-work(?:\s+(.+))?$", prompt.strip(), re.I | re.S)
raw = (m.group(1) or "").strip().strip("\"'") if m else ""
if not raw:
    print("<PROMETHEUS_PLAN_MODE>Start-work failed: provide plan path, e.g. /start-work .omg/plans/auth.md</PROMETHEUS_PLAN_MODE>")
    raise SystemExit(0)

base = Path(workspace)
plan_path = Path(raw)
if not plan_path.is_absolute():
    plan_path = base / raw
if not plan_path.is_file():
    # try under .omg/plans/
    alt = base / ".omg" / "plans" / Path(raw).name
    if alt.is_file():
        plan_path = alt
    else:
        print(f"<PROMETHEUS_PLAN_MODE>Start-work failed: plan not found: {raw}</PROMETHEUS_PLAN_MODE>")
        raise SystemExit(0)

try:
    rel = plan_path.resolve().relative_to(base.resolve())
    active_plan = str(rel).replace("\\", "/")
except ValueError:
    active_plan = str(plan_path)

if not active_plan.startswith(".omg/") or not active_plan.endswith(".md"):
    print("<PROMETHEUS_PLAN_MODE>Start-work failed: plan must be under .omg/ and end with .md</PROMETHEUS_PLAN_MODE>")
    raise SystemExit(0)

plan_name = Path(active_plan).stem
work_id = f"{plan_name}-work"

state = {
    "schema_version": 2,
    "active_work_id": work_id,
    "active_plan": active_plan,
    "plan_name": plan_name,
    "status": "active",
    "started_at": now,
    "updated_at": now,
    "session_ids": [session_id],
    "works": {
        work_id: {
            "work_id": work_id,
            "active_plan": active_plan,
            "plan_name": plan_name,
            "status": "active",
            "started_at": now,
            "updated_at": now,
            "session_ids": [session_id],
            "task_sessions": {},
        }
    },
}

boulder_file = base / ".omg" / "boulder.json"
boulder_file.parent.mkdir(parents=True, exist_ok=True)
boulder_file.write_text(json.dumps(state, indent=2) + "\n", encoding="utf-8")
PY

  printf '%s\n' '<PROMETHEUS_PLAN_MODE>Start-work: boulder.json activated. Execute the plan.</PROMETHEUS_PLAN_MODE>'
}

evaluate_prometheus_pre_tool() {
  prometheus_plan_mode_active || return 0
  local stdin_file="${1:-}"
  [ -n "$stdin_file" ] || return 0
  python3 - "$stdin_file" <<'PY'
import json
import os
import sys

workspace = os.environ.get("GROK_WORKSPACE_ROOT", "")
with open(sys.argv[1], encoding="utf-8") as f:
    data = json.load(f)
tool = (data.get("toolName") or data.get("tool_name") or "").lower()
block = data.get("toolInput") or data.get("tool_input") or {}
path = block.get("path") or block.get("file_path") or block.get("filePath") or ""
if tool not in ("write", "strreplace", "editnotebook", "delete"):
    raise SystemExit(0)
if not path:
    raise SystemExit(0)
rel = path.replace("\\", "/")
if workspace and rel.startswith(workspace):
    rel = rel[len(workspace) :].lstrip("/")
if rel.startswith(".omg/") and rel.endswith(".md"):
    raise SystemExit(0)
print(f"Prometheus plan mode: only .omg/**/*.md writes allowed; blocked: {path}")
raise SystemExit(2)
PY
}