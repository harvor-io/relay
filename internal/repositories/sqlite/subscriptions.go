package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/exalynt/turn/paginator/cursor"
	"github.com/exalynt/turn/store"
	turnsql "github.com/exalynt/turn/store/sql"
	"github.com/gofrs/uuid/v5"

	sqlitedb "github.com/harvor-io/relay/internal/database/sqlite"
	"github.com/harvor-io/relay/internal/models"
	"github.com/harvor-io/relay/internal/repositories"
)

// SubscriptionRepository is the SQLite-backed implementation of
// repositories.SubscriptionRepository, one row per destination and topic.
type SubscriptionRepository struct {
	db *sql.DB
}

var _ repositories.SubscriptionRepository = (*SubscriptionRepository)(nil)

// NewSubscriptionRepository returns a SubscriptionRepository backed by db.
func NewSubscriptionRepository(db *sql.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

const subscriptionColumns = `destination_id, topic, created_at`

// Create inserts a new subscription, assigning its CreatedAt when unset.
func (r *SubscriptionRepository) Create(ctx context.Context, subscription *models.Subscription) error {
	if subscription.CreatedAt.IsZero() {
		subscription.CreatedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO subscriptions (`+subscriptionColumns+`) VALUES (?, ?, ?)`,
		subscription.DestinationID.String(),
		subscription.Topic.String(),
		subscription.CreatedAt.Format(sqlitedb.TimeLayout),
	)
	if err != nil {
		if sqlitedb.IsUniqueViolation(err) {
			return repositories.ErrConflict
		}
		return fmt.Errorf("sqlite: insert subscription: %w", err)
	}
	return nil
}

// Get returns a destination's subscription to a topic.
func (r *SubscriptionRepository) Get(ctx context.Context, destinationID uuid.UUID, topic models.Topic) (*models.Subscription, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+subscriptionColumns+` FROM subscriptions WHERE destination_id = ? AND topic = ?`,
		destinationID.String(), topic.String())
	return scanSubscription(row)
}

// subscriptionsByTopic pages a destination's subscriptions by topic, which
// is unique within a destination, so the order is total.
var subscriptionsByTopic = sqlitedb.MustCursor(
	store.Asc("topic", func(s models.Subscription) string { return s.Topic.String() }),
)

// List returns the page of a destination's subscriptions query selects,
// ordered by topic. Each cursor is bound to its destination.
func (r *SubscriptionRepository) List(ctx context.Context, query repositories.SubscriptionQuery) (cursor.Page[models.Subscription], error) {
	destinationID := query.DestinationID.String()
	return subscriptionsByTopic.List(ctx, query.Page, func(ctx context.Context, q store.Query) ([]models.Subscription, error) {
		c := turnsql.Render(sqlitedb.Dialect, q, 2)
		rows, err := r.db.QueryContext(ctx,
			`SELECT `+subscriptionColumns+` FROM subscriptions WHERE destination_id = ?`+c.And()+c.Tail(),
			append([]any{destinationID}, c.Args...)...)
		if err != nil {
			return nil, fmt.Errorf("sqlite: list subscriptions: %w", err)
		}
		defer func() { _ = rows.Close() }()
		return scanSubscriptions(rows)
	}, "subscriptions", destinationID)
}

// ListByDestination returns every subscription belonging to destinationID,
// ordered by topic.
func (r *SubscriptionRepository) ListByDestination(ctx context.Context, destinationID uuid.UUID) ([]models.Subscription, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+subscriptionColumns+` FROM subscriptions WHERE destination_id = ? ORDER BY topic`,
		destinationID.String(),
	)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list subscriptions: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanSubscriptions(rows)
}

// Replace makes a destination's subscriptions exactly topics, in one
// transaction.
func (r *SubscriptionRepository) Replace(ctx context.Context, destinationID uuid.UUID, topics []models.Topic) (err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite: begin replace subscriptions: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	id := destinationID.String()
	keep := make([]any, 0, len(topics)+1)
	keep = append(keep, id)
	for _, topic := range topics {
		keep = append(keep, topic.String())
	}
	remove := `DELETE FROM subscriptions WHERE destination_id = ?`
	if len(topics) > 0 {
		remove += ` AND topic NOT IN (?` + strings.Repeat(`, ?`, len(topics)-1) + `)`
	}
	if _, err = tx.ExecContext(ctx, remove, keep...); err != nil {
		return fmt.Errorf("sqlite: remove subscriptions: %w", err)
	}

	now := time.Now().UTC().Format(sqlitedb.TimeLayout)
	for _, topic := range topics {
		if _, err = tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO subscriptions (`+subscriptionColumns+`) VALUES (?, ?, ?)`,
			id, topic.String(), now,
		); err != nil {
			return fmt.Errorf("sqlite: add subscription: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("sqlite: commit replace subscriptions: %w", err)
	}
	return nil
}

// Delete removes a destination's subscription to a topic.
func (r *SubscriptionRepository) Delete(ctx context.Context, destinationID uuid.UUID, topic models.Topic) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM subscriptions WHERE destination_id = ? AND topic = ?`,
		destinationID.String(), topic.String())
	if err != nil {
		return fmt.Errorf("sqlite: delete subscription: %w", err)
	}
	return sqlitedb.Affected(res)
}

// DeleteByDestination removes every subscription belonging to
// destinationID.
func (r *SubscriptionRepository) DeleteByDestination(ctx context.Context, destinationID uuid.UUID) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM subscriptions WHERE destination_id = ?`, destinationID.String()); err != nil {
		return fmt.Errorf("sqlite: delete destination subscriptions: %w", err)
	}
	return nil
}

func scanSubscriptions(rows *sql.Rows) ([]models.Subscription, error) {
	var subscriptions []models.Subscription
	for rows.Next() {
		subscription, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, *subscription)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: iterate subscriptions: %w", err)
	}
	return subscriptions, nil
}

func scanSubscription(s sqlitedb.Scanner) (*models.Subscription, error) {
	var (
		subscription                models.Subscription
		destinationID, topic, since string
	)
	if err := s.Scan(&destinationID, &topic, &since); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repositories.ErrNotFound
		}
		return nil, fmt.Errorf("sqlite: scan subscription: %w", err)
	}

	var err error
	if subscription.DestinationID, err = uuid.FromString(destinationID); err != nil {
		return nil, fmt.Errorf("sqlite: parse subscription destination_id: %w", err)
	}
	subscription.Topic = models.Topic(topic)
	if subscription.CreatedAt, err = sqlitedb.ParseTime(since); err != nil {
		return nil, fmt.Errorf("sqlite: parse subscription created_at: %w", err)
	}
	return &subscription, nil
}
