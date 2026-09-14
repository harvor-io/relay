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

// SourceRepository is the SQLite-backed implementation of
// repositories.SourceRepository.
type SourceRepository struct {
	db *sql.DB
}

var _ repositories.SourceRepository = (*SourceRepository)(nil)

// NewSourceRepository returns a SourceRepository backed by db.
func NewSourceRepository(db *sql.DB) *SourceRepository {
	return &SourceRepository{db: db}
}

const sourceColumns = `id, name, slug, description, is_active, created_at, updated_at`

// Create inserts a new source, assigning a UUIDv7 and timestamps when unset.
func (r *SourceRepository) Create(ctx context.Context, source *models.Source) error {
	if source.ID.IsNil() {
		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("sqlite: new source id: %w", err)
		}
		source.ID = id
	}
	now := time.Now().UTC()
	if source.CreatedAt.IsZero() {
		source.CreatedAt = now
	}
	source.UpdatedAt = now

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO sources (`+sourceColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		source.ID.String(),
		source.Name,
		source.Slug,
		sqlitedb.NullString(source.Description),
		source.IsActive,
		source.CreatedAt.Format(sqlitedb.TimeLayout),
		source.UpdatedAt.Format(sqlitedb.TimeLayout),
	)
	if err != nil {
		if sqlitedb.IsUniqueViolation(err) {
			return repositories.ErrConflict
		}
		return fmt.Errorf("sqlite: insert source: %w", err)
	}
	return nil
}

// Get returns a single source by ID.
func (r *SourceRepository) Get(ctx context.Context, id uuid.UUID) (*models.Source, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+sourceColumns+` FROM sources WHERE id = ?`, id.String())
	return scanSource(row)
}

// GetBySlug returns a single source by slug.
func (r *SourceRepository) GetBySlug(ctx context.Context, slug string) (*models.Source, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+sourceColumns+` FROM sources WHERE slug = ?`, slug)
	return scanSource(row)
}

// List returns every source ordered by name.
func (r *SourceRepository) List(ctx context.Context) ([]models.Source, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+sourceColumns+` FROM sources ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list sources: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var sources []models.Source
	for rows.Next() {
		source, err := scanSource(rows)
		if err != nil {
			return nil, err
		}
		sources = append(sources, *source)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: iterate sources: %w", err)
	}
	return sources, nil
}

// Update overwrites the mutable columns of an existing source.
func (r *SourceRepository) Update(ctx context.Context, source *models.Source) error {
	source.UpdatedAt = time.Now().UTC()
	res, err := r.db.ExecContext(ctx,
		`UPDATE sources SET name = ?, slug = ?, description = ?, is_active = ?, updated_at = ? WHERE id = ?`,
		source.Name,
		source.Slug,
		sqlitedb.NullString(source.Description),
		source.IsActive,
		source.UpdatedAt.Format(sqlitedb.TimeLayout),
		source.ID.String(),
	)
	if err != nil {
		if sqlitedb.IsUniqueViolation(err) {
			return repositories.ErrConflict
		}
		return fmt.Errorf("sqlite: update source: %w", err)
	}
	return sqlitedb.Affected(res)
}

// Delete removes a source by ID.
func (r *SourceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM sources WHERE id = ?`, id.String())
	if err != nil {
		return fmt.Errorf("sqlite: delete source: %w", err)
	}
	return sqlitedb.Affected(res)
}

func scanSource(s sqlitedb.Scanner) (*models.Source, error) {
	var (
		source               models.Source
		idStr                string
		description          sql.NullString
		createdAt, updatedAt string
	)
	if err := s.Scan(&idStr, &source.Name, &source.Slug, &description, &source.IsActive, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repositories.ErrNotFound
		}
		return nil, fmt.Errorf("sqlite: scan source: %w", err)
	}

	id, err := uuid.FromString(idStr)
	if err != nil {
		return nil, fmt.Errorf("sqlite: parse source id: %w", err)
	}
	source.ID = id
	source.Description = sqlitedb.StringPtr(description)
	if source.CreatedAt, err = sqlitedb.ParseTime(createdAt); err != nil {
		return nil, fmt.Errorf("sqlite: parse source created_at: %w", err)
	}
	if source.UpdatedAt, err = sqlitedb.ParseTime(updatedAt); err != nil {
		return nil, fmt.Errorf("sqlite: parse source updated_at: %w", err)
	}
	return &source, nil
}
