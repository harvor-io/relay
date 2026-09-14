package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gofrs/uuid/v5"
	"github.com/gosimple/slug"

	"github.com/harvor-io/relay/internal/models"
	"github.com/harvor-io/relay/internal/repositories"
)

// Bounds enforced on caller-supplied source fields. They are generous limits
// meant to reject obviously malformed input, not to encode product policy.
const (
	maxSourceNameLength        = 255
	maxSourceDescriptionLength = 1024
	maxSourceSlugLength        = 255

	// maxSlugCandidates caps how many numeric suffixes (name, name-2, name-3…)
	// the service tries when auto-deriving a unique slug before giving up.
	maxSlugCandidates = 50
)

// Errors returned by SourceService. Callers should compare against these with
// errors.Is rather than inspecting messages.
var (
	// ErrSourceNotFound means no source exists with the requested ID.
	ErrSourceNotFound = errors.New("source not found")

	// ErrSourceNameRequired means the supplied name was empty or whitespace.
	ErrSourceNameRequired = errors.New("source name is required")

	// ErrSourceNameTooLong means the supplied name exceeded the length bound.
	ErrSourceNameTooLong = fmt.Errorf("source name must be at most %d characters", maxSourceNameLength)

	// ErrSourceDescriptionTooLong means the supplied description exceeded the
	// length bound.
	ErrSourceDescriptionTooLong = fmt.Errorf("source description must be at most %d characters", maxSourceDescriptionLength)

	// ErrSourceSlugInvalid means an explicitly supplied slug is not URL-safe
	// (it must be lowercase letters, digits, hyphens, or underscores, and may
	// not start or end with a separator) or exceeds the length bound.
	ErrSourceSlugInvalid = fmt.Errorf("source slug must be URL-safe and at most %d characters", maxSourceSlugLength)

	// ErrSourceSlugTaken means the requested slug already belongs to another
	// source.
	ErrSourceSlugTaken = errors.New("source slug is already in use")

	// ErrSourceSlugUnderivable means no slug was supplied and none could be
	// derived from the name (for example, a name with no alphanumerics).
	ErrSourceSlugUnderivable = errors.New("cannot derive a slug from the source name; provide one explicitly")

	// ErrSourceIDTaken means an explicitly supplied ID already belongs to
	// another source.
	ErrSourceIDTaken = errors.New("source id is already in use")

	// ErrSourceIDInvalid means an explicitly supplied ID is not a valid UUID.
	ErrSourceIDInvalid = errors.New("source id must be a UUID")
)

// SourceService is the source-management use-case API: the operations callers
// (HTTP handlers today) invoke, and the errors they can expect back.
type SourceService interface {
	// Get returns the source with the given ID, or ErrSourceNotFound.
	Get(ctx context.Context, id uuid.UUID) (*models.Source, error)

	// GetByIDOrSlug returns the source identified by idOrSlug, trying it
	// first as a UUID and falling back to a slug lookup when it does not
	// parse as one. Returns ErrSourceNotFound if neither matches.
	GetByIDOrSlug(ctx context.Context, idOrSlug string) (*models.Source, error)

	// List returns every source, ordered by name.
	List(ctx context.Context) ([]models.Source, error)

	// Create validates input and persists a new source. It returns
	// ErrSourceNameRequired, ErrSourceNameTooLong, ErrSourceDescriptionTooLong,
	// ErrSourceSlugInvalid, ErrSourceSlugUnderivable, ErrSourceIDInvalid,
	// ErrSourceSlugTaken, or ErrSourceIDTaken when input is invalid or
	// conflicts with an existing source.
	Create(ctx context.Context, input CreateSourceInput) (*models.Source, error)

	// Update changes the name and/or description of the source with the given
	// ID. Fields left nil in input are unchanged; a non-nil Description that
	// is empty after trimming clears it, matching Create. It returns
	// ErrSourceNotFound, ErrSourceNameRequired, ErrSourceNameTooLong, or
	// ErrSourceDescriptionTooLong when input is invalid.
	Update(ctx context.Context, id uuid.UUID, input UpdateSourceInput) (*models.Source, error)

	// Delete removes the source with the given ID, or returns ErrSourceNotFound.
	Delete(ctx context.Context, id uuid.UUID) error
}

// sourceService is the default SourceService, backed by a
// repositories.SourceRepository.
type sourceService struct {
	sources repositories.SourceRepository
}

var _ SourceService = (*sourceService)(nil)

// NewSourceService returns a SourceService backed by sources.
func NewSourceService(sources repositories.SourceRepository) SourceService {
	return &sourceService{sources: sources}
}

// CreateSourceInput carries the caller-supplied fields for a new source. The
// service trims surrounding whitespace from every field and treats a
// description that is empty after trimming as absent.
type CreateSourceInput struct {
	Name        string
	Description *string

	// Slug is optional. When empty, the service derives a unique URL-safe slug
	// from Name. When set, it must already be URL-safe and unused.
	Slug string

	// ID is optional. When empty, the service generates a UUIDv7. When set,
	// it must be a valid UUID and must not already belong to another source.
	ID string
}

// UpdateSourceInput carries the caller-supplied fields to change on an
// existing source. A nil field is left unchanged. Name, when supplied, is
// trimmed and validated the same way as on Create. Description, when
// supplied, is trimmed and normalised the same way as on Create — including
// that a value which is empty after trimming clears the description; to
// leave the description untouched, leave this nil rather than supplying "".
type UpdateSourceInput struct {
	Name        *string
	Description *string
}

// Get returns the source with the given ID, or ErrSourceNotFound.
func (s *sourceService) Get(ctx context.Context, id uuid.UUID) (*models.Source, error) {
	source, err := s.sources.Get(ctx, id)
	if err != nil {
		return nil, mapSourceRepoError("get source", err)
	}
	return source, nil
}

// GetByIDOrSlug returns the source identified by idOrSlug, trying it first as
// a UUID and falling back to a slug lookup when it does not parse as one.
func (s *sourceService) GetByIDOrSlug(ctx context.Context, idOrSlug string) (*models.Source, error) {
	if id, err := uuid.FromString(idOrSlug); err == nil {
		return s.Get(ctx, id)
	}
	source, err := s.sources.GetBySlug(ctx, idOrSlug)
	if err != nil {
		return nil, mapSourceRepoError("get source by slug", err)
	}
	return source, nil
}

// List returns every source, ordered by name.
func (s *sourceService) List(ctx context.Context) ([]models.Source, error) {
	sources, err := s.sources.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("services: list sources: %w", err)
	}
	return sources, nil
}

// Create validates input and persists a new source, returning it with its
// assigned ID and timestamps. It returns ErrSourceNameRequired,
// ErrSourceNameTooLong, or ErrSourceDescriptionTooLong when input is invalid.
func (s *sourceService) Create(ctx context.Context, input CreateSourceInput) (*models.Source, error) {
	name := strings.TrimSpace(input.Name)
	switch {
	case name == "":
		return nil, ErrSourceNameRequired
	case len(name) > maxSourceNameLength:
		return nil, ErrSourceNameTooLong
	}

	description, err := normaliseDescription(input.Description)
	if err != nil {
		return nil, err
	}

	slg, err := s.resolveSlug(ctx, name, input.Slug)
	if err != nil {
		return nil, err
	}

	id, err := s.resolveID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	source := &models.Source{ID: id, Name: name, Slug: slg, Description: description}
	if err := s.sources.Create(ctx, source); err != nil {
		if errors.Is(err, repositories.ErrConflict) {
			return nil, ErrSourceSlugTaken
		}
		return nil, fmt.Errorf("services: create source: %w", err)
	}
	return source, nil
}

// Update changes the name and/or description of the source with the given
// ID, leaving fields nil in input unchanged. It returns ErrSourceNotFound,
// ErrSourceNameRequired, ErrSourceNameTooLong, or ErrSourceDescriptionTooLong
// when input is invalid.
func (s *sourceService) Update(ctx context.Context, id uuid.UUID, input UpdateSourceInput) (*models.Source, error) {
	source, err := s.sources.Get(ctx, id)
	if err != nil {
		return nil, mapSourceRepoError("get source", err)
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		switch {
		case name == "":
			return nil, ErrSourceNameRequired
		case len(name) > maxSourceNameLength:
			return nil, ErrSourceNameTooLong
		}
		source.Name = name
	}

	if input.Description != nil {
		description, err := normaliseDescription(input.Description)
		if err != nil {
			return nil, err
		}
		source.Description = description
	}

	if err := s.sources.Update(ctx, source); err != nil {
		if errors.Is(err, repositories.ErrConflict) {
			return nil, ErrSourceSlugTaken
		}
		return nil, mapSourceRepoError("update source", err)
	}
	return source, nil
}

// resolveSlug returns the slug to persist for a new source. An explicitly
// supplied slug is validated and checked for availability as-is; otherwise a
// URL-safe base is derived from name and the first free "base", "base-2",
// "base-3"… candidate is returned.
func (s *sourceService) resolveSlug(ctx context.Context, name, provided string) (string, error) {
	if provided = strings.TrimSpace(provided); provided != "" {
		if len(provided) > maxSourceSlugLength || !slug.IsSlug(provided) {
			return "", ErrSourceSlugInvalid
		}
		taken, err := s.slugTaken(ctx, provided)
		if err != nil {
			return "", err
		}
		if taken {
			return "", ErrSourceSlugTaken
		}
		return provided, nil
	}

	base := slug.Make(name)
	if len(base) > maxSourceSlugLength {
		base = strings.Trim(base[:maxSourceSlugLength], "-")
	}
	if base == "" {
		return "", ErrSourceSlugUnderivable
	}

	for i := 1; i <= maxSlugCandidates; i++ {
		candidate := base
		if i > 1 {
			candidate = fmt.Sprintf("%s-%d", base, i)
		}
		taken, err := s.slugTaken(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !taken {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("services: no available slug for name %q after %d attempts", name, maxSlugCandidates)
}

// slugTaken reports whether a source already uses slg.
func (s *sourceService) slugTaken(ctx context.Context, slg string) (bool, error) {
	switch _, err := s.sources.GetBySlug(ctx, slg); {
	case err == nil:
		return true, nil
	case errors.Is(err, repositories.ErrNotFound):
		return false, nil
	default:
		return false, fmt.Errorf("services: look up slug: %w", err)
	}
}

// resolveID returns the UUID to persist for a new source. An empty provided
// value returns the nil UUID, so the repository generates a UUIDv7. A
// non-empty value must parse as a UUID and must not already be in use.
func (s *sourceService) resolveID(ctx context.Context, provided string) (uuid.UUID, error) {
	provided = strings.TrimSpace(provided)
	if provided == "" {
		return uuid.Nil, nil
	}

	id, err := uuid.FromString(provided)
	if err != nil {
		return uuid.Nil, ErrSourceIDInvalid
	}

	taken, err := s.idTaken(ctx, id)
	if err != nil {
		return uuid.Nil, err
	}
	if taken {
		return uuid.Nil, ErrSourceIDTaken
	}
	return id, nil
}

// idTaken reports whether a source already uses id.
func (s *sourceService) idTaken(ctx context.Context, id uuid.UUID) (bool, error) {
	switch _, err := s.sources.Get(ctx, id); {
	case err == nil:
		return true, nil
	case errors.Is(err, repositories.ErrNotFound):
		return false, nil
	default:
		return false, fmt.Errorf("services: look up source id: %w", err)
	}
}

// Delete removes the source with the given ID, returning ErrSourceNotFound if
// it does not exist.
func (s *sourceService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.sources.Delete(ctx, id); err != nil {
		return mapSourceRepoError("delete source", err)
	}
	return nil
}

// normaliseDescription trims the optional description and collapses an empty
// result to nil. It returns ErrSourceDescriptionTooLong if the trimmed value
// exceeds the length bound.
func normaliseDescription(raw *string) (*string, error) {
	if raw == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil, nil
	}
	if len(trimmed) > maxSourceDescriptionLength {
		return nil, ErrSourceDescriptionTooLong
	}
	return &trimmed, nil
}

// mapSourceRepoError converts a repository error into a service-level error:
// repositories.ErrNotFound becomes ErrSourceNotFound; anything else is wrapped
// with context and surfaces as an internal failure.
func mapSourceRepoError(op string, err error) error {
	if errors.Is(err, repositories.ErrNotFound) {
		return ErrSourceNotFound
	}
	return fmt.Errorf("services: %s: %w", op, err)
}
