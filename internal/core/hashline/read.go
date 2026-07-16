package hashline

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MaxFileSize is the maximum file size (in bytes) that hashline will process.
const MaxFileSize = 10 * 1024 * 1024 // 10 MB

// MaxOperations is the maximum number of edits in a single hashline_edit call.
const MaxOperations = 200

// NewlineStyle represents the line ending style of a file.
type NewlineStyle string

const (
	NewlineLF   NewlineStyle = "lf"
	NewlineCRLF NewlineStyle = "crlf"
)

// FileIdentity contains metadata for stale detection.
type FileIdentity struct {
	Size       int64  `json:"size"`
	LineCount  int    `json:"lineCount"`
	SHA256     string `json:"sha256"`
	Newline    NewlineStyle `json:"newline"`
	HasFinalNL bool   `json:"hasFinalNewline"`
}

// Line represents a single line with its anchor.
type Line struct {
	Number  int    `json:"number"`
	Hash    string `json:"hash"`
	Content string `json:"content"`
}

// ReadResult is the output of hashline_read.
type ReadResult struct {
	Path           string       `json:"path"`
	CanonicalPath  string       `json:"canonicalPath"`
	Identity       FileIdentity `json:"identity"`
	TotalLines     int          `json:"totalLines"`
	Offset         int          `json:"offset"`
	Limit          int          `json:"limit"`
	Lines          []Line       `json:"lines"`
	Truncated      bool         `json:"truncated"`
}

// ReadFile reads a file and returns its lines with anchors.
// offset is 1-based; limit is the max number of lines (0 = all).
func ReadFile(path string, offset, limit int) (*ReadResult, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory")
	}
	if info.Size() > MaxFileSize {
		return nil, fmt.Errorf("file size %d exceeds limit %d", info.Size(), MaxFileSize)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	if isBinary(data) {
		return nil, fmt.Errorf("binary file not supported")
	}

	identity := computeIdentity(data, info.Size())
	lines := splitLines(data, identity.Newline)

	if offset < 1 {
		offset = 1
	}

	total := len(lines)
	if identity.HasFinalNL {
		// splitLines produces an empty trailing element for final newline;
		// total lines is the count of actual content lines
	}

	startIdx := offset - 1
	endIdx := total
	if limit > 0 && startIdx+limit < endIdx {
		endIdx = startIdx + limit
	}

	truncated := false
	if startIdx >= total {
		return &ReadResult{
			Path:          path,
			CanonicalPath: canonicalize(path),
			Identity:      identity,
			TotalLines:    total,
			Offset:        offset,
			Limit:         limit,
			Lines:         []Line{},
			Truncated:     false,
		}, nil
	}

	if limit > 0 && endIdx-startIdx < total-startIdx {
		truncated = true
	}

	out := make([]Line, 0, endIdx-startIdx)
	for i := startIdx; i < endIdx; i++ {
		content := lines[i]
		out = append(out, Line{
			Number:  i + 1,
			Hash:    ComputeLineHash(i+1, content),
			Content: content,
		})
	}

	return &ReadResult{
		Path:          path,
		CanonicalPath: canonicalize(path),
		Identity:      identity,
		TotalLines:    total,
		Offset:        offset,
		Limit:         limit,
		Lines:         out,
		Truncated:     truncated,
	}, nil
}

// computeIdentity computes file identity metadata from raw content.
func computeIdentity(data []byte, size int64) FileIdentity {
	sum := sha256.Sum256(data)
	newline := NewlineLF
	hasFinalNL := false
	if len(data) > 0 {
		if strings.HasSuffix(string(data), "\n") {
			hasFinalNL = true
		}
		if strings.Contains(string(data), "\r\n") {
			newline = NewlineCRLF
		}
	}

	lines := splitLines(data, newline)
	lineCount := len(lines)
	if hasFinalNL && lineCount > 0 && lines[lineCount-1] == "" {
		lineCount--
	}

	return FileIdentity{
		Size:       size,
		LineCount:  lineCount,
		SHA256:     hex.EncodeToString(sum[:]),
		Newline:    newline,
		HasFinalNL: hasFinalNL,
	}
}

// splitLines splits file content into lines, handling CRLF.
func splitLines(data []byte, newline NewlineStyle) []string {
	s := string(data)
	if newline == NewlineCRLF {
		s = strings.ReplaceAll(s, "\r\n", "\n")
	}
	// Remove a trailing newline so we don't get an empty last element
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return []string{}
	}
	return strings.Split(s, "\n")
}

// isBinary reports whether data appears to be binary.
func isBinary(data []byte) bool {
	for _, b := range data {
		if b == 0 {
			return true
		}
	}
	// Check for high proportion of non-text bytes
	nonText := 0
	for _, b := range data {
		if b < 9 || (b > 13 && b < 32) {
			nonText++
		}
	}
	if len(data) > 0 && nonText*100/len(data) > 30 {
		return true
	}
	return false
}

// canonicalize returns the cleaned absolute path.
func canonicalize(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return filepath.Clean(abs)
}

// ResolvePath resolves a path relative to workspaceRoot and validates containment.
// Returns the absolute path and an error if the path escapes the workspace.
func ResolvePath(raw, workspaceRoot string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("path is empty")
	}

	candidate := raw
	if !filepath.IsAbs(candidate) {
		if workspaceRoot == "" {
			return "", fmt.Errorf("relative path requires workspace root")
		}
		candidate = filepath.Join(workspaceRoot, raw)
	}

	abs, err := filepath.Abs(candidate)
	if err != nil {
		return "", fmt.Errorf("resolve: %w", err)
	}

	// Check for symlink escape
	if workspaceRoot != "" {
		absRoot, _ := filepath.Abs(workspaceRoot)
		absRoot = filepath.Clean(absRoot)
		absClean := filepath.Clean(abs)
		if !strings.HasPrefix(absClean+string(filepath.Separator), absRoot+string(filepath.Separator)) && absClean != absRoot {
			return "", fmt.Errorf("path %s escapes workspace %s", raw, workspaceRoot)
		}
	}

	return abs, nil
}
