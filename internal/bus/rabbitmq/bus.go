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
// events to. Consumers bind their own queues to it with whatever routing
// keys they care about.
const ingestExchange = "relay_ingest"

// retryQueueTTL is how long a message waits in a destination's retry queue
// before RabbitMQ automatically dead-letters it back to the delivery queue
// for another delivery attempt. It is a default meant to space out retries
// without callers having to manage the movement themselves.
const retryQueueTTL = 30 * time.Second

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

// deliveryQueueName, retryQueueName, and deadLetterQueueName return the
// queue names Apply and Remove use for destination's topology.
func deliveryQueueName(destination *models.Destination) string {
	return fmt.Sprintf("relay_destination_%s_delivery", destination.ID)
}

func retryQueueName(destination *models.Destination) string {
	return fmt.Sprintf("relay_destination_%s_retry", destination.ID)
}

func deadLetterQueueName(destination *models.Destination) string {
	return fmt.Sprintf("relay_destination_%s_dlq", destination.ID)
}

// bindingKey translates a models.Subscription into the routing key its
// binding on the ingest exchange uses. A subscription ending in ".*", which
// matches every topic under that prefix, becomes RabbitMQ's "#" wildcard
// rather than its "*": RabbitMQ's "*" matches exactly one dot-delimited
// word, but a topic can have more than one word after the prefix (e.g.
// "crm.*" must also match "crm.account.created", not just "crm.account").
// A subscription with no wildcard is used as an exact routing key.
func bindingKey(subscription models.Subscription) string {
	if prefix, ok := strings.CutSuffix(string(subscription), "*"); ok {
		return prefix + "#"
	}
	return string(subscription)
}

// Apply idempotently declares the queue topology destination needs and binds
// it to the ingest exchange, one binding per entry in destination.Subscriptions:
//
//   - a delivery queue, bound to the ingest exchange for each subscription,
//     that a delivery worker consumes from. A message it rejects (nack
//     without requeue) is dead-lettered to the retry queue.
//   - a retry queue that holds rejected deliveries for retryQueueTTL, after
//     which RabbitMQ automatically dead-letters them back to the delivery
//     queue for another attempt.
//   - a dead-letter queue: the terminal home for a message a delivery worker
//     gives up on after too many attempts.
//
// A destination with no subscriptions gets its queues declared but no
// bindings, so it receives nothing. Declaring a queue or binding with the
// same arguments as an existing one is a no-op in RabbitMQ, so Apply is safe
// to call repeatedly for the same destination — but note it only adds
// bindings for the subscriptions currently passed in; it does not remove
// bindings for subscriptions that were present on a previous call and are
// now gone (see bus.Bus.Apply).
func (b *Bus) Apply(ctx context.Context, destination *models.Destination) error {
	ch, err := b.conn.Channel()
	if err != nil {
		return fmt.Errorf("rabbitmq: open channel: %w", err)
	}
	defer func() { _ = ch.Close() }()

	deliveryQueue := deliveryQueueName(destination)
	retryQueue := retryQueueName(destination)
	deadLetterQueue := deadLetterQueueName(destination)

	if _, err := ch.QueueDeclare(deadLetterQueue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("rabbitmq: declare %s queue: %w", deadLetterQueue, err)
	}

	retryArgs := amqp.Table{
		"x-message-ttl":             int32(retryQueueTTL.Milliseconds()),
		"x-dead-letter-routing-key": deliveryQueue,
	}
	if _, err := ch.QueueDeclare(retryQueue, true, false, false, false, retryArgs); err != nil {
		return fmt.Errorf("rabbitmq: declare %s queue: %w", retryQueue, err)
	}

	deliveryArgs := amqp.Table{
		"x-dead-letter-routing-key": retryQueue,
	}
	if _, err := ch.QueueDeclare(deliveryQueue, true, false, false, false, deliveryArgs); err != nil {
		return fmt.Errorf("rabbitmq: declare %s queue: %w", deliveryQueue, err)
	}

	for _, subscription := range destination.Subscriptions {
		key := bindingKey(subscription)
		if err := ch.QueueBind(deliveryQueue, key, ingestExchange, false, nil); err != nil {
			return fmt.Errorf("rabbitmq: bind %s queue to %s exchange with key %q: %w", deliveryQueue, ingestExchange, key, err)
		}
	}

	return nil
}

// Remove idempotently deletes the delivery, retry, and dead-letter queues
// Apply provisioned for destination. A queue that does not exist (never
// created, or already removed) is treated as success rather than an error,
// so Remove is safe to call repeatedly for the same destination.
func (b *Bus) Remove(ctx context.Context, destination *models.Destination) error {
	queues := []string{
		deliveryQueueName(destination),
		retryQueueName(destination),
		deadLetterQueueName(destination),
	}
	for _, queue := range queues {
		if err := b.deleteQueue(queue); err != nil {
			return err
		}
	}
	return nil
}

// deleteQueue deletes the queue named name, treating it as already removed
// if it does not exist.
func (b *Bus) deleteQueue(name string) error {
	ch, err := b.conn.Channel()
	if err != nil {
		return fmt.Errorf("rabbitmq: open channel: %w", err)
	}
	defer func() { _ = ch.Close() }()

	_, err = ch.QueueDelete(name, false, false, false)
	if err == nil {
		return nil
	}

	var amqpErr *amqp.Error
	if errors.As(err, &amqpErr) && amqpErr.Code == amqp.NotFound {
		return nil
	}
	return fmt.Errorf("rabbitmq: delete %s queue: %w", name, err)
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
