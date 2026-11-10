// Package rabbitmq is the RabbitMQ-backed implementation of bus.Bus.
package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/harvor-io/relay/internal/bus"
	"github.com/harvor-io/relay/internal/models"
)

// DriverName is the config.Config.EventBusDriver value that selects this
// backend.
const DriverName = "rabbitmq"

// ingestExchange is the single topic exchange Relay publishes ingested
// events to. Each destination's queue is bound to it once per topic its
// subscriptions name.
const ingestExchange = "relay_ingest"

// destinationQueuePrefix starts the name of every destination's queue,
// which ends with the destination's ID.
const destinationQueuePrefix = "relay.destination."

// Bus is the RabbitMQ-backed implementation of bus.Bus.
type Bus struct {
	conn *amqp.Connection
}

var _ bus.Bus = (*Bus)(nil)

// New returns a Bus backed by conn.
func New(conn *amqp.Connection) *Bus {
	return &Bus{conn: conn}
}

// Setup declares the relay_ingest topic exchange if it does not already
// exist. Declaring an exchange in RabbitMQ is itself idempotent -
// redeclaring one with matching arguments is a no-op - so Setup is safe to
// call on every startup.
func (b *Bus) Setup(ctx context.Context) error {
	ch, err := b.conn.Channel()
	if err != nil {
		return fmt.Errorf("rabbitmq: open channel: %w", err)
	}
	defer func() { _ = ch.Close() }()

	err = ch.ExchangeDeclare(
		ingestExchange,     // name
		amqp.ExchangeTopic, // kind
		true,               // durable
		false,              // auto-deleted
		false,              // internal
		false,              // no-wait
		nil,                // arguments
	)
	if err != nil {
		return fmt.Errorf("rabbitmq: declare %s exchange: %w", ingestExchange, err)
	}
	return nil
}

// envelopeMessage is the wire format published to the ingest exchange: an
// explicit projection of models.Envelope so the message body is stable
// regardless of the Go struct's field layout.
type envelopeMessage struct {
	ID        string          `json:"id"`
	SourceID  string          `json:"source_id"`
	Source    string          `json:"source"`
	Type      string          `json:"type"`
	Topic     string          `json:"topic"`
	Data      json.RawMessage `json:"data"`
	CreatedAt time.Time       `json:"created_at"`
}

// Publish sends envelope to the relay_ingest exchange, routed by its Topic.
func (b *Bus) Publish(ctx context.Context, envelope *models.Envelope) error {
	ch, err := b.conn.Channel()
	if err != nil {
		return fmt.Errorf("rabbitmq: open channel: %w", err)
	}
	defer func() { _ = ch.Close() }()

	body, err := json.Marshal(envelopeMessage{
		ID:        envelope.ID.String(),
		SourceID:  envelope.SourceID.String(),
		Source:    envelope.Source,
		Type:      envelope.Type,
		Topic:     envelope.Topic().String(),
		Data:      envelope.Data,
		CreatedAt: envelope.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("rabbitmq: marshal envelope: %w", err)
	}

	err = ch.PublishWithContext(
		ctx,
		ingestExchange,
		envelope.Topic().String(),
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    envelope.CreatedAt,
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("rabbitmq: publish to %s exchange: %w", ingestExchange, err)
	}
	return nil
}

// destinationQueue returns the name of the durable queue that holds events
// routed to the destination with the given ID.
func destinationQueue(id fmt.Stringer) string {
	return destinationQueuePrefix + id.String()
}

// bindingKey translates a subscription's topic into a routing key pattern
// for the topic exchange. A trailing "*" becomes "#", which matches any
// number of parts. Event types can't contain "*" or "#", so an exact topic
// binds as itself and never acts as a wildcard.
func bindingKey(topic models.Topic) string {
	if prefix, ok := strings.CutSuffix(topic.String(), models.WildcardSuffix); ok {
		return prefix + ".#"
	}
	return topic.String()
}

// isNotFound reports whether err is the broker saying a queue or exchange
// doesn't exist. The broker closes the channel when it says so.
func isNotFound(err error) bool {
	var amqpErr *amqp.Error
	return errors.As(err, &amqpErr) && amqpErr.Code == amqp.NotFound
}

// withChannel runs fn on a channel of its own, closing it afterwards.
func (b *Bus) withChannel(fn func(ch *amqp.Channel) error) error {
	ch, err := b.conn.Channel()
	if err != nil {
		return fmt.Errorf("rabbitmq: open channel: %w", err)
	}
	defer func() { _ = ch.Close() }()
	return fn(ch)
}

// declareQueue declares the durable queue for the destination with the
// given ID. Redeclaring an existing queue with the same arguments is a
// no-op.
func declareQueue(ch *amqp.Channel, id fmt.Stringer) error {
	name := destinationQueue(id)
	if _, err := ch.QueueDeclare(
		name,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	); err != nil {
		return fmt.Errorf("rabbitmq: declare queue %s: %w", name, err)
	}
	return nil
}

// bindTopics binds the destination's queue to the ingest exchange for each
// topic. Binding a key that is already bound is a no-op.
func bindTopics(ch *amqp.Channel, id fmt.Stringer, topics []models.Topic) error {
	name := destinationQueue(id)
	for _, topic := range topics {
		if err := ch.QueueBind(name, bindingKey(topic), ingestExchange, false, nil); err != nil {
			return fmt.Errorf("rabbitmq: bind queue %s: %w", name, err)
		}
	}
	return nil
}

// unbindTopics removes the destination queue's binding for each topic.
// Unbinding a key that isn't bound is a no-op. A queue that doesn't exist
// has no bindings left, so it counts as success.
func unbindTopics(ch *amqp.Channel, id fmt.Stringer, topics []models.Topic) error {
	name := destinationQueue(id)
	for _, topic := range topics {
		if err := ch.QueueUnbind(name, bindingKey(topic), ingestExchange, nil); err != nil {
			if isNotFound(err) {
				return nil
			}
			return fmt.Errorf("rabbitmq: unbind queue %s: %w", name, err)
		}
	}
	return nil
}

// AddDestination declares the destination's durable queue.
func (b *Bus) AddDestination(_ context.Context, destination *models.Destination) error {
	return b.withChannel(func(ch *amqp.Channel) error {
		return declareQueue(ch, destination.ID)
	})
}

// RemoveDestination deletes the destination's queue, which removes its
// bindings and any messages waiting in it. A queue that doesn't exist
// counts as success.
func (b *Bus) RemoveDestination(_ context.Context, destination *models.Destination) error {
	return b.withChannel(func(ch *amqp.Channel) error {
		name := destinationQueue(destination.ID)
		if _, err := ch.QueueDelete(name, false, false, false); err != nil && !isNotFound(err) {
			return fmt.Errorf("rabbitmq: delete queue %s: %w", name, err)
		}
		return nil
	})
}

// Subscribe declares the destination's queue, then binds it for each of
// subscription's topics.
func (b *Bus) Subscribe(_ context.Context, subscription *models.Subscription) error {
	return b.withChannel(func(ch *amqp.Channel) error {
		if err := declareQueue(ch, subscription.DestinationID); err != nil {
			return err
		}
		return bindTopics(ch, subscription.DestinationID, subscription.Topics)
	})
}

// Unsubscribe removes the destination queue's binding for each of
// subscription's topics.
func (b *Bus) Unsubscribe(_ context.Context, subscription *models.Subscription) error {
	return b.withChannel(func(ch *amqp.Channel) error {
		return unbindTopics(ch, subscription.DestinationID, subscription.Topics)
	})
}

// ChangeSubscription declares the destination's queue and binds it for
// each of current's topics, then removes the bindings for the topics only
// previous has.
func (b *Bus) ChangeSubscription(_ context.Context, previous, current *models.Subscription) error {
	return b.withChannel(func(ch *amqp.Channel) error {
		if err := declareQueue(ch, current.DestinationID); err != nil {
			return err
		}
		if err := bindTopics(ch, current.DestinationID, current.Topics); err != nil {
			return err
		}
		return unbindTopics(ch, previous.DestinationID, removedTopics(previous.Topics, current.Topics))
	})
}

// removedTopics returns the topics in previous that current doesn't have.
func removedTopics(previous, current []models.Topic) []models.Topic {
	kept := make(map[models.Topic]bool, len(current))
	for _, topic := range current {
		kept[topic] = true
	}
	var removed []models.Topic
	for _, topic := range previous {
		if !kept[topic] {
			removed = append(removed, topic)
		}
	}
	return removed
}
