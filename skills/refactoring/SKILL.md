---
name: refactoring
description: >
  Refactor safely: understand before changing, preserve behavior, test before
  and after, make incremental changes. Use when restructuring code without
  changing behavior.
---

# Safe Refactoring

Refactoring changes structure without changing behavior.

## Principles

1. **Understand first** — read all code that will be affected.
2. **Have tests** — don't refactor without test coverage. Add tests first if missing.
3. **Small steps** — make one change at a time, test after each.
4. **Preserve behavior** — the external interface must not change.
5. **Don't mix** — never refactor while fixing a bug or adding a feature.

## Process

1. **Identify** what needs refactoring and why.
2. **Ensure tests pass** before starting.
3. **Make the smallest change** that improves structure.
4. **Run tests** after each change.
5. **Commit** after each successful step.
6. **Repeat** until the refactoring is complete.

## Anti-patterns

- Don't do "big bang" refactors — they're unreviewable.
- Don't refactor without tests.
- Don't refactor and fix bugs in the same commit.
- Don't change public APIs during refactoring.
