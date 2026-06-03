package hookenv

import (
	"strings"
)

func normalizeStopReason(stopReason string) string {
	sr := strings.ToLower(strings.TrimSpace(stopReason))
	sr = strings.ReplaceAll(sr, "-", "_")
	sr = strings.ReplaceAll(sr, " ", "_")
	return sr
}

// ShouldAllowStopOnAbort reports whether the harness stop is an abort or side
// effect (user cancel, error, active background work) where continuation hooks
// must not block. Routine model turn ends (EndTurn, completed, stop, …) return
// false so Ralph/todo/boulder continuations can run.
func ShouldAllowStopOnAbort(stopReason string, stopHookActive bool, backgroundTasks []map[string]any) bool {
	if stopHookActive {
		return true
	}
	active := map[string]struct{}{
		"running": {}, "pending": {}, "in_progress": {},
		"in-progress": {}, "active": {},
	}
	for _, t := range backgroundTasks {
		st, _ := t["status"].(string)
		if _, ok := active[strings.ToLower(st)]; ok {
			return true
		}
	}
	sr := normalizeStopReason(stopReason)
	if sr == "" {
		return false
	}
	switch sr {
	case "end_turn", "endturn", "completed", "complete", "stop", "finished", "done",
		"max_tokens", "max_turns", "length", "content_filter", "tool_use", "tool_calls":
		return false
	}
	switch sr {
	case "cancelled", "canceled", "user_cancel", "abort", "aborted",
		"interrupted", "interrupt", "error", "refused", "refusal":
		return true
	}
	// Unknown non-empty reason: preserve legacy fail-open for explicit harness stops.
	return true
}