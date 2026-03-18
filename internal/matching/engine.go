package matching

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/numofx/matching-backend/internal/config"
	"github.com/numofx/matching-backend/internal/orders"
)

type Engine struct {
	cfg      config.Config
	orders   *orders.Repository
	executor *ExecutorClient
}

func NewEngine(cfg config.Config, pool *pgxpool.Pool) *Engine {
	return &Engine{
		cfg:      cfg,
		orders:   orders.NewRepository(pool),
		executor: NewExecutorClient(cfg.ExecutorURL, cfg.ExecutorManagerData),
	}
}

func (e *Engine) Run(ctx context.Context) error {
	ticker := time.NewTicker(e.cfg.MatcherPollInterval)
	defer ticker.Stop()

	slog.Info("matcher started", "interval", e.cfg.MatcherPollInterval)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			e.tick(ctx)
		}
	}
}

func (e *Engine) tick(ctx context.Context) {
	now := time.Now()

	candidate, err := e.orders.AcquireMatchCandidate(ctx, e.cfg.BTCPerpAssetAddress, "0", now)
	if err != nil {
		slog.Error("acquire match candidate", "error", err)
		return
	}
	if candidate == nil {
		slog.Debug("matcher tick", "market", "BTC-PERP", "status", "book_not_crossed")
		return
	}

	release := true
	defer func() {
		if !release {
			return
		}
		if err := e.orders.ReleaseMatch(ctx, candidate.Taker.OrderID, candidate.Maker.OrderID); err != nil {
			slog.Error("release reserved orders", "taker_order_id", candidate.Taker.OrderID, "maker_order_id", candidate.Maker.OrderID, "error", err)
		}
	}()

	if candidate.Taker.IsExpired(now) || candidate.Maker.IsExpired(now) {
		slog.Debug("matcher tick", "market", "BTC-PERP", "status", "expired_order_present")
		return
	}

	priceCrossed, err := crosses(candidate.Taker, candidate.Maker)
	if err != nil {
		slog.Error("compare prices", "error", err)
		return
	}
	if !priceCrossed {
		slog.Debug("matcher tick", "market", "BTC-PERP", "status", "book_not_crossed")
		return
	}

	fillPrice := candidate.Maker.LimitPrice
	fillAmount, err := minDecimalString(remainingAmount(candidate.Taker), remainingAmount(candidate.Maker))
	if err != nil {
		slog.Error("compute fill amount", "error", err)
		return
	}
	if fillAmount == "0" {
		slog.Debug("matcher tick", "market", "BTC-PERP", "status", "zero_fill")
		return
	}

	executorResp, err := e.executor.SubmitMatch(ctx, *candidate, fillPrice, fillAmount)
	if err != nil {
		slog.Error("submit match", "taker_order_id", candidate.Taker.OrderID, "maker_order_id", candidate.Maker.OrderID, "error", err)
		_ = e.orders.MarkMatchFailed(ctx, []string{candidate.Taker.OrderID, candidate.Maker.OrderID}, err.Error())
		return
	}

	if err := e.orders.FinalizeMatch(ctx, candidate.Taker.OrderID, candidate.Maker.OrderID, fillAmount); err != nil {
		slog.Error("finalize match", "taker_order_id", candidate.Taker.OrderID, "maker_order_id", candidate.Maker.OrderID, "error", err)
		return
	}

	release = false

	slog.Info("match executed",
		"market", "BTC-PERP",
		"maker_order_id", candidate.Maker.OrderID,
		"taker_order_id", candidate.Taker.OrderID,
		"price", fillPrice,
		"amount", fillAmount,
		"tx_hash", executorResp.TxHash,
	)
}

func remainingAmount(order orders.Order) string {
	remaining, err := subtractDecimalString(order.DesiredAmount, order.FilledAmount)
	if err != nil {
		return "0"
	}
	return remaining
}

func crosses(taker orders.Order, maker orders.Order) (bool, error) {
	takerPrice, ok := new(big.Int).SetString(taker.LimitPrice, 10)
	if !ok {
		return false, slogError("invalid taker price")
	}
	makerPrice, ok := new(big.Int).SetString(maker.LimitPrice, 10)
	if !ok {
		return false, slogError("invalid maker price")
	}

	switch taker.Side {
	case orders.SideBuy:
		return takerPrice.Cmp(makerPrice) >= 0, nil
	case orders.SideSell:
		return takerPrice.Cmp(makerPrice) <= 0, nil
	default:
		return false, fmt.Errorf("unsupported taker side %q", taker.Side)
	}
}

func minDecimalString(a string, b string) (string, error) {
	left, ok := new(big.Int).SetString(a, 10)
	if !ok {
		return "", slogError("invalid decimal value")
	}
	right, ok := new(big.Int).SetString(b, 10)
	if !ok {
		return "", slogError("invalid decimal value")
	}
	if left.Cmp(right) <= 0 {
		return left.String(), nil
	}
	return right.String(), nil
}

func subtractDecimalString(a string, b string) (string, error) {
	left, ok := new(big.Int).SetString(a, 10)
	if !ok {
		return "", slogError("invalid decimal value")
	}
	right, ok := new(big.Int).SetString(b, 10)
	if !ok {
		return "", slogError("invalid decimal value")
	}
	if left.Cmp(right) < 0 {
		return "", slogError("filled amount exceeds desired amount")
	}
	return new(big.Int).Sub(left, right).String(), nil
}

func slogError(message string) error {
	return &matcherError{message: message}
}

type matcherError struct {
	message string
}

func (e *matcherError) Error() string {
	return e.message
}
