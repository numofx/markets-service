package api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/numofx/matching-backend/internal/instruments"
	"github.com/numofx/matching-backend/internal/orders"
)

func TestPresentOrderBTCVar30GoldenFields(t *testing.T) {
	instrument := instruments.Metadata{
		Symbol:           "BTCVAR30-PERP",
		TickSize:         "0.0001",
		PricingModel:     instruments.PricingModelVariance,
		PriceSemantics:   instruments.PricingModelVariance,
		DisplaySemantics: instruments.DisplayPriceVolPercent,
		DisplayName:      "BTC 30D Implied Volatility Perpetual",
		DisplayLabel:     "BTC 30D Vol Perp",
		QuotePrecision:   6,
	}
	order := orders.Order{
		OrderID:         "btcvar30-1",
		LimitPrice:      "0.2728",
		LimitPriceTicks: "2728",
	}

	payload := orderResponse{Order: presentOrder(order, instrument)}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}

	text := string(raw)
	required := []string{
		`"market":"BTCVAR30-PERP"`,
		`"limit_price":"0.2728"`,
		`"variance_price":0.2728`,
		`"vol_percent":52.23`,
		`"price_semantics":"variance"`,
		`"display_semantics":"vol_percent"`,
		`"display_name":"BTC 30D Implied Volatility Perpetual"`,
		`"tick_size":"0.0001"`,
	}
	for _, item := range required {
		if !strings.Contains(text, item) {
			t.Fatalf("response missing %s: %s", item, text)
		}
	}
}

func TestPresentMarketBTCVar30GoldenFields(t *testing.T) {
	market := presentMarket(instruments.Metadata{
		Symbol:           "BTCVAR30-PERP",
		AssetAddress:     "0xvar",
		SubID:            "0",
		TickSize:         "0.0001",
		PricingModel:     instruments.PricingModelVariance,
		PriceSemantics:   instruments.PricingModelVariance,
		DisplaySemantics: instruments.DisplayPriceVolPercent,
		DisplayName:      "BTC 30D Implied Volatility Perpetual",
		DisplayLabel:     "BTC 30D Vol Perp",
		SettlementNote:   "Internally priced and settled in 30D implied variance",
	})

	raw, err := json.Marshal(bookResponse{MarketPresentation: market})
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}

	text := string(raw)
	required := []string{
		`"market":"BTCVAR30-PERP"`,
		`"price_semantics":"variance"`,
		`"display_semantics":"vol_percent"`,
		`"display_name":"BTC 30D Implied Volatility Perpetual"`,
		`"tick_size":"0.0001"`,
	}
	for _, item := range required {
		if !strings.Contains(text, item) {
			t.Fatalf("response missing %s: %s", item, text)
		}
	}
}
