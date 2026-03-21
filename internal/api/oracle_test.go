package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/numofx/matching-backend/internal/config"
	"github.com/numofx/matching-backend/internal/instruments"
	oraclemodule "github.com/numofx/matching-backend/internal/oracles/btcvar30"
)

type stubOracle struct {
	latest  oraclemodule.Payload
	history []oraclemodule.Payload
	ok      bool
	err     error
}

func (s stubOracle) Latest() (oraclemodule.Payload, bool) {
	return s.latest, s.ok
}

func (s stubOracle) History(_ context.Context, _ int) ([]oraclemodule.Payload, error) {
	return s.history, s.err
}

func TestHandleBTCVar30Latest(t *testing.T) {
	server := NewServer(config.Config{}, nil, instruments.NewRegistry(nil), stubOracle{
		ok: true,
		latest: oraclemodule.Payload{
			Symbol:             oraclemodule.Symbol,
			Source:             oraclemodule.Source,
			Timestamp:          time.Unix(1_700_000_000, 0).UTC(),
			Vol30D:             61,
			Variance30D:        0.3721,
			MethodologyVersion: oraclemodule.MethodologyVersion,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/oracle/btcvar30/latest", nil)
	rec := httptest.NewRecorder()
	server.handleBTCVar30Latest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	var payload oraclePayloadResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if payload.Symbol != oraclemodule.Symbol {
		t.Fatalf("symbol = %s", payload.Symbol)
	}
	if payload.Market != "BTCVAR30-PERP" {
		t.Fatalf("market = %s", payload.Market)
	}
	if payload.PriceSemantics != "variance" {
		t.Fatalf("price semantics = %s", payload.PriceSemantics)
	}
	if payload.VolPercent == 0 {
		t.Fatal("expected vol_percent")
	}
}

func TestHandleBTCVar30History(t *testing.T) {
	server := NewServer(config.Config{}, nil, instruments.NewRegistry(nil), stubOracle{
		ok: true,
		history: []oraclemodule.Payload{
			{
				Symbol:             oraclemodule.Symbol,
				Source:             oraclemodule.Source,
				Timestamp:          time.Unix(1_700_000_000, 0).UTC(),
				Vol30D:             61,
				Variance30D:        0.3721,
				MethodologyVersion: oraclemodule.MethodologyVersion,
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/oracle/btcvar30/history?limit=5", nil)
	rec := httptest.NewRecorder()
	server.handleBTCVar30History(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	var response struct {
		Symbol         string                  `json:"symbol"`
		Market         string                  `json:"market"`
		PriceSemantics string                  `json:"price_semantics"`
		DisplayName    string                  `json:"display_name"`
		History        []oraclePayloadResponse `json:"history"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(response.History) != 1 {
		t.Fatalf("history length = %d", len(response.History))
	}
	if response.PriceSemantics != "variance" {
		t.Fatalf("price semantics = %s", response.PriceSemantics)
	}
	if response.History[0].VolPercent == 0 {
		t.Fatal("expected vol_percent in history response")
	}
}
