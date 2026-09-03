// Package config contains the app's small runtime configuration.
package config

import "os"

const defaultPort = "8080"

// Config contains the settings required to run the app.
type Config struct {
	Port string
}

// New loads configuration from the environment.
func New() *Config {
	return &Config{
		Port: envOrDefault("PORT", defaultPort),
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
