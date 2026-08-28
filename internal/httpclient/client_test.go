package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestDo_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", "yes")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := New(DefaultConfig())
	resp, err := c.Do(context.Background(), NewRequest("GET", srv.URL))
	if err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if resp.Header.Get("X-Test") != "yes" {
		t.Fatalf("missing expected response header")
	}
	if string(resp.Body) != `{"ok":true}` {
		t.Fatalf("body = %q", resp.Body)
	}
	if resp.Duration <= 0 {
		t.Fatalf("expected a positive duration")
	}
}

func TestDo_QueryAndHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("id") != "42" {
			t.Errorf("query param id = %q, want 42", r.URL.Query().Get("id"))
		}
		if r.Header.Get("X-Custom") != "value" {
			t.Errorf("header X-Custom = %q, want value", r.Header.Get("X-Custom"))
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(DefaultConfig())
	req := NewRequest("GET", srv.URL).WithQuery("id", "42").WithHeader("X-Custom", "value")
	if _, err := c.Do(context.Background(), req); err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
}

func TestDo_RetriesOnRetryableStatus(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.Retry = RetryPolicy{
		MaxAttempts:      3,
		BaseDelay:        time.Millisecond,
		MaxDelay:         5 * time.Millisecond,
		RetryStatusCodes: map[int]bool{503: true},
	}
	c := New(cfg)

	resp, err := c.Do(context.Background(), NewRequest("GET", srv.URL))
	if err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 after retries", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Fatalf("attempts = %d, want 3", got)
	}
}

func TestDo_DoesNotRetryNonRetryableStatus(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := New(DefaultConfig())
	resp, err := c.Do(context.Background(), NewRequest("GET", srv.URL))
	if err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Fatalf("attempts = %d, want 1 (404 is not retryable)", got)
	}
}

func TestDo_ContextCancellationAbortsRetries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.Retry = RetryPolicy{
		MaxAttempts:      5,
		BaseDelay:        50 * time.Millisecond,
		MaxDelay:         50 * time.Millisecond,
		RetryStatusCodes: map[int]bool{503: true},
	}
	c := New(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := c.Do(ctx, NewRequest("GET", srv.URL))
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected an error from cancelled context")
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("Do took %v, expected cancellation to cut retries short", elapsed)
	}
}

func TestDo_TimeoutProducesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.Timeout = 10 * time.Millisecond
	cfg.Retry = NoRetry()
	c := New(cfg)

	_, err := c.Do(context.Background(), NewRequest("GET", srv.URL))
	if err == nil {
		t.Fatal("expected a timeout error")
	}
}

func TestDo_MaxResponseBytesTruncates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("0123456789"))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.MaxResponseBytes = 4
	cfg.Retry = NoRetry()
	c := New(cfg)

	resp, err := c.Do(context.Background(), NewRequest("GET", srv.URL))
	if err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	if !resp.Truncated {
		t.Fatal("expected Truncated to be true")
	}
	if len(resp.Body) != 4 {
		t.Fatalf("body length = %d, want 4", len(resp.Body))
	}
}

func TestStream_ReturnsOpenBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("streamed"))
	}))
	defer srv.Close()

	c := New(DefaultConfig())
	stream, err := c.Stream(context.Background(), NewRequest("GET", srv.URL))
	if err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}
	defer stream.Close()

	buf := make([]byte, 8)
	n, err := stream.Body.Read(buf)
	if err != nil && n == 0 {
		t.Fatalf("read from stream: %v", err)
	}
	if string(buf[:n]) != "streamed"[:n] {
		t.Fatalf("unexpected stream content: %q", buf[:n])
	}
}
