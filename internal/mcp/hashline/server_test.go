package hashline

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runMCP sends a sequence of JSON-RPC lines and returns all responses.
func runMCP(t *testing.T, workspace string, requests []string) []map[string]any {
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

func TestMCPInitialize(t *testing.T) {
	responses := runMCP(t, t.TempDir(), []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
	})
	if len(responses) == 0 {
		t.Fatal("no response")
	}
	result, ok := responses[0]["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result: %v", responses[0])
	}
	if result["protocolVersion"] == nil {
		t.Error("missing protocolVersion")
	}
}

func TestMCPToolsList(t *testing.T) {
	responses := runMCP(t, t.TempDir(), []string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`,
	})
	if len(responses) == 0 {
		t.Fatal("no response")
	}
	result := responses[0]["result"].(map[string]any)
	tools := result["tools"].([]any)
	if len(tools) != 2 {
		t.Errorf("tools = %d, want 2", len(tools))
	}
	names := []string{}
	for _, tool := range tools {
		tm := tool.(map[string]any)
		names = append(names, tm["name"].(string))
	}
	if names[0] != "hashline_read" || names[1] != "hashline_edit" {
		t.Errorf("tool names = %v", names)
	}
}

func TestMCPReadFile(t *testing.T) {
	ws := t.TempDir()
	path := filepath.Join(ws, "test.txt")
	os.WriteFile(path, []byte("a\nb\nc\n"), 0o644)

	responses := runMCP(t, ws, []string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"hashline_read","arguments":{"path":"test.txt"}}}`,
	})
	if len(responses) == 0 {
		t.Fatal("no response")
	}
	result := responses[0]["result"].(map[string]any)
	content := result["content"].([]any)
	text := content[0].(map[string]any)["text"].(string)
	if !strings.Contains(text, "a") || !strings.Contains(text, "b") {
		t.Errorf("read result missing content: %s", text)
	}
}

func TestMCPEditFile(t *testing.T) {
	ws := t.TempDir()
	path := filepath.Join(ws, "test.txt")
	os.WriteFile(path, []byte("old\nb\n"), 0o644)

	// First read to get the anchor
	readResp := runMCP(t, ws, []string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"hashline_read","arguments":{"path":"test.txt"}}}`,
	})
	readResult := readResp[0]["result"].(map[string]any)
	readText := readResult["content"].([]any)[0].(map[string]any)["text"].(string)
	var readData struct {
		Lines []struct {
			Number int    `json:"number"`
			Hash   string `json:"hash"`
		} `json:"lines"`
	}
	json.Unmarshal([]byte(readText), &readData)
	anchor := "1#" + readData.Lines[0].Hash

	// Now edit
	editResp := runMCP(t, ws, []string{
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"hashline_edit","arguments":{"path":"test.txt","edits":[{"type":"replace_line","anchor":"` + anchor + `","content":"new"}]}}}`,
	})
	editResult := editResp[0]["result"].(map[string]any)
	editText := editResult["content"].([]any)[0].(map[string]any)["text"].(string)
	if !strings.Contains(editText, "diff") {
		t.Errorf("edit result should contain diff: %s", editText)
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "new") {
		t.Errorf("file should contain new content: %s", data)
	}
}

func TestMCPStaleAnchor(t *testing.T) {
	ws := t.TempDir()
	path := filepath.Join(ws, "test.txt")
	os.WriteFile(path, []byte("a\nb\n"), 0o644)

	responses := runMCP(t, ws, []string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"hashline_edit","arguments":{"path":"test.txt","edits":[{"type":"replace_line","anchor":"1#ZZ","content":"X"}]}}}`,
	})
	if len(responses) == 0 {
		t.Fatal("no response")
	}
	errObj := responses[0]["error"]
	if errObj == nil {
		t.Fatal("expected error for stale anchor")
	}
	errMap := errObj.(map[string]any)
	if errMap["code"].(float64) != -32603 {
		t.Errorf("error code = %v, want -32603", errMap["code"])
	}
	data, _ := errMap["data"].(map[string]any)
	if data["type"] != "stale_anchor" {
		t.Errorf("error data type = %v, want stale_anchor", data["type"])
	}
}

func TestMCPDryRun(t *testing.T) {
	ws := t.TempDir()
	path := filepath.Join(ws, "test.txt")
	os.WriteFile(path, []byte("a\nb\n"), 0o644)

	// Read first
	readResp := runMCP(t, ws, []string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"hashline_read","arguments":{"path":"test.txt"}}}`,
	})
	readText := readResp[0]["result"].(map[string]any)["content"].([]any)[0].(map[string]any)["text"].(string)
	var readData struct {
		Lines []struct {
			Number int    `json:"number"`
			Hash   string `json:"hash"`
		} `json:"lines"`
	}
	json.Unmarshal([]byte(readText), &readData)
	anchor := "1#" + readData.Lines[0].Hash

	// Dry run edit
	editResp := runMCP(t, ws, []string{
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"hashline_edit","arguments":{"path":"test.txt","dryRun":true,"edits":[{"type":"replace_line","anchor":"` + anchor + `","content":"X"}]}}}`,
	})
	editText := editResp[0]["result"].(map[string]any)["content"].([]any)[0].(map[string]any)["text"].(string)
	if !strings.Contains(editText, "diff") {
		t.Error("dry run should return diff")
	}

	// File should be unchanged
	data, _ := os.ReadFile(path)
	if string(data) != "a\nb\n" {
		t.Errorf("dry run should not modify file: %s", data)
	}
}

func TestMCPPathEscape(t *testing.T) {
	ws := t.TempDir()
	responses := runMCP(t, ws, []string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"hashline_read","arguments":{"path":"../../../etc/passwd"}}}`,
	})
	if len(responses) == 0 {
		t.Fatal("no response")
	}
	if responses[0]["error"] == nil {
		t.Error("expected error for path escape")
	}
}

func TestMCPUnknownTool(t *testing.T) {
	responses := runMCP(t, t.TempDir(), []string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"bogus_tool","arguments":{}}}`,
	})
	if len(responses) == 0 {
		t.Fatal("no response")
	}
	if responses[0]["error"] == nil {
		t.Error("expected error for unknown tool")
	}
}

func TestMCPUnknownMethod(t *testing.T) {
	responses := runMCP(t, t.TempDir(), []string{
		`{"jsonrpc":"2.0","id":1,"method":"bogus/method","params":{}}`,
	})
	if len(responses) == 0 {
		t.Fatal("no response")
	}
	if responses[0]["error"] == nil {
		t.Error("expected error for unknown method")
	}
}

func TestMCPMalformedJSON(t *testing.T) {
	responses := runMCP(t, t.TempDir(), []string{
		`{not valid json`,
	})
	if len(responses) == 0 {
		t.Fatal("no response")
	}
	if responses[0]["error"] == nil {
		t.Error("expected error for malformed JSON")
	}
}
