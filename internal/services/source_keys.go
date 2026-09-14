package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/gofrs/uuid/v5"

	"github.com/harvor-io/relay/internal/models"
	"github.com/harvor-io/relay/internal/repositories"
	"github.com/harvor-io/relay/pkg/encryptor"
	"github.com/harvor-io/relay/pkg/secrets"
)

// Bounds enforced on caller-supplied source key fields. Generous limits meant
// to reject obviously malformed input, not to encode product policy.
const (
	maxSourceKeyNameLength = 255

	// sourceKeySecretLength is the number of random bytes generated for a new
	// key's secret, hex-encoded before it is shown to the caller.
	sourceKeySecretLength = 32
)

// Errors returned by SourceKeyService. Callers should compare against these
// with errors.Is rather than inspecting messages.
var (
	// ErrSourceKeyNotFound means no key exists with the requested ID under
	// the given source.
	ErrSourceKeyNotFound = errors.New("source key not found")

	// ErrSourceKeyNameRequired means the supplied name was empty or
	// whitespace.
	ErrSourceKeyNameRequired = errors.New("source key name is required")

	// ErrSourceKeyNameTooLong means the supplied name exceeded the length
	// bound.
	ErrSourceKeyNameTooLong = fmt.Errorf("source key name must be at most %d characters", maxSourceKeyNameLength)
)

// SourceKeyService is the source-key-management use-case API: the operations
// callers (HTTP handlers today) invoke, and the errors they can expect back.
// A key's secret is generated and encrypted on Create and returned exactly
// once, in that response; it is never retrievable afterwards.
type SourceKeyService interface {
	// Create validates input, generates and encrypts a new secret, and
	// persists a key for the given source. It returns the key together with
	// the plaintext secret, hex-encoded. Returns ErrSourceNotFound if the
	// source does not exist, or ErrSourceKeyNameRequired / ErrSourceKeyNameTooLong
	// if input is invalid.
	Create(ctx context.Context, sourceID uuid.UUID, input CreateSourceKeyInput) (*models.SourceKey, string, error)

	// Get returns the key with the given ID belonging to sourceID, or
	// ErrSourceKeyNotFound.
	Get(ctx context.Context, sourceID uuid.UUID, id string) (*models.SourceKey, error)

	// ListBySource returns every key belonging to sourceID, ordered by name.
	// It returns ErrSourceNotFound if the source does not exist.
	ListBySource(ctx context.Context, sourceID uuid.UUID) ([]models.SourceKey, error)

	// Activate marks the key as active, or returns ErrSourceKeyNotFound.
	Activate(ctx context.Context, sourceID uuid.UUID, id string) (*models.SourceKey, error)

	// Deactivate marks the key as inactive, or returns ErrSourceKeyNotFound.
	Deactivate(ctx context.Context, sourceID uuid.UUID, id string) (*models.SourceKey, error)

	// Delete removes the key and its underlying secret, or returns
	// ErrSourceKeyNotFound.
	Delete(ctx context.Context, sourceID uuid.UUID, id string) error
}

// sourceKeyService is the default SourceKeyService, backed by a
// repositories.SourceKeyRepository for key metadata and a secrets.Store for
// the encrypted secret material.
type sourceKeyService struct {
	keys      repositories.SourceKeyRepository
	sources   repositories.SourceRepository
	secrets   secrets.Store
	encryptor encryptor.Encryptor
}

var _ SourceKeyService = (*sourceKeyService)(nil)

// NewSourceKeyService returns a SourceKeyService backed by keys for metadata,
// sources to validate that a source exists, secretStore for encrypted secret
// material, and enc to encrypt and decrypt it.
func NewSourceKeyService(
	keys repositories.SourceKeyRepository,
	sources repositories.SourceRepository,
	secretStore secrets.Store,
	enc encryptor.Encryptor,
) SourceKeyService {
	return &sourceKeyService{keys: keys, sources: sources, secrets: secretStore, encryptor: enc}
}

// CreateSourceKeyInput carries the caller-supplied fields for a new source
// key. The service trims surrounding whitespace from Name.
type CreateSourceKeyInput struct {
	Name string
}

// Create validates input, generates a random secret, encrypts and stores it,
// and persists a key referencing it. It returns the key together with the
// plaintext secret, hex-encoded; the secret is never retrievable again.
func (s *sourceKeyService) Create(ctx context.Context, sourceID uuid.UUID, input CreateSourceKeyInput) (*models.SourceKey, string, error) {
	if _, err := s.sources.Get(ctx, sourceID); err != nil {
		return nil, "", mapSourceRepoError("get source", err)
	}

	name := strings.TrimSpace(input.Name)
	switch {
	case name == "":
		return nil, "", ErrSourceKeyNameRequired
	case len(name) > maxSourceKeyNameLength:
		return nil, "", ErrSourceKeyNameTooLong
	}

	raw := make([]byte, sourceKeySecretLength)
	if _, err := rand.Read(raw); err != nil {
		return nil, "", fmt.Errorf("services: generate source key secret: %w", err)
	}
	ciphertext, err := s.encryptor.Encrypt(raw)
	if err != nil {
		return nil, "", fmt.Errorf("services: encrypt source key secret: %w", err)
	}

	secret := &secrets.Secret{Value: ciphertext}
	if err := s.secrets.Create(ctx, secret); err != nil {
		return nil, "", fmt.Errorf("services: store source key secret: %w", err)
	}

	key := &models.SourceKey{
		SourceID: sourceID,
		SecretID: secret.ID,
		Name:     name,
		IsActive: true,
	}
	if err := s.keys.Create(ctx, key); err != nil {
		return nil, "", fmt.Errorf("services: create source key: %w", err)
	}

	plaintext, err := s.decryptSecret(ctx, secret.ID)
	if err != nil {
		return nil, "", err
	}
	return key, plaintext, nil
}

// decryptSecret retrieves the encrypted secret with the given ID from the
// store and decrypts it, returning the plaintext as a hex string.
func (s *sourceKeyService) decryptSecret(ctx context.Context, secretID uuid.UUID) (string, error) {
	secret, err := s.secrets.Get(ctx, secretID)
	if err != nil {
		return "", fmt.Errorf("services: retrieve source key secret: %w", err)
	}
	raw, err := s.encryptor.Decrypt(secret.Value)
	if err != nil {
		return "", fmt.Errorf("services: decrypt source key secret: %w", err)
	}
	return hex.EncodeToString(raw), nil
}

// Get returns the key with the given ID belonging to sourceID, or
// ErrSourceKeyNotFound.
func (s *sourceKeyService) Get(ctx context.Context, sourceID uuid.UUID, id string) (*models.SourceKey, error) {
	return s.getOwned(ctx, sourceID, id)
}

// ListBySource returns every key belonging to sourceID, ordered by name. It
// returns ErrSourceNotFound if the source does not exist.
func (s *sourceKeyService) ListBySource(ctx context.Context, sourceID uuid.UUID) ([]models.SourceKey, error) {
	if _, err := s.sources.Get(ctx, sourceID); err != nil {
		return nil, mapSourceRepoError("get source", err)
	}
	keys, err := s.keys.ListBySource(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("services: list source keys: %w", err)
	}
	return keys, nil
}

// Activate marks the key as active, or returns ErrSourceKeyNotFound.
func (s *sourceKeyService) Activate(ctx context.Context, sourceID uuid.UUID, id string) (*models.SourceKey, error) {
	return s.setActive(ctx, sourceID, id, true)
}

// Deactivate marks the key as inactive, or returns ErrSourceKeyNotFound.
func (s *sourceKeyService) Deactivate(ctx context.Context, sourceID uuid.UUID, id string) (*models.SourceKey, error) {
	return s.setActive(ctx, sourceID, id, false)
}

func (s *sourceKeyService) setActive(ctx context.Context, sourceID uuid.UUID, id string, active bool) (*models.SourceKey, error) {
	key, err := s.getOwned(ctx, sourceID, id)
	if err != nil {
		return nil, err
	}
	key.IsActive = active
	if err := s.keys.Update(ctx, key); err != nil {
		return nil, mapSourceKeyRepoError("update source key", err)
	}
	return key, nil
}

// Delete removes the key and its underlying secret, or returns
// ErrSourceKeyNotFound. The secret is removed on a best-effort basis after
// the key row is gone; a secret that is already missing is not an error.
func (s *sourceKeyService) Delete(ctx context.Context, sourceID uuid.UUID, id string) error {
	key, err := s.getOwned(ctx, sourceID, id)
	if err != nil {
		return err
	}
	if err := s.keys.Delete(ctx, id); err != nil {
		return mapSourceKeyRepoError("delete source key", err)
	}
	if err := s.secrets.Delete(ctx, key.SecretID); err != nil && !errors.Is(err, secrets.ErrNotFound) {
		return fmt.Errorf("services: delete source key secret: %w", err)
	}
	return nil
}

// getOwned returns the key with the given ID, scoped to sourceID.
// ErrSourceKeyNotFound covers both a missing key and one that belongs to a
// different source, so callers cannot distinguish the two.
func (s *sourceKeyService) getOwned(ctx context.Context, sourceID uuid.UUID, id string) (*models.SourceKey, error) {
	key, err := s.keys.Get(ctx, id)
	if err != nil {
		return nil, mapSourceKeyRepoError("get source key", err)
	}
	if key.SourceID != sourceID {
		return nil, ErrSourceKeyNotFound
	}
	return key, nil
}

// mapSourceKeyRepoError converts a repository error into a service-level
// error: repositories.ErrNotFound becomes ErrSourceKeyNotFound; anything else
// is wrapped with context and surfaces as an internal failure.
func mapSourceKeyRepoError(op string, err error) error {
	if errors.Is(err, repositories.ErrNotFound) {
		return ErrSourceKeyNotFound
	}
	return fmt.Errorf("services: %s: %w", op, err)
}
