// Package environments models named sets of variables (Development,
// Staging, Production) that requests can reference by name, and the
// template renderer that resolves those references at request time.
package environments

// Variable is a single named value inside an Environment. Secret marks
// values such as API keys, tokens, or passwords that must never be written
// to logs, exported collection files, or the live test-output stream in
// plain text.
type Variable struct {
	Value  string `json:"value"`
	Secret bool   `json:"secret"`
}

// Environment is a named set of variables. Requests reference variables by
// name using {{name}} placeholders resolved against the active Environment.
type Environment struct {
	ID        string              `json:"id"`
	Name      string              `json:"name"`
	Variables map[string]Variable `json:"variables"`
}

// New creates an empty Environment with the given name.
func New(name string) *Environment {
	return &Environment{
		Name:      name,
		Variables: make(map[string]Variable),
	}
}

// Set adds or overwrites a variable.
func (e *Environment) Set(name, value string, secret bool) {
	if e.Variables == nil {
		e.Variables = make(map[string]Variable)
	}
	e.Variables[name] = Variable{Value: value, Secret: secret}
}

// Get returns the variable named name, if present.
func (e *Environment) Get(name string) (Variable, bool) {
	if e == nil {
		return Variable{}, false
	}
	v, ok := e.Variables[name]
	return v, ok
}

// Redacted returns a copy of the environment where every secret variable's
// value has been replaced with a fixed mask. It is safe to log, print, or
// send over the live test-output WebSocket.
func (e *Environment) Redacted() *Environment {
	out := &Environment{ID: e.ID, Name: e.Name, Variables: make(map[string]Variable, len(e.Variables))}
	for name, v := range e.Variables {
		if v.Secret {
			v.Value = "••••••••"
		}
		out.Variables[name] = v
	}
	return out
}

// ExportSafe returns a copy of the environment with secret variables
// stripped entirely, rather than masked, for writing to collection export
// files that a user might commit to source control or share with teammates.
func (e *Environment) ExportSafe() *Environment {
	out := &Environment{ID: e.ID, Name: e.Name, Variables: make(map[string]Variable)}
	for name, v := range e.Variables {
		if v.Secret {
			continue
		}
		out.Variables[name] = v
	}
	return out
}

// Lookup returns a resolver function suitable for Render. overrides take
// precedence over the environment's own variables, which is how values
// extracted mid-chain (e.g. an access_token pulled from a login response)
// win over a stale value already sitting in the environment.
func (e *Environment) Lookup(overrides map[string]string) func(name string) (string, bool) {
	return func(name string) (string, bool) {
		if v, ok := overrides[name]; ok {
			return v, true
		}
		v, ok := e.Get(name)
		if !ok {
			return "", false
		}
		return v.Value, true
	}
}
