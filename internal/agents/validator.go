// Package agents validates plugin agent definitions for correctness.
package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// AgentDef represents a parsed agent definition.
type AgentDef struct {
	Name           string
	Description    string
	PromptMode     string
	PermissionMode string
	Model          string
	AgentsMD       bool
	Tools          []string
	Body           string
	Path           string
	// Derived
	IsCoordinator bool
}

// ValidationResult holds the results of validation.
type ValidationResult struct {
	Errors   []string
	Warnings []string
	Agents   []AgentDef
}

// ValidPermissionModes is the set of known permission modes.
var ValidPermissionModes = map[string]bool{
	"default": true,
	"plan":    true,
}

// ValidPromptModes is the set of known prompt modes.
var ValidPromptModes = map[string]bool{
	"full": true,
}

// CoordinatorAgents can spawn subagents.
var CoordinatorAgents = map[string]bool{
	"sisyphus": true,
	"atlas":    true,
}

// ParseAgentFile parses an agent .md file and extracts frontmatter + body.
func ParseAgentFile(path string) (AgentDef, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return AgentDef{}, err
	}
	content := string(data)

	// Extract frontmatter
	fmRegex := regexp.MustCompile(`(?s)^---\n(.*?)\n---\n(.*)$`)
	matches := fmRegex.FindStringSubmatch(content)
	if matches == nil {
		return AgentDef{}, fmt.Errorf("no frontmatter found in %s", path)
	}
	fm := matches[1]
	body := matches[2]

	def := AgentDef{
		Path: path,
		Body: body,
	}

	lines := strings.Split(fm, "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		idx := strings.Index(trimmed, ":")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(trimmed[:idx])
		val := strings.TrimSpace(trimmed[idx+1:])

		// Handle YAML block scalars (> or |)
		if val == ">" || val == "|" {
			// Collect indented continuation lines
			var blockLines []string
			for j := i + 1; j < len(lines); j++ {
				nextLine := lines[j]
				if strings.TrimSpace(nextLine) == "" {
					blockLines = append(blockLines, "")
					continue
				}
				// Check if line is indented (part of block)
				if len(nextLine) > 0 && (nextLine[0] == ' ' || nextLine[0] == '\t') {
					blockLines = append(blockLines, strings.TrimSpace(nextLine))
				} else {
					i = j - 1
					break
				}
				if j == len(lines)-1 {
					i = j
				}
			}
			val = strings.Join(blockLines, " ")
			val = strings.TrimSpace(val)
		}

		val = strings.Trim(val, "\"'")

		switch key {
		case "name":
			def.Name = val
		case "description":
			def.Description = val
		case "prompt_mode":
			def.PromptMode = val
		case "permission_mode":
			def.PermissionMode = val
		case "model":
			def.Model = val
		case "agents_md":
			def.AgentsMD = val == "true"
		case "tools":
			// Parse YAML list: ["a", "b"] or ["a","b"]
			toolList := strings.Trim(val, "[]")
			for _, t := range strings.Split(toolList, ",") {
				t = strings.TrimSpace(t)
				t = strings.Trim(t, "\"'")
				if t != "" {
					def.Tools = append(def.Tools, t)
				}
			}
		}
	}

	def.IsCoordinator = CoordinatorAgents[def.Name]
	return def, nil
}

// ValidateDir validates all agent .md files in a directory.
func ValidateDir(agentsDir string) ValidationResult {
	result := ValidationResult{}
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("cannot read agents dir %s: %v", agentsDir, err))
		return result
	}

	names := map[string]bool{}
	var defs []AgentDef

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join(agentsDir, entry.Name())
		def, err := ParseAgentFile(path)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", entry.Name(), err))
			continue
		}
		defs = append(defs, def)
		result.Agents = append(result.Agents, def)
	}

	// Sort for deterministic output
	sort.Slice(defs, func(i, j int) bool { return defs[i].Name < defs[j].Name })

	for _, def := range defs {
		// Check required fields
		if def.Name == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: missing name", filepath.Base(def.Path)))
		}
		if def.Description == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: missing description", def.Name))
		}

		// Check valid values
		if def.PromptMode != "" && !ValidPromptModes[def.PromptMode] {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: invalid prompt_mode %q", def.Name, def.PromptMode))
		}
		if def.PermissionMode != "" && !ValidPermissionModes[def.PermissionMode] {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: invalid permission_mode %q", def.Name, def.PermissionMode))
		}

		// Check unique names
		if names[def.Name] {
			result.Errors = append(result.Errors, fmt.Sprintf("duplicate agent name: %s", def.Name))
		}
		names[def.Name] = true

		// Check model
		if def.Model != "" && def.Model != "inherit" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: model %q is not 'inherit'; ensure it exists in user's Grok catalog", def.Name, def.Model))
		}

		// Check leaf agents don't have coordinator capabilities
		if !def.IsCoordinator {
			bodyLower := strings.ToLower(def.Body)
			if strings.Contains(bodyLower, "spawn_subagent") && !strings.Contains(bodyLower, "cannot spawn") {
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s: leaf agent body mentions spawn_subagent — verify it instructs against spawning", def.Name))
			}
		}

		// Check coordinators mention delegation
		if def.IsCoordinator {
			if !strings.Contains(strings.ToLower(def.Body), "spawn_subagent") {
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s: coordinator agent should mention spawn_subagent in its instructions", def.Name))
			}
		}
	}

	return result
}

// HasErrors reports whether validation found any errors.
func (v ValidationResult) HasErrors() bool {
	return len(v.Errors) > 0
}

// Report returns a human-readable validation report.
func (v ValidationResult) Report() string {
	var sb strings.Builder
	if len(v.Agents) > 0 {
		sb.WriteString(fmt.Sprintf("Agents found: %d\n", len(v.Agents)))
		for _, a := range v.Agents {
			role := "leaf"
			if a.IsCoordinator {
				role = "coordinator"
			}
			sb.WriteString(fmt.Sprintf("  - %s (%s)\n", a.Name, role))
		}
	}
	if len(v.Errors) > 0 {
		sb.WriteString(fmt.Sprintf("\nErrors: %d\n", len(v.Errors)))
		for _, e := range v.Errors {
			sb.WriteString(fmt.Sprintf("  ERROR: %s\n", e))
		}
	}
	if len(v.Warnings) > 0 {
		sb.WriteString(fmt.Sprintf("\nWarnings: %d\n", len(v.Warnings)))
		for _, w := range v.Warnings {
			sb.WriteString(fmt.Sprintf("  WARN: %s\n", w))
		}
	}
	return sb.String()
}
