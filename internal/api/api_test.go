package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stan-ley-tech/ByteForge/internal/httpclient"
	"github.com/stan-ley-tech/ByteForge/internal/runner"
	"github.com/stan-ley-tech/ByteForge/internal/storage"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	store, err := storage.Open("")
	if err != nil {
		t.Fatalf("storage.Open returned error: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	cfg := httpclient.DefaultConfig()
	cfg.Retry = httpclient.NoRetry()
	rn := runner.New(httpclient.New(cfg))

	return NewServer(store, rn)
}

func doJSON(t *testing.T, srv *Server, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var reader *bytes.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(data)
	} else {
		reader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec
}

func TestHealthz(t *testing.T) {
	srv := newTestServer(t)
	rec := doJSON(t, srv, http.MethodGet, "/healthz", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestCollectionsCRUD(t *testing.T) {
	srv := newTestServer(t)

	create := doJSON(t, srv, http.MethodPost, "/api/collections", map[string]any{
		"name": "Demo",
		"requests": []map[string]any{
			{"name": "Ping", "method": "GET", "url": "https://example.com"},
		},
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201, body=%s", create.Code, create.Body.String())
	}

	var created map[string]any
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("created collection has no id")
	}

	get := doJSON(t, srv, http.MethodGet, "/api/collections/"+id, nil)
	if get.Code != http.StatusOK {
		t.Fatalf("get status = %d, want 200", get.Code)
	}

	list := doJSON(t, srv, http.MethodGet, "/api/collections", nil)
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200", list.Code)
	}
	var listed []map[string]any
	if err := json.Unmarshal(list.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("len(listed) = %d, want 1", len(listed))
	}

	del := doJSON(t, srv, http.MethodDelete, "/api/collections/"+id, nil)
	if del.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", del.Code)
	}

	missing := doJSON(t, srv, http.MethodGet, "/api/collections/"+id, nil)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("get after delete status = %d, want 404", missing.Code)
	}
}

func TestCreateCollection_RejectsInvalidMethod(t *testing.T) {
	srv := newTestServer(t)
	rec := doJSON(t, srv, http.MethodPost, "/api/collections", map[string]any{
		"name": "Demo",
		"requests": []map[string]any{
			{"name": "Bad", "method": "TRACE", "url": "https://example.com"},
		},
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body=%s", rec.Code, rec.Body.String())
	}
}

func TestEnvironments_ListNeverLeaksSecrets(t *testing.T) {
	srv := newTestServer(t)

	doJSON(t, srv, http.MethodPost, "/api/environments", map[string]any{
		"name": "Production",
		"variables": map[string]any{
			"API_KEY": map[string]any{"value": "sk-super-secret", "secret": true},
		},
	})

	list := doJSON(t, srv, http.MethodGet, "/api/environments", nil)
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200", list.Code)
	}
	if bytes.Contains(list.Body.Bytes(), []byte("sk-super-secret")) {
		t.Fatalf("environment listing leaked a secret value: %s", list.Body.String())
	}
}

func TestRunCollection_AgainstLiveServer(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id": 1}`))
	}))
	defer upstream.Close()

	srv := newTestServer(t)

	create := doJSON(t, srv, http.MethodPost, "/api/collections", map[string]any{
		"name": "Demo",
		"requests": []map[string]any{
			{
				"name":       "Get",
				"method":     "GET",
				"url":        upstream.URL,
				"assertions": []string{"status == 200", "response.body.id exists"},
			},
		},
	})
	var created map[string]any
	json.Unmarshal(create.Body.Bytes(), &created)
	id := created["id"].(string)

	run := doJSON(t, srv, http.MethodPost, "/api/collections/"+id+"/run", map[string]any{})
	if run.Code != http.StatusOK {
		t.Fatalf("run status = %d, want 200, body=%s", run.Code, run.Body.String())
	}

	var report map[string]any
	if err := json.Unmarshal(run.Body.Bytes(), &report); err != nil {
		t.Fatalf("decode run response: %v", err)
	}
	if report["failed"].(float64) != 0 {
		t.Fatalf("report has failures: %v", report)
	}

	runs := doJSON(t, srv, http.MethodGet, "/api/collections/"+id+"/runs", nil)
	if runs.Code != http.StatusOK {
		t.Fatalf("runs status = %d, want 200", runs.Code)
	}
	var runList []map[string]any
	json.Unmarshal(runs.Body.Bytes(), &runList)
	if len(runList) != 1 {
		t.Fatalf("len(runList) = %d, want 1 (run should have been persisted)", len(runList))
	}

	embeddedReport, ok := runList[0]["report"].(map[string]any)
	if !ok {
		t.Fatalf("report field is not an embedded JSON object: %v", runList[0]["report"])
	}
	if embeddedReport["collectionName"] != "Demo" {
		t.Fatalf("embedded report mismatch: %v", embeddedReport)
	}
}

func TestSendRequest_RecordsHistory(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	srv := newTestServer(t)
	rec := doJSON(t, srv, http.MethodPost, "/api/requests/send", map[string]any{
		"request": map[string]any{"name": "Ping", "method": "GET", "url": upstream.URL},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("send status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}

	history := doJSON(t, srv, http.MethodGet, "/api/history", nil)
	var entries []map[string]any
	json.Unmarshal(history.Body.Bytes(), &entries)
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
}
