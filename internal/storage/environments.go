package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/stan-ley-tech/ByteForge/internal/environments"
	"github.com/stan-ley-tech/ByteForge/internal/idgen"
)

// SaveEnvironment inserts env or overwrites it in place if its ID already
// exists. A missing ID is assigned before saving.
func (s *Store) SaveEnvironment(ctx context.Context, env *environments.Environment) error {
	if env.ID == "" {
		env.ID = idgen.New()
	}

	data, err := json.Marshal(env.Variables)
	if err != nil {
		return fmt.Errorf("storage: marshal environment: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO environments (id, name, data, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			data = excluded.data,
			updated_at = excluded.updated_at
	`, env.ID, env.Name, string(data), now, now)
	if err != nil {
		return fmt.Errorf("storage: save environment: %w", err)
	}
	return nil
}

// GetEnvironment returns the environment with the given ID, or ErrNotFound.
func (s *Store) GetEnvironment(ctx context.Context, id string) (*environments.Environment, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, name, data FROM environments WHERE id = ?`, id)
	return scanEnvironment(row)
}

// GetEnvironmentByName looks up an environment by its (unique) name, which
// is how the CLI's --env flag resolves an environment.
func (s *Store) GetEnvironmentByName(ctx context.Context, name string) (*environments.Environment, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, name, data FROM environments WHERE name = ?`, name)
	return scanEnvironment(row)
}

func scanEnvironment(row *sql.Row) (*environments.Environment, error) {
	var id, name, data string
	err := row.Scan(&id, &name, &data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("storage: get environment: %w", err)
	}

	env := &environments.Environment{ID: id, Name: name}
	if err := json.Unmarshal([]byte(data), &env.Variables); err != nil {
		return nil, fmt.Errorf("storage: decode environment: %w", err)
	}
	return env, nil
}

// ListEnvironments returns every saved environment, ordered by name.
func (s *Store) ListEnvironments(ctx context.Context) ([]*environments.Environment, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, data FROM environments ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("storage: list environments: %w", err)
	}
	defer rows.Close()

	out := []*environments.Environment{}
	for rows.Next() {
		var id, name, data string
		if err := rows.Scan(&id, &name, &data); err != nil {
			return nil, fmt.Errorf("storage: scan environment: %w", err)
		}
		env := &environments.Environment{ID: id, Name: name}
		if err := json.Unmarshal([]byte(data), &env.Variables); err != nil {
			return nil, fmt.Errorf("storage: decode environment: %w", err)
		}
		out = append(out, env)
	}
	return out, rows.Err()
}

// DeleteEnvironment removes an environment by ID, returning ErrNotFound if
// it didn't exist.
func (s *Store) DeleteEnvironment(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM environments WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("storage: delete environment: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
