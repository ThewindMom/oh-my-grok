package hookenv

import (
	"os"
	"path/filepath"
)

func GrokHome() string {
	if candidate := os.Getenv("GROK_HOME"); candidate != "" && filepath.IsAbs(candidate) {
		return candidate
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".grok")
	}
	return "/tmp/.grok"
}

func PluginRoot() (string, error) {
	if pr := os.Getenv("GROK_PLUGIN_ROOT"); pr != "" && filepath.IsAbs(pr) {
		return pr, nil
	}
	// internal/hookenv -> repo root is ../..
	here, err := os.Getwd()
	if err != nil {
		return "", err
	}
	candidates := []string{
		filepath.Join(here, "plugin.json"),
	}
	if exe, e := os.Executable(); e == nil {
		dir := filepath.Dir(exe)
		for i := 0; i < 6; i++ {
			candidates = append(candidates, filepath.Join(dir, "plugin.json"))
			dir = filepath.Dir(dir)
		}
	}
	for _, p := range candidates {
		root := filepath.Dir(p)
		if _, err := os.Stat(p); err == nil {
			return root, nil
		}
	}
	// Fallback: walk from cwd
	dir := here
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "plugin.json")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", os.ErrNotExist
}

func ApplyEvent(ev Event) {
	if ev.SessionID != "" && os.Getenv("GROK_SESSION_ID") == "" {
		_ = os.Setenv("GROK_SESSION_ID", ev.SessionID)
	}
	if ev.WorkspaceRoot != "" && os.Getenv("GROK_WORKSPACE_ROOT") == "" {
		_ = os.Setenv("GROK_WORKSPACE_ROOT", ev.WorkspaceRoot)
	}
}