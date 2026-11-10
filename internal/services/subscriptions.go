package services

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/exalynt/turn/paginator/cursor"
	"github.com/gofrs/uuid/v5"

	"github.com/harvor-io/relay/internal/bus"
	"github.com/harvor-io/relay/internal/models"
	"github.com/harvor-io/relay/internal/repositories"
)

// Errors returned by SubscriptionService. Callers should compare against
// these with errors.Is rather than inspecting messages.
var (
	// ErrSubscriptionNotFound means no subscription exists with the
	// requested ID under the given destination.
	ErrSubscriptionNotFound = errors.New("subscription not found")

	// ErrSubscriptionSourceInvalid means the supplied source ID is not the
	// ID of an existing source.
	ErrSubscriptionSourceInvalid = errors.New("subscription source_id must be the ID of an existing source")

	// ErrSubscriptionIDInvalid means an explicitly supplied ID is not a
	// valid UUID.
	ErrSubscriptionIDInvalid = errors.New("subscription id must be a UUID")

	// ErrSubscriptionIDTaken means an explicitly supplied ID already belongs
	// to another subscription.
	ErrSubscriptionIDTaken = errors.New("subscription id is already in use")

	// ErrSubscriptionExists means the destination already has a
	// subscription to the source.
	ErrSubscriptionExists = errors.New("destination is already subscribed to this source; add topics to its subscription instead")

	// ErrSubscriptionTopicsRequired means a new subscription named no
	// topics.
	ErrSubscriptionTopicsRequired = errors.New("subscription must name at least one topic")

	// ErrSubscriptionTopicInvalid means a supplied topic is not
	// "<source slug>.<event type>", optionally ending in ".*". The error
	// returned wraps it with the topic and the slug it must start with.
	ErrSubscriptionTopicInvalid = errors.New("subscription topic is invalid")

	// ErrSubscriptionLastTopic means an update would leave the subscription
	// with no topics.
	ErrSubscriptionLastTopic = errors.New("a subscription must keep at least one topic; unsubscribe from the source instead")
)

// SubscriptionService is the subscription-management use-case API: the
// operations callers (HTTP handlers today) invoke, and the errors they can
// expect back. While a destination is active, its subscriptions are kept
// routed on the eventbus; while it's inactive, they are only stored.
type SubscriptionService interface {
	// Get returns the subscription with the given ID belonging to
	// destinationID, or ErrSubscriptionNotFound.
	Get(ctx context.Context, destinationID, id uuid.UUID) (*models.Subscription, error)

	// List returns the page of destinationID's subscriptions page selects,
	// ordered by source slug, then ID. It returns ErrDestinationNotFound if
	// the destination does not exist, or ErrInvalidPage if the selection is
	// invalid.
	List(ctx context.Context, destinationID uuid.UUID, page cursor.Selector) (cursor.Page[models.Subscription], error)

	// Subscribe validates input and subscribes the destination to the
	// source's topics. It returns ErrDestinationNotFound,
	// ErrSubscriptionSourceInvalid, ErrSubscriptionTopicsRequired,
	// ErrSubscriptionTopicInvalid, ErrSubscriptionIDInvalid,
	// ErrSubscriptionIDTaken, or ErrSubscriptionExists.
	Subscribe(ctx context.Context, destinationID uuid.UUID, input SubscribeInput) (*models.Subscription, error)

	// ChangeTopics adds input.Add to the subscription's topics and removes
	// input.Remove from them. It returns ErrSubscriptionNotFound,
	// ErrSubscriptionTopicInvalid, or ErrSubscriptionLastTopic.
	ChangeTopics(ctx context.Context, destinationID, id uuid.UUID, input ChangeTopicsInput) (*models.Subscription, error)

	// Unsubscribe removes the subscription, or returns
	// ErrSubscriptionNotFound.
	Unsubscribe(ctx context.Context, destinationID, id uuid.UUID) error
}

// subscriptionService is the default SubscriptionService.
type subscriptionService struct {
	subscriptions repositories.SubscriptionRepository
	destinations  repositories.DestinationRepository
	sources       repositories.SourceRepository
	bus           bus.Bus
}

var _ SubscriptionService = (*subscriptionService)(nil)

// NewSubscriptionService returns a SubscriptionService backed by
// subscriptions for subscription records, destinations and sources to look
// up what a subscription links, and eventBus to route subscribed topics.
func NewSubscriptionService(
	subscriptions repositories.SubscriptionRepository,
	destinations repositories.DestinationRepository,
	sources repositories.SourceRepository,
	eventBus bus.Bus,
) SubscriptionService {
	return &subscriptionService{subscriptions: subscriptions, destinations: destinations, sources: sources, bus: eventBus}
}

// SubscribeInput carries the caller-supplied fields for a new subscription.
type SubscribeInput struct {
	// ID is optional. When empty, the service generates a UUIDv7. When set,
	// it must be a valid UUID and must not already belong to another
	// subscription.
	ID string

	// SourceID is the ID of the source to subscribe to. Required.
	SourceID string

	// Topics are the source's topics the destination wants, each
	// "<source slug>.<event type>", or a pattern ending in ".*". At least
	// one is required. Duplicates are dropped.
	Topics []string
}

// ChangeTopicsInput carries the topics to add to and remove from a
// subscription. A topic in Remove that the subscription doesn't have is
// ignored.
type ChangeTopicsInput struct {
	Add    []string
	Remove []string
}

// Get returns the subscription with the given ID belonging to
// destinationID, or ErrSubscriptionNotFound.
func (s *subscriptionService) Get(ctx context.Context, destinationID, id uuid.UUID) (*models.Subscription, error) {
	return s.getOwned(ctx, destinationID, id)
}

// List returns the page of destinationID's subscriptions page selects.
func (s *subscriptionService) List(ctx context.Context, destinationID uuid.UUID, page cursor.Selector) (cursor.Page[models.Subscription], error) {
	if _, err := s.destinations.Get(ctx, destinationID); err != nil {
		return cursor.Page[models.Subscription]{}, mapDestinationRepoError("get destination", err)
	}
	subscriptions, err := s.subscriptions.List(ctx, repositories.SubscriptionQuery{DestinationID: destinationID, Page: page})
	if err != nil {
		return cursor.Page[models.Subscription]{}, listError("list subscriptions", err)
	}
	return subscriptions, nil
}

// Subscribe validates input, stores the subscription, and, if the
// destination is active, routes its topics on the eventbus.
func (s *subscriptionService) Subscribe(ctx context.Context, destinationID uuid.UUID, input SubscribeInput) (*models.Subscription, error) {
	destination, err := s.destinations.Get(ctx, destinationID)
	if err != nil {
		return nil, mapDestinationRepoError("get destination", err)
	}

	source, err := s.lookUpSource(ctx, input.SourceID)
	if err != nil {
		return nil, err
	}

	if len(input.Topics) == 0 {
		return nil, ErrSubscriptionTopicsRequired
	}
	topics, err := subscriptionTopics(source.Slug, nil, input.Topics)
	if err != nil {
		return nil, err
	}

	id, err := s.resolveID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	subscription := &models.Subscription{
		ID:            id,
		DestinationID: destination.ID,
		SourceID:      source.ID,
		Source:        source.Slug,
		Topics:        topics,
	}
	if err := s.subscriptions.Create(ctx, subscription); err != nil {
		if errors.Is(err, repositories.ErrConflict) {
			return nil, ErrSubscriptionExists
		}
		return nil, fmt.Errorf("services: create subscription: %w", err)
	}

	if destination.IsActive {
		if err := s.bus.Subscribe(ctx, subscription); err != nil {
			return nil, fmt.Errorf("services: subscribe on eventbus: %w", err)
		}
	}
	return subscription, nil
}

// ChangeTopics adds and removes topics, refusing to leave none, and, if the
// destination is active, moves its routing on the eventbus to match.
func (s *subscriptionService) ChangeTopics(ctx context.Context, destinationID, id uuid.UUID, input ChangeTopicsInput) (*models.Subscription, error) {
	subscription, err := s.getOwned(ctx, destinationID, id)
	if err != nil {
		return nil, err
	}
	destination, err := s.destinations.Get(ctx, destinationID)
	if err != nil {
		return nil, mapDestinationRepoError("get destination", err)
	}

	topics, err := subscriptionTopics(subscription.Source, subscription.Topics, input.Add)
	if err != nil {
		return nil, err
	}
	topics = slices.DeleteFunc(topics, func(t models.Topic) bool {
		return slices.Contains(input.Remove, t.String())
	})
	if len(topics) == 0 {
		return nil, ErrSubscriptionLastTopic
	}

	previous := *subscription
	subscription.Topics = topics
	if err := s.subscriptions.Update(ctx, subscription); err != nil {
		return nil, mapSubscriptionRepoError("update subscription", err)
	}

	if destination.IsActive {
		if err := s.bus.ChangeSubscription(ctx, &previous, subscription); err != nil {
			return nil, fmt.Errorf("services: change subscription on eventbus: %w", err)
		}
	}
	return subscription, nil
}

// Unsubscribe removes the subscription and, if the destination is active,
// stops routing its topics on the eventbus.
func (s *subscriptionService) Unsubscribe(ctx context.Context, destinationID, id uuid.UUID) error {
	subscription, err := s.getOwned(ctx, destinationID, id)
	if err != nil {
		return err
	}
	destination, err := s.destinations.Get(ctx, destinationID)
	if err != nil {
		return mapDestinationRepoError("get destination", err)
	}

	if err := s.subscriptions.Delete(ctx, id); err != nil {
		return mapSubscriptionRepoError("delete subscription", err)
	}

	if destination.IsActive {
		if err := s.bus.Unsubscribe(ctx, subscription); err != nil {
			return fmt.Errorf("services: unsubscribe on eventbus: %w", err)
		}
	}
	return nil
}

// resolveID returns the UUID to persist for a new subscription. An empty
// provided value returns the nil UUID, so the repository generates a
// UUIDv7. A non-empty value must parse as a UUID and must not already be in
// use.
func (s *subscriptionService) resolveID(ctx context.Context, provided string) (uuid.UUID, error) {
	provided = strings.TrimSpace(provided)
	if provided == "" {
		return uuid.Nil, nil
	}
	id, err := uuid.FromString(provided)
	if err != nil {
		return uuid.Nil, ErrSubscriptionIDInvalid
	}
	switch _, err := s.subscriptions.Get(ctx, id); {
	case err == nil:
		return uuid.Nil, ErrSubscriptionIDTaken
	case errors.Is(err, repositories.ErrNotFound):
		return id, nil
	default:
		return uuid.Nil, fmt.Errorf("services: look up subscription id: %w", err)
	}
}

// lookUpSource returns the source a new subscription names, or
// ErrSubscriptionSourceInvalid.
func (s *subscriptionService) lookUpSource(ctx context.Context, sourceID string) (*models.Source, error) {
	id, err := uuid.FromString(strings.TrimSpace(sourceID))
	if err != nil {
		return nil, ErrSubscriptionSourceInvalid
	}
	source, err := s.sources.Get(ctx, id)
	switch {
	case errors.Is(err, repositories.ErrNotFound):
		return nil, ErrSubscriptionSourceInvalid
	case err != nil:
		return nil, fmt.Errorf("services: get source: %w", err)
	}
	return source, nil
}

// getOwned returns the subscription with the given ID, scoped to
// destinationID. ErrSubscriptionNotFound covers both a missing subscription
// and one that belongs to a different destination, so callers cannot
// distinguish the two.
func (s *subscriptionService) getOwned(ctx context.Context, destinationID, id uuid.UUID) (*models.Subscription, error) {
	subscription, err := s.subscriptions.Get(ctx, id)
	if err != nil {
		return nil, mapSubscriptionRepoError("get subscription", err)
	}
	if subscription.DestinationID != destinationID {
		return nil, ErrSubscriptionNotFound
	}
	return subscription, nil
}

// subscriptionTopics validates each of added against slug and returns the
// union of existing and added, de-duplicated and sorted.
func subscriptionTopics(slug string, existing []models.Topic, added []string) ([]models.Topic, error) {
	topics := slices.Clone(existing)
	for _, topic := range added {
		if err := validateSubscriptionTopic(slug, topic); err != nil {
			return nil, err
		}
		topics = append(topics, models.Topic(topic))
	}
	slices.Sort(topics)
	return slices.Compact(topics), nil
}

// validateSubscriptionTopic checks that topic is "<slug>.<event type>",
// "<slug>.<event type>.*", or "<slug>.*", within the length every topic is
// held to.
func validateSubscriptionTopic(slug, topic string) error {
	invalid := fmt.Errorf("%w: %q must be %q followed by an event type, optionally ending in %q, or %q alone, and at most %d characters",
		ErrSubscriptionTopicInvalid, topic, slug+".", models.WildcardSuffix, slug+models.WildcardSuffix, maxTopicLength)

	if len(topic) > maxTopicLength {
		return invalid
	}
	rest, ok := strings.CutPrefix(topic, slug+".")
	if !ok {
		return invalid
	}
	if rest == "*" {
		return nil
	}
	rest, _ = strings.CutSuffix(rest, models.WildcardSuffix)
	if !sourceEventTypePattern.MatchString(rest) {
		return invalid
	}
	return nil
}

// mapSubscriptionRepoError converts a repository error into a service-level
// error: repositories.ErrNotFound becomes ErrSubscriptionNotFound; anything
// else is wrapped with context and surfaces as an internal failure.
func mapSubscriptionRepoError(op string, err error) error {
	if errors.Is(err, repositories.ErrNotFound) {
		return ErrSubscriptionNotFound
	}
	return fmt.Errorf("services: %s: %w", op, err)
}
