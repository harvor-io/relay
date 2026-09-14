package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid/v5"

	sqlitedb "github.com/harvor-io/relay/internal/database/sqlite"
	"github.com/harvor-io/relay/internal/models"
	"github.com/harvor-io/relay/internal/repositories"
)

// SourceKeyRepository is the SQLite-backed implementation of
// repositories.SourceKeyRepository.
type SourceKeyRepository struct {
	db *sql.DB
}

var _ repositories.SourceKeyRepository = (*SourceKeyRepository)(nil)

// NewSourceKeyRepository returns a SourceKeyRepository backed by db.
func NewSourceKeyRepository(db *sql.DB) *SourceKeyRepository {
	return &SourceKeyRepository{db: db}
}

const sourceKeyColumns = `id, source_id, secret_id, name, is_active, created_at, updated_at`

// Create inserts a new source key, assigning an ID and timestamps when
// unset.
func (r *SourceKeyRepository) Create(ctx context.Context, key *models.SourceKey) error {
	if key.ID.IsNil() {
		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("sqlite: new source key id: %w", err)
		}
		key.ID = id
	}
	now := time.Now().UTC()
	if key.CreatedAt.IsZero() {
		key.CreatedAt = now
	}
	key.UpdatedAt = now

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO source_keys (`+sourceKeyColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		key.ID.String(),
		key.SourceID.String(),
		key.SecretID.String(),
		key.Name,
		key.IsActive,
		key.CreatedAt.Format(sqlitedb.TimeLayout),
		key.UpdatedAt.Format(sqlitedb.TimeLayout),
	)
	if err != nil {
		if sqlitedb.IsUniqueViolation(err) {
			return repositories.ErrConflict
		}
		return fmt.Errorf("sqlite: insert source key: %w", err)
	}
	return nil
}

// Get returns a single source key by ID.
func (r *SourceKeyRepository) Get(ctx context.Context, id uuid.UUID) (*models.SourceKey, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+sourceKeyColumns+` FROM source_keys WHERE id = ?`, id.String())
	return scanSourceKey(row)
}

// ListBySource returns every source key belonging to sourceID, ordered by
// name.
func (r *SourceKeyRepository) ListBySource(ctx context.Context, sourceID uuid.UUID) ([]models.SourceKey, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+sourceKeyColumns+` FROM source_keys WHERE source_id = ? ORDER BY name`,
		sourceID.String(),
	)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list source keys: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var keys []models.SourceKey
	for rows.Next() {
		key, err := scanSourceKey(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, *key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: iterate source keys: %w", err)
	}
	return keys, nil
}

// Update overwrites the mutable columns of an existing source key.
func (r *SourceKeyRepository) Update(ctx context.Context, key *models.SourceKey) error {
	key.UpdatedAt = time.Now().UTC()
	res, err := r.db.ExecContext(ctx,
		`UPDATE source_keys SET name = ?, is_active = ?, updated_at = ? WHERE id = ?`,
		key.Name,
		key.IsActive,
		key.UpdatedAt.Format(sqlitedb.TimeLayout),
		key.ID.String(),
	)
	if err != nil {
		return fmt.Errorf("sqlite: update source key: %w", err)
	}
	return sqlitedb.Affected(res)
}

// Delete removes a source key by ID.
func (r *SourceKeyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM source_keys WHERE id = ?`, id.String())
	if err != nil {
		return fmt.Errorf("sqlite: delete source key: %w", err)
	}
	return sqlitedb.Affected(res)
}

func scanSourceKey(s sqlitedb.Scanner) (*models.SourceKey, error) {
	var (
		key                             models.SourceKey
		idStr, sourceIDStr, secretIDStr string
		createdAt, updatedAt            string
	)
	if err := s.Scan(&idStr, &sourceIDStr, &secretIDStr, &key.Name, &key.IsActive, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repositories.ErrNotFound
		}
		return nil, fmt.Errorf("sqlite: scan source key: %w", err)
	}

	id, err := uuid.FromString(idStr)
	if err != nil {
		return nil, fmt.Errorf("sqlite: parse source key id: %w", err)
	}
	key.ID = id

	sourceID, err := uuid.FromString(sourceIDStr)
	if err != nil {
		return nil, fmt.Errorf("sqlite: parse source key source_id: %w", err)
	}
	key.SourceID = sourceID

	secretID, err := uuid.FromString(secretIDStr)
	if err != nil {
		return nil, fmt.Errorf("sqlite: parse source key secret_id: %w", err)
	}
	key.SecretID = secretID

	if key.CreatedAt, err = sqlitedb.ParseTime(createdAt); err != nil {
		return nil, fmt.Errorf("sqlite: parse source key created_at: %w", err)
	}
	if key.UpdatedAt, err = sqlitedb.ParseTime(updatedAt); err != nil {
		return nil, fmt.Errorf("sqlite: parse source key updated_at: %w", err)
	}
	return &key, nil
}
