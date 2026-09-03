// Package libsql manages connections to a hosted libSQL database such as Turso.
package libsql

import (
	"database/sql"
	"fmt"
	"net/url"

	_ "github.com/tursodatabase/libsql-client-go/libsql" // registers the "libsql" database/sql driver
)

// DriverName is the database/sql driver name registered by the libSQL client.
const DriverName = "libsql"

// Open opens the libSQL database at rawURL (for example
// "libsql://name-org.turso.io"). When authToken is non-empty it is added to
// the URL as the credential the server expects.
func Open(rawURL, authToken string) (*sql.DB, error) {
	dsn, err := dsn(rawURL, authToken)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open(DriverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("libsql: open: %w", err)
	}
	return db, nil
}

// dsn folds the auth token into the URL query string.
func dsn(rawURL, authToken string) (string, error) {
	if authToken == "" {
		return rawURL, nil
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("libsql: parse url: %w", err)
	}
	q := u.Query()
	q.Set("authToken", authToken)
	u.RawQuery = q.Encode()
	return u.String(), nil
}
