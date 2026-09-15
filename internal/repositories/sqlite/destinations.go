package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid/v5"

	sqlitedb "github.com/harvor-io/relay/internal/database/sqlite"
	"github.com/harvor-io/relay/internal/models"
	"github.com/harvor-io/relay/internal/repositories"
)

// DestinationRepository is the SQLite-backed implementation of
// repositories.DestinationRepository.
type DestinationRepository struct {
	db *sql.DB
}

var _ repositories.DestinationRepository = (*DestinationRepository)(nil)

// NewDestinationRepository returns a DestinationRepository backed by db.
func NewDestinationRepository(db *sql.DB) *DestinationRepository {
	return &DestinationRepository{db: db}
}

const destinationColumns = `id, name, type, config, description, subscriptions, is_active, created_at, updated_at`

// Create inserts a new destination, assigning a UUIDv7 and timestamps when
// unset.
func (r *DestinationRepository) Create(ctx context.Context, destination *models.Destination) error {
	if destination.ID.IsNil() {
		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("sqlite: new destination id: %w", err)
		}
		destination.ID = id
	}
	now := time.Now().UTC()
	if destination.CreatedAt.IsZero() {
		destination.CreatedAt = now
	}
	destination.UpdatedAt = now

	subscriptions, err := json.Marshal(destination.Subscriptions)
	if err != nil {
		return fmt.Errorf("sqlite: marshal destination subscriptions: %w", err)
	}

	_, err = r.db.ExecContext(ctx,
		`INSERT INTO destinations (`+destinationColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		destination.ID.String(),
		destination.Name,
		string(destination.Type),
		string(destination.Config),
		sqlitedb.NullString(destination.Description),
		string(subscriptions),
		destination.IsActive,
		destination.CreatedAt.Format(sqlitedb.TimeLayout),
		destination.UpdatedAt.Format(sqlitedb.TimeLayout),
	)
	if err != nil {
		if sqlitedb.IsUniqueViolation(err) {
			return repositories.ErrConflict
		}
		return fmt.Errorf("sqlite: insert destination: %w", err)
	}
	return nil
}

// Get returns a single destination by ID.
func (r *DestinationRepository) Get(ctx context.Context, id uuid.UUID) (*models.Destination, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+destinationColumns+` FROM destinations WHERE id = ?`, id.String())
	return scanDestination(row)
}

// List returns every destination ordered by name.
func (r *DestinationRepository) List(ctx context.Context) ([]models.Destination, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+destinationColumns+` FROM destinations ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list destinations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var destinations []models.Destination
	for rows.Next() {
		destination, err := scanDestination(rows)
		if err != nil {
			return nil, err
		}
		destinations = append(destinations, *destination)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: iterate destinations: %w", err)
	}
	return destinations, nil
}

// Update overwrites the mutable columns of an existing destination.
func (r *DestinationRepository) Update(ctx context.Context, destination *models.Destination) error {
	destination.UpdatedAt = time.Now().UTC()

	subscriptions, err := json.Marshal(destination.Subscriptions)
	if err != nil {
		return fmt.Errorf("sqlite: marshal destination subscriptions: %w", err)
	}

	res, err := r.db.ExecContext(ctx,
		`UPDATE destinations SET name = ?, description = ?, subscriptions = ?, is_active = ?, updated_at = ? WHERE id = ?`,
		destination.Name,
		sqlitedb.NullString(destination.Description),
		string(subscriptions),
		destination.IsActive,
		destination.UpdatedAt.Format(sqlitedb.TimeLayout),
		destination.ID.String(),
	)
	if err != nil {
		return fmt.Errorf("sqlite: update destination: %w", err)
	}
	return sqlitedb.Affected(res)
}

// Delete removes a destination by ID.
func (r *DestinationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM destinations WHERE id = ?`, id.String())
	if err != nil {
		return fmt.Errorf("sqlite: delete destination: %w", err)
	}
	return sqlitedb.Affected(res)
}

func scanDestination(s sqlitedb.Scanner) (*models.Destination, error) {
	var (
		destination                              models.Destination
		idStr, typeStr, configStr, subscriptions string
		description                              sql.NullString
		createdAt, updatedAt                     string
	)
	if err := s.Scan(&idStr, &destination.Name, &typeStr, &configStr, &description, &subscriptions, &destination.IsActive, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repositories.ErrNotFound
		}
		return nil, fmt.Errorf("sqlite: scan destination: %w", err)
	}

	id, err := uuid.FromString(idStr)
	if err != nil {
		return nil, fmt.Errorf("sqlite: parse destination id: %w", err)
	}
	destination.ID = id
	destination.Type = models.DestinationType(typeStr)
	destination.Config = json.RawMessage(configStr)
	destination.Description = sqlitedb.StringPtr(description)
	if err := json.Unmarshal([]byte(subscriptions), &destination.Subscriptions); err != nil {
		return nil, fmt.Errorf("sqlite: parse destination subscriptions: %w", err)
	}
	if destination.CreatedAt, err = sqlitedb.ParseTime(createdAt); err != nil {
		return nil, fmt.Errorf("sqlite: parse destination created_at: %w", err)
	}
	if destination.UpdatedAt, err = sqlitedb.ParseTime(updatedAt); err != nil {
		return nil, fmt.Errorf("sqlite: parse destination updated_at: %w", err)
	}
	return &destination, nil
}
