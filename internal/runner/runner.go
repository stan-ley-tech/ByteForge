// Package runner is ByteForge's test execution engine: it renders each
// request in a collection against an environment, sends it, evaluates its
// assertions, and threads any extracted variables into the requests that
// follow — the mechanism behind request chaining.
package runner

import (
	"context"
	"time"

	"github.com/stan-ley-tech/ByteForge/internal/assertions"
	"github.com/stan-ley-tech/ByteForge/internal/collections"
	"github.com/stan-ley-tech/ByteForge/internal/environments"
	"github.com/stan-ley-tech/ByteForge/internal/httpclient"
)

// Runner executes collections against a shared, pooled httpclient.Client.
type Runner struct {
	client *httpclient.Client
}

// New builds a Runner around client. Callers should reuse one Client (and
// therefore one Runner) across many runs so its connection pool is worth
// having.
func New(client *httpclient.Client) *Runner {
	return &Runner{client: client}
}

// Options configures how a collection is run.
type Options struct {
	// StopOnFailure halts the chain the moment a step's assertions fail,
	// instead of continuing with requests that likely depend on a variable
	// that failed step should have extracted.
	StopOnFailure bool

	// RequestTimeout bounds each individual request. Zero means the
	// underlying Client's own configured timeout applies.
	RequestTimeout time.Duration

	// OnStep, if set, is invoked synchronously right after each step
	// completes, before the next one starts. It's how the WebSocket
	// handler and the CLI's live output stream progress instead of
	// blocking until the whole run finishes.
	OnStep func(StepResult)
}

// Run executes coll's requests in order against env, threading extracted
// variables from each response into the requests that follow. The returned
// Report is populated even when Run also returns an error (e.g. the
// context was cancelled partway through) so callers can see what completed
// before the interruption.
func (rn *Runner) Run(ctx context.Context, coll *collections.Collection, env *environments.Environment, opts Options) (*Report, error) {
	report := &Report{CollectionName: coll.Name, Started: time.Now()}
	if env != nil {
		report.Environment = env.Name
	}

	vars := make(map[string]string)

	for _, req := range coll.Requests {
		step, err := rn.runStep(ctx, req, env, vars, opts)
		report.Steps = append(report.Steps, step)
		if step.Passed {
			report.Passed++
		} else {
			report.Failed++
		}
		if opts.OnStep != nil {
			opts.OnStep(step)
		}
		if err != nil {
			report.Duration = time.Since(report.Started)
			return report, err
		}
		if !step.Passed && opts.StopOnFailure {
			break
		}
	}

	report.Duration = time.Since(report.Started)
	return report, nil
}

// runStep executes a single request. It only returns a non-nil error for
// conditions the run can't meaningfully continue past (context
// cancellation); everything else — a bad template, a connection failure, a
// failed assertion — is recorded on the StepResult instead, because one bad
// request in a collection shouldn't abort the whole report.
func (rn *Runner) runStep(ctx context.Context, req collections.Request, env *environments.Environment, vars map[string]string, opts Options) (StepResult, error) {
	step := StepResult{Request: req.Name, Method: req.Method}

	hreq, err := renderRequest(req, env, vars)
	if err != nil {
		step.Error = err.Error()
		return step, nil
	}
	step.URL = hreq.URL

	reqCtx := ctx
	if opts.RequestTimeout > 0 {
		var cancel context.CancelFunc
		reqCtx, cancel = context.WithTimeout(ctx, opts.RequestTimeout)
		defer cancel()
	}

	resp, err := rn.client.Do(reqCtx, hreq)
	if err != nil {
		if ctx.Err() != nil {
			return step, ctx.Err()
		}
		step.Error = err.Error()
		return step, nil
	}

	step.Status = resp.StatusCode
	step.Duration = resp.Duration

	parsed, err := assertions.ParseAll(req.Assertions)
	if err != nil {
		step.Error = "invalid assertions: " + err.Error()
		return step, nil
	}

	assertCtx := assertions.Context{
		Status:  resp.StatusCode,
		Headers: resp.Header,
		Body:    resp.Body,
		Time:    resp.Duration,
	}

	step.Passed = true
	for _, a := range parsed {
		result := a.Evaluate(assertCtx)
		step.Assertions = append(step.Assertions, result)
		if !result.Passed {
			step.Passed = false
		}
	}

	if err := extractVariables(resp.Body, req.Extract, vars); err != nil {
		step.Error = err.Error()
		step.Passed = false
	}

	return step, nil
}
