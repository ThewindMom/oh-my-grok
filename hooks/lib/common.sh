#!/usr/bin/env bash
# Shared helpers for oh-my-grok plugin hooks.

set -euo pipefail

_common_lib_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Session state always under ~/.grok (not the plugin install dir).
resolve_grok_home() {
  local candidate="${GROK_HOME:-}"
  if [ -n "$candidate" ] && [[ "$candidate" != *'${'* ]] && [[ "$candidate" == /* ]]; then
    printf '%s' "$candidate"
    return 0
  fi
  if [ -n "${HOME:-}" ]; then
    printf '%s' "${HOME}/.grok"
    return 0
  fi
  printf '%s' "/tmp/.grok"
}

resolve_plugin_root() {
  local pr="${GROK_PLUGIN_ROOT:-}"
  if [ -n "$pr" ] && [[ "$pr" != *'${'* ]] && [[ "$pr" == /* ]]; then
    printf '%s' "$pr"
    return 0
  fi
  local hooks_dir
  hooks_dir="$(cd "${_common_lib_dir}/.." && pwd)"
  if [ -f "${hooks_dir}/../plugin.json" ]; then
    printf '%s' "$(cd "${hooks_dir}/.." && pwd)"
    return 0
  fi
  return 1
}

GROK_HOME="$(resolve_grok_home)"
PLUGIN_ROOT="$(resolve_plugin_root 2>/dev/null || true)"
export GROK_HOME
[ -n "$PLUGIN_ROOT" ] && export GROK_PLUGIN_ROOT="$PLUGIN_ROOT"

GROK_BIN="${GROK_BIN:-${GROK_HOME}/bin/grok}"
if [[ "${GROK_BIN}" == *'${'* ]] || [[ "${GROK_BIN}" != /* ]]; then
  GROK_BIN="${GROK_HOME}/bin/grok"
fi
export PATH="${GROK_HOME}/bin:${PATH:-/usr/bin:/bin}"

if [ -n "$PLUGIN_ROOT" ] && [ -f "${PLUGIN_ROOT}/skills/agent-skill-gate/SKILL.md" ]; then
  META_SKILL_PATH="${PLUGIN_ROOT}/skills/agent-skill-gate/SKILL.md"
  RULES_PATH="${PLUGIN_ROOT}/rules/00-agent-skill-gate.md"
else
  META_SKILL_PATH="${GROK_HOME}/skills/agent-skill-gate/SKILL.md"
  RULES_PATH="${GROK_HOME}/rules/00-agent-skill-gate.md"
fi

escape_for_json() {
  local s="$1"
  s="${s//\\/\\\\}"
  s="${s//\"/\\\"}"
  s="${s//$'\n'/\\n}"
  s="${s//$'\r'/\\r}"
  s="${s//$'\t'/\\t}"
  printf '%s' "$s"
}

session_state_dir() {
  local session_id="${GROK_SESSION_ID:-unknown}"
  printf '%s/state/skill-gate/%s' "$GROK_HOME" "$session_id"
}

all_skills_file() {
  printf '%s/all-skills.json' "$(session_state_dir)"
}

skills_loaded_file() {
  printf '%s/skills.loaded' "$(session_state_dir)"
}

reset_session_state() {
  local dir
  dir="$(session_state_dir)"
  mkdir -p "$dir"
  : > "$(skills_loaded_file)"
  printf '[]' > "$(all_skills_file)"
}

grok_bin() {
  if [ -x "$GROK_BIN" ]; then
    printf '%s' "$GROK_BIN"
    return 0
  fi
  if command -v grok >/dev/null 2>&1; then
    command -v grok
    return 0
  fi
  return 1
}

run_inspect() {
  local workspace="${GROK_WORKSPACE_ROOT:-${PWD:-}}"
  local grok_cmd
  grok_cmd="$(grok_bin)" || return 1
  if [ -n "$workspace" ] && [ -d "$workspace" ]; then
    (cd "$workspace" && "$grok_cmd" inspect --json 2>/dev/null) || return 1
  else
    "$grok_cmd" inspect --json 2>/dev/null || return 1
  fi
}

# Read hook stdin once; export session/workspace from JSON when env vars are missing.
apply_hook_env_from_stdin() {
  local input_file="${1:-}"
  [ -n "$input_file" ] || return 0
  [ -s "$input_file" ] || return 0
  eval "$(
    python3 - "$input_file" <<'PY'
import json, os, shlex, sys

path = sys.argv[1]
try:
    with open(path, encoding="utf-8") as f:
        data = json.load(f)
except (OSError, json.JSONDecodeError):
    raise SystemExit(0)

def pick(*keys):
    for k in keys:
        v = data.get(k)
        if isinstance(v, str) and v.strip():
            return v.strip()
    return ""

session = pick("sessionId", "session_id")
workspace = pick("workspaceRoot", "workspace_root", "cwd")

out = []
if session and not os.environ.get("GROK_SESSION_ID"):
    out.append(f"export GROK_SESSION_ID={shlex.quote(session)}")
if workspace and not os.environ.get("GROK_WORKSPACE_ROOT"):
    out.append(f"export GROK_WORKSPACE_ROOT={shlex.quote(workspace)}")
print("; ".join(out))
PY
  )"
}

discover_skills_on_disk() {
  local workspace="${GROK_WORKSPACE_ROOT:-${PWD:-}}"
  local out
  out="$(all_skills_file)"
  mkdir -p "$(session_state_dir)"
  WORKSPACE="$workspace" GROK_HOME="$GROK_HOME" GROK_PLUGIN_ROOT="${PLUGIN_ROOT:-}" python3 - "$out" <<'PY'
import json
import os
import re
import sys

workspace = os.environ.get("WORKSPACE") or os.getcwd()
grok_home = os.environ.get("GROK_HOME") or os.path.expanduser("~/.grok")
plugin_root = os.environ.get("GROK_PLUGIN_ROOT") or ""
out_path = sys.argv[1]

def skill_id_from_file(path, parent_name):
    try:
        with open(path, encoding="utf-8") as f:
            head = f.read(800)
        m = re.search(r"^name:\s*([^\n]+)", head, re.MULTILINE)
        if m:
            return m.group(1).strip().strip('"').strip("'")
    except OSError:
        pass
    return parent_name

def add_skill(skills, seen, path, scope):
    path = os.path.realpath(path)
    if path in seen or not os.path.isfile(path):
        return
    parent = os.path.basename(os.path.dirname(path))
    sid = skill_id_from_file(path, parent)
    if not sid:
        return
    seen.add(path)
    desc = ""
    try:
        with open(path, encoding="utf-8") as f:
            head = f.read(1200)
        m = re.search(r"^description:\s*>?\s*\n((?:[ \t]+[^\n]+\n?)+)", head, re.MULTILINE)
        if m:
            desc = " ".join(ln.strip() for ln in m.group(1).splitlines())[:500]
        else:
            m2 = re.search(r"^description:\s*(.+)$", head, re.MULTILINE)
            if m2:
                desc = m2.group(1).strip()[:500]
    except OSError:
        pass
    skills.append({"id": sid, "path": path, "scope": scope, "description": desc})

skills = []
seen = set()

def scan_root(root, scope):
    if not root or not os.path.isdir(root):
        return
    for base in (".agents/skills", ".grok/skills"):
        base_path = os.path.join(root, base)
        if not os.path.isdir(base_path):
            continue
        for name in sorted(os.listdir(base_path)):
            skill_md = os.path.join(base_path, name, "SKILL.md")
            add_skill(skills, seen, skill_md, scope)

scan_root(workspace, "project")
if plugin_root and os.path.isdir(plugin_root):
    scan_root(plugin_root, "plugin")
scan_root(grok_home, "user")

plugins = os.path.join(grok_home, "installed-plugins")
if os.path.isdir(plugins):
    for plugin_dir in sorted(os.listdir(plugins)):
        skills_root = os.path.join(plugins, plugin_dir, "skills")
        if not os.path.isdir(skills_root):
            continue
        for name in sorted(os.listdir(skills_root)):
            skill_md = os.path.join(skills_root, name, "SKILL.md")
            add_skill(skills, seen, skill_md, "plugin")

meta = os.path.join(
    plugin_root, "skills", "agent-skill-gate", "SKILL.md"
) if plugin_root else os.path.join(grok_home, "skills", "agent-skill-gate", "SKILL.md")
add_skill(skills, seen, meta, "plugin" if plugin_root else "user")

ids = {s["id"] for s in skills}
# stable dedupe by id (first wins)
deduped = []
seen_ids = set()
for s in skills:
    if s["id"] in seen_ids:
        continue
    seen_ids.add(s["id"])
    deduped.append(s)

with open(out_path, "w", encoding="utf-8") as f:
    json.dump(deduped, f, indent=2)
    f.write("\n")
print(len(deduped))
PY
}

ensure_skill_catalog() {
  local count
  count="$(catalog_count)"
  if [ "$count" -gt 0 ]; then
    return 0
  fi
  local inspect_json=""
  if inspect_json="$(run_inspect)"; then
    write_all_skills_from_inspect "$inspect_json" >/dev/null || true
    count="$(catalog_count)"
    if [ "$count" -gt 0 ]; then
      return 0
    fi
  fi
  discover_skills_on_disk >/dev/null || true
}

write_all_skills_from_inspect() {
  local inspect_json="$1"
  local out
  out="$(all_skills_file)"
  mkdir -p "$(session_state_dir)"
  META_SKILL_PATH="$META_SKILL_PATH" GROK_PLUGIN_ROOT="${PLUGIN_ROOT:-}" \
    INSPECT_JSON="$inspect_json" python3 - "$out" <<'PY'
import json
import os
import sys

out_path = sys.argv[1]
raw = os.environ.get("INSPECT_JSON", "")
try:
    data = json.loads(raw) if raw.strip() else {}
except json.JSONDecodeError:
    data = {}

skills = []
for entry in data.get("skills") or []:
    if not isinstance(entry, dict):
        continue
    name = entry.get("name") or ""
    desc = entry.get("description") or ""
    src = entry.get("source") or {}
    path = ""
    scope = "unknown"
    if isinstance(src, dict):
        path = src.get("path") or ""
        st = src.get("type") or "unknown"
        scope = st
    if not name or not path:
        continue
    skills.append(
        {
            "id": name,
            "path": path,
            "scope": scope,
            "description": desc[:500],
        }
    )

# Ensure meta-skill is always loadable (plugin path preferred)
meta = os.environ.get("META_SKILL_PATH") or ""
if not meta or not os.path.isfile(meta):
    plugin = os.environ.get("GROK_PLUGIN_ROOT") or ""
    if plugin:
        candidate = os.path.join(plugin, "skills", "agent-skill-gate", "SKILL.md")
        if os.path.isfile(candidate):
            meta = candidate
if not meta or not os.path.isfile(meta):
    plugins_dir = os.path.join(os.path.expanduser("~/.grok"), "installed-plugins")
    if os.path.isdir(plugins_dir):
        for name in sorted(os.listdir(plugins_dir)):
            if not name.startswith("oh-my-grok"):
                continue
            candidate = os.path.join(
                plugins_dir, name, "skills", "agent-skill-gate", "SKILL.md"
            )
            if os.path.isfile(candidate):
                meta = candidate
                break
ids = {s["id"] for s in skills}
if "agent-skill-gate" not in ids and meta and os.path.isfile(meta):
    skills.insert(
        0,
        {
            "id": "agent-skill-gate",
            "path": meta,
            "scope": "user",
            "description": "Skill gate meta-skill; read before mutating tools when unsure which skills apply.",
        },
    )

with open(out_path, "w", encoding="utf-8") as f:
    json.dump(skills, f, indent=2)
    f.write("\n")
print(len(skills))
PY
}

catalog_count() {
  python3 - "$(all_skills_file)" <<'PY'
import json, sys
path = sys.argv[1]
try:
    with open(path, encoding="utf-8") as f:
        data = json.load(f)
    print(len(data) if isinstance(data, list) else 0)
except (OSError, json.JSONDecodeError):
    print(0)
PY
}

loaded_ids() {
  local f
  f="$(skills_loaded_file)"
  [ -f "$f" ] || return 0
  grep -v '^[[:space:]]*$' "$f" 2>/dev/null || true
}

mark_skill_loaded() {
  local skill_id="$1"
  local loaded
  loaded="$(skills_loaded_file)"
  mkdir -p "$(session_state_dir)"
  if [ -f "$loaded" ] && grep -qxF "$skill_id" "$loaded" 2>/dev/null; then
    return 0
  fi
  printf '%s\n' "$skill_id" >>"$loaded"
}

skill_id_for_path() {
  local path="$1"
  python3 - "$path" "$(all_skills_file)" <<'PY'
import json, os, sys

path = os.path.realpath(os.path.expanduser(sys.argv[1]))
catalog_path = sys.argv[2]
try:
    with open(catalog_path, encoding="utf-8") as f:
        catalog = json.load(f)
except (OSError, json.JSONDecodeError):
    catalog = []

for entry in catalog:
    if not isinstance(entry, dict):
        continue
    p = entry.get("path") or ""
    if p and os.path.realpath(os.path.expanduser(p)) == path:
        print(entry.get("id") or "")
        raise SystemExit(0)

# Conventional layouts
for part in (".agents/skills/", ".grok/skills/"):
    if part in path and path.endswith("/SKILL.md"):
        seg = path.split(part, 1)[1]
        print(seg.split("/", 1)[0])
        raise SystemExit(0)

home = os.path.expanduser("~/.grok/skills/")
if path.startswith(os.path.realpath(home)) and path.endswith("/SKILL.md"):
    rel = path[len(os.path.realpath(home)) :]
    print(rel.split("/", 1)[0])
    raise SystemExit(0)

if "/installed-plugins/" in path and "/skills/" in path and path.endswith("/SKILL.md"):
    idx = path.index("/skills/") + len("/skills/")
    print(path[idx:].split("/", 1)[0])
    raise SystemExit(0)
PY
}

path_is_skill_file() {
  local path="$1"
  [[ "$path" == *"/SKILL.md" ]] || [[ "$path" == */SKILL.md ]]
}

list_unloaded_ids() {
  python3 - "$(all_skills_file)" "$(skills_loaded_file)" <<'PY'
import json, sys

def read_ids(path):
    try:
        with open(path, encoding="utf-8") as f:
            lines = [ln.strip() for ln in f if ln.strip()]
        return set(lines)
    except OSError:
        return set()

try:
    with open(sys.argv[1], encoding="utf-8") as f:
        catalog = json.load(f)
except (OSError, json.JSONDecodeError):
    catalog = []

loaded = read_ids(sys.argv[2])
ids = []
for entry in catalog:
    if isinstance(entry, dict):
        sid = entry.get("id") or ""
        if sid and sid not in loaded:
            ids.append(sid)
print("\n".join(ids))
PY
}

build_session_context_message() {
  local max_lines="${1:-20}"
  META_SKILL_PATH="$META_SKILL_PATH" python3 - "$max_lines" "$(all_skills_file)" "$RULES_PATH" <<'PY'
import json, sys

max_lines = int(sys.argv[1])
catalog_path = sys.argv[2]
rules_path = sys.argv[3]

try:
    with open(catalog_path, encoding="utf-8") as f:
        catalog = json.load(f)
except (OSError, json.JSONDecodeError):
    catalog = []

lines = [
    "<AGENT_SKILL_GATE>",
    "Skill gate is active. Before mutating tools (write/edit/delete), Read applicable skills from the catalog.",
    f"Full rules: {rules_path}",
    f"Catalog: {catalog_path} ({len(catalog)} skills)",
    "",
]
if not catalog:
    meta = os.environ.get("META_SKILL_PATH") or "agent-skill-gate (oh-my-grok plugin)"
    lines.append(f"Catalog empty — run `grok inspect` or Read {meta}")
else:
    lines.append("Available skills (use Read on path from inspect):")
    for entry in catalog[:max_lines]:
        if not isinstance(entry, dict):
            continue
        sid = entry.get("id") or "?"
        scope = entry.get("scope") or "?"
        desc = (entry.get("description") or "")[:120].replace("\n", " ")
        lines.append(f"- {sid} ({scope}): {desc}")
    if len(catalog) > max_lines:
        lines.append(f"- ... and {len(catalog) - max_lines} more in {catalog_path}")

lines.append("")
lines.append("Minimum: at least one catalog SKILL.md must be Read this session or mutating tools are blocked.")
lines.append("</AGENT_SKILL_GATE>")
print("\n".join(lines))
PY
}

build_prompt_reminder() {
  python3 - "$(all_skills_file)" "$(skills_loaded_file)" <<'PY'
import json, sys

def read_loaded(path):
    try:
        with open(path, encoding="utf-8") as f:
            return {ln.strip() for ln in f if ln.strip()}
    except OSError:
        return set()

try:
    with open(sys.argv[1], encoding="utf-8") as f:
        catalog = json.load(f)
except (OSError, json.JSONDecodeError):
    catalog = []

loaded = read_loaded(sys.argv[2])
unloaded = []
for entry in catalog:
    if isinstance(entry, dict):
        sid = entry.get("id") or ""
        if sid and sid not in loaded:
            unloaded.append(sid)

if not catalog:
    print("<AGENT_SKILL_GATE_REMINDER>Catalog empty. Read agent-skill-gate meta-skill before edits.</AGENT_SKILL_GATE_REMINDER>")
elif not loaded:
    sample = ", ".join(unloaded[:12])
    extra = f" (+{len(unloaded)-12} more)" if len(unloaded) > 12 else ""
    print(
        f"<AGENT_SKILL_GATE_REMINDER>No skills loaded yet. Read applicable skills before tools. Unloaded: {sample}{extra}</AGENT_SKILL_GATE_REMINDER>"
    )
elif unloaded:
    sample = ", ".join(unloaded[:8])
    print(
        f"<AGENT_SKILL_GATE_REMINDER>Loaded {len(loaded)} skill(s). Still unloaded: {sample}. Read task-matching skills before domain edits.</AGENT_SKILL_GATE_REMINDER>"
    )
else:
    print("<AGENT_SKILL_GATE_REMINDER>All catalog skills loaded this session.</AGENT_SKILL_GATE_REMINDER>")
PY
}

# Merge non-empty context chunks (UserPromptSubmit uses one JSON payload).
emit_user_prompt_context() {
  local merged=""
  local part
  for part in "$@"; do
    part="${part//$'\r'/}"
    [ -n "${part//[[:space:]]/}" ] || continue
    if [ -n "$merged" ]; then
      merged="${merged}

${part}"
    else
      merged="$part"
    fi
  done
  [ -n "$merged" ] || return 0
  emit_additional_context "$merged" "UserPromptSubmit"
}

emit_additional_context() {
  local message="$1"
  local hook_event="${2:-SessionStart}"
  local escaped
  escaped="$(escape_for_json "$message")"
  if [ -n "${CURSOR_PLUGIN_ROOT:-}" ]; then
    printf '{\n  "additional_context": "%s"\n}\n' "$escaped"
  elif [ -n "${CLAUDE_PLUGIN_ROOT:-}" ] && [ -z "${COPILOT_CLI:-}" ]; then
    printf '{\n  "hookSpecificOutput": {\n    "hookEventName": "%s",\n    "additionalContext": "%s"\n  }\n}\n' "$hook_event" "$escaped"
  else
    printf '{\n  "additionalContext": "%s"\n}\n' "$escaped"
  fi
}

emit_deny() {
  local reason="$1"
  local escaped
  escaped="$(escape_for_json "$reason")"
  printf '{"decision":"deny","reason":"%s"}\n' "$escaped"
  exit 2
}

emit_allow() {
  printf '{"decision":"allow"}\n'
  exit 0
}

stop_verify_state_dir() {
  local session_id="${GROK_SESSION_ID:-unknown}"
  printf '%s/state/stop-verify/%s' "$GROK_HOME" "$session_id"
}

emit_stop_allow() {
  printf '{}\n'
  exit 0
}

emit_stop_block() {
  local reason="$1"
  local escaped
  escaped="$(escape_for_json "$reason")"
  printf '{"decision":"block","reason":"%s"}\n' "$escaped"
  exit 0
}

read_hook_stdin_to_file() {
  local dest="$1"
  cat >"$dest"
}

extract_read_path_from_hook_input() {
  local input_file="$1"
  python3 - "$input_file" <<'PY'
import json, sys

try:
    with open(sys.argv[1], encoding="utf-8") as f:
        data = json.load(f)
except (OSError, json.JSONDecodeError):
    raise SystemExit(1)

def dig(obj, *keys):
    for k in keys:
        if isinstance(obj, dict) and k in obj:
            obj = obj[k]
        else:
            return None
    return obj

candidates = []
for key in ("toolInput", "tool_input", "input", "arguments", "rawInput"):
    block = data.get(key)
    if isinstance(block, dict):
        for pk in ("path", "file_path", "filePath", "target_file", "targetFile"):
            v = block.get(pk)
            if isinstance(v, str) and v:
                candidates.append(v)

for pk in ("path", "file_path"):
    v = data.get(pk)
    if isinstance(v, str) and v:
        candidates.append(v)

for c in candidates:
    print(c)
    break
PY
}

cleanup_session_state() {
  local dir
  dir="$(session_state_dir)"
  if [ -d "$dir" ]; then
    rm -rf "$dir"
  fi
}

cleanup_stop_verify_state() {
  local dir
  dir="$(stop_verify_state_dir)"
  if [ -d "$dir" ]; then
    rm -rf "$dir"
  fi
}