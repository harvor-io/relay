// Package config contains the app's small runtime configuration.
package config

import "os"

const (
	defaultPort = "8080"

	// defaultLogLevel is the minimum slog level emitted: one of "debug",
	// "info", "warn", or "error".
	defaultLogLevel = "info"

	// defaultLogFormat selects the slog handler: "json" for machine-readable
	// logs (the default, suited to production log pipelines) or "text" for
	// human-readable key=value lines.
	defaultLogFormat = "json"

	// defaultDatabaseDriver is the pure-Go SQLite driver, so the service runs
	// with no external database and no cgo by default.
	defaultDatabaseDriver = "sqlite"

	// defaultDatabaseURL points at a local SQLite file in the working
	// directory. It is enough to run the service with no external
	// dependencies; production deployments override DATABASE_URL.
	defaultDatabaseURL = "file:relay.db"

	// defaultRabbitMQURI matches the guest account RabbitMQ's official
	// Docker image enables out of the box, so the service can talk to a
	// local broker with no configuration.
	defaultRabbitMQURI = "amqp://guest:guest@localhost:5672/"

	// defaultEventBusDriver is the only eventbus backend Relay currently
	// supports.
	defaultEventBusDriver = "rabbitmq"
)

// Config contains the settings required to run the app.
type Config struct {
	Port string

	// LogLevel is the minimum severity written to the structured log:
	// "debug", "info", "warn", or "error". Anything else falls back to "info".
	LogLevel string

	// LogFormat selects the log handler: "json" (default) or "text".
	LogFormat string

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

	// Secrets configures the secret store used to encrypt-at-rest values
	// such as source keys.
	Secrets SecretsConfig

	// EventBusDriver selects the eventbus backend. "rabbitmq" (the default,
	// see internal/bus/rabbitmq) is the only backend currently supported.
	EventBusDriver string

	// RabbitMQ configures the connection to the RabbitMQ broker that backs
	// the eventbus (see internal/bus/rabbitmq).
	RabbitMQ RabbitMQConfig
}

// SecretsConfig configures the secret store.
type SecretsConfig struct {
	// Driver selects the registered database/sql driver to open for the
	// secret store, matching Config.DatabaseDriver's semantics. Defaults to
	// "sqlite".
	Driver string

	// DatabaseURL is the secret store's data source. Defaults to the main
	// Config.DatabaseURL so secrets live alongside the rest of the data;
	// override with SECRETS_DATABASE_URL to keep a dedicated secrets store.
	DatabaseURL string

	// EncryptionKey encrypts secrets (e.g. source keys) at rest. It must be
	// base64-encoded 32 random bytes, suitable for use as an AES-256 key
	// (e.g. `openssl rand -base64 32`).
	EncryptionKey string
}

// RabbitMQConfig configures the connection to the RabbitMQ broker.
type RabbitMQConfig struct {
	// URI is the AMQP connection URI, e.g.
	// "amqp://user:pass@host:5672/vhost". Any credentials, vhost, or query
	// parameters (e.g. TLS options) the broker needs belong in this single
	// connection string rather than as separate fields.
	URI string
}

// New loads configuration from the environment.
func New() *Config {
	databaseURL := envOrDefault("DATABASE_URL", defaultDatabaseURL)

	return &Config{
		Port:              envOrDefault("PORT", defaultPort),
		LogLevel:          envOrDefault("LOG_LEVEL", defaultLogLevel),
		LogFormat:         envOrDefault("LOG_FORMAT", defaultLogFormat),
		DatabaseDriver:    envOrDefault("DATABASE_DRIVER", defaultDatabaseDriver),
		DatabaseURL:       databaseURL,
		DatabaseAuthToken: os.Getenv("DATABASE_AUTH_TOKEN"),
		Secrets: SecretsConfig{
			Driver:        envOrDefault("SECRETS_DATABASE_DRIVER", defaultDatabaseDriver),
			DatabaseURL:   envOrDefault("SECRETS_DATABASE_URL", databaseURL),
			EncryptionKey: os.Getenv("SECRETS_ENCRYPTION_KEY"),
		},
		EventBusDriver: envOrDefault("EVENTBUS_DRIVER", defaultEventBusDriver),
		RabbitMQ: RabbitMQConfig{
			URI: envOrDefault("RABBITMQ_URI", defaultRabbitMQURI),
		},
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
