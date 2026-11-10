// Package rabbitmq is the RabbitMQ-backed implementation of bus.Bus.
package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
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

// encodeEnvelope returns envelope's message body.
func encodeEnvelope(envelope *models.Envelope) ([]byte, error) {
	return json.Marshal(envelopeMessage{
		ID:        envelope.ID.String(),
		SourceID:  envelope.SourceID.String(),
		Source:    envelope.Source,
		Type:      envelope.Type,
		Topic:     envelope.Topic().String(),
		Data:      envelope.Data,
		CreatedAt: envelope.CreatedAt,
	})
}

// decodeEnvelope returns the envelope a message body carries. The topic in
// the body isn't read back: an envelope derives it from Source and Type.
func decodeEnvelope(body []byte) (*models.Envelope, error) {
	var msg envelopeMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return nil, err
	}
	id, err := uuid.FromString(msg.ID)
	if err != nil {
		return nil, fmt.Errorf("parse id: %w", err)
	}
	sourceID, err := uuid.FromString(msg.SourceID)
	if err != nil {
		return nil, fmt.Errorf("parse source_id: %w", err)
	}
	return &models.Envelope{
		ID:        id,
		SourceID:  sourceID,
		Source:    msg.Source,
		Type:      msg.Type,
		Data:      msg.Data,
		CreatedAt: msg.CreatedAt,
	}, nil
}

// Publish sends envelope to the relay_ingest exchange, routed by its Topic.
func (b *Bus) Publish(ctx context.Context, envelope *models.Envelope) error {
	ch, err := b.conn.Channel()
	if err != nil {
		return fmt.Errorf("rabbitmq: open channel: %w", err)
	}
	defer func() { _ = ch.Close() }()

	body, err := encodeEnvelope(envelope)
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
// for the topic exchange. A "*" ending the pattern, or making up all of it,
// becomes "#", which matches any number of parts. Event types can't contain
// "*" or "#", so an exact topic binds as itself and never acts as a
// wildcard.
func bindingKey(topic models.Topic) string {
	if topic == models.Wildcard {
		return "#"
	}
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

// Subscribe declares the destination's queue, then binds it once for each
// subscription's topic, all on one channel. Binding a key that is already
// bound is a no-op.
func (b *Bus) Subscribe(_ context.Context, destination *models.Destination, subscriptions []models.Subscription) error {
	if len(subscriptions) == 0 {
		return nil
	}
	return b.withChannel(func(ch *amqp.Channel) error {
		if err := declareQueue(ch, destination.ID); err != nil {
			return err
		}
		name := destinationQueue(destination.ID)
		for i := range subscriptions {
			if err := ch.QueueBind(name, bindingKey(subscriptions[i].Topic), ingestExchange, false, nil); err != nil {
				return fmt.Errorf("rabbitmq: bind queue %s: %w", name, err)
			}
		}
		return nil
	})
}

// Unsubscribe removes the destination queue's binding for each
// subscription's topic, all on one channel. Unbinding a key that isn't
// bound is a no-op. A queue that doesn't exist has no bindings left, so it
// counts as success, and the driver stops there: the broker closes the
// channel when it reports a missing queue.
func (b *Bus) Unsubscribe(_ context.Context, destination *models.Destination, subscriptions []models.Subscription) error {
	if len(subscriptions) == 0 {
		return nil
	}
	return b.withChannel(func(ch *amqp.Channel) error {
		name := destinationQueue(destination.ID)
		for i := range subscriptions {
			if err := ch.QueueUnbind(name, bindingKey(subscriptions[i].Topic), ingestExchange, nil); err != nil {
				if isNotFound(err) {
					return nil
				}
				return fmt.Errorf("rabbitmq: unbind queue %s: %w", name, err)
			}
		}
		return nil
	})
}

// Consume takes envelopes off the destination's queue on a channel of its
// own, held until Consume returns. With a prefetch of one, the broker sends
// the next message only once the last is acked or nacked, so envelopes
// reach handle one at a time and in order, and a nacked one is redelivered
// ahead of those behind it. Closing the channel hands back any message
// still unacked.
//
// A message whose body won't decode can never be handled, so it is
// rejected without requeueing, which drops it, and Consume returns an
// error saying so.
func (b *Bus) Consume(ctx context.Context, destination *models.Destination, handle bus.EnvelopeHandler) error {
	return b.withChannel(func(ch *amqp.Channel) error {
		if err := declareQueue(ch, destination.ID); err != nil {
			return err
		}
		if err := ch.Qos(1, 0, false); err != nil {
			return fmt.Errorf("rabbitmq: set prefetch: %w", err)
		}

		name := destinationQueue(destination.ID)
		deliveries, err := ch.ConsumeWithContext(
			ctx,
			name,
			"",    // consumer tag, generated by the library
			false, // auto-ack
			false, // exclusive
			false, // no-local
			false, // no-wait
			nil,   // arguments
		)
		if err != nil {
			return fmt.Errorf("rabbitmq: consume queue %s: %w", name, err)
		}

		for {
			select {
			case <-ctx.Done():
				return nil
			case d, ok := <-deliveries:
				if !ok {
					// ConsumeWithContext closes deliveries when ctx is done,
					// too, so this can race the case above.
					if ctx.Err() != nil {
						return nil
					}
					return fmt.Errorf("rabbitmq: queue %s stopped delivering", name)
				}
				if err := deliver(ctx, d, handle); err != nil {
					return fmt.Errorf("rabbitmq: queue %s: %w", name, err)
				}
			}
		}
	})
}

// deliver hands d's envelope to handle, then acks d if handle succeeds or
// nacks it for redelivery if it fails. A body that won't decode is
// rejected for good and reported as an error.
func deliver(ctx context.Context, d amqp.Delivery, handle bus.EnvelopeHandler) error {
	envelope, err := decodeEnvelope(d.Body)
	if err != nil {
		if rejectErr := d.Reject(false); rejectErr != nil {
			return fmt.Errorf("reject undecodable message: %w", rejectErr)
		}
		return fmt.Errorf("dropped undecodable message: %w", err)
	}

	if err := handle(ctx, envelope); err != nil {
		if nackErr := d.Nack(false, true); nackErr != nil {
			return fmt.Errorf("nack envelope %s: %w", envelope.ID, nackErr)
		}
		return nil
	}
	if err := d.Ack(false); err != nil {
		return fmt.Errorf("ack envelope %s: %w", envelope.ID, err)
	}
	return nil
}
