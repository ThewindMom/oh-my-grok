package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeRuleFile(t *testing.T, path, content string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0o755)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestEngineAlwaysApply(t *testing.T) {
	ws := t.TempDir()
	writeRuleFile(t, filepath.Join(ws, "AGENTS.md"), "# Root rules\nAlways apply these.\n")

	engine := NewEngine()
	result := engine.DiscoverAndLoad(ws, ws)
	if len(result.Rules) != 1 {
		t.Fatalf("rules = %d, want 1", len(result.Rules))
	}
	if result.Rules[0].MatchReason != "alwaysApply" {
		t.Errorf("matchReason = %s, want alwaysApply", result.Rules[0].MatchReason)
	}
}

func TestEngineGlobMatch(t *testing.T) {
	ws := t.TempDir()
	writeRuleFile(t, filepath.Join(ws, ".grok", "rules", "go.md"), `---
description: Go rules
globs: ["**/*.go"]
---

Use gofmt.
`)
	sub := filepath.Join(ws, "src")
	os.MkdirAll(sub, 0o755)

	engine := NewEngine()
	result := engine.DiscoverAndLoad(ws, filepath.Join(sub, "main.go"))
	if len(result.Rules) != 1 {
		t.Fatalf("rules = %d, want 1", len(result.Rules))
	}
	if !strings.Contains(result.Rules[0].MatchReason, "glob") {
		t.Errorf("matchReason = %s, want glob:...", result.Rules[0].MatchReason)
	}
}

func TestEngineGlobNoMatch(t *testing.T) {
	ws := t.TempDir()
	writeRuleFile(t, filepath.Join(ws, ".grok", "rules", "go.md"), `---
description: Go rules
globs: ["**/*.go"]
---

Use gofmt.
`)

	engine := NewEngine()
	result := engine.DiscoverAndLoad(ws, filepath.Join(ws, "main.py"))
	if len(result.Rules) != 0 {
		t.Errorf("rules = %d, want 0 (no glob match)", len(result.Rules))
	}
}

func TestEngineNearestFirst(t *testing.T) {
	ws := t.TempDir()
	writeRuleFile(t, filepath.Join(ws, "AGENTS.md"), "# Root\n")
	sub := filepath.Join(ws, "sub")
	os.MkdirAll(sub, 0o755)
	writeRuleFile(t, filepath.Join(sub, "AGENTS.md"), "# Sub\n")

	engine := NewEngine()
	result := engine.DiscoverAndLoad(ws, sub)
	if len(result.Rules) != 2 {
		t.Fatalf("rules = %d, want 2", len(result.Rules))
	}
	if result.Rules[0].Distance != 0 {
		t.Error("nearest should be first")
	}
}

func TestEngineDedup(t *testing.T) {
	ws := t.TempDir()
	writeRuleFile(t, filepath.Join(ws, "AGENTS.md"), "same content\n")
	sub := filepath.Join(ws, "sub")
	os.MkdirAll(sub, 0o755)
	writeRuleFile(t, filepath.Join(sub, "AGENTS.md"), "same content\n")

	engine := NewEngine()
	result := engine.DiscoverAndLoad(ws, sub)
	if len(result.Rules) != 1 {
		t.Errorf("rules = %d, want 1 (deduped)", len(result.Rules))
	}
}

func TestEngineFormat(t *testing.T) {
	ws := t.TempDir()
	writeRuleFile(t, filepath.Join(ws, "AGENTS.md"), "# Rules\nUse Go.\n")

	engine := NewEngine()
	result := engine.DiscoverAndLoad(ws, ws)
	formatted := engine.Format(result.Rules)
	if !strings.Contains(formatted, "Use Go") {
		t.Error("format should contain rule content")
	}
	if !strings.Contains(formatted, "===") {
		t.Error("format should contain section header")
	}
}

func TestEngineFormatEmpty(t *testing.T) {
	engine := NewEngine()
	formatted := engine.Format(nil)
	if formatted != "" {
		t.Errorf("empty should return empty string, got %s", formatted)
	}
}

func TestEnginePathEscape(t *testing.T) {
	ws := t.TempDir()
	engine := NewEngine()
	result := engine.DiscoverAndLoad(ws, "/etc")
	if len(result.Diagnostics) == 0 {
		t.Error("expected diagnostic for path escape")
	}
}

func TestEngineMultipleSources(t *testing.T) {
	ws := t.TempDir()
	writeRuleFile(t, filepath.Join(ws, "AGENTS.md"), "# AGENTS\n")
	writeRuleFile(t, filepath.Join(ws, ".grok", "rules", "extra.md"), `---
description: Extra
alwaysApply: true
---

Extra rules.
`)
	writeRuleFile(t, filepath.Join(ws, ".agents", "rules", "more.md"), `---
description: More
alwaysApply: true
---

More rules.
`)

	engine := NewEngine()
	result := engine.DiscoverAndLoad(ws, ws)
	if len(result.Rules) < 3 {
		t.Errorf("rules = %d, want >= 3", len(result.Rules))
	}
}

func TestEngineTruncation(t *testing.T) {
	ws := t.TempDir()
	longContent := "# Rules\n" + strings.Repeat("line of content\n", 1000)
	writeRuleFile(t, filepath.Join(ws, "AGENTS.md"), longContent)

	engine := NewEngine()
	engine.maxRuleChars = 100
	result := engine.DiscoverAndLoad(ws, ws)
	if len(result.Rules) != 1 {
		t.Fatalf("rules = %d, want 1", len(result.Rules))
	}
	if !strings.Contains(result.Rules[0].Body, "truncated") {
		t.Error("body should be truncated")
	}
}

func TestEngineLoadAndFormat(t *testing.T) {
	ws := t.TempDir()
	writeRuleFile(t, filepath.Join(ws, "AGENTS.md"), "# Rules\nUse Go.\n")

	engine := NewEngine()
	formatted, diags := engine.LoadAndFormat(ws, ws)
	if len(diags) != 0 {
		t.Errorf("diagnostics = %d, want 0", len(diags))
	}
	if !strings.Contains(formatted, "Use Go") {
		t.Error("should contain rule content")
	}
}

func TestParseFrontmatter(t *testing.T) {
	fm := parseFrontmatter(`description: Test rule
globs: ["**/*.go", "**/*.ts"]
alwaysApply: false`)
	if fm.Description != "Test rule" {
		t.Errorf("description = %s", fm.Description)
	}
	if len(fm.Globs) != 2 {
		t.Errorf("globs = %d, want 2", len(fm.Globs))
	}
	if fm.AlwaysApply {
		t.Error("alwaysApply should be false")
	}
}

func TestParseFrontmatterAlwaysApply(t *testing.T) {
	fm := parseFrontmatter(`description: Always
alwaysApply: true`)
	if !fm.AlwaysApply {
		t.Error("alwaysApply should be true")
	}
}

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"**/*.go", "src/main.go", true},
		{"**/*.go", "src/sub/main.go", true},
		{"**/*.go", "src/main.py", false},
		{"*.go", "main.go", true},
		{"*.go", "src/main.go", false},
		{"**/*", "anything/here", true},
	}

	for _, tt := range tests {
		got := matchGlob(tt.pattern, tt.path)
		if got != tt.want {
			t.Errorf("matchGlob(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
		}
	}
}

func TestEngineNoFrontmatter(t *testing.T) {
	ws := t.TempDir()
	writeRuleFile(t, filepath.Join(ws, "AGENTS.md"), "Just plain content without frontmatter.\n")

	engine := NewEngine()
	result := engine.DiscoverAndLoad(ws, ws)
	if len(result.Rules) != 1 {
		t.Fatalf("rules = %d, want 1", len(result.Rules))
	}
	if result.Rules[0].MatchReason != "alwaysApply" {
		t.Errorf("matchReason = %s, want alwaysApply", result.Rules[0].MatchReason)
	}
}

func TestEngineClaudeRules(t *testing.T) {
	ws := t.TempDir()
	writeRuleFile(t, filepath.Join(ws, ".claude", "rules", "custom.md"), `---
description: Claude rule
alwaysApply: true
---

Claude-specific rules.
`)

	engine := NewEngine()
	result := engine.DiscoverAndLoad(ws, ws)
	if len(result.Rules) != 1 {
		t.Fatalf("rules = %d, want 1", len(result.Rules))
	}
	if result.Rules[0].Source != SourceClaudeRules {
		t.Errorf("source = %s, want .claude/rules", result.Rules[0].Source)
	}
}

func TestEngineCopilotInstructions(t *testing.T) {
	ws := t.TempDir()
	writeRuleFile(t, filepath.Join(ws, ".github", "copilot-instructions.md"), "Copilot rules.\n")

	engine := NewEngine()
	result := engine.DiscoverAndLoad(ws, ws)
	if len(result.Rules) != 1 {
		t.Fatalf("rules = %d, want 1", len(result.Rules))
	}
	if result.Rules[0].Source != SourceCopilotInstr {
		t.Errorf("source = %s, want copilot-instructions", result.Rules[0].Source)
	}
}
