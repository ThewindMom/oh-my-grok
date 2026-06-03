# Grok hooks layout

Plugin manifest: **`hooks/hooks.json`** (loaded via `GROK_PLUGIN_ROOT`). **Do not** add parallel `~/.grok/hooks/*.json` for this stack — use `grok plugin install github:mihazs/oh-my-grok --trust` (or `$(pwd)` from a local clone).

## Event map

| Event | Script | Role |
|-------|--------|------|
| `SessionStart` | `session-start.sh` | Skill catalog + skill-gate rules |
| `UserPromptSubmit` | **`user-prompt.sh`** | **One** merged `additionalContext` (see below) |
| `PreToolUse` | `pre-tool-mutate.sh` | Block writes until a skill is Read |
| `PostToolUse` (Read) | `post-tool-read.sh` | Skill gate + hashline read cache |
| `PostToolUse` (TodoWrite) | `post-tool-todo-write.sh` | Mirror todos → `.omg/todos/<session>.json` |
| `PostToolUse` (Write\|StrReplace) | `post-tool-lsp.sh` | LSP diagnostics → `~/.grok/state/lsp-diagnostics/<session>.json` |
| `Stop` | `stop-hook.sh` | Continuation chain (`lib/stop-chain.sh`) |
| `SessionEnd` | `session-end.sh` | Reset session state |

## UserPromptSubmit (merged)

**`user-prompt.sh`** collects and emits a single JSON payload:

1. `using-superpowers` (first prompt only)
2. Workspace `AGENTS.md` + plugin `rules/*.md` (every prompt; size-capped)
3. Ralph / ultrawork (`lib/ralph-loop.sh`)
4. **IntentGate** (`lib/intent-gate.sh`) — search / analyze / team / hyperplan banners (`OMG_INTENT_GATE`)
5. **Prometheus** (`lib/prometheus.sh`) — `/plan`, `/start-work`, plan-mode state
6. `/handoff`, `/stop-continuation`, `/resume-continuation`
7. Boulder context (`.omg/boulder.json`)
8. **LSP** (`lib/lsp.sh`) — `<LSP_DIAGNOSTICS>` from session stash
9. **Hashline** (`lib/hashline.sh`) — `<HASHLINE_CACHE>` for recently read files
10. Skill-gate reminder

## Stop (priority chain)

`lib/stop-chain.sh` — **first block wins**:

1. **Ralph / ultrawork** — not affected by `/stop-continuation` (but `/stop-continuation` clears loop state)
2. **Boulder** — `.omg/plans/*.md` progress
3. **Todo continuation** — incomplete `TodoWrite` items (**todo enforcer**: 5s cooldown, 3s abort window on non-`end_turn` stops; state in `~/.grok/state/todo-enforcer/<session>/state.json`)
4. **LSP** — error diagnostics in stash (`lib/lsp.sh`; skip when `OMG_LSP_ENFORCE=0`)
5. **plan.md** — root/session unchecked boxes (fallback)

Grok fires **`Stop`** (not Claude Code’s `session.idle`).

After `/stop-continuation`, steps 2–5 are skipped until `/resume-continuation` or `SessionEnd`.

**PreToolUse** (`pre-tool-mutate.sh`): prometheus plan-mode deny → hashline stale `LINE#ID` deny → skill gate.

## Workspace state (`.omg/`)

| Path | Purpose |
|------|---------|
| `.omg/boulder.json` | Active plan work (omo-compatible schema) |
| `.omg/plans/*.md` | Prometheus-style plans |
| `.omg/todos/<session>.json` | Todo mirror |
| `.omg/run-continuation/<session>.json` | Pause marker (with `~/.grok/state/stop-continuation/`) |
| `.omg/ralph-loop.local.md` | Ralph / ultrawork loop |
| `.omg/handoffs/*.md` | Saved handoff summaries |

Session hook state under **`~/.grok/state/`**: skill-gate, stop-continuation, **hashline** (`state/hashline/<session>/`), **lsp-diagnostics** (`state/lsp-diagnostics/<session>.json`), **todo-enforcer**.

Bundled MCP (optional): `.mcp.json` — `ast_grep`, `lsp` under `vendor/` (see `skills/ast-grep`, `skills/lsp`).

## Plugin overlap

**superpowers** also registers `SessionStart`. Both may run; expect skill-gate + superpowers bootstrap on startup.

## Tests

From repo root with `GROK_PLUGIN_ROOT` set (see main README):

```bash
export GROK_PLUGIN_ROOT="$(pwd)"
bash hooks/test-stop-verify.sh
bash hooks/test-ralph-loop.sh
bash hooks/test-ulw-loop.sh
bash hooks/test-todo-boulder.sh
bash hooks/test-using-superpowers-first-prompt.sh
bash hooks/test-handoff.sh
bash hooks/test-intent-gate.sh
bash hooks/test-prometheus.sh
bash hooks/test-hashline.sh
bash hooks/test-lsp.sh
```

`OMG_*` toggles: [docs/configuration.md](../docs/configuration.md).