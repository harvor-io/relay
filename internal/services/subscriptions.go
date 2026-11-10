package services

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/exalynt/turn/paginator/cursor"
	"github.com/gofrs/uuid/v5"
	"github.com/gosimple/slug"

	"github.com/harvor-io/relay/internal/bus"
	"github.com/harvor-io/relay/internal/models"
	"github.com/harvor-io/relay/internal/repositories"
)

// Errors returned by SubscriptionService. Callers should compare against
// these with errors.Is rather than inspecting messages.
var (
	// ErrSubscriptionNotFound means the destination isn't subscribed to the
	// topic.
	ErrSubscriptionNotFound = errors.New("subscription not found")

	// ErrSubscriptionTopicInvalid means a supplied topic is not a source's
	// slug followed by an event type, optionally ending in ".*", the slug
	// followed by ".*", or "*" alone. The error returned wraps it with the
	// topic.
	ErrSubscriptionTopicInvalid = errors.New("subscription topic is invalid")
)

// SubscriptionService is the subscription-management use-case API: the
// operations callers (HTTP handlers today) invoke, and the errors they can
// expect back. A subscription links a destination to one topic or pattern,
// and is identified by the two. While a destination is active, its
// subscriptions are kept routed on the eventbus; while it's inactive, they
// are only stored.
type SubscriptionService interface {
	// Get returns destinationID's subscription to topic, or
	// ErrSubscriptionNotFound.
	Get(ctx context.Context, destinationID uuid.UUID, topic string) (*models.Subscription, error)

	// List returns the page of destinationID's subscriptions page selects,
	// ordered by topic. It returns ErrDestinationNotFound if the destination
	// does not exist, or ErrInvalidPage if the selection is invalid.
	List(ctx context.Context, destinationID uuid.UUID, page cursor.Selector) (cursor.Page[models.Subscription], error)

	// Subscribe subscribes destinationID to topic, and reports whether the
	// subscription is new. Subscribing again changes nothing. It returns
	// ErrDestinationNotFound or ErrSubscriptionTopicInvalid.
	Subscribe(ctx context.Context, destinationID uuid.UUID, topic string) (*models.Subscription, bool, error)

	// Replace makes destinationID's subscriptions exactly topics: it
	// subscribes to the ones it isn't subscribed to and unsubscribes from
	// the ones topics doesn't name. Empty topics unsubscribes from every
	// topic. It returns ErrDestinationNotFound or ErrSubscriptionTopicInvalid,
	// in which case nothing changes.
	Replace(ctx context.Context, destinationID uuid.UUID, topics []string) error

	// Unsubscribe removes destinationID's subscription to topic. Not being
	// subscribed is not an error. It returns ErrDestinationNotFound.
	Unsubscribe(ctx context.Context, destinationID uuid.UUID, topic string) error
}

// subscriptionService is the default SubscriptionService.
type subscriptionService struct {
	subscriptions repositories.SubscriptionRepository
	destinations  repositories.DestinationRepository
	bus           bus.Bus
}

var _ SubscriptionService = (*subscriptionService)(nil)

// NewSubscriptionService returns a SubscriptionService backed by
// subscriptions for subscription records, destinations to look up the
// destination a subscription belongs to, and eventBus to route subscribed
// topics.
func NewSubscriptionService(
	subscriptions repositories.SubscriptionRepository,
	destinations repositories.DestinationRepository,
	eventBus bus.Bus,
) SubscriptionService {
	return &subscriptionService{subscriptions: subscriptions, destinations: destinations, bus: eventBus}
}

// Get returns destinationID's subscription to topic, or
// ErrSubscriptionNotFound.
func (s *subscriptionService) Get(ctx context.Context, destinationID uuid.UUID, topic string) (*models.Subscription, error) {
	subscription, err := s.subscriptions.Get(ctx, destinationID, models.Topic(topic))
	if err != nil {
		return nil, mapSubscriptionRepoError("get subscription", err)
	}
	return subscription, nil
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

// Subscribe stores the subscription if it's new and, if the destination is
// active, routes its topic on the eventbus. Routing an already-routed topic
// is a no-op, so a repeated call also repairs routing that failed before.
func (s *subscriptionService) Subscribe(ctx context.Context, destinationID uuid.UUID, topic string) (*models.Subscription, bool, error) {
	destination, err := s.destinations.Get(ctx, destinationID)
	if err != nil {
		return nil, false, mapDestinationRepoError("get destination", err)
	}
	if err := validateSubscriptionTopic(topic); err != nil {
		return nil, false, err
	}

	subscription, err := s.subscriptions.Get(ctx, destinationID, models.Topic(topic))
	created := false
	switch {
	case errors.Is(err, repositories.ErrNotFound):
		subscription = &models.Subscription{DestinationID: destinationID, Topic: models.Topic(topic)}
		if err := s.subscriptions.Create(ctx, subscription); err != nil {
			return nil, false, fmt.Errorf("services: create subscription: %w", err)
		}
		created = true
	case err != nil:
		return nil, false, fmt.Errorf("services: get subscription: %w", err)
	}

	if destination.IsActive {
		if err := s.bus.Subscribe(ctx, destination, []models.Subscription{*subscription}); err != nil {
			return nil, false, fmt.Errorf("services: subscribe on eventbus: %w", err)
		}
	}
	return subscription, created, nil
}

// Replace validates every topic before changing anything, stores exactly
// those subscriptions and, if the destination is active, routes the new
// topics and stops routing the dropped ones. New topics are routed before
// dropped ones stop, so a topic both covers is never interrupted.
func (s *subscriptionService) Replace(ctx context.Context, destinationID uuid.UUID, topics []string) error {
	destination, err := s.destinations.Get(ctx, destinationID)
	if err != nil {
		return mapDestinationRepoError("get destination", err)
	}
	wanted := make([]models.Topic, 0, len(topics))
	for _, topic := range topics {
		if err := validateSubscriptionTopic(topic); err != nil {
			return err
		}
		wanted = append(wanted, models.Topic(topic))
	}
	slices.Sort(wanted)
	wanted = slices.Compact(wanted)

	before, err := s.subscriptions.ListByDestination(ctx, destinationID)
	if err != nil {
		return fmt.Errorf("services: list subscriptions: %w", err)
	}
	if err := s.subscriptions.Replace(ctx, destinationID, wanted); err != nil {
		return fmt.Errorf("services: replace subscriptions: %w", err)
	}
	if !destination.IsActive {
		return nil
	}

	// Every wanted topic is routed, not only the new ones: routing is
	// idempotent, so this also repairs routing that failed before.
	routed := make([]models.Subscription, 0, len(wanted))
	for _, topic := range wanted {
		routed = append(routed, models.Subscription{DestinationID: destinationID, Topic: topic})
	}
	if err := s.bus.Subscribe(ctx, destination, routed); err != nil {
		return fmt.Errorf("services: subscribe on eventbus: %w", err)
	}
	dropped := slices.DeleteFunc(before, func(subscription models.Subscription) bool {
		_, kept := slices.BinarySearch(wanted, subscription.Topic)
		return kept
	})
	if err := s.bus.Unsubscribe(ctx, destination, dropped); err != nil {
		return fmt.Errorf("services: unsubscribe on eventbus: %w", err)
	}
	return nil
}

// Unsubscribe removes the subscription, if there is one, and, if the
// destination is active, stops routing its topic on the eventbus.
func (s *subscriptionService) Unsubscribe(ctx context.Context, destinationID uuid.UUID, topic string) error {
	destination, err := s.destinations.Get(ctx, destinationID)
	if err != nil {
		return mapDestinationRepoError("get destination", err)
	}
	subscription := models.Subscription{DestinationID: destinationID, Topic: models.Topic(topic)}
	if err := s.subscriptions.Delete(ctx, destinationID, subscription.Topic); err != nil && !errors.Is(err, repositories.ErrNotFound) {
		return fmt.Errorf("services: delete subscription: %w", err)
	}
	// Unrouting runs even when nothing was stored, so a repeated call also
	// repairs routing that a failed unsubscribe left behind.
	if destination.IsActive {
		if err := s.bus.Unsubscribe(ctx, destination, []models.Subscription{subscription}); err != nil {
			return fmt.Errorf("services: unsubscribe on eventbus: %w", err)
		}
	}
	return nil
}

// validateSubscriptionTopic checks that topic is "*", "<slug>.*",
// "<slug>.<event type>", or "<slug>.<event type>.*", within the length
// every topic is held to. The slug needn't belong to a source yet.
func validateSubscriptionTopic(topic string) error {
	invalid := fmt.Errorf(`%w: %q must be a source's slug followed by "." and an event type, optionally ending in %q; a slug followed by %q; or %q alone; and at most %d characters`,
		ErrSubscriptionTopicInvalid, topic, models.WildcardSuffix, models.WildcardSuffix, models.Wildcard, maxTopicLength)

	if topic == models.Wildcard {
		return nil
	}
	if len(topic) > maxTopicLength {
		return invalid
	}
	prefix, rest, ok := strings.Cut(topic, ".")
	if !ok || !slug.IsSlug(prefix) {
		return invalid
	}
	if rest == models.Wildcard {
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
