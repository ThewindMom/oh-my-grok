package hookenv

import (
	"encoding/json"
	"io"
)

// Event is the subset of Grok hook stdin JSON used across subcommands.
type Event struct {
	SessionID     string
	WorkspaceRoot string
	ToolName      string
	ToolInput     map[string]any
	Prompt        string
	StopReason    string
	HookEventName string
}

type rawEvent map[string]any

func ReadEvent(r io.Reader) (Event, error) {
	var raw rawEvent
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return Event{}, err
	}
	return Event{
		SessionID:     pickString(raw, "sessionId", "session_id"),
		WorkspaceRoot: pickString(raw, "workspaceRoot", "workspace_root", "cwd"),
		ToolName:      pickString(raw, "toolName", "tool_name", "tool"),
		ToolInput:     pickMap(raw, "toolInput", "tool_input", "input", "arguments", "rawInput"),
		Prompt:        pickString(raw, "prompt", "userPrompt", "user_prompt"),
		StopReason:    pickString(raw, "stopReason", "stop_reason"),
		HookEventName: pickString(raw, "hookEventName", "hook_event_name"),
	}, nil
}

func pickString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		v, ok := m[k]
		if !ok {
			continue
		}
		s, ok := v.(string)
		if ok && s != "" {
			return s
		}
	}
	return ""
}

func pickMap(m map[string]any, keys ...string) map[string]any {
	for _, k := range keys {
		v, ok := m[k]
		if !ok {
			continue
		}
		if mm, ok := v.(map[string]any); ok {
			return mm
		}
	}
	return nil
}