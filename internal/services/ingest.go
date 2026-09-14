package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/gofrs/uuid/v5"

	"github.com/harvor-io/relay/internal/models"
	"github.com/harvor-io/relay/internal/repositories"
)

// maxIngestTypeLength bounds the caller-supplied event type. A generous limit
// meant to reject obviously malformed input, not to encode product policy.
const maxIngestTypeLength = 255

// ingestTypePattern matches a valid event type: alphanumeric characters plus
// ".", "_", and "-", with no whitespace.
var ingestTypePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// Errors returned by IngestService. Callers should compare against these
// with errors.Is rather than inspecting messages.
var (
	// ErrIngestTypeRequired means the request body's type field was empty.
	ErrIngestTypeRequired = errors.New("type is required")

	// ErrIngestTypeInvalid means the supplied type contained characters
	// other than alphanumerics, ".", "_", or "-".
	ErrIngestTypeInvalid = errors.New("type must contain only alphanumeric characters, '.', '_', and '-'")

	// ErrIngestTypeTooLong means the supplied type exceeded the length
	// bound.
	ErrIngestTypeTooLong = fmt.Errorf("type must be at most %d characters", maxIngestTypeLength)

	// ErrIngestDataRequired means the request body's data field was absent.
	ErrIngestDataRequired = errors.New("data is required")

	// ErrIngestEnvelopeIDInvalid means an explicitly supplied envelope_id is
	// not a valid UUID.
	ErrIngestEnvelopeIDInvalid = errors.New("envelope_id must be a UUID")

	// ErrIngestEnvelopeIDTaken means an explicitly supplied envelope_id
	// already belongs to another envelope.
	ErrIngestEnvelopeIDTaken = errors.New("envelope_id is already in use")

	// ErrIngestSourceNotActive means the ingesting source has been
	// deactivated and is not currently accepting events.
	ErrIngestSourceNotActive = errors.New("source not active")
)

// IngestService is the event-ingestion use-case API: it accepts an event on
// behalf of an already-authenticated Source and persists it as an Envelope.
type IngestService interface {
	// Ingest validates input and persists a new Envelope for source. Its
	// type is combined with source's slug to form the topic
	// ("<slug>.<type>") used later to route the Envelope to Destinations.
	// It returns ErrIngestSourceNotActive if source has been deactivated,
	// ErrIngestTypeRequired, ErrIngestTypeInvalid, ErrIngestTypeTooLong, or
	// ErrIngestDataRequired when input is invalid, ErrIngestEnvelopeIDInvalid
	// if a supplied envelope ID is not a UUID, or ErrIngestEnvelopeIDTaken if
	// it already belongs to another Envelope.
	Ingest(ctx context.Context, source *models.Source, input IngestInput) (*models.Envelope, error)
}

// ingestService is the default IngestService, backed by a
// repositories.EnvelopeRepository.
type ingestService struct {
	envelopes repositories.EnvelopeRepository
}

var _ IngestService = (*ingestService)(nil)

// NewIngestService returns an IngestService backed by envelopes.
func NewIngestService(envelopes repositories.EnvelopeRepository) IngestService {
	return &ingestService{envelopes: envelopes}
}

// IngestInput carries the caller-supplied fields for a new Envelope.
type IngestInput struct {
	// Type identifies the kind of event within source, e.g. "user.created".
	// Combined with source's slug, it forms the Envelope's topic. Required;
	// must contain only alphanumeric characters, ".", "_", or "-".
	Type string

	// Data is the event payload, stored verbatim as the Envelope's message.
	// Required; may be any JSON value, including an explicit null.
	Data json.RawMessage

	// EnvelopeID is an optional client-supplied UUID for the Envelope. When
	// empty, a UUIDv7 is generated. When set, it must be a valid UUID and
	// must not already belong to another Envelope.
	EnvelopeID string
}

// Ingest validates input and persists a new Envelope for source.
func (s *ingestService) Ingest(ctx context.Context, source *models.Source, input IngestInput) (*models.Envelope, error) {
	if !source.IsActive {
		return nil, ErrIngestSourceNotActive
	}

	switch {
	case input.Type == "":
		return nil, ErrIngestTypeRequired
	case len(input.Type) > maxIngestTypeLength:
		return nil, ErrIngestTypeTooLong
	case !ingestTypePattern.MatchString(input.Type):
		return nil, ErrIngestTypeInvalid
	case len(input.Data) == 0:
		return nil, ErrIngestDataRequired
	}

	id, err := s.resolveEnvelopeID(ctx, input.EnvelopeID)
	if err != nil {
		return nil, err
	}

	envelope := &models.Envelope{
		ID:       id,
		SourceID: source.ID,
		Source:   source.Slug,
		Type:     input.Type,
		Data:     input.Data,
	}
	if err := s.envelopes.Create(ctx, envelope); err != nil {
		return nil, fmt.Errorf("services: create envelope: %w", err)
	}
	return envelope, nil

	// TODO: transactionally publish to our eventbus
}

// resolveEnvelopeID returns the UUID to persist for a new Envelope. An empty
// provided value returns the nil UUID, so the repository generates a
// UUIDv7. A non-empty value must parse as a UUID and must not already be in
// use.
func (s *ingestService) resolveEnvelopeID(ctx context.Context, provided string) (uuid.UUID, error) {
	if provided == "" {
		return uuid.Nil, nil
	}

	id, err := uuid.FromString(provided)
	if err != nil {
		return uuid.Nil, ErrIngestEnvelopeIDInvalid
	}

	switch _, err := s.envelopes.Get(ctx, id); {
	case err == nil:
		return uuid.Nil, ErrIngestEnvelopeIDTaken
	case errors.Is(err, repositories.ErrNotFound):
		return id, nil
	default:
		return uuid.Nil, fmt.Errorf("services: look up envelope id: %w", err)
	}
}
