#!/usr/bin/env python3
"""Hashline line-ID computation (oh-my-openagent hashline-core compatible)."""
from __future__ import annotations

import sys
import unicodedata

NIBBLE_STR = "ZPMQVRWSNKTXJBYH"
HASHLINE_DICT = [
    f"{NIBBLE_STR[i >> 4]}{NIBBLE_STR[i & 0x0F]}" for i in range(256)
]

PRIME32_1 = 0x9E3779B1
PRIME32_2 = 0x85EBCA77
PRIME32_3 = 0xC2B2AE3D
PRIME32_4 = 0x27D4EB2F
PRIME32_5 = 0x165667B1


def _u32(value: int) -> int:
    return value & 0xFFFFFFFF


def _imul(a: int, b: int) -> int:
    return _u32((_u32(a) * _u32(b)) & 0xFFFFFFFFFFFFFFFF)


def _rotl32(value: int, bits: int) -> int:
    value = _u32(value)
    return _u32((value << bits) | (value >> (32 - bits)))


def _read_uint32_le(data: bytes, offset: int) -> int:
    return _u32(
        (data[offset] if offset < len(data) else 0)
        | ((data[offset + 1] if offset + 1 < len(data) else 0) << 8)
        | ((data[offset + 2] if offset + 2 < len(data) else 0) << 16)
        | ((data[offset + 3] if offset + 3 < len(data) else 0) << 24)
    )


def _round32(accumulator: int, value: int) -> int:
    added = _u32(accumulator + _imul(value, PRIME32_2))
    return _imul(_rotl32(added, 13), PRIME32_1)


def xxhash32(data: bytes, seed: int) -> int:
    """xxHash32 over raw bytes (matches hashline-core xxHash32Js)."""
    offset = 0
    length = len(data)
    seed = _u32(seed)

    if length >= 16:
        limit = length - 16
        value1 = _u32(seed + PRIME32_1 + PRIME32_2)
        value2 = _u32(seed + PRIME32_2)
        value3 = seed
        value4 = _u32(seed - PRIME32_1)

        while offset <= limit:
            value1 = _round32(value1, _read_uint32_le(data, offset))
            offset += 4
            value2 = _round32(value2, _read_uint32_le(data, offset))
            offset += 4
            value3 = _round32(value3, _read_uint32_le(data, offset))
            offset += 4
            value4 = _round32(value4, _read_uint32_le(data, offset))
            offset += 4

        hash_value = _u32(_rotl32(value1, 1) + _rotl32(value2, 7))
        hash_value = _u32(hash_value + _rotl32(value3, 12))
        hash_value = _u32(hash_value + _rotl32(value4, 18))
    else:
        hash_value = _u32(seed + PRIME32_5)

    hash_value = _u32(hash_value + length)

    while offset + 4 <= length:
        hash_value = _u32(hash_value + _imul(_read_uint32_le(data, offset), PRIME32_3))
        hash_value = _imul(_rotl32(hash_value, 17), PRIME32_4)
        offset += 4

    while offset < length:
        hash_value = _u32(hash_value + _imul(data[offset], PRIME32_5))
        hash_value = _imul(_rotl32(hash_value, 11), PRIME32_1)
        offset += 1

    hash_value = _u32(hash_value ^ (hash_value >> 15))
    hash_value = _imul(hash_value, PRIME32_2)
    hash_value = _u32(hash_value ^ (hash_value >> 13))
    hash_value = _imul(hash_value, PRIME32_3)
    return _u32(hash_value ^ (hash_value >> 16))


def _has_significant_char(text: str) -> bool:
    """True when content has a Unicode letter or number (\\p{L}\\p{N})."""
    for ch in text:
        cat = unicodedata.category(ch)
        if cat and (cat[0] == "L" or cat[0] == "N"):
            return True
    return False


def _compute_normalized_line_hash(line_number: int, normalized_content: str) -> str:
    seed = 0 if _has_significant_char(normalized_content) else line_number
    digest = xxhash32(normalized_content.encode("utf-8"), seed)
    return HASHLINE_DICT[digest % 256]


def compute_line_hash(line_number: int, content: str) -> str:
    normalized = content.replace("\r", "").rstrip()
    return _compute_normalized_line_hash(line_number, normalized)


def format_hash_line(line_number: int, content: str) -> str:
    line_hash = compute_line_hash(line_number, content)
    return f"{line_number}#{line_hash}|{content}"


def main(argv: list[str] | None = None) -> int:
    args = list(argv if argv is not None else sys.argv[1:])
    if len(args) < 3 or args[0] != "compute":
        print(
            "usage: hashline.py compute <line_number> <content>",
            file=sys.stderr,
        )
        return 2

    line_number = int(args[1])
    content = args[2]
    line_hash = compute_line_hash(line_number, content)
    print(f"{line_number}#{line_hash}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())