package environments

import (
	"regexp"
	"strings"
)

var placeholderPattern = regexp.MustCompile(`\{\{\s*([A-Za-z0-9_.]+)\s*\}\}`)

// Render substitutes every {{name}} placeholder in s with the value lookup
// returns for name. In strict mode, any placeholder that lookup can't
// resolve is reported as an error and the original string is discarded;
// otherwise unresolved placeholders are left untouched so a request can
// still be previewed without a fully populated environment.
func Render(s string, lookup func(name string) (string, bool), strict bool) (string, error) {
	if !strings.Contains(s, "{{") {
		return s, nil
	}

	var missing []string
	result := placeholderPattern.ReplaceAllStringFunc(s, func(match string) string {
		name := placeholderPattern.FindStringSubmatch(match)[1]
		if val, ok := lookup(name); ok {
			return val
		}
		missing = append(missing, name)
		return match
	})

	if strict && len(missing) > 0 {
		return "", &UnresolvedVariableError{Names: missing}
	}
	return result, nil
}

// Names returns the set of placeholder names referenced in s, in order of
// first appearance, without resolving them. Useful for UI hints like
// "this request uses: BASE_URL, ACCESS_TOKEN".
func Names(s string) []string {
	matches := placeholderPattern.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]bool, len(matches))
	names := make([]string, 0, len(matches))
	for _, m := range matches {
		if !seen[m[1]] {
			seen[m[1]] = true
			names = append(names, m[1])
		}
	}
	return names
}
