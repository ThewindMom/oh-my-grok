package hookio

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

func escapeJSON(s string) string {
	var b strings.Builder
	_ = json.NewEncoder(&b).Encode(s)
	out := strings.TrimSpace(b.String())
	if len(out) >= 2 && out[0] == '"' {
		return out[1 : len(out)-1]
	}
	return out
}

func EmitAllow(w io.Writer) {
	fmt.Fprint(w, `{"decision":"allow"}`+"\n")
}

func EmitDeny(w io.Writer, reason string) int {
	fmt.Fprintf(w, `{"decision":"deny","reason":"%s"}`+"\n", escapeJSON(reason))
	return 2
}

func EmitStopBlock(w io.Writer, reason string) {
	fmt.Fprintf(w, `{"decision":"block","reason":"%s"}`+"\n", escapeJSON(reason))
}

func EmitStopAllow(w io.Writer) {
	fmt.Fprint(w, "{}\n")
}

func EmitAdditionalContext(w io.Writer, message, hookEvent string) {
	escaped := escapeJSON(message)
	if os.Getenv("CURSOR_PLUGIN_ROOT") != "" {
		fmt.Fprintf(w, `{"additional_context":"%s"}`+"\n", escaped)
		return
	}
	if os.Getenv("CLAUDE_PLUGIN_ROOT") != "" && os.Getenv("COPILOT_CLI") == "" {
		fmt.Fprintf(w, `{"hookSpecificOutput":{"hookEventName":"%s","additionalContext":"%s"}}`+"\n",
			escapeJSON(hookEvent), escaped)
		return
	}
	fmt.Fprintf(w, `{"additionalContext":"%s"}`+"\n", escaped)
}