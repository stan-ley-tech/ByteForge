package assertions

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// parse compiles a single assertion expression such as:
//
//	status == 200
//	response.body.email == "test@example.com"
//	response.time < 500ms
//	response.body.id exists
//
// The grammar is intentionally flat: <operand> <operator> [<literal>].
// There is no operator precedence or boolean composition (no "and"/"or") —
// each line is one check, which keeps both the parser and the test report
// output honest about what actually ran.
func parse(expr string) (Assertion, error) {
	trimmed := strings.TrimSpace(expr)
	tokens := tokenize(trimmed)
	if len(tokens) < 2 {
		return Assertion{}, fmt.Errorf("assertions: cannot parse %q: expected an operand and an operator", expr)
	}

	left, err := parseOperand(tokens[0])
	if err != nil {
		return Assertion{}, fmt.Errorf("assertions: %q: %w", expr, err)
	}

	op, remaining, err := parseOperator(tokens[1:])
	if err != nil {
		return Assertion{}, fmt.Errorf("assertions: %q: %w", expr, err)
	}

	a := Assertion{raw: trimmed, left: left, op: op}

	if op == opExists || op == opNotExists {
		if len(remaining) != 0 {
			return Assertion{}, fmt.Errorf("assertions: %q: unexpected tokens after %q", expr, op.symbol())
		}
		return a, nil
	}

	if len(remaining) != 1 {
		return Assertion{}, fmt.Errorf("assertions: %q: expected exactly one value after %q", expr, op.symbol())
	}

	right, err := parseLiteral(remaining[0])
	if err != nil {
		return Assertion{}, fmt.Errorf("assertions: %q: %w", expr, err)
	}
	a.right = right
	return a, nil
}

// tokenize splits on whitespace but keeps double-quoted strings intact as a
// single token, so `response.body.email == "jane doe"` doesn't get split
// inside the literal.
func tokenize(expr string) []string {
	var tokens []string
	var cur strings.Builder
	inQuotes := false

	flush := func() {
		if cur.Len() > 0 {
			tokens = append(tokens, cur.String())
			cur.Reset()
		}
	}

	for _, r := range expr {
		switch {
		case r == '"':
			cur.WriteRune(r)
			inQuotes = !inQuotes
		case r == ' ' && !inQuotes:
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return tokens
}

func parseOperator(tokens []string) (operator, []string, error) {
	if len(tokens) == 0 {
		return 0, nil, fmt.Errorf("missing operator")
	}

	switch tokens[0] {
	case "==":
		return opEquals, tokens[1:], nil
	case "!=":
		return opNotEquals, tokens[1:], nil
	case "<":
		return opLessThan, tokens[1:], nil
	case "<=":
		return opLessOrEqual, tokens[1:], nil
	case ">":
		return opGreaterThan, tokens[1:], nil
	case ">=":
		return opGreaterOrEqual, tokens[1:], nil
	case "contains":
		return opContains, tokens[1:], nil
	case "exists":
		return opExists, tokens[1:], nil
	case "not":
		if len(tokens) >= 2 && tokens[1] == "exists" {
			return opNotExists, tokens[2:], nil
		}
		return 0, nil, fmt.Errorf(`expected "exists" after "not"`)
	default:
		return 0, nil, fmt.Errorf("unknown operator %q", tokens[0])
	}
}

// parseOperand recognizes the left-hand paths an assertion can reference:
// status, response.time, response.header.<Name>, and response.body[.<path>]
// (the "response." prefix is optional on body/header for brevity).
func parseOperand(token string) (operand, error) {
	switch {
	case token == "status" || token == "response.status":
		return operand{kind: kindStatus, raw: token}, nil

	case token == "response.time":
		return operand{kind: kindTime, raw: token}, nil

	case strings.HasPrefix(token, "response.header."):
		name := strings.TrimPrefix(token, "response.header.")
		if name == "" {
			return operand{}, fmt.Errorf("response.header. requires a header name")
		}
		return operand{kind: kindHeader, header: name, raw: token}, nil

	// A response body that is itself a JSON array (a list endpoint) is
	// referenced as response.body[0].id, with no dot before the bracket —
	// bodyPath handles both that and the dotted response.body.id form.
	case token == "response.body" || strings.HasPrefix(token, "response.body.") || strings.HasPrefix(token, "response.body["):
		return operand{kind: kindBodyPath, path: bodyPath(token, "response.body"), raw: token}, nil

	case token == "body" || strings.HasPrefix(token, "body.") || strings.HasPrefix(token, "body["):
		return operand{kind: kindBodyPath, path: bodyPath(token, "body"), raw: token}, nil

	default:
		return operand{}, fmt.Errorf(
			"unrecognized operand %q (expected status, response.time, response.body.<path>, or response.header.<name>)", token)
	}
}

// bodyPath strips the leading "response.body"/"body" operand prefix off
// token, leaving whatever jsonpath.Query expects: either empty (the whole
// body), a dotted path ("id", ".id" -> "id"), or a bracketed one
// ("[0].id") left untouched, since jsonpath tolerates an empty leading
// segment name either way.
func bodyPath(token, prefix string) string {
	return strings.TrimPrefix(strings.TrimPrefix(token, prefix), ".")
}

// parseLiteral recognizes the right-hand values an assertion can compare
// against: quoted strings, booleans, durations (500ms, 2s), and numbers.
func parseLiteral(token string) (operand, error) {
	if len(token) >= 2 && strings.HasPrefix(token, `"`) && strings.HasSuffix(token, `"`) {
		return operand{kind: kindLiteral, literal: token[1 : len(token)-1], raw: token}, nil
	}
	if token == "true" || token == "false" {
		return operand{kind: kindLiteral, literal: token == "true", raw: token}, nil
	}
	if d, err := time.ParseDuration(token); err == nil {
		return operand{kind: kindLiteral, literal: d, raw: token}, nil
	}
	if f, err := strconv.ParseFloat(token, 64); err == nil {
		return operand{kind: kindLiteral, literal: f, raw: token}, nil
	}
	return operand{}, fmt.Errorf("unrecognized literal %q", token)
}
