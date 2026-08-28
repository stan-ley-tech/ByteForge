package assertions

import (
	"fmt"
	"strings"
	"time"
)

func compare(left any, op operator, right any) (bool, error) {
	switch op {
	case opEquals:
		return equal(left, right), nil
	case opNotEquals:
		return !equal(left, right), nil
	case opContains:
		return contains(left, right)
	case opLessThan, opLessOrEqual, opGreaterThan, opGreaterOrEqual:
		return numericCompare(left, op, right)
	default:
		return false, fmt.Errorf("unsupported operator %s", op.symbol())
	}
}

// toFloat64 coerces the numeric types that can show up on either side of a
// comparison — a JSON number (float64), an int status code, or a
// time.Duration measured in nanoseconds — onto a common scale.
func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case time.Duration:
		return float64(n), true
	default:
		return 0, false
	}
}

func equal(a, b any) bool {
	if af, aok := toFloat64(a); aok {
		if bf, bok := toFloat64(b); bok {
			return af == bf
		}
	}
	if as, aok := a.(string); aok {
		if bs, bok := b.(string); bok {
			return as == bs
		}
	}
	if ab, aok := a.(bool); aok {
		if bb, bok := b.(bool); bok {
			return ab == bb
		}
	}
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return fmt.Sprint(a) == fmt.Sprint(b)
}

func numericCompare(left any, op operator, right any) (bool, error) {
	lf, lok := toFloat64(left)
	rf, rok := toFloat64(right)
	if !lok || !rok {
		return false, fmt.Errorf("cannot compare %s and %s with %s", describeType(left), describeType(right), op.symbol())
	}

	switch op {
	case opLessThan:
		return lf < rf, nil
	case opLessOrEqual:
		return lf <= rf, nil
	case opGreaterThan:
		return lf > rf, nil
	case opGreaterOrEqual:
		return lf >= rf, nil
	default:
		return false, fmt.Errorf("unsupported operator %s", op.symbol())
	}
}

func contains(left, right any) (bool, error) {
	switch l := left.(type) {
	case string:
		rs, ok := right.(string)
		if !ok {
			return false, fmt.Errorf("contains: right-hand side must be a string when the left side is a string")
		}
		return strings.Contains(l, rs), nil
	case []any:
		for _, item := range l {
			if equal(item, right) {
				return true, nil
			}
		}
		return false, nil
	default:
		return false, fmt.Errorf("contains: left-hand side must be a string or array, got %s", describeType(left))
	}
}

func describeType(v any) string {
	if v == nil {
		return "null"
	}
	switch v.(type) {
	case string:
		return "string"
	case bool:
		return "boolean"
	case float64, int, int64:
		return "number"
	case time.Duration:
		return "duration"
	case []any:
		return "array"
	default:
		return fmt.Sprintf("%T", v)
	}
}
