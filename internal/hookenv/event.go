package hookenv

import (
	"encoding/json"
	"io"
)

// Event is the subset of Grok hook stdin JSON used across subcommands.
type Event struct {
	SessionID            string
	WorkspaceRoot      string
	ToolName             string
	ToolInput            map[string]any
	Prompt               string
	StopReason           string
	HookEventName        string
	LastAssistantMessage string
	StopHookActive       bool
	BackgroundTasks      []map[string]any
}

type rawEvent map[string]any

func ReadEvent(r io.Reader) (Event, error) {
	var raw rawEvent
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return Event{}, err
	}
	return Event{
		SessionID:            pickString(raw, "sessionId", "session_id"),
		WorkspaceRoot:        pickString(raw, "workspaceRoot", "workspace_root", "cwd"),
		ToolName:             pickString(raw, "toolName", "tool_name", "tool"),
		ToolInput:            pickMap(raw, "toolInput", "tool_input", "input", "arguments", "rawInput"),
		Prompt:               pickString(raw, "prompt", "userPrompt", "user_prompt", "message"),
		StopReason:           pickString(raw, "stopReason", "stop_reason", "stop_reason_code"),
		HookEventName:        pickString(raw, "hookEventName", "hook_event_name"),
		LastAssistantMessage: pickString(raw, "last_assistant_message", "lastAssistantMessage", "last_assistant_message_text"),
		StopHookActive:       pickBool(raw, "stop_hook_active", "stopHookActive"),
		BackgroundTasks:      pickTaskList(raw, "background_tasks", "backgroundTasks"),
	}, nil
}

func pickBool(m map[string]any, keys ...string) bool {
	for _, k := range keys {
		v, ok := m[k]
		if !ok {
			continue
		}
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func pickTaskList(m map[string]any, keys ...string) []map[string]any {
	for _, k := range keys {
		v, ok := m[k]
		if !ok {
			continue
		}
		arr, ok := v.([]any)
		if !ok {
			continue
		}
		out := make([]map[string]any, 0, len(arr))
		for _, item := range arr {
			if mm, ok := item.(map[string]any); ok {
				out = append(out, mm)
			}
		}
		return out
	}
	return nil
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