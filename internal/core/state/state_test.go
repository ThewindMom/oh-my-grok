package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestWriteReadRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "state", "test.json")

	data := map[string]any{"key": "value", "num": 42}
	if err := WriteJSON(path, data); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	var got map[string]any
	if err := ReadJSON(path, &got); err != nil {
		t.Fatalf("ReadJSON: %v", err)
	}
	if got["key"] != "value" {
		t.Errorf("key = %v, want value", got["key"])
	}
}

func TestAtomicWriteNoTempLeft(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "state", "test.json")

	if err := WriteJSON(path, map[string]any{"a": 1}); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	entries, _ := os.ReadDir(filepath.Dir(path))
	for _, e := range entries {
		if filepath.Base(e.Name()) != "test.json" {
			t.Errorf("unexpected temp file left: %s", e.Name())
		}
	}
}

func TestConcurrentWrites(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "state", "concurrent.json")

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			data := map[string]any{"writer": n}
			if err := WriteJSON(path, data); err != nil {
				t.Errorf("writer %d: %v", n, err)
			}
		}(i)
	}
	wg.Wait()

	// File should be valid JSON after concurrent writes
	var got map[string]any
	if err := ReadJSON(path, &got); err != nil {
		t.Fatalf("corrupt after concurrent writes: %v", err)
	}
}

func TestReadWithBackupValid(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "valid.json")
	os.WriteFile(path, []byte(`{"ok": true}`), 0o644)

	data, backup, err := ReadWithBackup(path)
	if err != nil {
		t.Fatalf("ReadWithBackup: %v", err)
	}
	if backup != "" {
		t.Errorf("backup should be empty for valid file, got %s", backup)
	}
	if string(data) != `{"ok": true}` {
		t.Errorf("unexpected data: %s", data)
	}
}

func TestReadWithBackupCorrupt(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "corrupt.json")
	os.WriteFile(path, []byte(`{not valid json`), 0o644)

	_, backup, err := ReadWithBackup(path)
	if err == nil {
		t.Fatal("expected error for corrupt JSON")
	}
	if backup == "" {
		t.Error("expected backup path for corrupt file")
	}
	if _, err := os.Stat(backup); err != nil {
		t.Errorf("backup file should exist: %v", err)
	}
}

func TestMigrateNoFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "missing.json")
	migrations := map[int]func(map[string]any) map[string]any{
		1: func(d map[string]any) map[string]any {
			d["migrated"] = true
			return d
		},
	}
	if err := Migrate(path, migrations); err != nil {
		t.Fatalf("Migrate on missing file should not error: %v", err)
	}
}

func TestMigrateApplies(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "state.json")
	// v1 format
	os.WriteFile(path, []byte(`{"schemaVersion": 1, "oldField": "data"}`), 0o644)

	migrations := map[int]func(map[string]any) map[string]any{
		1: func(d map[string]any) map[string]any {
			d["newField"] = d["oldField"]
			delete(d, "oldField")
			return d
		},
	}
	if err := Migrate(path, migrations); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	var got map[string]any
	if err := ReadJSON(path, &got); err != nil {
		t.Fatalf("ReadJSON: %v", err)
	}
	if got["schemaVersion"].(float64) != 2 {
		t.Errorf("schemaVersion = %v, want 2", got["schemaVersion"])
	}
	if got["newField"] != "data" {
		t.Errorf("newField = %v, want data", got["newField"])
	}
	if _, exists := got["oldField"]; exists {
		t.Error("oldField should be removed after migration")
	}
}

func TestMigrateAlreadyCurrent(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "state.json")
	os.WriteFile(path, []byte(`{"schemaVersion": 2, "data": "x"}`), 0o644)

	migrations := map[int]func(map[string]any) map[string]any{
		1: func(d map[string]any) map[string]any {
			d["migrated"] = true
			return d
		},
	}
	if err := Migrate(path, migrations); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	var got map[string]any
	ReadJSON(path, &got)
	if _, exists := got["migrated"]; exists {
		t.Error("should not migrate already-current file")
	}
}

func TestMigrateCreatesBackup(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "state.json")
	original := `{"schemaVersion": 1, "val": "old"}`
	os.WriteFile(path, []byte(original), 0o644)

	migrations := map[int]func(map[string]any) map[string]any{
		1: func(d map[string]any) map[string]any {
			d["val"] = "new"
			return d
		},
	}
	if err := Migrate(path, migrations); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Find backup file
	entries, _ := os.ReadDir(tmp)
	foundBackup := false
	for _, e := range entries {
		name := e.Name()
		if len(name) > len("state.json") && name[:10] == "state.json" {
			bdata, _ := os.ReadFile(filepath.Join(tmp, name))
			if string(bdata) == original {
				foundBackup = true
			}
		}
	}
	if !foundBackup {
		t.Error("expected a pre-migration backup file")
	}
}

func TestPreservePermissions(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "perms.json")
	os.WriteFile(path, []byte(`{"v":1}`), 0o600)

	WriteJSON(path, map[string]any{"v": 2})

	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o600 {
		t.Errorf("permissions changed: got %o, want 0600", info.Mode().Perm())
	}
}

func TestJSONRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "f.json")

	f := File{
		SchemaVersion: CurrentSchemaVersion,
		UpdatedAt:    "2026-01-01T00:00:00Z",
		Data:         map[string]any{"test": true},
	}
	if err := WriteJSON(path, f); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	var got File
	if err := ReadJSON(path, &got); err != nil {
		t.Fatalf("ReadJSON: %v", err)
	}
	if got.SchemaVersion != CurrentSchemaVersion {
		t.Errorf("schemaVersion = %d, want %d", got.SchemaVersion, CurrentSchemaVersion)
	}
}

func TestConcurrentReadDuringWrite(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "rw.json")
	WriteJSON(path, map[string]any{"init": true})

	var wg sync.WaitGroup
	done := make(chan error, 2)

	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			if err := WriteJSON(path, map[string]any{"i": i}); err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			var got map[string]any
			if err := ReadJSON(path, &got); err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()
	wg.Wait()
	close(done)
	for err := range done {
		if err != nil {
			t.Errorf("concurrent error: %v", err)
		}
	}
}

func TestReadMissingFile(t *testing.T) {
	_, err := Read("/nonexistent/file.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestCorruptJSONUnmarshal(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "bad.json")
	os.WriteFile(path, []byte(`{broken`), 0o644)

	var v map[string]any
	if err := ReadJSON(path, &v); err == nil {
		t.Error("expected error for corrupt JSON")
	}

	// Verify it's a JSON error
	data, _ := Read(path)
	var probe any
	if err := json.Unmarshal(data, &probe); err == nil {
		t.Error("json.Unmarshal should fail on corrupt data")
	}
}
