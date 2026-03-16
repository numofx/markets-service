package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/numofx/matching-backend/internal/config"
	"github.com/numofx/matching-backend/internal/db"
	"github.com/numofx/matching-backend/internal/matching"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	engine := matching.NewEngine(cfg, pool)
	if err := engine.Run(ctx); err != nil {
		slog.Error("run matcher", "error", err)
		os.Exit(1)
	}
}
