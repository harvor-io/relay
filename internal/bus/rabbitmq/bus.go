// Package rabbitmq is the RabbitMQ-backed implementation of bus.Bus.
package rabbitmq

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/harvor-io/relay/internal/bus"
)

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
