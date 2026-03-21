package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/numofx/matching-backend/internal/api"
	"github.com/numofx/matching-backend/internal/config"
	"github.com/numofx/matching-backend/internal/db"
	"github.com/numofx/matching-backend/internal/funding"
	"github.com/numofx/matching-backend/internal/instruments"
	btcvar30instrument "github.com/numofx/matching-backend/internal/instruments/btcvar30"
	"github.com/numofx/matching-backend/internal/marketdata/deribit"
	oraclemodule "github.com/numofx/matching-backend/internal/oracles/btcvar30"
	"github.com/numofx/matching-backend/internal/orders"
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

	registry := instruments.DefaultRegistry(cfg)
	ordersRepo := orders.NewRepository(pool)
	if err := ordersRepo.BackfillLimitPriceTicks(ctx, registry); err != nil {
		slog.Error("backfill limit price ticks", "error", err)
		os.Exit(1)
	}

	deribitClient := deribit.NewClient(deribit.Config{
		BaseURL: cfg.DeribitBaseURL,
		WSURL:   cfg.DeribitWSURL,
		Logger:  slog.Default(),
	})
	oracleRepo := oraclemodule.NewRepository(pool)
	oracleSigner := oraclemodule.NewDeterministicSigner(cfg.BTCVar30OracleSigningKey)
	oracleService := oraclemodule.NewService(
		deribitClient,
		oracleRepo,
		oracleSigner,
		slog.Default(),
		cfg.BTCVar30OraclePollInterval,
		cfg.BTCVar30OracleStaleAfter,
	)

	if cfg.BTCVar30Enabled {
		go func() {
			if err := oracleService.Run(ctx); err != nil && err != context.Canceled {
				slog.Error("run btcvar30 oracle service", "error", err)
			}
		}()

		if instrument, ok := registry.BySymbol(btcvar30instrument.Symbol); ok {
			fundingService := funding.NewService(
				ordersRepo,
				oracleService,
				instrument,
				cfg.BTCVar30FundingInterval,
				cfg.BTCVar30FundingCoeff,
				cfg.BTCVar30FundingCap,
				slog.Default(),
			)
			go func() {
				if err := fundingService.Run(ctx); err != nil && err != context.Canceled {
					slog.Error("run btcvar30 funding service", "error", err)
				}
			}()
		}
	}

	server := api.NewServer(cfg, pool, registry, oracleService)
	if err := server.Run(); err != nil {
		slog.Error("run api server", "error", err)
		os.Exit(1)
	}
}
