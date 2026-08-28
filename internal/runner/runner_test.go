package runner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stan-ley-tech/ByteForge/internal/collections"
	"github.com/stan-ley-tech/ByteForge/internal/environments"
	"github.com/stan-ley-tech/ByteForge/internal/httpclient"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"access_token": "secret-token-123"})
	})
	mux.HandleFunc("/profile", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret-token-123" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": 1, "email": "test@example.com"})
	})
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func newTestRunner() *Runner {
	cfg := httpclient.DefaultConfig()
	cfg.Retry = httpclient.NoRetry()
	return New(httpclient.New(cfg))
}

func TestRun_ChainsExtractedVariableIntoNextRequest(t *testing.T) {
	srv := newTestServer(t)
	env := environments.New("Test")
	env.Set("BASE_URL", srv.URL, false)

	coll := collections.New("Chain")
	coll.AddRequest(collections.Request{
		Name:       "Login",
		Method:     "POST",
		URL:        "{{BASE_URL}}/login",
		Assertions: []string{"status == 200"},
		Extract:    []collections.Extraction{{Variable: "access_token", From: "body.access_token"}},
	})
	coll.AddRequest(collections.Request{
		Name:   "Profile",
		Method: "GET",
		URL:    "{{BASE_URL}}/profile",
		Auth:   collections.Auth{Type: collections.AuthBearer, Token: "{{access_token}}"},
		Assertions: []string{
			"status == 200",
			"response.body.id exists",
			`response.body.email == "test@example.com"`,
		},
	})

	rn := newTestRunner()
	report, err := rn.Run(context.Background(), coll, env, Options{})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if report.Failed != 0 {
		t.Fatalf("expected all steps to pass, report = %+v", report)
	}
	if len(report.Steps) != 2 {
		t.Fatalf("len(report.Steps) = %d, want 2", len(report.Steps))
	}
	if !report.Steps[1].Passed {
		t.Fatalf("profile step failed, meaning the extracted token wasn't chained: %+v", report.Steps[1])
	}
}

func TestRun_StopOnFailureHaltsChain(t *testing.T) {
	srv := newTestServer(t)
	env := environments.New("Test")
	env.Set("BASE_URL", srv.URL, false)

	coll := collections.New("Chain")
	coll.AddRequest(collections.Request{
		Name:       "Profile without token",
		Method:     "GET",
		URL:        "{{BASE_URL}}/profile",
		Assertions: []string{"status == 200"},
	})
	coll.AddRequest(collections.Request{
		Name:   "Login",
		Method: "POST",
		URL:    "{{BASE_URL}}/login",
	})

	rn := newTestRunner()
	report, err := rn.Run(context.Background(), coll, env, Options{StopOnFailure: true})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if len(report.Steps) != 1 {
		t.Fatalf("len(report.Steps) = %d, want 1 (StopOnFailure should have halted the chain)", len(report.Steps))
	}
}

func TestRun_UnresolvedVariableFailsTheStepNotThePanic(t *testing.T) {
	env := environments.New("Test")
	coll := collections.New("Broken")
	coll.AddRequest(collections.Request{
		Name:   "No base url set",
		Method: "GET",
		URL:    "{{BASE_URL}}/users",
	})

	rn := newTestRunner()
	report, err := rn.Run(context.Background(), coll, env, Options{})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if report.Steps[0].Error == "" {
		t.Fatal("expected the step to record a rendering error for the unresolved variable")
	}
	if report.Steps[0].Passed {
		t.Fatal("a step with an unresolved variable should not be reported as passed")
	}
}

func TestRun_OnStepCallbackFiresPerStep(t *testing.T) {
	srv := newTestServer(t)
	env := environments.New("Test")
	env.Set("BASE_URL", srv.URL, false)

	coll := collections.New("Chain")
	coll.AddRequest(collections.Request{Name: "Login", Method: "POST", URL: "{{BASE_URL}}/login"})

	var seen []string
	rn := newTestRunner()
	_, err := rn.Run(context.Background(), coll, env, Options{
		OnStep: func(s StepResult) { seen = append(seen, s.Request) },
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(seen) != 1 || seen[0] != "Login" {
		t.Fatalf("OnStep callbacks = %v, want [Login]", seen)
	}
}

func TestRun_ContextCancellationStopsChain(t *testing.T) {
	srv := newTestServer(t)
	env := environments.New("Test")
	env.Set("BASE_URL", srv.URL, false)

	coll := collections.New("Chain")
	coll.AddRequest(collections.Request{Name: "Slow", Method: "GET", URL: "{{BASE_URL}}/slow"})
	coll.AddRequest(collections.Request{Name: "Login", Method: "POST", URL: "{{BASE_URL}}/login"})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	rn := newTestRunner()
	report, err := rn.Run(ctx, coll, env, Options{})
	if err == nil {
		t.Fatal("expected an error from context cancellation")
	}
	if len(report.Steps) != 1 {
		t.Fatalf("len(report.Steps) = %d, want 1 (cancellation should stop the chain after the in-flight request)", len(report.Steps))
	}
}

func TestRunConcurrent_ExecutesAllRequestsWithBoundedParallelism(t *testing.T) {
	srv := newTestServer(t)
	env := environments.New("Test")
	env.Set("BASE_URL", srv.URL, false)

	reqs := make([]collections.Request, 5)
	for i := range reqs {
		reqs[i] = collections.Request{
			Name:       "Login",
			Method:     "POST",
			URL:        "{{BASE_URL}}/login",
			Assertions: []string{"status == 200"},
		}
	}

	rn := newTestRunner()
	results := rn.RunConcurrent(context.Background(), reqs, env, Options{}, 2)

	if len(results) != 5 {
		t.Fatalf("len(results) = %d, want 5", len(results))
	}
	for i, r := range results {
		if !r.Passed {
			t.Fatalf("result[%d] failed: %+v", i, r)
		}
	}
}
