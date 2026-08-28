// Package storage persists collections, environments, and run/request
// history to a local SQLite database. It is the only package that knows
// SQL is involved — everything else in ByteForge works with the domain
// types from collections, environments, and runner.
package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "modernc.org/sqlite" // pure-Go SQLite driver, registered as "sqlite"
)

// ErrNotFound is returned by lookups that find no matching row.
var ErrNotFound = errors.New("storage: not found")

// Store is ByteForge's local database handle.
type Store struct {
	db *sql.DB
}

// Open opens (creating if necessary) the SQLite database at path and
// applies the schema. An empty path opens an in-memory database, which is
// primarily useful for tests.
func Open(path string) (*Store, error) {
	dsn := path
	if dsn == "" {
		dsn = ":memory:"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("storage: open %s: %w", path, err)
	}

	// SQLite serializes writers regardless of how many connections you give
	// it; capping the pool at one avoids "database is locked" errors
	// surfacing as flaky request failures instead of being handled here.
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		db.Close()
		return nil, fmt.Errorf("storage: enable foreign keys: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// Close releases the underlying database handle.
func (s *Store) Close() error {
	return s.db.Close()
}
