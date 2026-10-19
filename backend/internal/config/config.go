// Package config loads runtime configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	HTTPAddr        string
	DatabaseURL     string
	ContentDir      string
	LogLevel        string
	ShutdownTimeout time.Duration
}

// Load reads configuration from the environment, applying defaults where sensible.
func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:        getenv("HTTP_ADDR", ":8080"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		ContentDir:      getenv("CONTENT_DIR", "/content"),
		LogLevel:        getenv("LOG_LEVEL", "info"),
		ShutdownTimeout: 10 * time.Second,
	}
	if v := os.Getenv("SHUTDOWN_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("invalid SHUTDOWN_TIMEOUT %q: %w", v, err)
		}
		cfg.ShutdownTimeout = d
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	return cfg, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
