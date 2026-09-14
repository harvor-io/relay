// Package pagination provides database-agnostic building blocks for
// cursor-based pagination. It knows nothing about SQL or any particular
// store: callers encode a row's sort-key values into an opaque Cursor,
// decode a caller-supplied Cursor back into those values to resume a query,
// and use Slice to turn a lookahead-fetched batch of rows into a Page.
package pagination

import (
	"encoding/base64"
	"encoding/json"
	"errors"
)

// ErrInvalidCursor is returned by Decode when a Cursor is malformed, was
// encoded with a different number of key values, or was tampered with.
var ErrInvalidCursor = errors.New("pagination: invalid cursor")

// DefaultLimit is the page size Limit substitutes when a caller does not
// request one.
const DefaultLimit = 50

// MaxLimit is the largest page size Limit will return, regardless of what a
// caller requests.
const MaxLimit = 200

// Cursor is an opaque, URL-safe token identifying a position in an ordered
// list of rows. Treat it as opaque: produce it with Encode and read it with
// Decode, never by parsing or comparing it directly. The empty Cursor means
// "start from the beginning".
type Cursor string

// Encode returns a Cursor for the given ordered key values. keys should be
// the same fields, in the same order, used to sort the underlying query
// (for example a timestamp followed by a tiebreaking ID for a query ordered
// by "created_at DESC, id DESC"), and must be JSON-marshalable.
func Encode(keys ...any) Cursor {
	b, err := json.Marshal(keys)
	if err != nil {
		// keys are expected to be simple, JSON-marshalable sort-key values
		// (strings, numbers, times); a marshal failure means a caller
		// passed something else, which is a programmer error.
		panic("pagination: encode cursor: " + err.Error())
	}
	return Cursor(base64.RawURLEncoding.EncodeToString(b))
}

// Decode parses a Cursor previously produced by Encode into dest, which
// must be pointers to the same types, in the same order, as the values
// passed to Encode. It returns ErrInvalidCursor if cursor is malformed or
// does not decode into exactly len(dest) values.
func Decode(cursor Cursor, dest ...any) error {
	b, err := base64.RawURLEncoding.DecodeString(string(cursor))
	if err != nil {
		return ErrInvalidCursor
	}

	var raw []json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return ErrInvalidCursor
	}
	if len(raw) != len(dest) {
		return ErrInvalidCursor
	}
	for i, r := range raw {
		if err := json.Unmarshal(r, dest[i]); err != nil {
			return ErrInvalidCursor
		}
	}
	return nil
}

// Limit clamps a caller-requested page size to the (0, MaxLimit] range,
// substituting DefaultLimit when requested is not positive.
func Limit(requested int) int {
	switch {
	case requested <= 0:
		return DefaultLimit
	case requested > MaxLimit:
		return MaxLimit
	default:
		return requested
	}
}

// Page is a single page of cursor-paginated results.
type Page[T any] struct {
	// Items is this page's results, in the order the query produced them.
	Items []T

	// NextCursor fetches the next page when passed back to the same query.
	// It is "" once Items reaches the end of the list.
	NextCursor Cursor
}

// Slice splits rows into a page of at most limit items and reports whether
// more rows remain. Callers query for limit+1 rows ordered by their cursor
// fields, pass the result to Slice to trim the lookahead row, and — when
// hasMore is true — call Encode on the last returned item's sort-key values
// to populate Page.NextCursor.
func Slice[T any](rows []T, limit int) (items []T, hasMore bool) {
	if len(rows) > limit {
		return rows[:limit], true
	}
	return rows, false
}
