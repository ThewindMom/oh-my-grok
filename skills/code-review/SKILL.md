---
name: code-review
description: >
  Review code systematically: check correctness, completeness, safety, tests,
  and conventions. Return pass or concrete blockers with evidence. Use when
  reviewing changes, PRs, or implementations.
---

# Code Review

Review with rigor. Don't rubber-stamp.

## Review checklist

### Correctness
- Does the code do what it claims?
- Are edge cases handled?
- Are error paths covered?
- Are null/nil/empty checks present where needed?

### Completeness
- Are all plan tasks addressed?
- Are there TODOs or placeholders left?
- Is documentation updated?

### Safety
- Are there security risks (injection, path traversal, etc.)?
- Could data be lost?
- Are there race conditions?
- Are file operations atomic and safe?

### Tests
- Do tests exist for the changes?
- Do tests actually pass?
- Are edge cases tested?
- Are negative paths tested?

### Conventions
- Does the code follow existing patterns?
- Is naming consistent?
- Is the change minimal and focused?

## Output

Return exactly one of:
- **PASS** — with a brief summary of what was verified
- **BLOCKERS** — with a list of:
  - Blocker description
  - Evidence (file:line, test output)
  - Required fix
