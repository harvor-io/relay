// Package config contains the app's small runtime configuration.
package config

import "os"

const (
	defaultPort = "8080"

	// defaultDatabaseDriver is the pure-Go SQLite driver, so the service runs
	// with no external database and no cgo by default.
	defaultDatabaseDriver = "sqlite"

	// defaultDatabaseURL points at a local SQLite file in the working
	// directory. It is enough to run the service with no external
	// dependencies; production deployments override DATABASE_URL.
	defaultDatabaseURL = "file:relay.db"
)

// Config contains the settings required to run the app.
type Config struct {
	Port string

	// DatabaseDriver selects the registered database/sql driver to open.
	// "sqlite" (the default) is the embedded pure-Go SQLite engine; "libsql"
	// talks to a hosted libSQL database such as Turso.
	DatabaseDriver string

	// DatabaseURL is the SQLite data source. It accepts either a local file
	// DSN ("file:relay.db", "file:/var/lib/relay/relay.db") or the URL of a
	// hosted libSQL database such as Turso ("libsql://name-org.turso.io").
	DatabaseURL string

	// DatabaseAuthToken authenticates against a hosted libSQL database. It is
	// empty for local file databases.
	DatabaseAuthToken string
}

// New loads configuration from the environment.
func New() *Config {
	return &Config{
		Port:              envOrDefault("PORT", defaultPort),
		DatabaseDriver:    envOrDefault("DATABASE_DRIVER", defaultDatabaseDriver),
		DatabaseURL:       envOrDefault("DATABASE_URL", defaultDatabaseURL),
		DatabaseAuthToken: os.Getenv("DATABASE_AUTH_TOKEN"),
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
