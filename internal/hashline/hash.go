package hashline

import (
	"fmt"
	"strings"
	"unicode"
)

const nibbleStr = "ZPMQVRWSNKTXJBYH"

var hashlineDict [256]string

func init() {
	for i := 0; i < 256; i++ {
		hashlineDict[i] = string([]byte{nibbleStr[i>>4], nibbleStr[i&0x0F]})
	}
}

const (
	prime32_1 = 0x9E3779B1
	prime32_2 = 0x85EBCA77
	prime32_3 = 0xC2B2AE3D
	prime32_4 = 0x27D4EB2F
	prime32_5 = 0x165667B1
)

func u32(v uint32) uint32 { return v }

func imul(a, b uint32) uint32 {
	return uint32(uint64(a) * uint64(b))
}

func rotl32(v uint32, bits uint) uint32 {
	return (v << bits) | (v >> (32 - bits))
}

func readUint32LE(data []byte, offset int) uint32 {
	var b0, b1, b2, b3 byte
	if offset < len(data) {
		b0 = data[offset]
	}
	if offset+1 < len(data) {
		b1 = data[offset+1]
	}
	if offset+2 < len(data) {
		b2 = data[offset+2]
	}
	if offset+3 < len(data) {
		b3 = data[offset+3]
	}
	return uint32(b0) | uint32(b1)<<8 | uint32(b2)<<16 | uint32(b3)<<24
}

func round32(accumulator, value uint32) uint32 {
	added := accumulator + imul(value, prime32_2)
	return imul(rotl32(added, 13), prime32_1)
}

func xxhash32(data []byte, seed uint32) uint32 {
	offset := 0
	length := len(data)
	seed = u32(seed)

	var hashValue uint32
	if length >= 16 {
		limit := length - 16
		v1 := u32(seed + prime32_1 + prime32_2)
		v2 := u32(seed + prime32_2)
		v3 := seed
		v4 := u32(seed - prime32_1)

		for offset <= limit {
			v1 = round32(v1, readUint32LE(data, offset))
			offset += 4
			v2 = round32(v2, readUint32LE(data, offset))
			offset += 4
			v3 = round32(v3, readUint32LE(data, offset))
			offset += 4
			v4 = round32(v4, readUint32LE(data, offset))
			offset += 4
		}

		hashValue = u32(rotl32(v1, 1) + rotl32(v2, 7))
		hashValue = u32(hashValue + rotl32(v3, 12))
		hashValue = u32(hashValue + rotl32(v4, 18))
	} else {
		hashValue = u32(seed + prime32_5)
	}

	hashValue = u32(hashValue + uint32(length))

	for offset+4 <= length {
		hashValue = u32(hashValue + imul(readUint32LE(data, offset), prime32_3))
		hashValue = imul(rotl32(hashValue, 17), prime32_4)
		offset += 4
	}

	for offset < length {
		hashValue = u32(hashValue + imul(uint32(data[offset]), prime32_5))
		hashValue = imul(rotl32(hashValue, 11), prime32_1)
		offset++
	}

	hashValue = u32(hashValue ^ (hashValue >> 15))
	hashValue = imul(hashValue, prime32_2)
	hashValue = u32(hashValue ^ (hashValue >> 13))
	hashValue = imul(hashValue, prime32_3)
	return u32(hashValue ^ (hashValue >> 16))
}

func hasSignificantChar(text string) bool {
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return true
		}
	}
	return false
}

func computeNormalizedLineHash(lineNumber int, normalizedContent string) string {
	seed := uint32(0)
	if !hasSignificantChar(normalizedContent) {
		seed = uint32(lineNumber)
	}
	digest := xxhash32([]byte(normalizedContent), seed)
	return hashlineDict[digest%256]
}

// ComputeLineHash returns the two-char hash tag for a line (e.g. "ST" for line 1 "  hello  ").
func ComputeLineHash(lineNumber int, content string) string {
	normalized := strings.TrimRight(strings.ReplaceAll(content, "\r", ""), " \t\n\r")
	return computeNormalizedLineHash(lineNumber, normalized)
}

// FormatHashLine returns "N#TAG|content" for display in read output.
func FormatHashLine(lineNumber int, content string) string {
	return fmt.Sprintf("%d#%s|%s", lineNumber, ComputeLineHash(lineNumber, content), content)
}