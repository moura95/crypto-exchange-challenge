package config

import (
	"os"
)

type Config struct {
	HTTPServerAddress string
}

func Load() (*Config, error) {
	cfg := &Config{
		HTTPServerAddress: getEnv("HTTP_SERVER_ADDRESS", "0.0.0.0:8080"),
	}
	if cfg.HTTPServerAddress == "" {
		cfg.HTTPServerAddress = "0.0.0.0:8080"
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
