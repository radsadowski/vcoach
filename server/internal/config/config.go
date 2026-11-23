// Package config loads the server configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all server configuration. It is deliberately flat: every field
// maps to one environment variable documented in .env.example.
type Config struct {
	DatabaseURL  string
	Port         string
	LLMBaseURL   string
	LLMAPIKey    string
	LLMModel     string
	GarminAPIKey string
	SyncInterval time.Duration
}

// Load reads the configuration from the process environment.
func Load() (Config, error) {
	return load(os.Getenv)
}

func load(getenv func(string) string) (Config, error) {
	cfg := Config{
		DatabaseURL:  getenv("DATABASE_URL"),
		Port:         getenv("PORT"),
		LLMBaseURL:   getenv("LLM_BASE_URL"),
		LLMAPIKey:    getenv("LLM_API_KEY"),
		LLMModel:     getenv("LLM_MODEL"),
		GarminAPIKey: getenv("GARMIN_API_KEY"),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("config: DATABASE_URL is required")
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	syncIntervalMin := getenv("SYNC_INTERVAL_MIN")
	if syncIntervalMin == "" {
		syncIntervalMin = "60"
	}
	minutes, err := strconv.Atoi(syncIntervalMin)
	if err != nil || minutes <= 0 {
		return Config{}, fmt.Errorf("config: SYNC_INTERVAL_MIN must be a positive integer, got %q", syncIntervalMin)
	}
	cfg.SyncInterval = time.Duration(minutes) * time.Minute

	return cfg, nil
}
