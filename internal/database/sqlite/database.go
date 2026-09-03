// Package sqlite manages connections to the embedded, pure-Go SQLite engine and
// provides the storage-encoding helpers (timestamp layout, NULL handling,
// bool/int mapping) shared by SQLite-backed repositories.
package sqlite

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite" // registers the "sqlite" database/sql driver
)

// DriverName is the database/sql driver name registered by modernc.org/sqlite.
const DriverName = "sqlite"

// pragmas are applied to every connection. WAL journaling lets readers run
// while a write is in progress, the busy timeout makes writers wait for a lock
// instead of failing, and foreign key enforcement is off by default in SQLite.
var pragmas = []string{
	"busy_timeout(5000)",
	"journal_mode(WAL)",
	"foreign_keys(ON)",
}

// Open opens the SQLite database identified by url (for example
// "file:relay.db" or "file:/var/lib/relay/relay.db") with relay's standard
// pragmas applied.
func Open(url string) (*sql.DB, error) {
	db, err := sql.Open(DriverName, dsn(url))
	if err != nil {
		return nil, fmt.Errorf("sqlite: open: %w", err)
	}

	// SQLite allows only one writer at a time; capping the pool at a single
	// connection turns lock contention into ordinary queuing.
	db.SetMaxOpenConns(1)

	return db, nil
}

// dsn appends the pragma query parameters that modernc.org/sqlite understands.
func dsn(url string) string {
	sep := "?"
	if strings.Contains(url, "?") {
		sep = "&"
	}
	var b strings.Builder
	b.WriteString(url)
	for i, p := range pragmas {
		if i == 0 {
			b.WriteString(sep)
		} else {
			b.WriteString("&")
		}
		b.WriteString("_pragma=")
		b.WriteString(p)
	}
	return b.String()
}
