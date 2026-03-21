package btcvar30

import (
	"context"
	"testing"
	"time"
)

func TestServiceLatestMarksStale(t *testing.T) {
	service := &Service{
		staleAfter:    time.Minute,
		lastSuccessAt: time.Now().Add(-2 * time.Minute),
		latest: Payload{
			Timestamp: time.Now().Add(-2 * time.Minute),
		},
		hasLatest: true,
	}

	payload, ok := service.Latest()
	if !ok {
		t.Fatal("expected latest payload")
	}
	if !payload.Stale {
		t.Fatal("expected payload to be stale")
	}
}

func TestServiceLatestUsesLastSuccessAtForFreshness(t *testing.T) {
	service := &Service{
		staleAfter:    time.Minute,
		lastSuccessAt: time.Now().Add(-10 * time.Second),
		latest: Payload{
			Timestamp: time.Now().Add(-2 * time.Minute),
		},
		hasLatest: true,
	}

	payload, ok := service.Latest()
	if !ok {
		t.Fatal("expected latest payload")
	}
	if payload.Stale {
		t.Fatal("expected payload to remain fresh when refresh succeeded recently")
	}
}

func TestDeterministicSigner(t *testing.T) {
	signer := NewDeterministicSigner("secret")
	signature, err := signer.Sign(Payload{
		Symbol:             Symbol,
		Source:             Source,
		Timestamp:          time.Unix(1_700_000_000, 0).UTC(),
		Vol30D:             60,
		Variance30D:        0.36,
		MethodologyVersion: MethodologyVersion,
	})
	if err != nil {
		t.Fatalf("Sign returned error: %v", err)
	}
	if signature == "" {
		t.Fatal("expected non-empty signature")
	}
}

func TestServiceHydratesLatestFromRepository(t *testing.T) {
	payload := Payload{
		Symbol:             Symbol,
		Source:             Source,
		Timestamp:          time.Unix(1_700_000_000, 0).UTC(),
		Vol30D:             60,
		Variance30D:        0.36,
		MethodologyVersion: MethodologyVersion,
	}

	service := &Service{
		repo: stubStorage{latest: payload},
	}

	if err := service.hydrateLatest(context.Background()); err != nil {
		t.Fatalf("hydrateLatest returned error: %v", err)
	}
	if !service.hasLatest {
		t.Fatal("expected service to hydrate latest payload")
	}
	if service.latest.Symbol != payload.Symbol {
		t.Fatalf("hydrated symbol = %s", service.latest.Symbol)
	}
}

type stubStorage struct {
	latest Payload
	err    error
}

func (s stubStorage) Insert(context.Context, Payload) error {
	return nil
}

func (s stubStorage) Latest(context.Context) (Payload, error) {
	return s.latest, s.err
}

func (s stubStorage) History(context.Context, int) ([]Payload, error) {
	return nil, s.err
}

func TestServiceHydrateLatestError(t *testing.T) {
	service := &Service{
		repo: stubStorage{err: context.DeadlineExceeded},
	}

	if err := service.hydrateLatest(context.Background()); err == nil {
		t.Fatal("expected hydrateLatest to fail")
	}
}
