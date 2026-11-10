package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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
// repositories.SubscriptionRepository. A subscription's topics are stored as
// a JSON array in one column.
type SubscriptionRepository struct {
	db *sql.DB
}

var _ repositories.SubscriptionRepository = (*SubscriptionRepository)(nil)

// NewSubscriptionRepository returns a SubscriptionRepository backed by db.
func NewSubscriptionRepository(db *sql.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

const subscriptionColumns = `id, destination_id, source_id, source_slug, topics, created_at, updated_at`

// Create inserts a new subscription, assigning an ID and timestamps when
// unset.
func (r *SubscriptionRepository) Create(ctx context.Context, subscription *models.Subscription) error {
	if subscription.ID.IsNil() {
		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("sqlite: new subscription id: %w", err)
		}
		subscription.ID = id
	}
	now := time.Now().UTC()
	if subscription.CreatedAt.IsZero() {
		subscription.CreatedAt = now
	}
	subscription.UpdatedAt = now

	topics, err := json.Marshal(subscription.Topics)
	if err != nil {
		return fmt.Errorf("sqlite: encode subscription topics: %w", err)
	}

	_, err = r.db.ExecContext(ctx,
		`INSERT INTO subscriptions (`+subscriptionColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		subscription.ID.String(),
		subscription.DestinationID.String(),
		subscription.SourceID.String(),
		subscription.Source,
		string(topics),
		subscription.CreatedAt.Format(sqlitedb.TimeLayout),
		subscription.UpdatedAt.Format(sqlitedb.TimeLayout),
	)
	if err != nil {
		if sqlitedb.IsUniqueViolation(err) {
			return repositories.ErrConflict
		}
		return fmt.Errorf("sqlite: insert subscription: %w", err)
	}
	return nil
}

// Get returns a single subscription by ID.
func (r *SubscriptionRepository) Get(ctx context.Context, id uuid.UUID) (*models.Subscription, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+subscriptionColumns+` FROM subscriptions WHERE id = ?`, id.String())
	return scanSubscription(row)
}

// subscriptionsBySource pages a destination's subscriptions by source slug,
// then ID, so the order is total.
var subscriptionsBySource = sqlitedb.MustCursor(
	store.Asc("source_slug", func(s models.Subscription) string { return s.Source }),
	store.Asc("id", func(s models.Subscription) string { return s.ID.String() }),
)

// List returns the page of a destination's subscriptions query selects,
// ordered by source slug, then ID. Each cursor is bound to its destination.
func (r *SubscriptionRepository) List(ctx context.Context, query repositories.SubscriptionQuery) (cursor.Page[models.Subscription], error) {
	destinationID := query.DestinationID.String()
	return subscriptionsBySource.List(ctx, query.Page, func(ctx context.Context, q store.Query) ([]models.Subscription, error) {
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
// ordered by source slug.
func (r *SubscriptionRepository) ListByDestination(ctx context.Context, destinationID uuid.UUID) ([]models.Subscription, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+subscriptionColumns+` FROM subscriptions WHERE destination_id = ? ORDER BY source_slug`,
		destinationID.String(),
	)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list subscriptions: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanSubscriptions(rows)
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

// Update overwrites an existing subscription's topics.
func (r *SubscriptionRepository) Update(ctx context.Context, subscription *models.Subscription) error {
	topics, err := json.Marshal(subscription.Topics)
	if err != nil {
		return fmt.Errorf("sqlite: encode subscription topics: %w", err)
	}
	subscription.UpdatedAt = time.Now().UTC()
	res, err := r.db.ExecContext(ctx,
		`UPDATE subscriptions SET topics = ?, updated_at = ? WHERE id = ?`,
		string(topics),
		subscription.UpdatedAt.Format(sqlitedb.TimeLayout),
		subscription.ID.String(),
	)
	if err != nil {
		return fmt.Errorf("sqlite: update subscription: %w", err)
	}
	return sqlitedb.Affected(res)
}

// Delete removes a subscription by ID.
func (r *SubscriptionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM subscriptions WHERE id = ?`, id.String())
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

func scanSubscription(s sqlitedb.Scanner) (*models.Subscription, error) {
	var (
		subscription                         models.Subscription
		idStr, destinationIDStr, sourceIDStr string
		topics, createdAt, updatedAt         string
	)
	if err := s.Scan(&idStr, &destinationIDStr, &sourceIDStr, &subscription.Source, &topics, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repositories.ErrNotFound
		}
		return nil, fmt.Errorf("sqlite: scan subscription: %w", err)
	}

	var err error
	if subscription.ID, err = uuid.FromString(idStr); err != nil {
		return nil, fmt.Errorf("sqlite: parse subscription id: %w", err)
	}
	if subscription.DestinationID, err = uuid.FromString(destinationIDStr); err != nil {
		return nil, fmt.Errorf("sqlite: parse subscription destination_id: %w", err)
	}
	if subscription.SourceID, err = uuid.FromString(sourceIDStr); err != nil {
		return nil, fmt.Errorf("sqlite: parse subscription source_id: %w", err)
	}
	if err := json.Unmarshal([]byte(topics), &subscription.Topics); err != nil {
		return nil, fmt.Errorf("sqlite: parse subscription topics: %w", err)
	}
	if subscription.CreatedAt, err = sqlitedb.ParseTime(createdAt); err != nil {
		return nil, fmt.Errorf("sqlite: parse subscription created_at: %w", err)
	}
	if subscription.UpdatedAt, err = sqlitedb.ParseTime(updatedAt); err != nil {
		return nil, fmt.Errorf("sqlite: parse subscription updated_at: %w", err)
	}
	return &subscription, nil
}
