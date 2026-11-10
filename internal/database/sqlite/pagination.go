package sqlite

import (
	"github.com/exalynt/turn/codec/plain"
	"github.com/exalynt/turn/store"
	turnsql "github.com/exalynt/turn/store/sql"
)

// Dialect is how turn renders the pagination parts of a SQLite query: ?
// placeholders, row value comparison for the keyset (SQLite has it since
// 3.15), and LIMIT … OFFSET.
var Dialect turnsql.Dialect = dialect{}

type dialect struct{}

func (dialect) Placeholder(int) string { return "?" }
func (dialect) RowValues() bool        { return true }
func (dialect) Limit(limit, offset string) string {
	if offset == "" {
		return "LIMIT " + limit
	}
	return "LIMIT " + limit + " OFFSET " + offset
}

// MustCursor builds a listing's cursor paginator with turn's default page
// sizes and unsigned codec, ordered by keys. It panics on invalid keys, so
// call it once per listing, at package initialization.
func MustCursor[T any](keys ...store.Key[T]) *store.Cursor[T] {
	c, err := store.NewCursor(store.CursorOptions{Codec: plain.Codec[store.Position]{}}, keys...)
	if err != nil {
		panic(err)
	}
	return c
}
