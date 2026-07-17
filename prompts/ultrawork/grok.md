# Ultrawork — Grok-Optimized Variant

Ultrawork mode active. Execute with maximum efficiency.

## Protocol
1. Classify: trivial → direct, complex → delegate
2. Fire parallel `spawn_subagent` for independent work
3. Track in boulder state
4. Require tests — fix failures
5. Final review via `momus`
6. Auto-continue through Stop — never ask permission

## Delegation
- explore: codebase search (background)
- hephaestus: implementation (foreground)
- oracle: architecture decisions (foreground)
- momus: final review (foreground)

## Rules
- One level deep — no recursive spawning
- Verify after every delegation: read files, run tests
- Auto-continue until done or cancelled
- No excuses, no partial work, no scope reduction
