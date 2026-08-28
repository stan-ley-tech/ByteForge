package environments

import "strings"

// UnresolvedVariableError is returned by Render in strict mode when one or
// more placeholders have no matching variable.
type UnresolvedVariableError struct {
	Names []string
}

func (e *UnresolvedVariableError) Error() string {
	return "environments: unresolved variable(s): " + strings.Join(e.Names, ", ")
}
