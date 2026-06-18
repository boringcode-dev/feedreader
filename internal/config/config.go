package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	DBPath               string
	RefreshIntervalHours int
	ItemsPerSource       int
	RequestTimeoutSec    float64
	UserAgent            string
	Host                 string
	Port                 int
}

func Load() (Config, error) {
	cfg := Config{
		DBPath:               envOrDefault("FEEDREADER_DB_PATH", "./data/feedreader.db"),
		RefreshIntervalHours: envInt("FEEDREADER_REFRESH_INTERVAL_HOURS", 3),
		ItemsPerSource:       envInt("FEEDREADER_ITEMS_PER_SOURCE", 20),
		RequestTimeoutSec:    envFloat("FEEDREADER_REQUEST_TIMEOUT_SECONDS", 20),
		UserAgent:            envOrDefault("FEEDREADER_USER_AGENT", "feedreader/0.1"),
		Host:                 envOrDefault("FEEDREADER_HOST", "0.0.0.0"),
		Port:                 envInt("FEEDREADER_PORT", 8080),
	}
	if cfg.RefreshIntervalHours < 1 {
		cfg.RefreshIntervalHours = 1
	}
	if cfg.ItemsPerSource < 1 {
		cfg.ItemsPerSource = 1
	}
	if cfg.RequestTimeoutSec < 1 {
		cfg.RequestTimeoutSec = 1
	}
	abs, err := filepath.Abs(cfg.DBPath)
	if err != nil {
		return Config{}, fmt.Errorf("resolve db path: %w", err)
	}
	cfg.DBPath = abs
	return cfg, nil
}

func (c Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return fallback
}

func envFloat(key string, fallback float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return fallback
}
