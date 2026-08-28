// Package httpclient wraps net/http with the behavior an API testing tool
// needs: pooled connections shared across many requests, bounded timeouts,
// automatic retries on transient failures, and cancellation that actually
// aborts in-flight work instead of just giving up on waiting for it.
package httpclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"
)

// Config controls how a Client talks to the network.
type Config struct {
	// Timeout bounds a single request attempt end to end: connect, TLS
	// handshake, redirects and reading the response body.
	Timeout time.Duration

	// MaxIdleConns and MaxIdleConnsPerHost tune the shared connection pool.
	// Reusing connections matters here because a test run can fire dozens of
	// requests at the same host in quick succession.
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	IdleConnTimeout     time.Duration

	// InsecureSkipVerify disables TLS certificate verification. It exists
	// for local development against self-signed endpoints and should never
	// be the default.
	InsecureSkipVerify bool

	// MaxResponseBytes caps how much of a response body is buffered into
	// memory by Do. Zero means unlimited. Use Stream for bodies that should
	// never be fully buffered regardless of size.
	MaxResponseBytes int64

	// MaxRedirects caps how many redirects a single attempt will follow.
	MaxRedirects int

	Retry RetryPolicy
}

// DefaultConfig returns sane defaults for interactive use: a generous but
// finite timeout, a small pool, and a conservative retry policy.
func DefaultConfig() Config {
	return Config{
		Timeout:             30 * time.Second,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		MaxResponseBytes:    10 << 20, // 10 MiB
		MaxRedirects:        10,
		Retry:               DefaultRetryPolicy(),
	}
}

// Client executes HTTP requests with pooling, retries and cancellation.
// A Client is safe for concurrent use and is meant to be reused across many
// requests rather than constructed per call, so the underlying transport's
// connection pool actually does its job.
type Client struct {
	http *http.Client
	cfg  Config
}

// New builds a Client from cfg. Zero-valued fields fall back to
// DefaultConfig's values where that makes sense.
func New(cfg Config) *Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultConfig().Timeout
	}
	if cfg.MaxRedirects <= 0 {
		cfg.MaxRedirects = DefaultConfig().MaxRedirects
	}

	transport := &http.Transport{
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.MaxIdleConnsPerHost,
		IdleConnTimeout:     cfg.IdleConnTimeout,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: cfg.InsecureSkipVerify},
	}

	maxRedirects := cfg.MaxRedirects
	return &Client{
		cfg: cfg,
		http: &http.Client{
			Transport: transport,
			Timeout:   cfg.Timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= maxRedirects {
					return fmt.Errorf("httpclient: stopped after %d redirects", maxRedirects)
				}
				return nil
			},
		},
	}
}

// Do sends req, retrying according to the Client's RetryPolicy, and returns
// the buffered Response. The context governs the entire operation including
// retries: cancelling it aborts any in-flight attempt and stops further
// retries immediately.
func (c *Client) Do(ctx context.Context, req *Request) (*Response, error) {
	attempts := c.cfg.Retry.MaxAttempts
	if attempts < 1 {
		attempts = 1
	}

	var lastResp *Response
	var lastErr error

	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			if err := c.cfg.Retry.wait(ctx, attempt); err != nil {
				return nil, err
			}
		}

		resp, err := c.do(ctx, req)
		if err != nil {
			lastErr = err
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			if !c.cfg.Retry.shouldRetryError(err) {
				return nil, err
			}
			continue
		}

		lastResp, lastErr = resp, nil
		if !c.cfg.Retry.shouldRetryStatus(resp.StatusCode) {
			return resp, nil
		}
	}

	// Every attempt either errored (returned above) or came back with a
	// retryable status code. Surface the last response we actually got.
	return lastResp, lastErr
}

// do performs exactly one attempt: build the *http.Request, send it, buffer
// the body up to MaxResponseBytes, and time the round trip.
func (c *Client) do(ctx context.Context, req *Request) (*Response, error) {
	httpReq, err := req.toHTTP(ctx)
	if err != nil {
		return nil, &RequestError{URL: req.URL, Err: err}
	}

	start := time.Now()
	httpResp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, &RequestError{URL: req.URL, Err: err}
	}
	defer httpResp.Body.Close()

	body, truncated, err := readLimited(httpResp.Body, c.cfg.MaxResponseBytes)
	duration := time.Since(start)
	if err != nil {
		return nil, &RequestError{URL: req.URL, Err: err}
	}

	return &Response{
		StatusCode: httpResp.StatusCode,
		Status:     httpResp.Status,
		Header:     httpResp.Header,
		Body:       body,
		Truncated:  truncated,
		Duration:   duration,
		Proto:      httpResp.Proto,
	}, nil
}

// Stream sends req and returns the live response with its body left open
// for the caller to read incrementally, instead of buffering it. This is
// how large downloads and chunked/streamed responses are handled without
// pulling the whole payload into memory. The caller owns the returned
// StreamResponse and must call Close when done, which also releases the
// underlying connection back to the pool.
func (c *Client) Stream(ctx context.Context, req *Request) (*StreamResponse, error) {
	httpReq, err := req.toHTTP(ctx)
	if err != nil {
		return nil, &RequestError{URL: req.URL, Err: err}
	}

	start := time.Now()
	httpResp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, &RequestError{URL: req.URL, Err: err}
	}

	return &StreamResponse{
		StatusCode: httpResp.StatusCode,
		Status:     httpResp.Status,
		Header:     httpResp.Header,
		Proto:      httpResp.Proto,
		Started:    start,
		Body:       httpResp.Body,
	}, nil
}

// CloseIdleConnections releases pooled connections that are currently idle.
// Callers that create a Client per environment (rather than one shared
// instance) should call this on teardown.
func (c *Client) CloseIdleConnections() {
	c.http.CloseIdleConnections()
}
