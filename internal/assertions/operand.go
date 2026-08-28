package assertions

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/stan-ley-tech/ByteForge/internal/jsonpath"
)

type operandKind int

const (
	kindStatus operandKind = iota
	kindTime
	kindHeader
	kindBodyPath
	kindLiteral
)

// operand is one side of an assertion: either something resolved from the
// response (status, timing, a header, a JSON body path) or a literal value
// parsed straight out of the expression text.
type operand struct {
	kind    operandKind
	header  string
	path    string
	literal any
	raw     string
}

func (o operand) describe() string {
	return o.raw
}

// resolve extracts this operand's value from ctx. A nil, nil return means
// "not present" (a missing header, a JSON path with no match) which exists
// and not-exists assertions rely on; every other failure is a real error,
// e.g. a body path evaluated against a non-JSON response.
func (o operand) resolve(ctx Context) (any, error) {
	switch o.kind {
	case kindStatus:
		return ctx.Status, nil

	case kindTime:
		return ctx.Time, nil

	case kindHeader:
		values := ctx.Headers.Values(o.header)
		if len(values) == 0 {
			return nil, nil
		}
		return values[0], nil

	case kindBodyPath:
		if len(ctx.Body) == 0 {
			return nil, nil
		}
		var data any
		if err := json.Unmarshal(ctx.Body, &data); err != nil {
			return nil, fmt.Errorf("response body is not valid JSON: %w", err)
		}
		if o.path == "" {
			return data, nil
		}
		val, err := jsonpath.Query(data, o.path)
		if err != nil {
			if errors.Is(err, jsonpath.ErrNotFound) {
				return nil, nil
			}
			return nil, err
		}
		return val, nil

	case kindLiteral:
		return o.literal, nil
	}
	return nil, fmt.Errorf("assertions: unknown operand kind %d", o.kind)
}
