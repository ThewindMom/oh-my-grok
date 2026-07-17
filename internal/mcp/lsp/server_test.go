package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func runLSPMCP(t *testing.T, workspace string, requests []string) []map[string]any {
	t.Helper()
	server := NewServer(workspace)
	rIn, wIn := io.Pipe()
	rOut := &bytes.Buffer{}
	server.in = bufio.NewReader(rIn)
	server.out = rOut

	done := make(chan struct{})
	go func() {
		server.Run()
		close(done)
	}()

	for _, req := range requests {
		wIn.Write([]byte(req + "\n"))
	}
	wIn.Close()
	<-done

	var responses []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(rOut.String()), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err == nil {
			responses = append(responses, m)
		}
	}
	return responses
}

func TestLSPInitialize(t *testing.T) {
	responses := runLSPMCP(t, t.TempDir(), []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
	})
	if len(responses) == 0 {
		t.Fatal("no response")
	}
	result := responses[0]["result"].(map[string]any)
	if result["protocolVersion"] == nil {
		t.Error("missing protocolVersion")
	}
}

func TestLSPToolsList(t *testing.T) {
	responses := runLSPMCP(t, t.TempDir(), []string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`,
	})
	if len(responses) == 0 {
		t.Fatal("no response")
	}
	result := responses[0]["result"].(map[string]any)
	tools := result["tools"].([]any)
	if len(tools) != 4 {
		t.Errorf("tools = %d, want 4", len(tools))
	}
}

func TestLSPDiagnostics(t *testing.T) {
	ws := t.TempDir()
	responses := runLSPMCP(t, ws, []string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"lsp_diagnostics","arguments":{"filePath":"test.go"}}}`,
	})
	if len(responses) == 0 {
		t.Fatal("no response")
	}
	result := responses[0]["result"].(map[string]any)
	content := result["content"].([]any)
	text := content[0].(map[string]any)["text"].(string)
	if !strings.Contains(text, "diagnostics") {
		t.Errorf("should contain diagnostics: %s", text)
	}
}

func TestLSPStatus(t *testing.T) {
	responses := runLSPMCP(t, t.TempDir(), []string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"lsp_status","arguments":{}}}`,
	})
	if len(responses) == 0 {
		t.Fatal("no response")
	}
	result := responses[0]["result"].(map[string]any)
	content := result["content"].([]any)
	text := content[0].(map[string]any)["text"].(string)
	if !strings.Contains(text, "workspaceRoot") {
		t.Errorf("should contain workspaceRoot: %s", text)
	}
}

func TestLSPUnknownTool(t *testing.T) {
	responses := runLSPMCP(t, t.TempDir(), []string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"bogus","arguments":{}}}`,
	})
	if len(responses) == 0 {
		t.Fatal("no response")
	}
	if responses[0]["error"] == nil {
		t.Error("expected error for unknown tool")
	}
}

func TestLSPMalformedJSON(t *testing.T) {
	responses := runLSPMCP(t, t.TempDir(), []string{
		`{not valid json`,
	})
	if len(responses) == 0 {
		t.Fatal("no response")
	}
	if responses[0]["error"] == nil {
		t.Error("expected error for malformed JSON")
	}
}

func TestGetLanguageServer(t *testing.T) {
	tests := []struct {
		ext  string
		want bool
	}{
		{".go", true},
		{".ts", true},
		{".py", true},
		{".rs", true},
		{".unknown", false},
	}

	for _, tt := range tests {
		cmd := getLanguageServer(tt.ext)
		got := cmd != nil
		if got != tt.want {
			t.Errorf("getLanguageServer(%q) = %v, want %v", tt.ext, got, tt.want)
		}
	}
}

func TestGetAvailableServers(t *testing.T) {
	servers := getAvailableServers()
	if len(servers) == 0 {
		t.Error("should have available servers")
	}
	found := false
	for _, s := range servers {
		if strings.Contains(s, "Go") {
			found = true
		}
	}
	if !found {
		t.Error("should list Go server")
	}
}
