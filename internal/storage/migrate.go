package storage

import (
	"context"
	"fmt"
)

// schema is applied with CREATE TABLE IF NOT EXISTS / CREATE INDEX IF NOT
// EXISTS, so re-running it on every Open is idempotent. That's enough for a
// single-file local database with one schema version; a tool like this
// doesn't need a migration framework on top of it.
const schema = `
CREATE TABLE IF NOT EXISTS collections (
	id          TEXT PRIMARY KEY,
	name        TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	data        TEXT NOT NULL,
	created_at  TEXT NOT NULL,
	updated_at  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS environments (
	id         TEXT PRIMARY KEY,
	name       TEXT NOT NULL UNIQUE,
	data       TEXT NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS runs (
	id               TEXT PRIMARY KEY,
	collection_id    TEXT NOT NULL,
	collection_name  TEXT NOT NULL,
	environment_name TEXT NOT NULL DEFAULT '',
	report           TEXT NOT NULL,
	passed           INTEGER NOT NULL,
	failed           INTEGER NOT NULL,
	started_at       TEXT NOT NULL,
	duration_ms      INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_runs_collection_id ON runs(collection_id);
CREATE INDEX IF NOT EXISTS idx_runs_started_at ON runs(started_at);

CREATE TABLE IF NOT EXISTS request_history (
	id           TEXT PRIMARY KEY,
	request_name TEXT NOT NULL DEFAULT '',
	method       TEXT NOT NULL,
	url          TEXT NOT NULL,
	status       INTEGER NOT NULL,
	duration_ms  INTEGER NOT NULL,
	executed_at  TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_request_history_executed_at ON request_history(executed_at);
`

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("storage: migrate schema: %w", err)
	}
	return nil
}
