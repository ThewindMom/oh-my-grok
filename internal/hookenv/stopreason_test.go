package hookenv

import "testing"

func TestShouldAllowStopOnAbort(t *testing.T) {
	cases := []struct {
		reason string
		allow  bool
	}{
		{"", false},
		{"end_turn", false},
		{"EndTurn", false},
		{"completed", false},
		{"stop", false},
		{"cancelled", true},
		{"user_cancel", true},
		{"error", true},
	}
	for _, tc := range cases {
		got := ShouldAllowStopOnAbort(tc.reason, false, nil)
		if got != tc.allow {
			t.Errorf("ShouldAllowStopOnAbort(%q) = %v, want %v", tc.reason, got, tc.allow)
		}
	}
}

func TestWorkspaceFromEnv(t *testing.T) {
	t.Setenv("GROK_WORKSPACE_ROOT", "/tmp/ws-test")
	ev := Event{SessionID: "s1", Prompt: "/ralph-loop x"}
	if got := Workspace(ev); got != "/tmp/ws-test" {
		t.Fatalf("Workspace() = %q, want /tmp/ws-test", got)
	}
}