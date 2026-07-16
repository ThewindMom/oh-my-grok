---
name: disciplined-implementation
description: >
  Follow a disciplined implementation process: understand before editing, make
  minimal changes, test after each change, verify before declaring done. Use
  when implementing features, fixing bugs, or making changes to source code.
---

# Disciplined Implementation

Follow this process for every implementation task.

## Before editing

1. **Read** the relevant source files completely.
2. **Understand** the existing patterns, conventions, and architecture.
3. **Identify** the minimal set of changes needed.
4. **Check** for existing tests that cover the area you're changing.

## During editing

1. **Make minimal changes** — don't refactor unrelated code.
2. **Use hashline tools** when hashline mode is strict for precise, conflict-free edits.
3. **One logical change at a time** — don't mix features, fixes, and refactors.
4. **Keep commits atomic** — each commit should be independently meaningful.

## After editing

1. **Run focused tests** for the changed package.
2. **Run broader tests** if the change affects shared code.
3. **Verify** the change does what was intended.
4. **Check** for regressions in related areas.

## Completion checklist

- [ ] All tests pass
- [ ] No new warnings or errors
- [ ] Code follows existing conventions
- [ ] Change is minimal and focused
- [ ] No debug code left behind
