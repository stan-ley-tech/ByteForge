package runner

import (
	"time"

	"github.com/stan-ley-tech/ByteForge/internal/assertions"
)

// StepResult is the outcome of executing one request within a run, ready to
// be rendered as a CLI line, an API response, or a live WebSocket event.
type StepResult struct {
	Request    string              `json:"request"`
	Method     string              `json:"method"`
	URL        string              `json:"url"`
	Status     int                 `json:"status,omitempty"`
	Duration   time.Duration       `json:"durationMs"`
	Assertions []assertions.Result `json:"assertions,omitempty"`
	Passed     bool                `json:"passed"`
	// Error is set when the request itself couldn't be completed (a
	// template failed to render, the connection failed, extraction failed)
	// as distinct from an assertion simply not passing.
	Error string `json:"error,omitempty"`
}

// Report is the result of running a whole collection: one StepResult per
// request, in execution order, plus pass/fail totals.
type Report struct {
	CollectionName string        `json:"collectionName"`
	Environment    string        `json:"environment,omitempty"`
	Steps          []StepResult  `json:"steps"`
	Started        time.Time     `json:"started"`
	Duration       time.Duration `json:"durationMs"`
	Passed         int           `json:"passed"`
	Failed         int           `json:"failed"`
}

// AllPassed reports whether every step in the run passed. The CLI's `test`
// command uses this to decide its process exit code, which is what makes
// ByteForge usable as a CI gate.
func (r *Report) AllPassed() bool {
	return r.Failed == 0
}
