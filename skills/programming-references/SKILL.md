---
name: programming-references
description: >
  Language-specific programming references for Go, Python, Rust, and TypeScript.
  Covers idioms, error handling, testing, concurrency, and common patterns.
  Use when writing or reviewing code in these languages.
---

# Programming References

Quick reference for common patterns across supported languages.

## Go

### Error handling
```go
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

### Testing
```go
func TestX(t *testing.T) {
    got := X()
    want := expected
    if got != want {
        t.Errorf("X() = %v, want %v", got, want)
    }
}
```

### Concurrency
```go
var wg sync.WaitGroup
for i := 0; i < n; i++ {
    wg.Add(1)
    go func(i int) {
        defer wg.Done()
        // work
    }(i)
}
wg.Wait()
```

### Interface design
- Keep interfaces small (1-3 methods)
- Accept interfaces, return structs
- Define interfaces where they're used, not where they're implemented

## Python

### Error handling
```python
try:
    result = operation()
except SpecificError as e:
    raise RuntimeError(f"operation failed: {e}") from e
```

### Testing
```python
def test_x():
    got = x()
    want = expected
    assert got == want, f"x() = {got}, want {want}"
```

### Type hints
```python
def add(a: int, b: int) -> int:
    return a + b
```

## Rust

### Error handling
```rust
fn operation() -> Result<Output, Error> {
    let value = step()?;
    Ok(value)
}
```

### Testing
```rust
#[test]
fn test_x() {
    let got = x();
    let want = expected;
    assert_eq!(got, want);
}
```

### Ownership patterns
- Use `&T` for read-only borrows
- Use `&mut T` for mutable borrows
- Use `Arc<T>` for shared ownership across threads
- Use `Rc<T>` for shared ownership within a thread

## TypeScript

### Error handling
```typescript
try {
    const result = await operation();
} catch (error) {
    throw new Error(`operation failed: ${error}`);
}
```

### Testing
```typescript
test('x returns expected', () => {
    const got = x();
    const want = expected;
    expect(got).toBe(want);
});
```

### Type patterns
```typescript
interface User {
    id: string;
    name: string;
}

type Result<T> = { ok: true; value: T } | { ok: false; error: string };
```
