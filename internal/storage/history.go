package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/stan-ley-tech/ByteForge/internal/idgen"
)

// HistoryEntry is a single ad-hoc request execution, independent of any
// collection run — the equivalent of a browser's address bar history for
// requests fired from the builder.
type HistoryEntry struct {
	ID          string
	RequestName string
	Method      string
	URL         string
	Status      int
	DurationMS  int64
	ExecutedAt  time.Time
}

// AddHistory records a single request execution.
func (s *Store) AddHistory(ctx context.Context, h HistoryEntry) error {
	if h.ID == "" {
		h.ID = idgen.New()
	}
	if h.ExecutedAt.IsZero() {
		h.ExecutedAt = time.Now()
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO request_history (id, request_name, method, url, status, duration_ms, executed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, h.ID, h.RequestName, h.Method, h.URL, h.Status, h.DurationMS, h.ExecutedAt.UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("storage: add history entry: %w", err)
	}
	return nil
}

// ListHistory returns the most recent request executions, most recent
// first, capped at limit entries.
func (s *Store) ListHistory(ctx context.Context, limit int) ([]HistoryEntry, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, request_name, method, url, status, duration_ms, executed_at
		FROM request_history
		ORDER BY executed_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("storage: list history: %w", err)
	}
	defer rows.Close()

	out := []HistoryEntry{}
	for rows.Next() {
		var h HistoryEntry
		var executedAt string
		if err := rows.Scan(&h.ID, &h.RequestName, &h.Method, &h.URL, &h.Status, &h.DurationMS, &executedAt); err != nil {
			return nil, fmt.Errorf("storage: scan history entry: %w", err)
		}
		h.ExecutedAt, err = time.Parse(time.RFC3339, executedAt)
		if err != nil {
			return nil, fmt.Errorf("storage: parse history timestamp: %w", err)
		}
		out = append(out, h)
	}
	return out, rows.Err()
}
