package funding

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/numofx/matching-backend/internal/instruments"
	oraclemodule "github.com/numofx/matching-backend/internal/oracles/btcvar30"
	"github.com/numofx/matching-backend/internal/orders"
	"github.com/numofx/matching-backend/internal/pricing"
)

type OracleReader interface {
	Latest() (oraclemodule.Payload, bool)
	IsStale(time.Time) bool
}

type Snapshot struct {
	Symbol           string    `json:"symbol"`
	Timestamp        time.Time `json:"timestamp"`
	MarkPrice        float64   `json:"mark_price"`
	OracleVariance30 float64   `json:"oracle_variance_30d"`
	Premium          float64   `json:"premium"`
	FundingRate      float64   `json:"funding_rate"`
	Paused           bool      `json:"paused"`
	Reason           string    `json:"reason,omitempty"`
}

type Service struct {
	orders      *orders.Repository
	oracle      OracleReader
	instrument  instruments.Metadata
	interval    time.Duration
	coefficient float64
	cap         float64
	logger      *slog.Logger

	mu     sync.RWMutex
	latest Snapshot
}

func NewService(ordersRepo *orders.Repository, oracle OracleReader, instrument instruments.Metadata, interval time.Duration, coefficient float64, cap float64, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}

	return &Service{
		orders:      ordersRepo,
		oracle:      oracle,
		instrument:  instrument,
		interval:    interval,
		coefficient: coefficient,
		cap:         math.Abs(cap),
		logger:      logger,
	}
}

func (s *Service) Run(ctx context.Context) error {
	if !s.instrument.Enabled {
		s.logger.Info("btcvar30 instrument disabled", "symbol", s.instrument.Symbol)
		return nil
	}

	if err := s.refresh(ctx); err != nil {
		s.logger.Warn("btcvar30 funding initial refresh failed", "error", err)
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := s.refresh(ctx); err != nil {
				s.logger.Error("btcvar30 funding refresh failed", "error", err)
			}
		}
	}
}

func (s *Service) Latest() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.latest
}

func (s *Service) refresh(ctx context.Context) error {
	now := time.Now().UTC()
	snapshot := Snapshot{
		Symbol:    s.instrument.Symbol,
		Timestamp: now,
	}

	payload, ok := s.oracle.Latest()
	if !ok {
		snapshot.Paused = true
		snapshot.Reason = "oracle_unavailable"
		s.store(snapshot)
		return fmt.Errorf("btcvar30 funding paused: oracle unavailable")
	}
	if s.oracle.IsStale(now) || payload.Stale {
		snapshot.Paused = true
		snapshot.Reason = "oracle_stale"
		snapshot.OracleVariance30 = payload.Variance30D
		s.store(snapshot)
		s.logger.Warn("btcvar30 funding paused", "reason", snapshot.Reason, "oracle_timestamp", payload.Timestamp)
		return nil
	}

	markPrice, err := s.currentMarkPrice(ctx)
	if err != nil {
		snapshot.Paused = true
		snapshot.Reason = "mark_unavailable"
		snapshot.OracleVariance30 = payload.Variance30D
		s.store(snapshot)
		return err
	}

	snapshot.MarkPrice = markPrice
	snapshot.OracleVariance30 = payload.Variance30D
	snapshot.Premium = snapshot.MarkPrice - snapshot.OracleVariance30
	snapshot.FundingRate = clamp(snapshot.Premium*s.coefficient, -s.cap, s.cap)

	s.store(snapshot)
	s.logger.Info("btcvar30 funding calculation",
		"symbol", snapshot.Symbol,
		"mark_price", snapshot.MarkPrice,
		"oracle_variance_30d", snapshot.OracleVariance30,
		"funding_rate", snapshot.FundingRate,
		"paused", snapshot.Paused,
	)
	return nil
}

func (s *Service) store(snapshot Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.latest = snapshot
}

func (s *Service) currentMarkPrice(ctx context.Context) (float64, error) {
	bid, ask, err := s.orders.BestBidAndAsk(ctx, s.instrument.AssetAddress, s.instrument.SubID)
	if err != nil {
		return 0, err
	}
	if bid == nil && ask == nil {
		return 0, fmt.Errorf("no book for %s", s.instrument.Symbol)
	}

	switch {
	case bid != nil && ask != nil:
		bidValue, err := s.priceTicksToFloat(bid.LimitPriceTicks)
		if err != nil {
			return 0, err
		}
		askValue, err := s.priceTicksToFloat(ask.LimitPriceTicks)
		if err != nil {
			return 0, err
		}
		if askValue < bidValue {
			return bidValue, nil
		}
		return (bidValue + askValue) / 2.0, nil
	case bid != nil:
		return s.priceTicksToFloat(bid.LimitPriceTicks)
	default:
		return s.priceTicksToFloat(ask.LimitPriceTicks)
	}
}

func CalculateFunding(markPrice float64, oracleVariance float64, coefficient float64, cap float64) float64 {
	return clamp((markPrice-oracleVariance)*coefficient, -math.Abs(cap), math.Abs(cap))
}

func clamp(value float64, floor float64, cap float64) float64 {
	return math.Max(floor, math.Min(cap, value))
}

func (s *Service) priceTicksToFloat(value string) (float64, error) {
	converter, err := pricing.NewConverter(s.instrument)
	if err != nil {
		return 0, err
	}
	display, err := converter.FormatTicks(value)
	if err != nil {
		return 0, err
	}
	parsed, err := strconv.ParseFloat(display, 64)
	if err != nil {
		return 0, fmt.Errorf("parse decimal %q: %w", display, err)
	}
	return parsed, nil
}
