// Package workspace provides scoped AGENTS.md discovery and injection.
//
// It discovers AGENTS.md files from the workspace root to the target path,
// preserving nearest-file precedence. It enforces byte limits and avoids
// injecting duplicate content.
//
// This is an original implementation. It does not derive from any SUL-covered source.
package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MaxRuleBytes is the maximum bytes of rules to inject per file.
const MaxRuleBytes = 4096

// MaxTotalBytes is the maximum total bytes of rules to inject.
const MaxTotalBytes = 32768

// RuleFile represents a discovered AGENTS.md file.
type RuleFile struct {
	Path     string `json:"path"`
	RelPath   string `json:"relPath"`
	Content  string `json:"content"`
	Bytes    int    `json:"bytes"`
	Depth    int    `json:"depth"` // 0 = nearest, higher = further up
}

// DiscoverRules finds AGENTS.md files from the workspace root to the target path.
// Returns files in nearest-first order (closest to targetPath first).
func DiscoverRules(workspaceRoot, targetPath string) ([]RuleFile, error) {
	if workspaceRoot == "" {
		return nil, fmt.Errorf("workspace root is required")
	}

	absRoot, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return nil, err
	}
	absRoot = filepath.Clean(absRoot)

	absTarget := absRoot
	if targetPath != "" {
		absTarget, err = filepath.Abs(targetPath)
		if err != nil {
			return nil, err
		}
		absTarget = filepath.Clean(absTarget)
	}

	// Verify target is within workspace
	if !strings.HasPrefix(absTarget+string(filepath.Separator), absRoot+string(filepath.Separator)) && absTarget != absRoot {
		return nil, fmt.Errorf("target path %s escapes workspace %s", targetPath, workspaceRoot)
	}

	// Walk from target up to root, collecting AGENTS.md files
	var files []RuleFile
	dir := absTarget
	depth := 0

	for {
		agentsPath := filepath.Join(dir, "AGENTS.md")
		if content, err := os.ReadFile(agentsPath); err == nil {
			relPath, _ := filepath.Rel(absRoot, agentsPath)
			if relPath == "" {
				relPath = "AGENTS.md"
			}
			truncated := truncateContent(string(content), MaxRuleBytes)
			files = append(files, RuleFile{
				Path:    agentsPath,
				RelPath:  relPath,
				Content:  truncated,
				Bytes:    len(truncated),
				Depth:    depth,
			})
		}

		// Also check .grok/AGENTS.md and .agents/AGENTS.md
		for _, subDir := range []string{".grok", ".agents"} {
			subPath := filepath.Join(dir, subDir, "AGENTS.md")
			if content, err := os.ReadFile(subPath); err == nil {
				relPath, _ := filepath.Rel(absRoot, subPath)
				truncated := truncateContent(string(content), MaxRuleBytes)
				files = append(files, RuleFile{
					Path:    subPath,
					RelPath:  relPath,
					Content:  truncated,
					Bytes:    len(truncated),
					Depth:    depth,
				})
			}
		}

		// Stop at workspace root
		if dir == absRoot {
			break
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
		depth++
	}

	return files, nil
}

// FormatRules formats discovered rules into a single injection string.
// Files are ordered nearest-first. Total bytes are capped at MaxTotalBytes.
func FormatRules(files []RuleFile) string {
	if len(files) == 0 {
		return ""
	}

	var sb strings.Builder
	totalBytes := 0
	seen := map[string]bool{} // deduplicate by content hash

	for _, f := range files {
		if totalBytes >= MaxTotalBytes {
			break
		}

		// Deduplicate by content
		if seen[f.Content] {
			continue
		}
		seen[f.Content] = true

		remaining := MaxTotalBytes - totalBytes
		content := f.Content
		if len(content) > remaining {
			content = truncateContent(content, remaining)
		}

		sb.WriteString(fmt.Sprintf("=== %s ===\n", f.RelPath))
		sb.WriteString(content)
		if !strings.HasSuffix(content, "\n") {
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
		totalBytes += len(content)
	}

	return sb.String()
}

// truncateContent truncates content to maxBytes, ending at a line boundary.
func truncateContent(content string, maxBytes int) string {
	if len(content) <= maxBytes {
		return content
	}

	// Find a line boundary near the limit
	cutoff := maxBytes
	for cutoff > 0 && content[cutoff-1] != '\n' {
		cutoff--
	}
	if cutoff == 0 {
		cutoff = maxBytes
	}

	return content[:cutoff] + "\n... (truncated)\n"
}

// InjectRules discovers and formats rules for injection.
// This is the main entry point for hook context injection.
func InjectRules(workspaceRoot, targetPath string) (string, error) {
	files, err := DiscoverRules(workspaceRoot, targetPath)
	if err != nil {
		return "", err
	}
	return FormatRules(files), nil
}
