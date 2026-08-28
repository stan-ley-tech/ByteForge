package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func writeTempCollection(t *testing.T, baseURL string) string {
	t.Helper()

	raw := map[string]any{
		"name": "CLI Test",
		"requests": []map[string]any{
			{
				"name":       "Get",
				"method":     "GET",
				"url":        baseURL + "/users/1",
				"assertions": []string{"status == 200", "response.body.id exists"},
			},
		},
	}
	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}

	path := filepath.Join(t.TempDir(), "collection.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func newTestUpstream(t *testing.T, status int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write([]byte(`{"id": 1}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestRunCommand_PrintsReportAndExitsCleanly(t *testing.T) {
	upstream := newTestUpstream(t, http.StatusOK)
	path := writeTempCollection(t, upstream.URL)

	cmd := newRunCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("run command returned error: %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("1/1 PASSED")) {
		t.Fatalf("output missing pass summary:\n%s", out.String())
	}
}

func TestTestCommand_FailsBuildOnAssertionFailure(t *testing.T) {
	upstream := newTestUpstream(t, http.StatusInternalServerError)
	path := writeTempCollection(t, upstream.URL)

	cmd := newTestCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{path})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected the test command to fail when an assertion fails")
	}
}

func TestTestCommand_PassesWhenAssertionsHold(t *testing.T) {
	upstream := newTestUpstream(t, http.StatusOK)
	path := writeTempCollection(t, upstream.URL)

	cmd := newTestCommand()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetArgs([]string{path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestRunCommand_RejectsMissingFile(t *testing.T) {
	cmd := newRunCommand()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetArgs([]string{filepath.Join(t.TempDir(), "does-not-exist.json")})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected an error for a missing collection file")
	}
}

func TestRunCommand_AppliesVarOverrides(t *testing.T) {
	upstream := newTestUpstream(t, http.StatusOK)

	raw := map[string]any{
		"name": "Templated",
		"requests": []map[string]any{
			{"name": "Get", "method": "GET", "url": "{{BASE_URL}}/users/1", "assertions": []string{"status == 200"}},
		},
	}
	data, _ := json.Marshal(raw)
	path := filepath.Join(t.TempDir(), "collection.json")
	os.WriteFile(path, data, 0o644)

	cmd := newRunCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{path, "--var", "BASE_URL=" + upstream.URL})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("run command returned error: %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("1/1 PASSED")) {
		t.Fatalf("--var override did not resolve the URL template:\n%s", out.String())
	}
}
