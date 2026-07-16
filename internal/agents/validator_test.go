package agents

import (
	"os"
	"path/filepath"
	"testing"
)

func writeAgentFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestValidateValidAgents(t *testing.T) {
	dir := t.TempDir()
	writeAgentFile(t, dir, "sisyphus.md", `---
name: sisyphus
description: Coordinator agent
prompt_mode: full
model: inherit
permission_mode: default
agents_md: true
---

You are Sisyphus. Use spawn_subagent to delegate work.
`)
	writeAgentFile(t, dir, "hephaestus.md", `---
name: hephaestus
description: Implementation specialist
prompt_mode: full
model: inherit
permission_mode: default
agents_md: true
tools: ["read_file", "grep"]
---

You are Hephaestus. You cannot spawn subagents.
`)

	result := ValidateDir(dir)
	if result.HasErrors() {
		t.Errorf("expected no errors, got: %v", result.Errors)
	}
	if len(result.Agents) != 2 {
		t.Errorf("agents = %d, want 2", len(result.Agents))
	}
}

func TestValidateDuplicateName(t *testing.T) {
	dir := t.TempDir()
	writeAgentFile(t, dir, "a.md", `---
name: same
description: First
prompt_mode: full
---
body
`)
	writeAgentFile(t, dir, "b.md", `---
name: same
description: Second
prompt_mode: full
---
body
`)

	result := ValidateDir(dir)
	if !result.HasErrors() {
		t.Error("expected error for duplicate name")
	}
}

func TestValidateMissingName(t *testing.T) {
	dir := t.TempDir()
	writeAgentFile(t, dir, "bad.md", `---
description: No name
prompt_mode: full
---
body
`)

	result := ValidateDir(dir)
	if !result.HasErrors() {
		t.Error("expected error for missing name")
	}
}

func TestValidateInvalidPermissionMode(t *testing.T) {
	dir := t.TempDir()
	writeAgentFile(t, dir, "bad.md", `---
name: bad
description: Bad permission
prompt_mode: full
permission_mode: bogus
---
body
`)

	result := ValidateDir(dir)
	if !result.HasErrors() {
		t.Error("expected error for invalid permission_mode")
	}
}

func TestValidateInvalidPromptMode(t *testing.T) {
	dir := t.TempDir()
	writeAgentFile(t, dir, "bad.md", `---
name: bad
description: Bad prompt mode
prompt_mode: bogus
---
body
`)

	result := ValidateDir(dir)
	if !result.HasErrors() {
		t.Error("expected error for invalid prompt_mode")
	}
}

func TestValidateCoordinatorMentionsSpawn(t *testing.T) {
	dir := t.TempDir()
	writeAgentFile(t, dir, "sisyphus.md", `---
name: sisyphus
description: Coordinator
prompt_mode: full
permission_mode: default
---
You coordinate things but don't mention delegation.
`)

	result := ValidateDir(dir)
	found := false
	for _, w := range result.Warnings {
		if contains(w, "should mention spawn_subagent") {
			found = true
		}
	}
	if !found {
		t.Error("expected warning about coordinator not mentioning spawn_subagent")
	}
}

func TestValidateLeafMentionsSpawn(t *testing.T) {
	dir := t.TempDir()
	writeAgentFile(t, dir, "hephaestus.md", `---
name: hephaestus
description: Leaf agent
prompt_mode: full
permission_mode: default
---
You should use spawn_subagent to delegate.
`)

	result := ValidateDir(dir)
	found := false
	for _, w := range result.Warnings {
		if contains(w, "leaf agent body mentions spawn_subagent") {
			found = true
		}
	}
	if !found {
		t.Error("expected warning about leaf agent mentioning spawn_subagent")
	}
}

func TestValidateModelWarning(t *testing.T) {
	dir := t.TempDir()
	writeAgentFile(t, dir, "custom.md", `---
name: custom
description: Custom model
prompt_mode: full
model: grok-build
---
body
`)

	result := ValidateDir(dir)
	found := false
	for _, w := range result.Warnings {
		if contains(w, "model") && contains(w, "catalog") {
			found = true
		}
	}
	if !found {
		t.Error("expected warning about non-inherit model")
	}
}

func TestParseAgentFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	os.WriteFile(path, []byte(`---
name: test
description: Test agent
prompt_mode: full
model: inherit
permission_mode: plan
agents_md: true
tools: ["read_file", "grep", "list_dir"]
---

Body text here.
`), 0o644)

	def, err := ParseAgentFile(path)
	if err != nil {
		t.Fatalf("ParseAgentFile: %v", err)
	}
	if def.Name != "test" {
		t.Errorf("name = %q", def.Name)
	}
	if def.PromptMode != "full" {
		t.Errorf("prompt_mode = %q", def.PromptMode)
	}
	if def.PermissionMode != "plan" {
		t.Errorf("permission_mode = %q", def.PermissionMode)
	}
	if len(def.Tools) != 3 {
		t.Errorf("tools = %d, want 3", len(def.Tools))
	}
	if def.Tools[0] != "read_file" {
		t.Errorf("tools[0] = %q", def.Tools[0])
	}
}

func TestValidateRealAgents(t *testing.T) {
	// Validate the actual agents directory
	result := ValidateDir("../../agents")
	if result.HasErrors() {
		t.Errorf("real agents have validation errors:\n%s", result.Report())
	}
	if len(result.Agents) < 9 {
		t.Errorf("expected at least 9 agents, got %d", len(result.Agents))
	}
	// Check we have the required agents
	required := []string{"sisyphus", "hephaestus", "prometheus", "metis", "momus", "oracle", "librarian", "explore", "atlas"}
	names := map[string]bool{}
	for _, a := range result.Agents {
		names[a.Name] = true
	}
	for _, req := range required {
		if !names[req] {
			t.Errorf("missing required agent: %s", req)
		}
	}
	// Check leaf agents
	for _, a := range result.Agents {
		if !a.IsCoordinator {
			// Leaf agents should not be coordinators
			if CoordinatorAgents[a.Name] {
				t.Errorf("%s should be a leaf agent but is marked coordinator", a.Name)
			}
		}
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
