---
name: git-master
description: >
  Advanced git workflow: interactive rebasing, cherry-picking, conflict
  resolution, bisecting, stash management, and branch cleanup. Use when
  performing advanced git operations beyond simple commits.
---

# Git Master

Advanced git operations for complex workflows.

## Interactive rebase

Clean up commit history before merging:

```bash
git rebase -i HEAD~N  # Rebase last N commits
```

Common squash operations:
- `pick` — keep the commit
- `squash` — combine with previous commit
- `reword` — change the commit message
- `drop` — remove the commit

## Cherry-picking

Apply a specific commit to the current branch:

```bash
git cherry-pick <commit-hash>
```

## Conflict resolution

1. Identify conflicted files: `git status`
2. Open each file and resolve conflicts (look for `<<<<<<<`, `=======`, `>>>>>>>`)
3. Stage resolved files: `git add <file>`
4. Continue: `git rebase --continue` or `git merge --continue`
5. If you need to abort: `git rebase --abort` or `git merge --abort`

## Bisecting

Find which commit introduced a bug:

```bash
git bisect start
git bisect bad          # Current commit is bad
git bisect good <hash>  # This commit was good
# Git will check out commits. Test each one:
git bisect good         # or git bisect bad
# When done:
git bisect reset
```

## Stash management

```bash
git stash               # Stash current changes
git stash list          # List stashes
git stash pop           # Apply and remove latest stash
git stash apply         # Apply without removing
git stash drop          # Remove a stash
```

## Branch cleanup

```bash
git branch --merged     # List merged branches
git branch -d <name>    # Delete merged branch
git branch -D <name>    # Force delete unmerged branch
git remote prune origin # Remove stale remote tracking branches
```

## Safety

- Never force-push to shared branches (main, master, develop)
- Always create a backup branch before rebasing: `git branch backup`
- Never commit during a rebase — use `git rebase --continue`
- If something goes wrong, `git reflog` shows recent operations
