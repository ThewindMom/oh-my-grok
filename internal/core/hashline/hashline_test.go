package hashline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func anchor(l Line) string {
	return fmt.Sprintf("%d#%s", l.Number, l.Hash)
}

// === Hash computation tests ===

func TestComputeLineHashDeterministic(t *testing.T) {
	h1 := ComputeLineHash(1, "hello world")
	h2 := ComputeLineHash(1, "hello world")
	if h1 != h2 {
		t.Error("hash should be deterministic")
	}
	if len(h1) != 2 {
		t.Errorf("hash length = %d, want 2", len(h1))
	}
}

func TestComputeLineHashTrailingWhitespace(t *testing.T) {
	h1 := ComputeLineHash(1, "hello")
	h2 := ComputeLineHash(1, "hello   ")
	h3 := ComputeLineHash(1, "hello\t")
	if h1 != h2 || h1 != h3 {
		t.Error("trailing whitespace should be normalized")
	}
}

func TestComputeLineHashCRLF(t *testing.T) {
	h1 := ComputeLineHash(1, "hello")
	h2 := ComputeLineHash(1, "hello\r")
	if h1 != h2 {
		t.Error("CR should be stripped in normalization")
	}
}

func TestComputeLineHashUnicode(t *testing.T) {
	h := ComputeLineHash(1, "héllo wörld 世界")
	if len(h) != 2 {
		t.Errorf("unicode hash length = %d, want 2", len(h))
	}
}

func TestComputeLineHashVeryLongLine(t *testing.T) {
	long := strings.Repeat("x", 10000)
	h := ComputeLineHash(1, long)
	if len(h) != 2 {
		t.Errorf("long line hash length = %d, want 2", len(h))
	}
}

func TestComputeLineHashEmptyLine(t *testing.T) {
	h := ComputeLineHash(5, "")
	if len(h) != 2 {
		t.Errorf("empty line hash length = %d, want 2", len(h))
	}
}

func TestParseAnchor(t *testing.T) {
	a, err := ParseAnchor("12#ZP")
	if err != nil {
		t.Fatalf("ParseAnchor: %v", err)
	}
	if a.Line != 12 {
		t.Errorf("line = %d, want 12", a.Line)
	}
	if a.Hash != "ZP" {
		t.Errorf("hash = %q, want ZP", a.Hash)
	}
}

func TestParseAnchorInvalid(t *testing.T) {
	bad := []string{"", "abc", "12#", "#ZP", "0#ZP", "12#Z", "12#ZPM", "12#AC"}
	for _, s := range bad {
		_, err := ParseAnchor(s)
		if err == nil {
			t.Errorf("expected error for invalid anchor %q", s)
		}
	}
}

func TestFormatHashLine(t *testing.T) {
	s := FormatHashLine(5, "hello")
	if !strings.HasPrefix(s, "5#") {
		t.Errorf("FormatHashLine should start with line number: %s", s)
	}
	if !strings.Contains(s, "|hello") {
		t.Errorf("FormatHashLine should contain |content: %s", s)
	}
}

// === Read tests ===

func writeTestFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadFullFile(t *testing.T) {
	path := writeTestFile(t, "line one\nline two\nline three\n")
	result, err := ReadFile(path, 1, 0)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(result.Lines) != 3 {
		t.Fatalf("lines = %d, want 3", len(result.Lines))
	}
	if result.Lines[0].Content != "line one" {
		t.Errorf("line 1 = %q", result.Lines[0].Content)
	}
	if result.Lines[0].Number != 1 {
		t.Errorf("line 1 number = %d", result.Lines[0].Number)
	}
	if result.Lines[0].Hash == "" {
		t.Error("line 1 hash should not be empty")
	}
}

func TestReadOffsetLimit(t *testing.T) {
	path := writeTestFile(t, "a\nb\nc\nd\ne\n")
	result, err := ReadFile(path, 2, 2)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(result.Lines) != 2 {
		t.Fatalf("lines = %d, want 2", len(result.Lines))
	}
	if result.Lines[0].Content != "b" {
		t.Errorf("line 1 = %q, want b", result.Lines[0].Content)
	}
	if result.Lines[1].Content != "c" {
		t.Errorf("line 2 = %q, want c", result.Lines[1].Content)
	}
}

func TestReadEmptyFile(t *testing.T) {
	path := writeTestFile(t, "")
	result, err := ReadFile(path, 1, 0)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(result.Lines) != 0 {
		t.Errorf("lines = %d, want 0", len(result.Lines))
	}
}

func TestReadNoFinalNewline(t *testing.T) {
	path := writeTestFile(t, "a\nb")
	result, err := ReadFile(path, 1, 0)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if result.Identity.HasFinalNL {
		t.Error("should have no final newline")
	}
	if len(result.Lines) != 2 {
		t.Errorf("lines = %d, want 2", len(result.Lines))
	}
}

func TestReadCRLF(t *testing.T) {
	path := writeTestFile(t, "a\r\nb\r\n")
	result, err := ReadFile(path, 1, 0)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if result.Identity.Newline != NewlineCRLF {
		t.Errorf("newline = %q, want crlf", result.Identity.Newline)
	}
	if len(result.Lines) != 2 {
		t.Errorf("lines = %d, want 2", len(result.Lines))
	}
	if result.Lines[0].Content != "a" {
		t.Errorf("line 1 = %q, want a", result.Lines[0].Content)
	}
}

func TestReadBinaryRejection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bin.dat")
	os.WriteFile(path, []byte{0x00, 0x01, 0x02, 0x00, 0x03}, 0o644)
	_, err := ReadFile(path, 1, 0)
	if err == nil {
		t.Error("binary file should be rejected")
	}
}

func TestReadPathEscape(t *testing.T) {
	tmp := t.TempDir()
	_, err := ResolvePath("../../../etc/passwd", tmp)
	if err == nil {
		t.Error("path escape should be rejected")
	}
}

// === Edit tests ===

func TestEditReplaceLine(t *testing.T) {
	path := writeTestFile(t, "old line\nb\nc\n")
	result, err := ReadFile(path, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	anchor := anchor(result.Lines[0])

	editResult, err := ApplyEdits(EditRequest{
		Path: path,
		Edits: []EditOp{
			{Type: OpReplaceLine, Anchor: anchor, Content: "new line"},
		},
	}, filepath.Dir(path))
	if err != nil {
		t.Fatalf("ApplyEdits: %v", err)
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "new line") {
		t.Errorf("file should contain new content: %s", data)
	}
	if editResult.Diff == "" {
		t.Error("diff should not be empty")
	}
}

func TestEditReplaceRange(t *testing.T) {
	path := writeTestFile(t, "a\nb\nc\nd\ne\n")
	result, _ := ReadFile(path, 1, 0)
	startAnchor := anchor(result.Lines[0])
	endAnchor := anchor(result.Lines[2])

	_, err := ApplyEdits(EditRequest{
		Path: path,
		Edits: []EditOp{
			{Type: OpReplaceRange, Anchor: startAnchor, EndAnchor: endAnchor, Content: "X\nY"},
		},
	}, filepath.Dir(path))
	if err != nil {
		t.Fatalf("ApplyEdits: %v", err)
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "X\nY") {
		t.Errorf("file should contain replacement: %s", data)
	}
	if strings.Contains(string(data), "\nb\n") {
		t.Errorf("old line b should be gone: %s", data)
	}
}

func TestEditInsertBefore(t *testing.T) {
	path := writeTestFile(t, "a\nb\n")
	result, _ := ReadFile(path, 1, 0)
	anchor := anchor(result.Lines[0])

	_, err := ApplyEdits(EditRequest{
		Path: path,
		Edits: []EditOp{
			{Type: OpInsertBefore, Anchor: anchor, Content: "inserted"},
		},
	}, filepath.Dir(path))
	if err != nil {
		t.Fatalf("ApplyEdits: %v", err)
	}

	data, _ := os.ReadFile(path)
	if !strings.HasPrefix(string(data), "inserted\n") {
		t.Errorf("inserted line should be first: %s", data)
	}
}

func TestEditInsertAfter(t *testing.T) {
	path := writeTestFile(t, "a\nb\n")
	result, _ := ReadFile(path, 1, 0)
	anchor := anchor(result.Lines[0])

	_, err := ApplyEdits(EditRequest{
		Path: path,
		Edits: []EditOp{
			{Type: OpInsertAfter, Anchor: anchor, Content: "inserted"},
		},
	}, filepath.Dir(path))
	if err != nil {
		t.Fatalf("ApplyEdits: %v", err)
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "a\ninserted\nb") {
		t.Errorf("inserted line should be after line 1: %s", data)
	}
}

func TestEditDeleteLine(t *testing.T) {
	path := writeTestFile(t, "a\nb\nc\n")
	result, _ := ReadFile(path, 1, 0)
	anchor := anchor(result.Lines[1])

	_, err := ApplyEdits(EditRequest{
		Path: path,
		Edits: []EditOp{
			{Type: OpDeleteLine, Anchor: anchor},
		},
	}, filepath.Dir(path))
	if err != nil {
		t.Fatalf("ApplyEdits: %v", err)
	}

	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "\nb\n") {
		t.Errorf("line b should be deleted: %s", data)
	}
}

func TestEditDeleteRange(t *testing.T) {
	path := writeTestFile(t, "a\nb\nc\nd\ne\n")
	result, _ := ReadFile(path, 1, 0)
	startAnchor := anchor(result.Lines[1])
	endAnchor := anchor(result.Lines[3])

	_, err := ApplyEdits(EditRequest{
		Path: path,
		Edits: []EditOp{
			{Type: OpDeleteRange, Anchor: startAnchor, EndAnchor: endAnchor},
		},
	}, filepath.Dir(path))
	if err != nil {
		t.Fatalf("ApplyEdits: %v", err)
	}

	data, _ := os.ReadFile(path)
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	if len(lines) != 2 {
		t.Errorf("lines = %d, want 2: %s", len(lines), data)
	}
	if lines[0] != "a" || lines[1] != "e" {
		t.Errorf("expected a,e: %s", data)
	}
}

func TestEditPrepend(t *testing.T) {
	path := writeTestFile(t, "a\n")
	_, err := ApplyEdits(EditRequest{
		Path: path,
		Edits: []EditOp{
			{Type: OpPrepend, Content: "first"},
		},
	}, filepath.Dir(path))
	if err != nil {
		t.Fatalf("ApplyEdits: %v", err)
	}
	data, _ := os.ReadFile(path)
	if !strings.HasPrefix(string(data), "first\n") {
		t.Errorf("prepend should be first: %s", data)
	}
}

func TestEditAppend(t *testing.T) {
	path := writeTestFile(t, "a\n")
	_, err := ApplyEdits(EditRequest{
		Path: path,
		Edits: []EditOp{
			{Type: OpAppend, Content: "last"},
		},
	}, filepath.Dir(path))
	if err != nil {
		t.Fatalf("ApplyEdits: %v", err)
	}
	data, _ := os.ReadFile(path)
	if !strings.HasSuffix(string(data), "last\n") {
		t.Errorf("append should be last: %s", data)
	}
}

func TestEditMultipleIndependent(t *testing.T) {
	path := writeTestFile(t, "a\nb\nc\nd\ne\n")
	result, _ := ReadFile(path, 1, 0)
	a1 := anchor(result.Lines[0])
	a4 := anchor(result.Lines[3])

	_, err := ApplyEdits(EditRequest{
		Path: path,
		Edits: []EditOp{
			{Type: OpReplaceLine, Anchor: a4, Content: "D"},
			{Type: OpReplaceLine, Anchor: a1, Content: "A"},
		},
	}, filepath.Dir(path))
	if err != nil {
		t.Fatalf("ApplyEdits: %v", err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "A\nb\nc\nD\ne") {
		t.Errorf("both replacements should apply: %s", data)
	}
}

func TestEditUnsorted(t *testing.T) {
	path := writeTestFile(t, "a\nb\nc\n")
	result, _ := ReadFile(path, 1, 0)
	a1 := anchor(result.Lines[0])
	a3 := anchor(result.Lines[2])

	// Edits in reverse order should still work
	_, err := ApplyEdits(EditRequest{
		Path: path,
		Edits: []EditOp{
			{Type: OpReplaceLine, Anchor: a3, Content: "C"},
			{Type: OpReplaceLine, Anchor: a1, Content: "A"},
		},
	}, filepath.Dir(path))
	if err != nil {
		t.Fatalf("ApplyEdits: %v", err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "A\nb\nC") {
		t.Errorf("unsorted edits should apply correctly: %s", data)
	}
}

func TestEditOverlapRejection(t *testing.T) {
	path := writeTestFile(t, "a\nb\nc\nd\n")
	result, _ := ReadFile(path, 1, 0)
	a2 := anchor(result.Lines[1])
	startAnchor := anchor(result.Lines[0])
	endAnchor := anchor(result.Lines[2])

	_, err := ApplyEdits(EditRequest{
		Path: path,
		Edits: []EditOp{
			{Type: OpReplaceRange, Anchor: startAnchor, EndAnchor: endAnchor, Content: "X"},
			{Type: OpReplaceLine, Anchor: a2, Content: "Y"},
		},
	}, filepath.Dir(path))
	if err == nil {
		t.Error("overlapping edits should be rejected")
	}

	// File should be unchanged
	data, _ := os.ReadFile(path)
	if string(data) != "a\nb\nc\nd\n" {
		t.Errorf("file should be unchanged after rejection: %s", data)
	}
}

func TestEditStaleAnchor(t *testing.T) {
	path := writeTestFile(t, "a\nb\nc\n")
	// Use a fake anchor that doesn't match
	_, err := ApplyEdits(EditRequest{
		Path: path,
		Edits: []EditOp{
			{Type: OpReplaceLine, Anchor: "1#ZZ", Content: "X"},
		},
	}, filepath.Dir(path))
	if err == nil {
		t.Error("stale anchor should be rejected")
	}

	// File should be unchanged
	data, _ := os.ReadFile(path)
	if string(data) != "a\nb\nc\n" {
		t.Errorf("file should be unchanged after stale rejection: %s", data)
	}
}

func TestEditStaleAnchorHasNearby(t *testing.T) {
	path := writeTestFile(t, "a\nb\nc\n")
	_, err := ApplyEdits(EditRequest{
		Path: path,
		Edits: []EditOp{
			{Type: OpReplaceLine, Anchor: "2#ZZ", Content: "X"},
		},
	}, filepath.Dir(path))
	if err == nil {
		t.Fatal("expected stale anchor error")
	}
	// Should be a StaleAnchorError with nearby lines
	if _, ok := err.(*StaleAnchorError); !ok {
		t.Errorf("expected StaleAnchorError, got %T: %v", err, err)
	}
}

func TestEditDryRun(t *testing.T) {
	path := writeTestFile(t, "a\nb\n")
	result, _ := ReadFile(path, 1, 0)
	anchor := anchor(result.Lines[0])

	editResult, err := ApplyEdits(EditRequest{
		Path:    path,
		DryRun:  true,
		Edits: []EditOp{
			{Type: OpReplaceLine, Anchor: anchor, Content: "X"},
		},
	}, filepath.Dir(path))
	if err != nil {
		t.Fatalf("ApplyEdits: %v", err)
	}
	if !editResult.DryRun {
		t.Error("result should indicate dry run")
	}
	if editResult.Diff == "" {
		t.Error("dry run should still produce diff")
	}

	// File should be unchanged
	data, _ := os.ReadFile(path)
	if string(data) != "a\nb\n" {
		t.Errorf("dry run should not write: %s", data)
	}
}

func TestEditIdentityMismatch(t *testing.T) {
	path := writeTestFile(t, "a\nb\n")
	_, err := ApplyEdits(EditRequest{
		Path: path,
		ExpectedIdentity: &FileIdentity{SHA256: "0000000000000000000000000000000000000000000000000000000000000000"},
		Edits: []EditOp{
			{Type: OpAppend, Content: "x"},
		},
	}, filepath.Dir(path))
	if err == nil {
		t.Error("identity mismatch should be rejected")
	}
}

func TestEditCRLFPreserved(t *testing.T) {
	path := filepath.Join(t.TempDir(), "crlf.txt")
	os.WriteFile(path, []byte("a\r\nb\r\n"), 0o644)
	result, _ := ReadFile(path, 1, 0)
	anchor := anchor(result.Lines[0])

	_, err := ApplyEdits(EditRequest{
		Path: path,
		Edits: []EditOp{
			{Type: OpReplaceLine, Anchor: anchor, Content: "X"},
		},
	}, filepath.Dir(path))
	if err != nil {
		t.Fatalf("ApplyEdits: %v", err)
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "\r\n") {
		t.Errorf("CRLF should be preserved: %s", data)
	}
}

func TestEditFinalNewlinePreserved(t *testing.T) {
	path := writeTestFile(t, "a\nb\n")
	result, _ := ReadFile(path, 1, 0)
	anchor := anchor(result.Lines[0])

	_, err := ApplyEdits(EditRequest{
		Path: path,
		Edits: []EditOp{
			{Type: OpReplaceLine, Anchor: anchor, Content: "X"},
		},
	}, filepath.Dir(path))
	if err != nil {
		t.Fatalf("ApplyEdits: %v", err)
	}

	data, _ := os.ReadFile(path)
	if !strings.HasSuffix(string(data), "\n") {
		t.Errorf("final newline should be preserved: %s", data)
	}
}

func TestEditPermissionsPreserved(t *testing.T) {
	path := filepath.Join(t.TempDir(), "perms.txt")
	os.WriteFile(path, []byte("a\nb\n"), 0o600)
	result, _ := ReadFile(path, 1, 0)
	anchor := anchor(result.Lines[0])

	ApplyEdits(EditRequest{
		Path: path,
		Edits: []EditOp{
			{Type: OpReplaceLine, Anchor: anchor, Content: "X"},
		},
	}, filepath.Dir(path))

	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o600 {
		t.Errorf("permissions changed: got %o, want 0600", info.Mode().Perm())
	}
}

func TestEditConcurrentSameFile(t *testing.T) {
	path := writeTestFile(t, "a\nb\nc\nd\ne\nf\ng\nh\n")
	result, _ := ReadFile(path, 1, 0)

	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for i := 0; i < 4; i++ {
		idx := i * 2
		if idx >= len(result.Lines) {
			continue
		}
		anchor := anchor(result.Lines[idx])
		wg.Add(1)
		go func(a string, n int) {
			defer wg.Done()
			_, err := ApplyEdits(EditRequest{
				Path: path,
				Edits: []EditOp{
					{Type: OpReplaceLine, Anchor: a, Content: "edited" + string(rune('0'+n))},
				},
			}, filepath.Dir(path))
			errs <- err
		}(anchor, i)
	}
	wg.Wait()
	close(errs)

	// At least some should succeed; file should be valid JSON
	data, _ := os.ReadFile(path)
	if len(data) == 0 {
		t.Error("file should not be empty after concurrent edits")
	}
}

func TestEditNoOps(t *testing.T) {
	path := writeTestFile(t, "a\n")
	_, err := ApplyEdits(EditRequest{
		Path:  path,
		Edits: []EditOp{},
	}, filepath.Dir(path))
	if err == nil {
		t.Error("empty edits should be rejected")
	}
}

func TestEditTooManyOps(t *testing.T) {
	path := writeTestFile(t, "a\n")
	edits := make([]EditOp, MaxOperations+1)
	for i := range edits {
		edits[i] = EditOp{Type: OpAppend, Content: "x"}
	}
	_, err := ApplyEdits(EditRequest{
		Path:  path,
		Edits: edits,
	}, filepath.Dir(path))
	if err == nil {
		t.Error("too many ops should be rejected")
	}
}

func TestResolvePathRelative(t *testing.T) {
	ws := t.TempDir()
	abs, err := ResolvePath("foo/bar.txt", ws)
	if err != nil {
		t.Fatalf("ResolvePath: %v", err)
	}
	expected := filepath.Join(ws, "foo", "bar.txt")
	if abs != expected {
		t.Errorf("got %s, want %s", abs, expected)
	}
}

func TestResolvePathAbsolute(t *testing.T) {
	ws := t.TempDir()
	absPath := filepath.Join(ws, "test.txt")
	abs, err := ResolvePath(absPath, ws)
	if err != nil {
		t.Fatalf("ResolvePath: %v", err)
	}
	if abs != absPath {
		t.Errorf("got %s, want %s", abs, absPath)
	}
}

func TestGenerateDiff(t *testing.T) {
	old := []string{"a", "b", "c"}
	new := []string{"a", "X", "c"}
	diff := generateUnifiedDiff(old, new, 3)
	if !strings.Contains(diff, "-b") {
		t.Error("diff should contain removed line")
	}
	if !strings.Contains(diff, "+X") {
		t.Error("diff should contain added line")
	}
}
