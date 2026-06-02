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

echo "Done. Install or refresh: grok plugin install github:mihazs/oh-my-grok --trust"
echo "Reload hooks: Ctrl+L → Hooks, or start a new session."