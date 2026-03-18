package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
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
	ExecutorManagerData string
	ExpectedOrderOwner  string
	ExpectedOrderSigner string
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
		ExecutorManagerData: "0x",
		ExpectedOrderOwner:  os.Getenv("EXPECTED_ORDER_OWNER"),
		ExpectedOrderSigner: os.Getenv("EXPECTED_ORDER_SIGNER"),
	}

	managerData, err := loadExecutorManagerData()
	if err != nil {
		return Config{}, err
	}
	cfg.ExecutorManagerData = managerData

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

func loadExecutorManagerData() (string, error) {
	if path := strings.TrimSpace(os.Getenv("EXECUTOR_MANAGER_DATA_FILE")); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read EXECUTOR_MANAGER_DATA_FILE: %w", err)
		}
		return parseExecutorManagerData(data)
	}

	value := strings.TrimSpace(os.Getenv("EXECUTOR_MANAGER_DATA"))
	if value == "" {
		return "0x", nil
	}
	return value, nil
}

func parseExecutorManagerData(data []byte) (string, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return "0x", nil
	}

	if strings.HasPrefix(trimmed, "{") {
		var payload struct {
			ManagerData string `json:"manager_data"`
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			return "", fmt.Errorf("parse EXECUTOR_MANAGER_DATA_FILE json: %w", err)
		}
		if strings.TrimSpace(payload.ManagerData) == "" {
			return "", fmt.Errorf("EXECUTOR_MANAGER_DATA_FILE json missing manager_data")
		}
		return strings.TrimSpace(payload.ManagerData), nil
	}
	return trimmed, nil
}
