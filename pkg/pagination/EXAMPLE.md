# Using `pagination` in a repository and handler

This walks through wiring `pkg/pagination` into a keyset-paginated
`EnvelopeRepository.List`, end to end: repository interface, SQLite
implementation, service passthrough, and HTTP handler. It's illustrative —
the real `EnvelopeRepository.List` in this repo still returns every row; this
is the shape it would take if paginated.

Envelopes are ordered `created_at DESC`, but `created_at` alone isn't unique,
so the cursor key is the pair `(created_at, id)` — the same tiebreak the
query itself uses.

## 1. Repository interface

`internal/repositories/envelopes.go`:

```go
import "github.com/harvor-io/relay/pkg/pagination"

type EnvelopeRepository interface {
	Create(ctx context.Context, envelope *models.Envelope) error
	Get(ctx context.Context, id uuid.UUID) (*models.Envelope, error)

	// List returns a page of Envelopes, newest first. cursor resumes after
	// the page previously returned by List; the zero Cursor starts from the
	// beginning.
	List(ctx context.Context, cursor pagination.Cursor, limit int) (pagination.Page[models.Envelope], error)
}
```

## 2. SQLite implementation

`internal/repositories/sqlite/envelopes.go`:

```go
func (r *EnvelopeRepository) List(ctx context.Context, cursor pagination.Cursor, limit int) (pagination.Page[models.Envelope], error) {
	// Decode the incoming cursor into the (created_at, id) position to
	// resume from. A zero Cursor decodes nothing, leaving the zero values,
	// which the WHERE clause below treats as "no lower bound".
	var (
		afterCreatedAt string
		afterID        string
	)
	if cursor != "" {
		if err := pagination.Decode(cursor, &afterCreatedAt, &afterID); err != nil {
			return pagination.Page[models.Envelope]{}, fmt.Errorf("sqlite: decode envelope cursor: %w", err)
		}
	}

	// Fetch one row past limit; Slice below uses the extra row to decide
	// whether a next page exists, without a separate COUNT query.
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+envelopeColumns+` FROM envelopes
		 WHERE ? = '' OR (created_at, id) < (?, ?)
		 ORDER BY created_at DESC, id DESC
		 LIMIT ?`,
		afterCreatedAt, afterCreatedAt, afterID, limit+1)
	if err != nil {
		return pagination.Page[models.Envelope]{}, fmt.Errorf("sqlite: list envelopes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var envelopes []models.Envelope
	for rows.Next() {
		envelope, err := scanEnvelope(rows)
		if err != nil {
			return pagination.Page[models.Envelope]{}, err
		}
		envelopes = append(envelopes, *envelope)
	}
	if err := rows.Err(); err != nil {
		return pagination.Page[models.Envelope]{}, fmt.Errorf("sqlite: iterate envelopes: %w", err)
	}

	items, hasMore := pagination.Slice(envelopes, limit)
	page := pagination.Page[models.Envelope]{Items: items}
	if hasMore {
		last := items[len(items)-1]
		page.NextCursor = pagination.Encode(
			last.CreatedAt.Format(sqlitedb.TimeLayout),
			last.ID.String(),
		)
	}
	return page, nil
}
```

`pagination.Decode`/`Encode` don't know anything about SQL — they just
round-trip the `(created_at, id)` values through an opaque token. The
`(created_at, id) < (?, ?)` keyset comparison and the `LIMIT ?` are ordinary
SQLite, so a Postgres implementation would decode/encode the same two
values but write its own dialect-appropriate query.

## 3. Service

`internal/services/envelopes.go` (or wherever it lives) mostly just
passes the cursor and limit through, clamping the limit with
`pagination.Limit`:

```go
func (s *envelopeService) List(ctx context.Context, cursor pagination.Cursor, limit int) (pagination.Page[models.Envelope], error) {
	return s.envelopes.List(ctx, cursor, pagination.Limit(limit))
}
```

## 4. HTTP handler

`internal/handlers/envelopes.go`:

```go
// envelopePageResource is the JSON representation of a page of envelopes.
type envelopePageResource struct {
	Items      []envelopeResource `json:"items"`
	NextCursor string             `json:"next_cursor,omitempty"`
}

func (h *EnvelopeHandler) ListEnvelopes(w http.ResponseWriter, r *http.Request) {
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		limit = 0 // pagination.Limit substitutes DefaultLimit
	}
	cursor := pagination.Cursor(r.URL.Query().Get("cursor"))

	page, err := h.envelopes.List(r.Context(), cursor, limit)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			renderError(w, r, http.StatusBadRequest, "invalid cursor")
			return
		}
		renderError(w, r, http.StatusInternalServerError, "internal server error")
		return
	}

	render.JSON(w, r, envelopePageResource{
		Items:      newEnvelopeResourceList(page.Items),
		NextCursor: string(page.NextCursor),
	})
}
```

A client fetches the first page with `GET /envelopes?limit=50`, reads
`next_cursor` from the response, and requests
`GET /envelopes?limit=50&cursor=<next_cursor>` to continue — with no
knowledge of `created_at`/`id` or how the store orders rows.
