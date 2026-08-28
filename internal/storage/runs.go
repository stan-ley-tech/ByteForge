package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/stan-ley-tech/ByteForge/internal/idgen"
)

// RunRecord is a stored test run. Report holds the full runner.Report as
// JSON; storage doesn't depend on the runner package directly so that the
// dependency graph stays one-directional (runner produces results, storage
// persists bytes — it doesn't need to understand their shape).
type RunRecord struct {
	ID              string
	CollectionID    string
	CollectionName  string
	EnvironmentName string
	Report          []byte
	Passed          int
	Failed          int
	StartedAt       time.Time
	DurationMS      int64
}

// SaveRun persists a completed test run. A missing ID is assigned before
// saving.
func (s *Store) SaveRun(ctx context.Context, r RunRecord) (string, error) {
	if r.ID == "" {
		r.ID = idgen.New()
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO runs (id, collection_id, collection_name, environment_name, report, passed, failed, started_at, duration_ms)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, r.ID, r.CollectionID, r.CollectionName, r.EnvironmentName, string(r.Report),
		r.Passed, r.Failed, r.StartedAt.UTC().Format(time.RFC3339), r.DurationMS)
	if err != nil {
		return "", fmt.Errorf("storage: save run: %w", err)
	}
	return r.ID, nil
}

// ListRuns returns the most recent runs, most recent first, optionally
// filtered to a single collection. A limit of 0 returns all runs.
func (s *Store) ListRuns(ctx context.Context, collectionID string, limit int) ([]RunRecord, error) {
	query := `SELECT id, collection_id, collection_name, environment_name, report, passed, failed, started_at, duration_ms FROM runs`
	args := []any{}
	if collectionID != "" {
		query += ` WHERE collection_id = ?`
		args = append(args, collectionID)
	}
	query += ` ORDER BY started_at DESC`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("storage: list runs: %w", err)
	}
	defer rows.Close()

	out := []RunRecord{}
	for rows.Next() {
		var r RunRecord
		var startedAt string
		if err := rows.Scan(&r.ID, &r.CollectionID, &r.CollectionName, &r.EnvironmentName,
			&r.Report, &r.Passed, &r.Failed, &startedAt, &r.DurationMS); err != nil {
			return nil, fmt.Errorf("storage: scan run: %w", err)
		}
		r.StartedAt, err = time.Parse(time.RFC3339, startedAt)
		if err != nil {
			return nil, fmt.Errorf("storage: parse run timestamp: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
