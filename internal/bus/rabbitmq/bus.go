// Package rabbitmq is the RabbitMQ-backed implementation of bus.Bus.
package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
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
