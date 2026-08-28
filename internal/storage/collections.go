package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/stan-ley-tech/ByteForge/internal/collections"
)

// SaveCollection inserts c or, if a collection with the same ID already
// exists, overwrites it in place. Missing IDs are assigned before saving.
func (s *Store) SaveCollection(ctx context.Context, c *collections.Collection) error {
	c.AssignMissingIDs()
	if err := c.Validate(); err != nil {
		return err
	}

	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("storage: marshal collection: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO collections (id, name, description, data, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			description = excluded.description,
			data = excluded.data,
			updated_at = excluded.updated_at
	`, c.ID, c.Name, c.Description, string(data), now, now)
	if err != nil {
		return fmt.Errorf("storage: save collection: %w", err)
	}
	return nil
}

// GetCollection returns the collection with the given ID, or ErrNotFound.
func (s *Store) GetCollection(ctx context.Context, id string) (*collections.Collection, error) {
	var data string
	err := s.db.QueryRowContext(ctx, `SELECT data FROM collections WHERE id = ?`, id).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("storage: get collection: %w", err)
	}

	var c collections.Collection
	if err := json.Unmarshal([]byte(data), &c); err != nil {
		return nil, fmt.Errorf("storage: decode collection: %w", err)
	}
	return &c, nil
}

// ListCollections returns every saved collection, most recently updated
// first.
func (s *Store) ListCollections(ctx context.Context) ([]*collections.Collection, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT data FROM collections ORDER BY updated_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("storage: list collections: %w", err)
	}
	defer rows.Close()

	out := []*collections.Collection{}
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("storage: scan collection: %w", err)
		}
		var c collections.Collection
		if err := json.Unmarshal([]byte(data), &c); err != nil {
			return nil, fmt.Errorf("storage: decode collection: %w", err)
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

// DeleteCollection removes a collection by ID, returning ErrNotFound if it
// didn't exist.
func (s *Store) DeleteCollection(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM collections WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("storage: delete collection: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
