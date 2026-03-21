package btcvar30

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/numofx/matching-backend/internal/marketdata/deribit"
)

type Service struct {
	client     *deribit.Client
	repo       storage
	signer     Signer
	logger     *slog.Logger
	pollEvery  time.Duration
	staleAfter time.Duration

	mu              sync.RWMutex
	latest          Payload
	hasLatest       bool
	lastSuccessAt   time.Time
	lastFailureAt   time.Time
	lastUpdateError error
}

type storage interface {
	Insert(context.Context, Payload) error
	Latest(context.Context) (Payload, error)
	History(context.Context, int) ([]Payload, error)
}

func NewService(client *deribit.Client, repo storage, signer Signer, logger *slog.Logger, pollEvery time.Duration, staleAfter time.Duration) *Service {
	if logger == nil {
		logger = slog.Default()
	}

	return &Service{
		client:     client,
		repo:       repo,
		signer:     signer,
		logger:     logger,
		pollEvery:  pollEvery,
		staleAfter: staleAfter,
	}
}

func (s *Service) Run(ctx context.Context) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("btcvar30 oracle service is not configured")
	}

	if err := s.hydrateLatest(ctx); err != nil {
		s.logger.Warn("btcvar30 oracle hydrate failed", "error", err)
	}

	if err := s.Refresh(ctx); err != nil {
		s.logger.Warn("btcvar30 oracle initial refresh failed", "error", err)
	}

	ticker := time.NewTicker(s.pollEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := s.Refresh(ctx); err != nil {
				s.logger.Error("btcvar30 oracle update failure", "error", err)
			}
			if payload, ok := s.Latest(); ok && payload.Stale {
				s.logger.Warn("btcvar30 oracle stale", "timestamp", payload.Timestamp, "stale_after_ms", s.staleAfter.Milliseconds())
			}
		}
	}
}

func (s *Service) Refresh(ctx context.Context) error {
	now := time.Now().UTC()
	start := now.Add(-2 * time.Hour)

	resp, err := s.client.GetVolatilityIndexData(ctx, deribit.VolatilityIndexDataParams{
		Currency:       "BTC",
		StartTimestamp: start.UnixMilli(),
		EndTimestamp:   now.UnixMilli(),
		Resolution:     "60",
	})
	if err != nil {
		s.mu.Lock()
		s.lastFailureAt = now
		s.lastUpdateError = err
		s.mu.Unlock()
		return err
	}
	if len(resp.Data) == 0 {
		return errors.New("deribit volatility index returned no candles")
	}

	last := resp.Data[len(resp.Data)-1]
	derived := Derive(last.Close)
	payload := Payload{
		Symbol:             Symbol,
		Source:             Source,
		Timestamp:          time.UnixMilli(last.Timestamp).UTC(),
		Vol30D:             derived.Vol30D,
		Variance30D:        derived.Variance30D,
		MethodologyVersion: MethodologyVersion,
	}

	signature, err := s.signer.Sign(payload)
	if err != nil {
		return err
	}
	payload.Signature = signature

	if s.repo != nil {
		if err := s.repo.Insert(ctx, payload); err != nil {
			return fmt.Errorf("persist btcvar30 oracle payload: %w", err)
		}
	}

	s.mu.Lock()
	s.latest = payload
	s.hasLatest = true
	s.lastSuccessAt = now
	s.lastUpdateError = nil
	s.mu.Unlock()

	s.logger.Info("btcvar30 oracle update success",
		"timestamp", payload.Timestamp,
		"vol_30d", payload.Vol30D,
		"variance_30d", payload.Variance30D,
		"latency_ms", time.Since(now).Milliseconds(),
	)
	return nil
}

func (s *Service) hydrateLatest(ctx context.Context) error {
	if s.repo == nil {
		return nil
	}

	payload, err := s.repo.Latest(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.hasLatest {
		return nil
	}
	s.latest = payload
	s.hasLatest = true
	s.lastSuccessAt = payload.Timestamp
	return nil
}

func (s *Service) Latest() (Payload, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.hasLatest {
		return Payload{}, false
	}

	payload := s.latest
	reference := s.lastSuccessAt
	if reference.IsZero() {
		reference = payload.Timestamp
	}
	payload.Stale = time.Since(reference) > s.staleAfter
	return payload, true
}

func (s *Service) History(ctx context.Context, limit int) ([]Payload, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("btcvar30 history persistence not configured")
	}

	items, err := s.repo.History(ctx, limit)
	if err != nil {
		return nil, err
	}

	for i := range items {
		items[i].Stale = time.Since(items[i].Timestamp) > s.staleAfter
	}
	return items, nil
}

func (s *Service) IsStale(at time.Time) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.hasLatest {
		return true
	}

	reference := s.lastSuccessAt
	if reference.IsZero() {
		reference = s.latest.Timestamp
	}
	return at.Sub(reference) > s.staleAfter
}
