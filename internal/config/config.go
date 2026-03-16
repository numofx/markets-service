package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	AppEnv              string
	APIAddr             string
	DatabaseURL         string
	MatcherPollInterval time.Duration
	ChainID             string
	MatchingAddress     string
	TradeModuleAddress  string
	BTCPerpAssetAddress string
	ExecutorURL         string
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:              getenvDefault("APP_ENV", "dev"),
		APIAddr:             getenvDefault("API_ADDR", ":8080"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		ChainID:             os.Getenv("CHAIN_ID"),
		MatchingAddress:     os.Getenv("MATCHING_ADDRESS"),
		TradeModuleAddress:  os.Getenv("TRADE_MODULE_ADDRESS"),
		BTCPerpAssetAddress: os.Getenv("BTC_PERP_ASSET_ADDRESS"),
		ExecutorURL:         os.Getenv("EXECUTOR_URL"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	pollInterval, err := time.ParseDuration(getenvDefault("MATCHER_POLL_INTERVAL", "250ms"))
	if err != nil {
		return Config{}, fmt.Errorf("parse MATCHER_POLL_INTERVAL: %w", err)
	}
	cfg.MatcherPollInterval = pollInterval

	return cfg, nil
}

func getenvDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
