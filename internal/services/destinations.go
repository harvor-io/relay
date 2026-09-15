package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/gofrs/uuid/v5"

	"github.com/harvor-io/relay/internal/bus"
	"github.com/harvor-io/relay/internal/models"
	"github.com/harvor-io/relay/internal/repositories"
	"github.com/harvor-io/relay/pkg/encryptor"
	"github.com/harvor-io/relay/pkg/secrets"
)

// Bounds enforced on caller-supplied destination fields. They are generous
// limits meant to reject obviously malformed input, not to encode product
// policy.
const (
	maxDestinationNameLength        = 255
	maxDestinationDescriptionLength = 1024

	// maxDestinationSubscriptionLength bounds each caller-supplied
	// subscription topic.
	maxDestinationSubscriptionLength = 255
)

// destinationSubscriptionPattern matches an exact subscription topic: the
// same character restrictions as an event's type — alphanumeric characters
// plus ".", "_", and "-", with no whitespace.
var destinationSubscriptionPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// destinationSubscriptionWildcardPattern matches a subscription ending in
// ".*", which denotes a catch-all for every topic under that prefix (for
// example, "crm.*" matches "crm.account.created").
var destinationSubscriptionWildcardPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+\.\*$`)

// Errors returned by DestinationService. Callers should compare against
// these with errors.Is rather than inspecting messages.
var (
	// ErrDestinationNotFound means no destination exists with the requested
	// ID.
	ErrDestinationNotFound = errors.New("destination not found")

	// ErrDestinationNameRequired means the supplied name was empty or
	// whitespace.
	ErrDestinationNameRequired = errors.New("destination name is required")

	// ErrDestinationNameTooLong means the supplied name exceeded the length
	// bound.
	ErrDestinationNameTooLong = fmt.Errorf("destination name must be at most %d characters", maxDestinationNameLength)

	// ErrDestinationDescriptionTooLong means the supplied description
	// exceeded the length bound.
	ErrDestinationDescriptionTooLong = fmt.Errorf("destination description must be at most %d characters", maxDestinationDescriptionLength)

	// ErrDestinationIDTaken means an explicitly supplied ID already belongs
	// to another destination.
	ErrDestinationIDTaken = errors.New("destination id is already in use")

	// ErrDestinationIDInvalid means an explicitly supplied ID is not a valid
	// UUID.
	ErrDestinationIDInvalid = errors.New("destination id must be a UUID")

	// ErrDestinationTypeInvalid means an explicitly supplied type is not one
	// of the supported DestinationType values.
	ErrDestinationTypeInvalid = fmt.Errorf("destination type must be one of: %s", destinationTypeWebhookName)

	// ErrDestinationConfigRequired means no config was supplied.
	ErrDestinationConfigRequired = errors.New("destination config is required")

	// ErrDestinationConfigInvalid means the supplied config was not valid
	// JSON matching the shape required by the destination's type.
	ErrDestinationConfigInvalid = errors.New("destination config is not valid JSON")

	// ErrDestinationConfigURLRequired means a webhook config was supplied
	// without a url.
	ErrDestinationConfigURLRequired = errors.New("destination config url is required")

	// ErrDestinationAuthTypeInvalid means a webhook config's auth.type was
	// not one of the supported AuthType values.
	ErrDestinationAuthTypeInvalid = fmt.Errorf("destination auth type must be one of: %s", authTypeHMACName)

	// ErrDestinationHMACSecretRequired means an hmac auth config was
	// supplied without a signing secret.
	ErrDestinationHMACSecretRequired = errors.New("destination auth config secret is required")

	// ErrDestinationHMACConfigInvalid means an hmac auth config was missing
	// one of its other required fields (algorithm, signature_header,
	// encoding, signing_template, or timestamp_header).
	ErrDestinationHMACConfigInvalid = errors.New("destination auth config is missing required hmac fields")

	// ErrDestinationSubscriptionInvalid means a supplied subscription
	// contained characters other than alphanumerics, ".", "_", or "-", or
	// misused the "*" wildcard.
	ErrDestinationSubscriptionInvalid = errors.New(`destination subscriptions must contain only alphanumeric characters, '.', '_', and '-', optionally ending in ".*" to match every topic under a prefix`)

	// ErrDestinationSubscriptionTooLong means a supplied subscription
	// exceeded the length bound.
	ErrDestinationSubscriptionTooLong = fmt.Errorf("destination subscription must be at most %d characters", maxDestinationSubscriptionLength)
)

// authTypeHMACName is models.AuthTypeHMAC's string value, used to build
// ErrDestinationAuthTypeInvalid's message and to validate caller-supplied
// auth types.
const authTypeHMACName = string(models.AuthTypeHMAC)

// destinationTypeWebhookName is models.DestinationTypeWebhook's string value,
// used to build ErrDestinationTypeInvalid's message and to validate
// caller-supplied types.
const destinationTypeWebhookName = string(models.DestinationTypeWebhook)

// DestinationService is the destination-management use-case API: the
// operations callers (HTTP handlers today) invoke, and the errors they can
// expect back.
type DestinationService interface {
	// Get returns the destination with the given ID, or
	// ErrDestinationNotFound.
	Get(ctx context.Context, id uuid.UUID) (*models.Destination, error)

	// List returns every destination, ordered by name.
	List(ctx context.Context) ([]models.Destination, error)

	// Create validates input and persists a new destination, which starts
	// active. When the config's auth.type is "hmac", the caller-supplied
	// plaintext signing secret is encrypted and stored in the secrets
	// store, and only the resulting secret ID is persisted in the
	// destination's config; the plaintext is never stored or returned. It
	// returns ErrDestinationNameRequired, ErrDestinationNameTooLong,
	// ErrDestinationDescriptionTooLong, ErrDestinationTypeInvalid,
	// ErrDestinationConfigRequired, ErrDestinationConfigInvalid,
	// ErrDestinationConfigURLRequired, ErrDestinationAuthTypeInvalid,
	// ErrDestinationHMACSecretRequired, ErrDestinationHMACConfigInvalid,
	// ErrDestinationSubscriptionInvalid, ErrDestinationSubscriptionTooLong,
	// ErrDestinationIDInvalid, or ErrDestinationIDTaken when input is
	// invalid or conflicts with an existing destination.
	Create(ctx context.Context, input CreateDestinationInput) (*models.Destination, error)

	// Update changes the name, description, and/or subscriptions of the
	// destination with the given ID. Fields left nil in input are
	// unchanged; a non-nil Description that is empty after trimming clears
	// it, matching Create. It returns ErrDestinationNotFound,
	// ErrDestinationNameRequired, ErrDestinationNameTooLong,
	// ErrDestinationDescriptionTooLong, ErrDestinationSubscriptionInvalid,
	// or ErrDestinationSubscriptionTooLong when input is invalid.
	Update(ctx context.Context, id uuid.UUID, input UpdateDestinationInput) (*models.Destination, error)

	// Activate marks the destination as active, or returns
	// ErrDestinationNotFound.
	Activate(ctx context.Context, id uuid.UUID) (*models.Destination, error)

	// Deactivate marks the destination as inactive, or returns
	// ErrDestinationNotFound.
	Deactivate(ctx context.Context, id uuid.UUID) (*models.Destination, error)

	// Delete removes the destination with the given ID, or returns
	// ErrDestinationNotFound. If the destination's config referenced an
	// hmac secret, the secret is removed on a best-effort basis after the
	// destination is gone. Its eventbus topology (delivery, retry, and
	// dead-letter queues) is torn down the same way.
	Delete(ctx context.Context, id uuid.UUID) error
}

// destinationService is the default DestinationService, backed by a
// repositories.DestinationRepository for destination records, a
// secrets.Store for secret values referenced from a destination's config
// (for example, an hmac auth config's signing secret), and a bus.Bus to
// provision the eventbus topology that delivers envelopes matching a
// destination's subscriptions.
type destinationService struct {
	destinations repositories.DestinationRepository
	secrets      secrets.Store
	encryptor    encryptor.Encryptor
	bus          bus.Bus
}

var _ DestinationService = (*destinationService)(nil)

// NewDestinationService returns a DestinationService backed by destinations
// for destination records, secretStore for secret values referenced from a
// destination's config, enc to encrypt and decrypt them, and eventBus to
// provision the eventbus topology backing each destination's subscriptions.
func NewDestinationService(
	destinations repositories.DestinationRepository,
	secretStore secrets.Store,
	enc encryptor.Encryptor,
	eventBus bus.Bus,
) DestinationService {
	return &destinationService{destinations: destinations, secrets: secretStore, encryptor: enc, bus: eventBus}
}

// CreateDestinationInput carries the caller-supplied fields for a new
// destination. The service trims surrounding whitespace from every field and
// treats a description that is empty after trimming as absent.
type CreateDestinationInput struct {
	Name        string
	Description *string

	// Type is optional. When empty, it defaults to
	// models.DestinationTypeWebhook, currently the only supported type. When
	// set, it must equal "webhook".
	Type string

	// Config is the Type-specific delivery configuration. It is required and
	// validated against the shape Type expects: for
	// models.DestinationTypeWebhook, it must decode as models.WebhookConfig
	// with a non-empty URL, and its Auth, if present, must have Type "hmac".
	Config json.RawMessage

	// Subscriptions lists the topic patterns this destination wants to
	// receive envelopes for. Each entry is trimmed and must contain only
	// alphanumeric characters, ".", "_", and "-" — the same restriction as
	// an event's type — optionally ending in ".*" to match every topic
	// under that prefix (for example, "crm.*" matches
	// "crm.account.created"). Optional; a destination with no subscriptions
	// receives nothing.
	Subscriptions []string

	// ID is optional. When empty, the service generates a UUIDv7. When set,
	// it must be a valid UUID and must not already belong to another
	// destination.
	ID string
}

// UpdateDestinationInput carries the caller-supplied fields to change on an
// existing destination. A nil field is left unchanged. Name, when supplied,
// is trimmed and validated the same way as on Create. Description, when
// supplied, is trimmed and normalised the same way as on Create — including
// that a value which is empty after trimming clears the description; to
// leave the description untouched, leave this nil rather than supplying "".
// Subscriptions, when supplied, replaces the destination's subscriptions
// wholesale and is validated the same way as on Create; a non-nil pointer to
// an empty slice clears the subscriptions.
type UpdateDestinationInput struct {
	Name          *string
	Description   *string
	Subscriptions *[]string
}

// Get returns the destination with the given ID, or ErrDestinationNotFound.
func (s *destinationService) Get(ctx context.Context, id uuid.UUID) (*models.Destination, error) {
	destination, err := s.destinations.Get(ctx, id)
	if err != nil {
		return nil, mapDestinationRepoError("get destination", err)
	}
	return destination, nil
}

// List returns every destination, ordered by name.
func (s *destinationService) List(ctx context.Context) ([]models.Destination, error) {
	destinations, err := s.destinations.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("services: list destinations: %w", err)
	}
	return destinations, nil
}

// Create validates input and persists a new destination, returning it with
// its assigned ID and timestamps. It also provisions the destination's
// eventbus topology, bound to receive envelopes matching its subscriptions.
// It returns ErrDestinationNameRequired, ErrDestinationNameTooLong,
// ErrDestinationDescriptionTooLong, ErrDestinationTypeInvalid,
// ErrDestinationConfigRequired, ErrDestinationConfigInvalid,
// ErrDestinationConfigURLRequired, ErrDestinationAuthTypeInvalid,
// ErrDestinationHMACSecretRequired, ErrDestinationHMACConfigInvalid,
// ErrDestinationSubscriptionInvalid, or ErrDestinationSubscriptionTooLong
// when input is invalid.
func (s *destinationService) Create(ctx context.Context, input CreateDestinationInput) (*models.Destination, error) {
	name := strings.TrimSpace(input.Name)
	switch {
	case name == "":
		return nil, ErrDestinationNameRequired
	case len(name) > maxDestinationNameLength:
		return nil, ErrDestinationNameTooLong
	}

	description, err := normaliseDestinationDescription(input.Description)
	if err != nil {
		return nil, err
	}

	typ, err := resolveDestinationType(input.Type)
	if err != nil {
		return nil, err
	}

	cfg, err := s.resolveDestinationConfig(ctx, typ, input.Config)
	if err != nil {
		return nil, err
	}

	subscriptions, err := resolveDestinationSubscriptions(input.Subscriptions)
	if err != nil {
		return nil, err
	}

	id, err := s.resolveID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	destination := &models.Destination{ID: id, Name: name, Type: typ, Config: cfg, Description: description, Subscriptions: subscriptions, IsActive: true}
	if err := s.destinations.Create(ctx, destination); err != nil {
		if errors.Is(err, repositories.ErrConflict) {
			return nil, ErrDestinationIDTaken
		}
		return nil, fmt.Errorf("services: create destination: %w", err)
	}

	if err := s.bus.Apply(ctx, destination); err != nil {
		return nil, fmt.Errorf("services: provision destination eventbus topology: %w", err)
	}
	return destination, nil
}

// Update changes the name, description, and/or subscriptions of the
// destination with the given ID, leaving fields nil in input unchanged. When
// Subscriptions is supplied, the destination's eventbus topology is rebound
// from scratch to match. It returns ErrDestinationNotFound,
// ErrDestinationNameRequired, ErrDestinationNameTooLong,
// ErrDestinationDescriptionTooLong, ErrDestinationSubscriptionInvalid, or
// ErrDestinationSubscriptionTooLong when input is invalid.
func (s *destinationService) Update(ctx context.Context, id uuid.UUID, input UpdateDestinationInput) (*models.Destination, error) {
	destination, err := s.destinations.Get(ctx, id)
	if err != nil {
		return nil, mapDestinationRepoError("get destination", err)
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		switch {
		case name == "":
			return nil, ErrDestinationNameRequired
		case len(name) > maxDestinationNameLength:
			return nil, ErrDestinationNameTooLong
		}
		destination.Name = name
	}

	if input.Description != nil {
		description, err := normaliseDestinationDescription(input.Description)
		if err != nil {
			return nil, err
		}
		destination.Description = description
	}

	rebind := false
	if input.Subscriptions != nil {
		subscriptions, err := resolveDestinationSubscriptions(*input.Subscriptions)
		if err != nil {
			return nil, err
		}
		destination.Subscriptions = subscriptions
		rebind = true
	}

	if err := s.destinations.Update(ctx, destination); err != nil {
		return nil, mapDestinationRepoError("update destination", err)
	}

	if rebind {
		// Apply only adds bindings for the subscriptions it's given; it
		// doesn't remove ones left over from before. Remove and reapply to
		// rebind from scratch.
		if err := s.bus.Remove(ctx, destination); err != nil {
			return nil, fmt.Errorf("services: remove destination eventbus topology: %w", err)
		}
		if err := s.bus.Apply(ctx, destination); err != nil {
			return nil, fmt.Errorf("services: provision destination eventbus topology: %w", err)
		}
	}
	return destination, nil
}

// Activate marks the destination as active, or returns
// ErrDestinationNotFound.
func (s *destinationService) Activate(ctx context.Context, id uuid.UUID) (*models.Destination, error) {
	return s.setActive(ctx, id, true)
}

// Deactivate marks the destination as inactive, or returns
// ErrDestinationNotFound.
func (s *destinationService) Deactivate(ctx context.Context, id uuid.UUID) (*models.Destination, error) {
	return s.setActive(ctx, id, false)
}

func (s *destinationService) setActive(ctx context.Context, id uuid.UUID, active bool) (*models.Destination, error) {
	destination, err := s.destinations.Get(ctx, id)
	if err != nil {
		return nil, mapDestinationRepoError("get destination", err)
	}
	destination.IsActive = active
	if err := s.destinations.Update(ctx, destination); err != nil {
		return nil, mapDestinationRepoError("update destination", err)
	}
	return destination, nil
}

// Delete removes the destination with the given ID, returning
// ErrDestinationNotFound if it does not exist. Its eventbus topology
// (delivery, retry, and dead-letter queues) is torn down after the
// destination row is gone. If the destination's config referenced an hmac
// secret, the secret is removed on a best-effort basis after that; a secret
// that is already missing is not an error.
func (s *destinationService) Delete(ctx context.Context, id uuid.UUID) error {
	destination, err := s.destinations.Get(ctx, id)
	if err != nil {
		return mapDestinationRepoError("get destination", err)
	}

	if err := s.destinations.Delete(ctx, id); err != nil {
		return mapDestinationRepoError("delete destination", err)
	}

	if err := s.bus.Remove(ctx, destination); err != nil {
		return fmt.Errorf("services: remove destination eventbus topology: %w", err)
	}

	secretID, ok := hmacSecretID(destination.Type, destination.Config)
	if !ok {
		return nil
	}
	if err := s.secrets.Delete(ctx, secretID); err != nil && !errors.Is(err, secrets.ErrNotFound) {
		return fmt.Errorf("services: delete destination hmac secret: %w", err)
	}
	return nil
}

// resolveID returns the UUID to persist for a new destination. An empty
// provided value returns the nil UUID, so the repository generates a
// UUIDv7. A non-empty value must parse as a UUID and must not already be in
// use.
func (s *destinationService) resolveID(ctx context.Context, provided string) (uuid.UUID, error) {
	provided = strings.TrimSpace(provided)
	if provided == "" {
		return uuid.Nil, nil
	}

	id, err := uuid.FromString(provided)
	if err != nil {
		return uuid.Nil, ErrDestinationIDInvalid
	}

	taken, err := s.idTaken(ctx, id)
	if err != nil {
		return uuid.Nil, err
	}
	if taken {
		return uuid.Nil, ErrDestinationIDTaken
	}
	return id, nil
}

// idTaken reports whether a destination already uses id.
func (s *destinationService) idTaken(ctx context.Context, id uuid.UUID) (bool, error) {
	switch _, err := s.destinations.Get(ctx, id); {
	case err == nil:
		return true, nil
	case errors.Is(err, repositories.ErrNotFound):
		return false, nil
	default:
		return false, fmt.Errorf("services: look up destination id: %w", err)
	}
}

// resolveDestinationType returns the DestinationType to persist for a new
// destination. An empty provided value defaults to
// models.DestinationTypeWebhook; any other value must match a supported
// DestinationType exactly, or ErrDestinationTypeInvalid is returned.
func resolveDestinationType(provided string) (models.DestinationType, error) {
	provided = strings.TrimSpace(provided)
	if provided == "" {
		return models.DestinationTypeWebhook, nil
	}
	if provided != destinationTypeWebhookName {
		return "", ErrDestinationTypeInvalid
	}
	return models.DestinationType(provided), nil
}

// resolveDestinationSubscriptions trims and validates each caller-supplied
// subscription topic and returns the models.Subscriptions to persist. Each
// entry must contain only alphanumeric characters, ".", "_", and "-" —
// the same restriction as an event's type — optionally ending in ".*" to
// match every topic under that prefix.
func resolveDestinationSubscriptions(provided []string) ([]models.Subscription, error) {
	subscriptions := make([]models.Subscription, 0, len(provided))
	for _, raw := range provided {
		trimmed := strings.TrimSpace(raw)
		switch {
		case len(trimmed) > maxDestinationSubscriptionLength:
			return nil, ErrDestinationSubscriptionTooLong
		case !destinationSubscriptionPattern.MatchString(trimmed) && !destinationSubscriptionWildcardPattern.MatchString(trimmed):
			return nil, ErrDestinationSubscriptionInvalid
		}
		subscriptions = append(subscriptions, models.Subscription(trimmed))
	}
	return subscriptions, nil
}

// resolveDestinationConfig validates raw against the config shape typ
// requires and returns the config to persist. Today typ is always
// models.DestinationTypeWebhook, so this always resolves raw as a
// models.WebhookConfig.
func (s *destinationService) resolveDestinationConfig(ctx context.Context, typ models.DestinationType, raw json.RawMessage) (json.RawMessage, error) {
	switch typ {
	case models.DestinationTypeWebhook:
		return s.resolveWebhookConfig(ctx, raw)
	default:
		return nil, ErrDestinationTypeInvalid
	}
}

// resolveWebhookConfig checks raw decodes as a models.WebhookConfig with a
// non-empty URL. When Auth is present, its Type must be "hmac" — the only
// AuthType supported today — and its Config is resolved by
// resolveHMACAuthConfig, which replaces the caller-supplied plaintext
// secret with a reference to where it is stored. A webhook with no Auth is
// returned unchanged.
func (s *destinationService) resolveWebhookConfig(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		return nil, ErrDestinationConfigRequired
	}

	var cfg models.WebhookConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, ErrDestinationConfigInvalid
	}
	if strings.TrimSpace(cfg.URL) == "" {
		return nil, ErrDestinationConfigURLRequired
	}
	if cfg.Auth == nil {
		return raw, nil
	}
	if cfg.Auth.Type != models.AuthTypeHMAC {
		return nil, ErrDestinationAuthTypeInvalid
	}

	authConfig, err := s.resolveHMACAuthConfig(ctx, cfg.Auth.Config)
	if err != nil {
		return nil, err
	}

	resolved, err := json.Marshal(models.WebhookConfig{
		URL:  cfg.URL,
		Auth: &models.AuthConfig{Type: models.AuthTypeHMAC, Config: authConfig},
	})
	if err != nil {
		return nil, fmt.Errorf("services: marshal destination config: %w", err)
	}
	return resolved, nil
}

// hmacAuthConfigInput is the caller-supplied shape of AuthConfig.Config when
// AuthConfig.Type is models.AuthTypeHMAC. The caller supplies the plaintext
// signing secret directly; resolveHMACAuthConfig encrypts and stores it, and
// persists only a reference (see models.HMACAuthConfig).
type hmacAuthConfigInput struct {
	// Secret is the plaintext HMAC signing secret. It is never persisted
	// verbatim: it is encrypted and stored in the secrets store, and only
	// the resulting secret ID is written to the destination's config.
	Secret string `json:"secret"`

	Algorithm       string `json:"algorithm"`
	SignatureHeader string `json:"signature_header"`
	Encoding        string `json:"encoding"`
	SigningTemplate string `json:"signing_template"`
	TimestampHeader string `json:"timestamp_header"`
}

// resolveHMACAuthConfig validates raw as an hmacAuthConfigInput, encrypts and
// stores its plaintext secret in the secrets store, and returns the
// models.HMACAuthConfig to persist: the same fields, with the secret
// replaced by the stored secret's ID. It returns
// ErrDestinationHMACSecretRequired if secret is empty, or
// ErrDestinationHMACConfigInvalid if any other required field is empty.
func (s *destinationService) resolveHMACAuthConfig(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
	var input hmacAuthConfigInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, ErrDestinationConfigInvalid
	}

	secret := strings.TrimSpace(input.Secret)
	if secret == "" {
		return nil, ErrDestinationHMACSecretRequired
	}
	if strings.TrimSpace(input.Algorithm) == "" ||
		strings.TrimSpace(input.SignatureHeader) == "" ||
		strings.TrimSpace(input.Encoding) == "" ||
		strings.TrimSpace(input.SigningTemplate) == "" ||
		strings.TrimSpace(input.TimestampHeader) == "" {
		return nil, ErrDestinationHMACConfigInvalid
	}

	ciphertext, err := s.encryptor.Encrypt([]byte(secret))
	if err != nil {
		return nil, fmt.Errorf("services: encrypt destination hmac secret: %w", err)
	}
	stored := &secrets.Secret{Value: ciphertext}
	if err := s.secrets.Create(ctx, stored); err != nil {
		return nil, fmt.Errorf("services: store destination hmac secret: %w", err)
	}

	resolved, err := json.Marshal(models.HMACAuthConfig{
		SecretID:        stored.ID.String(),
		Algorithm:       input.Algorithm,
		SignatureHeader: input.SignatureHeader,
		Encoding:        input.Encoding,
		SigningTemplate: input.SigningTemplate,
		TimestampHeader: input.TimestampHeader,
	})
	if err != nil {
		return nil, fmt.Errorf("services: marshal destination hmac auth config: %w", err)
	}
	return resolved, nil
}

// hmacSecretID extracts the secrets store ID referenced by config's hmac
// auth, if any. It reports ok == false whenever config does not carry an
// hmac secret reference — including a non-webhook typ, unparseable config,
// absent or non-hmac auth, or a malformed secret ID — since Delete treats
// secret cleanup as best-effort.
func hmacSecretID(typ models.DestinationType, config json.RawMessage) (id uuid.UUID, ok bool) {
	if typ != models.DestinationTypeWebhook {
		return uuid.Nil, false
	}
	var cfg models.WebhookConfig
	if err := json.Unmarshal(config, &cfg); err != nil || cfg.Auth == nil || cfg.Auth.Type != models.AuthTypeHMAC {
		return uuid.Nil, false
	}
	var auth models.HMACAuthConfig
	if err := json.Unmarshal(cfg.Auth.Config, &auth); err != nil || auth.SecretID == "" {
		return uuid.Nil, false
	}
	id, err := uuid.FromString(auth.SecretID)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

// normaliseDestinationDescription trims the optional description and
// collapses an empty result to nil. It returns
// ErrDestinationDescriptionTooLong if the trimmed value exceeds the length
// bound.
func normaliseDestinationDescription(raw *string) (*string, error) {
	if raw == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil, nil
	}
	if len(trimmed) > maxDestinationDescriptionLength {
		return nil, ErrDestinationDescriptionTooLong
	}
	return &trimmed, nil
}

// mapDestinationRepoError converts a repository error into a service-level
// error: repositories.ErrNotFound becomes ErrDestinationNotFound; anything
// else is wrapped with context and surfaces as an internal failure.
func mapDestinationRepoError(op string, err error) error {
	if errors.Is(err, repositories.ErrNotFound) {
		return ErrDestinationNotFound
	}
	return fmt.Errorf("services: %s: %w", op, err)
}
