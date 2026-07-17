package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverRules(t *testing.T) {
	ws := t.TempDir()
	os.WriteFile(filepath.Join(ws, "AGENTS.md"), []byte("# Root rules\n"), 0o644)

	sub := filepath.Join(ws, "sub")
	os.MkdirAll(sub, 0o755)
	os.WriteFile(filepath.Join(sub, "AGENTS.md"), []byte("# Sub rules\n"), 0o644)

	files, err := DiscoverRules(ws, sub)
	if err != nil {
		t.Fatalf("DiscoverRules: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("files = %d, want 2", len(files))
	}
	// Nearest should be first
	if !strings.Contains(files[0].Content, "Sub rules") {
		t.Error("nearest file should be first")
	}
}

func TestDiscoverRulesNearestFirst(t *testing.T) {
	ws := t.TempDir()
	os.WriteFile(filepath.Join(ws, "AGENTS.md"), []byte("# Root\n"), 0o644)

	a := filepath.Join(ws, "a")
	os.MkdirAll(a, 0o755)
	os.WriteFile(filepath.Join(a, "AGENTS.md"), []byte("# A\n"), 0o644)

	b := filepath.Join(a, "b")
	os.MkdirAll(b, 0o755)
	os.WriteFile(filepath.Join(b, "AGENTS.md"), []byte("# B\n"), 0o644)

	files, err := DiscoverRules(ws, b)
	if err != nil {
		t.Fatalf("DiscoverRules: %v", err)
	}
	if len(files) != 3 {
		t.Errorf("files = %d, want 3", len(files))
	}
	// B (nearest) should be first, Root (furthest) should be last
	if !strings.Contains(files[0].Content, "# B") {
		t.Error("nearest (B) should be first")
	}
	if !strings.Contains(files[2].Content, "# Root") {
		t.Error("furthest (Root) should be last")
	}
}

func TestDiscoverRulesNoFiles(t *testing.T) {
	ws := t.TempDir()
	files, err := DiscoverRules(ws, ws)
	if err != nil {
		t.Fatalf("DiscoverRules: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("files = %d, want 0", len(files))
	}
}

func TestDiscoverRulesPathEscape(t *testing.T) {
	ws := t.TempDir()
	_, err := DiscoverRules(ws, "/etc")
	if err == nil {
		t.Error("expected error for path escape")
	}
}

func TestFormatRules(t *testing.T) {
	files := []RuleFile{
		{RelPath: "AGENTS.md", Content: "# Root rules\n", Bytes: 13, Depth: 1},
		{RelPath: "sub/AGENTS.md", Content: "# Sub rules\n", Bytes: 12, Depth: 0},
	}
	result := FormatRules(files)
	if !strings.Contains(result, "Root rules") {
		t.Error("should contain root rules")
	}
	if !strings.Contains(result, "Sub rules") {
		t.Error("should contain sub rules")
	}
}

func TestFormatRulesEmpty(t *testing.T) {
	result := FormatRules(nil)
	if result != "" {
		t.Errorf("empty should return empty string, got %s", result)
	}
}

func TestFormatRulesDedup(t *testing.T) {
	files := []RuleFile{
		{RelPath: "AGENTS.md", Content: "same content\n", Bytes: 13, Depth: 0},
		{RelPath: "sub/AGENTS.md", Content: "same content\n", Bytes: 13, Depth: 1},
	}
	result := FormatRules(files)
	// Should only appear once
	count := strings.Count(result, "same content")
	if count != 1 {
		t.Errorf("dedup count = %d, want 1", count)
	}
}

func TestTruncateContent(t *testing.T) {
	content := "line1\nline2\nline3\nline4\n"
	truncated := truncateContent(content, 12)
	if !strings.Contains(truncated, "truncated") {
		t.Error("should contain truncation marker")
	}
}

func TestTruncateContentShort(t *testing.T) {
	content := "short"
	truncated := truncateContent(content, 100)
	if truncated != "short" {
		t.Errorf("short content should not be truncated: %s", truncated)
	}
}

func TestInjectRules(t *testing.T) {
	ws := t.TempDir()
	os.WriteFile(filepath.Join(ws, "AGENTS.md"), []byte("# Rules\nUse Go.\n"), 0o644)

	result, err := InjectRules(ws, ws)
	if err != nil {
		t.Fatalf("InjectRules: %v", err)
	}
	if !strings.Contains(result, "Use Go") {
		t.Error("should contain rules content")
	}
}

func TestDiscoverRulesGrokdir(t *testing.T) {
	ws := t.TempDir()
	os.MkdirAll(filepath.Join(ws, ".grok"), 0o755)
	os.WriteFile(filepath.Join(ws, ".grok", "AGENTS.md"), []byte("# Grok rules\n"), 0o644)

	files, err := DiscoverRules(ws, ws)
	if err != nil {
		t.Fatalf("DiscoverRules: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("files = %d, want 1", len(files))
	}
	if !strings.Contains(files[0].Content, "Grok rules") {
		t.Error("should contain .grok/AGENTS.md content")
	}
}

func TestFormatRulesByteLimit(t *testing.T) {
	// Create many files that would exceed the limit
	var files []RuleFile
	for i := 0; i < 100; i++ {
		files = append(files, RuleFile{
			RelPath: filepath.Join("dir", "AGENTS.md"),
			Content: strings.Repeat("x", 1000) + "\n",
			Bytes:   1001,
			Depth:    i,
		})
	}
	result := FormatRules(files)
	if len(result) > MaxTotalBytes+1000 { // allow some overhead for headers
		t.Errorf("result too large: %d bytes (limit %d)", len(result), MaxTotalBytes)
	}
}
