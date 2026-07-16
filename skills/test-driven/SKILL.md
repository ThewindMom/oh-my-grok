---
name: test-driven
description: >
  Follow test-driven development: write the test first, see it fail, implement
  the minimum to pass, then refactor. Use when adding features or fixing bugs
  with test coverage.
---

# Test-Driven Development

Write tests first. Always.

## Red-Green-Refactor

1. **Red**: Write a test that describes the desired behavior. Run it. It should fail.
2. **Green**: Write the minimum code to make the test pass. Don't over-engineer.
3. **Refactor**: Improve the code structure while keeping tests green.

## Guidelines

- Write one test at a time.
- Test behavior, not implementation.
- Name tests descriptively: `TestX_whenY_shouldZ`.
- Test edge cases: empty, nil, max, negative, Unicode.
- Test error paths, not just happy paths.
- Don't test framework or library code — test your code.

## When fixing bugs

1. Write a test that reproduces the bug.
2. Confirm the test fails.
3. Fix the bug.
4. Confirm the test passes.
5. Run the full suite to check for regressions.

## Anti-patterns

- Don't write tests after implementation (you'll test what you built, not what you should build).
- Don't skip the "see it fail" step (a test that always passes is worthless).
- Don't test implementation details (they change).
