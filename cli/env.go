package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/stan-ley-tech/ByteForge/internal/environments"
)

// loadEnvironment builds an Environment from an optional JSON file plus any
// --var KEY=VALUE overrides, which are applied last and win. Both are
// optional: a CI job can skip --env entirely and pass secrets straight
// through from its own secret store via --var, so nothing sensitive ever
// touches disk.
func loadEnvironment(path string, overrides []string) (*environments.Environment, error) {
	env := environments.New("cli")

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read environment file: %w", err)
		}
		if err := json.Unmarshal(data, env); err != nil {
			return nil, fmt.Errorf("parse environment file %s: %w", path, err)
		}
	}

	for _, kv := range overrides {
		name, value, ok := strings.Cut(kv, "=")
		if !ok {
			return nil, fmt.Errorf("invalid --var %q, expected KEY=VALUE", kv)
		}
		env.Set(name, value, false)
	}

	return env, nil
}
