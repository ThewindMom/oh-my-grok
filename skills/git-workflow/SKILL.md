---
name: git-workflow
description: >
  Follow a clean git workflow: stage intentionally, write conventional commit
  messages, never commit secrets. Use when committing changes, creating
  branches, or managing git history.
---

# Git Workflow

## Before committing

1. **Review** what will be committed with `git diff` and `git diff --staged`.
2. **Stage** only the relevant files — don't `git add .` blindly.
3. **Check** for debug code, print statements, or temporary files.
4. **Verify** no secrets, API keys, or credentials are in the diff.

## Commit messages

Follow conventional commits:
- `feat:` new feature
- `fix:` bug fix
- `docs:` documentation
- `refactor:` code restructuring
- `test:` adding tests
- `chore:` maintenance

Format:
```
type(scope): concise description

Optional body explaining why, not what.
```

## Branches

- Use descriptive branch names: `feat/add-hashline`, `fix/anchor-parse`.
- Keep branches short-lived.
- Rebase before merging to keep history clean.

## Safety

- Never force-push to shared branches.
- Never commit secrets — if a secret is committed, rotate it immediately.
- Don't commit generated files unless they're meant to be tracked.
- Check `.gitignore` is correct for your project.
