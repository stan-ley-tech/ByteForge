// Package assertions parses and evaluates the small expression language
// used to check a response, e.g.:
//
//	status == 200
//	response.body.id exists
//	response.body.email == "test@example.com"
//	response.time < 500ms
//
// An Assertion is parsed once and evaluated against a Context built from an
// executed request's response, producing a pass/fail Result with a
// human-readable message for test reports.
package assertions

import (
	"fmt"
	"net/http"
	"time"
)

// Context is everything an assertion can inspect. It's built by the runner
// package from an httpclient.Response after a request executes.
type Context struct {
	Status  int
	Headers http.Header
	Body    []byte
	Time    time.Duration
}

// Result is the outcome of evaluating a single Assertion.
type Result struct {
	Expression string
	Passed     bool
	Message    string
	Err        error
}

// Assertion is a parsed expression, ready to be evaluated repeatedly
// against different Contexts.
type Assertion struct {
	raw   string
	left  operand
	op    operator
	right operand
}

// String returns the original expression text.
func (a Assertion) String() string {
	return a.raw
}

// Parse compiles a single assertion expression. It returns an error for
// anything the grammar doesn't recognize, rather than silently accepting a
// malformed assertion that would always (or never) pass.
func Parse(expr string) (Assertion, error) {
	return parse(expr)
}

// ParseAll parses one assertion per non-blank line of exprs.
func ParseAll(exprs []string) ([]Assertion, error) {
	out := make([]Assertion, 0, len(exprs))
	for _, e := range exprs {
		if isBlank(e) {
			continue
		}
		a, err := Parse(e)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

func isBlank(s string) bool {
	for _, r := range s {
		if r != ' ' && r != '\t' {
			return false
		}
	}
	return true
}

// Evaluate runs the assertion against ctx.
func (a Assertion) Evaluate(ctx Context) Result {
	res := Result{Expression: a.raw}

	leftVal, err := a.left.resolve(ctx)
	if err != nil {
		res.Err = err
		res.Message = fmt.Sprintf("could not evaluate %s: %v", a.left.describe(), err)
		return res
	}

	if a.op == opExists || a.op == opNotExists {
		exists := leftVal != nil
		res.Passed = exists == (a.op == opExists)
		res.Message = existsMessage(a.left, exists, a.op == opExists)
		return res
	}

	rightVal, err := a.right.resolve(ctx)
	if err != nil {
		res.Err = err
		res.Message = fmt.Sprintf("could not evaluate %s: %v", a.raw, err)
		return res
	}

	passed, err := compare(leftVal, a.op, rightVal)
	if err != nil {
		res.Err = err
		res.Message = fmt.Sprintf("%s: %v", a.raw, err)
		return res
	}

	res.Passed = passed
	if passed {
		res.Message = fmt.Sprintf("%s %s %s", formatVal(leftVal), a.op.symbol(), formatVal(rightVal))
	} else {
		res.Message = fmt.Sprintf("got %s, expected %s %s", formatVal(leftVal), a.op.symbol(), formatVal(rightVal))
	}
	return res
}

func existsMessage(left operand, exists bool, want bool) string {
	if exists == want {
		if want {
			return fmt.Sprintf("%s exists", left.describe())
		}
		return fmt.Sprintf("%s does not exist", left.describe())
	}
	if exists {
		return fmt.Sprintf("%s exists, expected it not to", left.describe())
	}
	return fmt.Sprintf("%s does not exist, expected it to", left.describe())
}
