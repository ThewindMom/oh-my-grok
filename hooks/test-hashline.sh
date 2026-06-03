#!/usr/bin/env bash
set -euo pipefail
HOOKS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HASHLINE_PY="${HOOKS_DIR}/lib/hashline.py"
NIBBLE='[ZPMQVRWSNKTXJBYH]'

# Golden from omo hashline-core: computeLineHash(1, "  hello  ") -> ST
result="$(python3 "$HASHLINE_PY" compute 1 "  hello  ")"
test "$result" = "1#ST" || { echo "golden hash mismatch: got $result want 1#ST"; exit 1; }

# Trailing whitespace ignored (trimEnd)
h1="$(python3 "$HASHLINE_PY" compute 1 "function hello() {")"
h2="$(python3 "$HASHLINE_PY" compute 1 "function hello() {  ")"
test "$h1" = "$h2" || { echo "trimEnd mismatch: $h1 vs $h2"; exit 1; }

# Non-significant lines mix line number into seed
p1="$(python3 "$HASHLINE_PY" compute 1 "{}")"
p2="$(python3 "$HASHLINE_PY" compute 2 "{}")"
test "$p1" != "$p2" || { echo "expected different hashes for {} on lines 1 and 2"; exit 1; }

# format_hash_line via python -c
formatted="$(python3 -c "
import sys
sys.path.insert(0, '${HOOKS_DIR}/lib')
from hashline import format_hash_line
print(format_hash_line(42, 'const x = 42'))
")"
echo "$formatted" | rg -q '^42#'"${NIBBLE}"'{2}\|const x = 42$' || {
  echo "format_hash_line mismatch: $formatted"
  exit 1
}

# HASHLINE_DICT length and charset
python3 -c "
import sys
sys.path.insert(0, '${HOOKS_DIR}/lib')
from hashline import HASHLINE_DICT, NIBBLE_STR
assert len(NIBBLE_STR) == 16
assert len(HASHLINE_DICT) == 256
for entry in HASHLINE_DICT:
    assert len(entry) == 2
    assert all(c in NIBBLE_STR for c in entry)
"

echo "hashline: OK"