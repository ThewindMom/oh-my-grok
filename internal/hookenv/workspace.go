package hookenv

import (
	"os"
	"strings"
)

// Workspace returns the workspace root from the hook event stdin JSON or the
// process environment Grok sets when invoking plugin hooks.
func Workspace(ev Event) string {
	if ws := strings.TrimSpace(ev.WorkspaceRoot); ws != "" {
		return ws
	}
	for _, key := range []string{"GROK_WORKSPACE_ROOT", "CLAUDE_PROJECT_DIR"} {
		if ws := strings.TrimSpace(os.Getenv(key)); ws != "" {
			return ws
		}
	}
	return ""
}

// WorkspaceFromRaw picks workspace from decoded stdin before Event is built.
func WorkspaceFromRaw(raw map[string]any) string {
	ws := pickString(raw, "workspaceRoot", "workspace_root", "cwd")
	if ws != "" {
		return ws
	}
	if arr, ok := raw["workspaceRoots"].([]any); ok && len(arr) > 0 {
		if s, ok := arr[0].(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	if arr, ok := raw["workspace_roots"].([]any); ok && len(arr) > 0 {
		if s, ok := arr[0].(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	for _, key := range []string{"GROK_WORKSPACE_ROOT", "CLAUDE_PROJECT_DIR"} {
		if ws := strings.TrimSpace(os.Getenv(key)); ws != "" {
			return ws
		}
	}
	return ""
}