package assertions

import (
	"fmt"
	"strconv"
	"time"
)

// formatVal renders a resolved value for test-report messages the way a
// user would type it, not the way Go's %v would print it (JSON numbers in
// particular come back as float64, and "200" reads a lot better than
// "200.000000" in a failure message).
func formatVal(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case string:
		return strconv.Quote(x)
	case time.Duration:
		return x.String()
	case float64:
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", x)
	}
}
