package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestExportCommand_NormalizesAndAssignsIDs(t *testing.T) {
	raw := `{"name": "Export Demo", "requests": [{"name": "Ping", "method": "GET", "url": "https://example.com"}]}`
	path := filepath.Join(t.TempDir(), "collection.json")
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	cmd := newExportCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("export command returned error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("export output is not valid JSON: %v\n%s", err, out.String())
	}
	if decoded["id"] == nil || decoded["id"] == "" {
		t.Fatal("export did not assign a collection ID")
	}
}

func TestExportCommand_WritesToOutputFile(t *testing.T) {
	raw := `{"name": "Export Demo", "requests": []}`
	path := filepath.Join(t.TempDir(), "collection.json")
	os.WriteFile(path, []byte(raw), 0o644)

	outPath := filepath.Join(t.TempDir(), "out.json")

	cmd := newExportCommand()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetArgs([]string{path, "--output", outPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("export command returned error: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("expected output file to be written: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("output file is empty")
	}
}

func TestExportCommand_RejectsInvalidCollection(t *testing.T) {
	raw := `{"name": "Bad", "requests": [{"name": "X", "method": "TRACE", "url": "https://example.com"}]}`
	path := filepath.Join(t.TempDir(), "collection.json")
	os.WriteFile(path, []byte(raw), 0o644)

	cmd := newExportCommand()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetArgs([]string{path})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected export to reject an invalid collection")
	}
}
