---
name: systematic-debugging
description: >
  Debug systematically: reproduce, isolate, hypothesize, test, verify. Avoid
  trial-and-error. Use when debugging bugs, investigating failures, or
  diagnosing unexpected behavior.
---

# Systematic Debugging

Never guess. Follow the scientific method.

## Step 1: Reproduce

- Establish a reliable reproduction steps.
- Note the exact environment, inputs, and conditions.
- If you can't reproduce it, you can't fix it.

## Step 2: Isolate

- Narrow the scope: which module, function, or line is involved?
- Use binary search if the scope is unclear.
- Check logs, error messages, and stack traces for clues.

## Step 3: Hypothesize

- Form a specific, testable hypothesis about the root cause.
- Consider competing explanations.
- Rank hypotheses by likelihood and testability.

## Step 4: Test

- Test the hypothesis with the smallest possible change.
- Add a test that would fail if the hypothesis is wrong.
- If the test passes, the hypothesis may be correct.
- If the test fails, revise the hypothesis.

## Step 5: Verify

- Confirm the fix resolves the original issue.
- Confirm no new issues are introduced.
- Run the full test suite for the affected area.
- Clean up any debug code.

## Anti-patterns

- Don't change multiple things at once and hope.
- Don't add print statements without a hypothesis.
- Don't fix symptoms without understanding the cause.
- Don't declare fixed without verification.
