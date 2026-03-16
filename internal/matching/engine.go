package matching

import (
	"context"
	"log"
	"log/slog"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/numofx/matching-backend/internal/config"
	"github.com/numofx/matching-backend/internal/orders"
)

type Engine struct {
	cfg    config.Config
	pool   *pgxpool.Pool
	orders *orders.Repository
}

type MatchInstruction struct {
	Market        string       `json:"market"`
	AssetAddress  string       `json:"asset_address"`
	ModuleAddress string       `json:"module_address"`
	Taker         orders.Order `json:"taker"`
	Maker         orders.Order `json:"maker"`
	Price         string       `json:"price"`
	Amount        string       `json:"amount"`
	ExecutorURL   string       `json:"executor_url"`
}

func NewEngine(cfg config.Config, pool *pgxpool.Pool) *Engine {
	return &Engine{
		cfg:    cfg,
		pool:   pool,
		orders: orders.NewRepository(pool),
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
	bid, ask, err := e.orders.BestBidAndAsk(ctx, e.cfg.BTCPerpAssetAddress, "0")
	if err != nil {
		slog.Error("load best bid and ask", "error", err)
		return
	}
	if bid == nil || ask == nil {
		slog.Debug("matcher tick", "market", "BTC-PERP", "status", "book_not_crossed")
		return
	}
	if bid.IsExpired(time.Now()) || ask.IsExpired(time.Now()) {
		slog.Debug("matcher tick", "market", "BTC-PERP", "status", "expired_order_present")
		return
	}

	priceCrossed, err := crosses(bid.LimitPrice, ask.LimitPrice)
	if err != nil {
		slog.Error("compare prices", "error", err)
		return
	}
	if !priceCrossed {
		slog.Debug("matcher tick", "market", "BTC-PERP", "status", "book_not_crossed")
		return
	}

	amount, err := minDecimalString(remainingAmount(*bid), remainingAmount(*ask))
	if err != nil {
		slog.Error("compute fill amount", "error", err)
		return
	}

	instruction := MatchInstruction{
		Market:        "BTC-PERP",
		AssetAddress:  e.cfg.BTCPerpAssetAddress,
		ModuleAddress: e.cfg.TradeModuleAddress,
		Taker:         *ask,
		Maker:         *bid,
		Price:         bid.LimitPrice,
		Amount:        amount,
		ExecutorURL:   e.cfg.ExecutorURL,
	}

	slog.Info("match candidate ready",
		"market", instruction.Market,
		"maker_order_id", instruction.Maker.OrderID,
		"taker_order_id", instruction.Taker.OrderID,
		"price", instruction.Price,
		"amount", instruction.Amount,
	)
	log.Printf("TODO submit MatchInstruction to executor: %+v", instruction)
}

func remainingAmount(order orders.Order) string {
	remaining, err := subtractDecimalString(order.DesiredAmount, order.FilledAmount)
	if err != nil {
		return "0"
	}
	return remaining
}

func crosses(bidPrice string, askPrice string) (bool, error) {
	bid, ok := new(big.Int).SetString(bidPrice, 10)
	if !ok {
		return false, slogError("invalid bid price")
	}
	ask, ok := new(big.Int).SetString(askPrice, 10)
	if !ok {
		return false, slogError("invalid ask price")
	}
	return bid.Cmp(ask) >= 0, nil
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
