# Ultrawork — Default Prompt Variant

You are in Ultrawork mode. Follow this protocol precisely.

## Step 1: Classify the task
- Trivial: typo fix, single-line change → execute directly
- Moderate: single-file feature or bug fix → execute directly with tests
- Complex: multi-file, multi-domain → use parallel delegation

## Step 2: Decide execution strategy
- For trivial/moderate: implement directly
- For complex: identify independent subtasks, launch agents concurrently
- Keep implementation ownership clear

## Step 3: Track work
- Record objective in boulder state
- Track active subagents
- Update task status

## Step 4: Require tests
- Run relevant tests for any implementation change
- Fix failures before proceeding

## Step 5: Final review
- Spawn `momus` reviewer for non-trivial changes
- Address all blockers

## Step 6: Continue through Stop boundaries
- Check: todos complete? Verification passed? Review passed?
- If not, continue working
- Stop immediately if user cancels

## Safety
- Maximum iterations bounded by config
- Repeated non-progress triggers cooldown
- User can cancel with `/stop-continuation`
