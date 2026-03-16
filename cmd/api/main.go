package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/numofx/matching-backend/internal/api"
	"github.com/numofx/matching-backend/internal/config"
	"github.com/numofx/matching-backend/internal/db"
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

	server := api.NewServer(cfg, pool)
	if err := server.Run(); err != nil {
		slog.Error("run api server", "error", err)
		os.Exit(1)
	}
}
