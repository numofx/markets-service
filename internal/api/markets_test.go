package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/numofx/matching-backend/internal/config"
	"github.com/numofx/matching-backend/internal/instruments"
)

func TestHandleMarketsIncludesDeliverableFutureMetadata(t *testing.T) {
	registry := instruments.DefaultRegistry(config.Config{
		BTCPerpAssetAddress:           "0xbtc",
		CNGNApr2026FutureAssetAddress: "0xf000000000000000000000000000000000000123",
		CNGNApr2026FutureSubID:        "1777507200",
	})

	server := NewServer(config.Config{}, nil, registry, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/markets", nil)
	rec := httptest.NewRecorder()
	server.handleMarkets(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	var markets []marketPresentation
	if err := json.Unmarshal(rec.Body.Bytes(), &markets); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	var found *marketPresentation
	for i := range markets {
		if markets[i].Market == instruments.CNGNApr2026Symbol {
			found = &markets[i]
			break
		}
	}
	if found == nil {
		t.Fatal("deliverable future missing from markets response")
	}

	if found.ContractType != "deliverable_fx_future" {
		t.Fatalf("contract type = %q", found.ContractType)
	}
	if found.SettlementType != "physical_delivery" {
		t.Fatalf("settlement type = %q", found.SettlementType)
	}
	if found.AssetAddress != "0xf000000000000000000000000000000000000123" {
		t.Fatalf("asset address = %q", found.AssetAddress)
	}
	if found.SubID != "1777507200" {
		t.Fatalf("sub id = %q", found.SubID)
	}
	if found.ExpiryTimestamp != 1777507200 || found.LastTradeTimestamp != 1777420800 {
		t.Fatalf("unexpected expiry window %+v", found)
	}
	if found.BaseAssetSymbol != "USDC" || found.QuoteAssetSymbol != "cNGN" {
		t.Fatalf("unexpected base/quote %q/%q", found.BaseAssetSymbol, found.QuoteAssetSymbol)
	}
	if found.TickSize != "1" {
		t.Fatalf("tick size = %q", found.TickSize)
	}
}
