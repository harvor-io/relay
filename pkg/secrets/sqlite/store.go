// Package sqlite provides the SQLite-backed implementation of secrets.Store.
// It depends only on database/sql; the caller opens the *sql.DB (see
// internal/database). Unlike the internal repositories, the Store creates its
// own table on construction rather than relying on a goose migration.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/gofrs/uuid/v5"

	sqlitedb "github.com/harvor-io/relay/internal/database/sqlite"
	"github.com/harvor-io/relay/pkg/secrets"
)

// Store is the SQLite-backed implementation of secrets.Store.
type Store struct {
	db *sql.DB
}

var _ secrets.Store = (*Store)(nil)

// createTableSQL creates the secrets table if it does not already exist.
// Unlike the other repositories, the secrets Store owns its schema directly
// rather than relying on the goose migrations under internal/database, since
// pkg/secrets is meant to be usable standalone.
const createTableSQL = `
CREATE TABLE IF NOT EXISTS secrets (
	id     TEXT PRIMARY KEY,
	secret BLOB NOT NULL
)`

// NewStore returns a Store backed by db, creating the secrets table if it
// does not already exist.
func NewStore(ctx context.Context, db *sql.DB) (*Store, error) {
	if _, err := db.ExecContext(ctx, createTableSQL); err != nil {
		return nil, fmt.Errorf("sqlite: create secrets table: %w", err)
	}
	return &Store{db: db}, nil
}

// Create inserts a new secret, assigning a UUIDv7 when unset.
func (s *Store) Create(ctx context.Context, secret *secrets.Secret) error {
	if secret.ID.IsNil() {
		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("sqlite: new secret id: %w", err)
		}
		secret.ID = id
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO secrets (id, secret) VALUES (?, ?)`,
		secret.ID.String(),
		secret.Value,
	)
	if err != nil {
		if sqlitedb.IsUniqueViolation(err) {
			return secrets.ErrConflict
		}
		return fmt.Errorf("sqlite: insert secret: %w", err)
	}
	return nil
}

// Get returns a single secret by ID.
func (s *Store) Get(ctx context.Context, id uuid.UUID) (*secrets.Secret, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, secret FROM secrets WHERE id = ?`, id.String())
	return scanSecret(row)
}

// Delete removes a secret by ID.
func (s *Store) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM secrets WHERE id = ?`, id.String())
	if err != nil {
		return fmt.Errorf("sqlite: delete secret: %w", err)
	}
	return affected(res)
}

func scanSecret(row *sql.Row) (*secrets.Secret, error) {
	var (
		secret secrets.Secret
		idStr  string
	)
	if err := row.Scan(&idStr, &secret.Value); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, secrets.ErrNotFound
		}
		return nil, fmt.Errorf("sqlite: scan secret: %w", err)
	}

	id, err := uuid.FromString(idStr)
	if err != nil {
		return nil, fmt.Errorf("sqlite: parse secret id: %w", err)
	}
	secret.ID = id
	return &secret, nil
}

func affected(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("sqlite: rows affected: %w", err)
	}
	if n == 0 {
		return secrets.ErrNotFound
	}
	return nil
}
