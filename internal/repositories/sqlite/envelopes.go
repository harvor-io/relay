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

// EnvelopeRepository is the SQLite-backed implementation of
// repositories.EnvelopeRepository.
type EnvelopeRepository struct {
	db *sql.DB
}

var _ repositories.EnvelopeRepository = (*EnvelopeRepository)(nil)

// NewEnvelopeRepository returns an EnvelopeRepository backed by db.
func NewEnvelopeRepository(db *sql.DB) *EnvelopeRepository {
	return &EnvelopeRepository{db: db}
}

const envelopeColumns = `id, source_id, source, type, message, created_at`

// Create inserts a new envelope, assigning a UUIDv7 and CreatedAt when unset.
func (r *EnvelopeRepository) Create(ctx context.Context, envelope *models.Envelope) error {
	if envelope.ID.IsNil() {
		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("sqlite: new envelope id: %w", err)
		}
		envelope.ID = id
	}
	if envelope.CreatedAt.IsZero() {
		envelope.CreatedAt = time.Now().UTC()
	}

	message := envelope.Message
	if len(message) == 0 {
		message = json.RawMessage("null")
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO envelopes (`+envelopeColumns+`) VALUES (?, ?, ?, ?, ?, ?)`,
		envelope.ID.String(),
		envelope.SourceID.String(),
		envelope.Source,
		envelope.Type,
		string(message),
		envelope.CreatedAt.Format(sqlitedb.TimeLayout),
	)
	if err != nil {
		return fmt.Errorf("sqlite: insert envelope: %w", err)
	}
	return nil
}

// Get returns a single envelope by ID.
func (r *EnvelopeRepository) Get(ctx context.Context, id uuid.UUID) (*models.Envelope, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+envelopeColumns+` FROM envelopes WHERE id = ?`, id.String())
	return scanEnvelope(row)
}

// List returns every envelope, newest first.
func (r *EnvelopeRepository) List(ctx context.Context) ([]models.Envelope, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+envelopeColumns+` FROM envelopes ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list envelopes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var envelopes []models.Envelope
	for rows.Next() {
		envelope, err := scanEnvelope(rows)
		if err != nil {
			return nil, err
		}
		envelopes = append(envelopes, *envelope)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: iterate envelopes: %w", err)
	}
	return envelopes, nil
}

func scanEnvelope(s sqlitedb.Scanner) (*models.Envelope, error) {
	var (
		envelope  models.Envelope
		idStr     string
		sourceID  string
		message   string
		createdAt string
	)
	if err := s.Scan(&idStr, &sourceID, &envelope.Source, &envelope.Type, &message, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repositories.ErrNotFound
		}
		return nil, fmt.Errorf("sqlite: scan envelope: %w", err)
	}

	id, err := uuid.FromString(idStr)
	if err != nil {
		return nil, fmt.Errorf("sqlite: parse envelope id: %w", err)
	}
	envelope.ID = id

	envelope.SourceID, err = uuid.FromString(sourceID)
	if err != nil {
		return nil, fmt.Errorf("sqlite: parse envelope source_id: %w", err)
	}

	envelope.Message = json.RawMessage(message)
	if envelope.CreatedAt, err = sqlitedb.ParseTime(createdAt); err != nil {
		return nil, fmt.Errorf("sqlite: parse envelope created_at: %w", err)
	}
	return &envelope, nil
}
