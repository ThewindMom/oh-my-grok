#!/usr/bin/env bash
# Pending-work checks for Stop hook. Sources common.sh first.

# Evaluate pending work; prints block reason to stdout and exits 0 if should block,
# exits 1 if no block needed. Uses stdin JSON file path in STOP_STDIN_FILE.
evaluate_stop_pending_work() {
  local stdin_file="${1:-}"
  local session_id="${GROK_SESSION_ID:-}"
  local grok_home="${GROK_HOME:-}"
  STOP_STDIN_FILE="$stdin_file" GROK_SESSION_ID="$session_id" GROK_HOME="$grok_home" \
    python3 - <<'PY'
import json
import os
import re
import sys
from pathlib import Path

stdin_path = os.environ.get("STOP_STDIN_FILE") or ""
session_id = (os.environ.get("GROK_SESSION_ID") or "").strip()
grok_home = Path(os.environ.get("GROK_HOME") or Path.home() / ".grok")

data = {}
if stdin_path and os.path.isfile(stdin_path):
    try:
        with open(stdin_path, encoding="utf-8") as f:
            data = json.load(f)
    except (OSError, json.JSONDecodeError):
        data = {}


def pick(*keys):
    for k in keys:
        v = data.get(k)
        if isinstance(v, str) and v.strip():
            return v.strip()
    return ""


def allow_exit():
    raise SystemExit(1)


def block_exit(reason: str):
    print(reason.strip())
    raise SystemExit(0)


# User interrupt / explicit stop — never block.
stop_reason = pick("stopReason", "stop_reason", "stop_reason_code")
if stop_reason and stop_reason.lower() not in ("end_turn", "endturn", ""):
    allow_exit()

# Already continuing from a prior Stop block — avoid infinite loops.
if data.get("stop_hook_active") is True or data.get("stopHookActive") is True:
    allow_exit()

# Background work still running — session is paused, not finished.
bg = data.get("background_tasks") or data.get("backgroundTasks") or []
if isinstance(bg, list):
    active = {"running", "pending", "in_progress", "in-progress", "active"}
    for task in bg:
        if not isinstance(task, dict):
            continue
        status = str(task.get("status") or "").lower()
        if status in active:
            allow_exit()

crons = data.get("session_crons") or data.get("sessionCrons") or []
if isinstance(crons, list) and crons:
    allow_exit()

if not session_id:
    allow_exit()

state_dir = grok_home / "state" / "stop-verify" / session_id
state_dir.mkdir(parents=True, exist_ok=True)
blocks_file = state_dir / "blocks.json"
blocks = 0
try:
    with open(blocks_file, encoding="utf-8") as f:
        blocks = int(json.load(f).get("count", 0))
except (OSError, json.JSONDecodeError, TypeError, ValueError):
    blocks = 0

MAX_BLOCKS = 8
if blocks >= MAX_BLOCKS:
    allow_exit()

workspace = pick("workspaceRoot", "workspace_root", "cwd")
if not workspace:
    workspace = os.environ.get("GROK_WORKSPACE_ROOT") or ""

reasons = []


def find_session_dir(sid: str):
    """Resolve ~/.grok/sessions/<encoded-workspace>/<session_id>/ without rglob."""
    root = grok_home / "sessions"
    if not root.is_dir() or not sid:
        return None
    # Fast path: only child directories named exactly like the session id.
    try:
        for workspace_dir in root.iterdir():
            if not workspace_dir.is_dir():
                continue
            candidate = workspace_dir / sid
            if (candidate / "resources_state.json").is_file():
                return candidate
    except OSError:
        return None
    return None


def pending_todos_from_resources(session_dir: Path) -> list[str]:
    resources = session_dir / "resources_state.json"
    if not resources.is_file():
        return []
    try:
        with open(resources, encoding="utf-8") as f:
            state = json.load(f)
    except (OSError, json.JSONDecodeError):
        return []
    pending = []
    for key, entry in (state or {}).items():
        if not isinstance(entry, dict):
            continue
        if "TodoState" not in key and "todo_write" not in key:
            continue
        raw = entry.get("state")
        if not raw:
            continue
        try:
            todo_data = json.loads(raw) if isinstance(raw, str) else raw
        except json.JSONDecodeError:
            continue
        todos = todo_data.get("todos") if isinstance(todo_data, dict) else None
        if not isinstance(todos, list):
            continue
        for t in todos:
            if not isinstance(t, dict):
                continue
            status = str(t.get("status") or "").lower()
            if status in ("pending", "in_progress", "in-progress"):
                content = (t.get("content") or t.get("id") or "todo")[:120]
                pending.append(content)
    return pending


def pending_plan_checkboxes(session_dir: Path, workspace_root: str) -> list[str]:
    candidates = []
    if workspace_root:
        candidates.append(Path(workspace_root) / "plan.md")
    candidates.append(session_dir / "plan.md")
    unchecked = []
    pat = re.compile(r"^\s*-\s*\[\s*\]\s+(.+)$")
    for plan in candidates:
        if not plan.is_file():
            continue
        try:
            text = plan.read_text(encoding="utf-8", errors="replace")
        except OSError:
            continue
        for line in text.splitlines():
            m = pat.match(line)
            if m:
                unchecked.append(m.group(1).strip()[:120])
        if unchecked:
            return unchecked
    return unchecked


session_dir = find_session_dir(session_id)
if session_dir:
    # Todo continuation hook handles TodoWrite lists (omo todo-continuation-enforcer).
    plan_items = pending_plan_checkboxes(session_dir, workspace)
    if plan_items:
        sample = "; ".join(plan_items[:5])
        extra = f" (+{len(plan_items) - 5} more)" if len(plan_items) > 5 else ""
        reasons.append(f"plan.md has {len(plan_items)} unchecked step(s): {sample}{extra}")

if not reasons:
    allow_exit()

blocks += 1
with open(blocks_file, "w", encoding="utf-8") as f:
    json.dump({"count": blocks}, f)
    f.write("\n")

msg = (
    "Stop hook: unfinished work detected. Continue until complete, then summarize. "
    + " | ".join(reasons)
)
if blocks >= MAX_BLOCKS:
    msg += f" (block {blocks}/{MAX_BLOCKS}; this is the final allowed block)"
block_exit(msg)
PY
}