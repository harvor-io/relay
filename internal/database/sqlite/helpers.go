package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/harvor-io/relay/internal/repositories"
)

// TimeLayout is how timestamps are stored in TEXT columns: RFC 3339 with
// nanosecond precision. Values are always written in UTC so string ordering
// matches chronological ordering.
const TimeLayout = time.RFC3339Nano

// Scanner is implemented by both *sql.Row and *sql.Rows, letting a single scan
// helper serve Get and List.
type Scanner interface {
	Scan(dest ...any) error
}

// NullString maps an optional string to the value a driver expects: the string
// itself, or nil for a SQL NULL.
func NullString(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

// StringPtr converts a scanned nullable column back into an optional string.
func StringPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	s := ns.String
	return &s
}

// BoolToInt stores a bool as SQLite's 0/1 integer.
func BoolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ParseTime reads a timestamp written in TimeLayout.
func ParseTime(s string) (time.Time, error) {
	return time.Parse(TimeLayout, s)
}

// Affected turns a write result into repositories.ErrNotFound when it changed
// no rows.
func Affected(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("sqlite: rows affected: %w", err)
	}
	if n == 0 {
		return repositories.ErrNotFound
	}
	return nil
}
