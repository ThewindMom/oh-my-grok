#!/usr/bin/env bash
# One-time cleanup: remove global oh-my-grok copies so only the plugin provides hooks/skills/rules.
set -euo pipefail

ARCHIVE="${HOME}/.grok/archive/removed-global-oh-my-grok-$(date +%Y%m%d)"
mkdir -p "$ARCHIVE"

if [ -d "${HOME}/.grok/hooks" ]; then
  cp -a "${HOME}/.grok/hooks" "$ARCHIVE/hooks"
  rm -rf "${HOME}/.grok/hooks"
  echo "Removed ~/.grok/hooks (archived to $ARCHIVE/hooks)"
fi

for skill in agent-skill-gate ralph-loop ulw-loop cancel-ralph; do
  if [ -d "${HOME}/.grok/skills/${skill}" ]; then
    cp -a "${HOME}/.grok/skills/${skill}" "$ARCHIVE/skills-${skill}"
    rm -rf "${HOME}/.grok/skills/${skill}"
    echo "Removed ~/.grok/skills/${skill}"
  fi
done

for rule in 00-agent-skill-gate.md 10-ralph-loop.md 12-todo-boulder.md; do
  if [ -f "${HOME}/.grok/rules/${rule}" ]; then
    mkdir -p "$ARCHIVE/rules"
    cp -a "${HOME}/.grok/rules/${rule}" "$ARCHIVE/rules/"
    rm -f "${HOME}/.grok/rules/${rule}"
    echo "Removed ~/.grok/rules/${rule}"
  fi
done

if command -v grok >/dev/null 2>&1; then
  grok plugin enable oh-my-grok 2>/dev/null || true
fi

# Sanitize stale plugin IDs in config (old "user/<hash>/name" entries from before
# full plugin system; these can cause reload_plugins to report 0 hooks or skip
# registration of current oh-my-grok hooks manifest).
CONFIG="${HOME}/.grok/config.toml"
if [ -f "$CONFIG" ]; then
  cp -a "$CONFIG" "$ARCHIVE/config.toml.bak" 2>/dev/null || true
  if command -v python3 >/dev/null 2>&1; then
    python3 - "$CONFIG" <<'PY'
import sys, re
cfg = sys.argv[1]
with open(cfg) as f:
    content = f.read()
def clean(m):
    items = re.findall(r'"([^"]+)"', m.group(0))
    keep = [i for i in items if not re.match(r'user/[0-9a-f]+/(oh-my-grok|superpowers)', i)]
    for canon in ('oh-my-grok', 'superpowers'):
        if canon not in keep:
            keep.append(canon)
    return 'enabled = [\n' + ',\n'.join(f'    "{k}"' for k in keep) + '\n]'
newc = re.sub(r'enabled\s*=\s*\[[^\]]*\]', clean, content, flags=re.S)
if newc != content:
    with open(cfg, 'w') as f:
        f.write(newc)
    print("Sanitized stale user/* ids from [plugins] enabled (backup archived)")
PY
  else
    # fallback: crude line delete for known bad patterns
    sed -i '/user\/[0-9a-f]*\/oh-my-grok/d; /user\/[0-9a-f]*\/superpowers/d' "$CONFIG"
    echo "Sanitized (sed fallback) stale user/* ids from $CONFIG"
  fi
fi

echo "Done. Install or refresh: grok plugin install github:mihazs/oh-my-grok --trust"
echo "Reload hooks: Ctrl+L → Hooks (or r in Plugins tab), or start a new session."
echo "Verify: grok plugin list; ls ~/.grok/state/skill-gate/ (recent non-test-* entries mean SessionStart hook ran)"