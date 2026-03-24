package api

import (
	"testing"
	"time"

	"github.com/numofx/matching-backend/internal/instruments"
	"github.com/numofx/matching-backend/internal/orders"
)

func TestPresentOrderForBTCVar30AddsVarianceAndVolDisplay(t *testing.T) {
	order := orders.Order{
		LimitPrice:      "0.2728",
		LimitPriceTicks: "2728",
	}
	instrument := instruments.Metadata{
		Symbol:           "BTCVAR30-PERP",
		TickSize:         "0.0001",
		QuotePrecision:   6,
		PricingModel:     instruments.PricingModelVariance,
		PriceSemantics:   instruments.PricingModelVariance,
		DisplayPriceKind: instruments.DisplayPriceVolPercent,
	}

	presented := presentOrder(order, instrument)
	if presented.VariancePrice != 0.2728 {
		t.Fatalf("variance price = %v", presented.VariancePrice)
	}
	if presented.VolPercent != 52.23 {
		t.Fatalf("vol percent = %v", presented.VolPercent)
	}
	if presented.PriceSemantics != instruments.PricingModelVariance {
		t.Fatalf("price semantics = %s", presented.PriceSemantics)
	}
}

func TestPresentOrderForLinearInstrumentLeavesVarianceFieldsEmpty(t *testing.T) {
	order := orders.Order{
		LimitPrice:      "100",
		LimitPriceTicks: "100",
	}
	instrument := instruments.Metadata{
		Symbol:           "BTCUSDC-CVXPERP",
		TickSize:         "1",
		QuotePrecision:   8,
		PricingModel:     instruments.PricingModelLinear,
		PriceSemantics:   instruments.PricingModelLinear,
		DisplayPriceKind: instruments.DisplayPriceDirect,
	}

	presented := presentOrder(order, instrument)
	if presented.VariancePrice != 0 || presented.VolPercent != 0 {
		t.Fatalf("unexpected variance presentation %+v", presented)
	}
}

func TestPresentTradesIncludesDeliverableMetadata(t *testing.T) {
	items := []orders.TradeFill{{
		TradeID:       1,
		AssetAddress:  "0xf000000000000000000000000000000000000123",
		SubID:         "1777507200",
		Price:         "1605.25",
		Size:          "100000000000000000",
		AggressorSide: orders.SideBuy,
		TakerOrderID:  "taker-1",
		MakerOrderID:  "maker-1",
		CreatedAt:     time.Unix(1777507200, 0).UTC(),
	}}
	instrument := instruments.Metadata{
		Symbol:         instruments.CNGNApr2026Symbol,
		ContractType:   "deliverable_fx_future",
		SettlementType: "physical_delivery",
	}

	presented := presentTrades(items, instrument)
	if len(presented) != 1 {
		t.Fatalf("len = %d", len(presented))
	}
	if presented[0].Market != instruments.CNGNApr2026Symbol {
		t.Fatalf("market = %q", presented[0].Market)
	}
	if presented[0].ContractType != "deliverable_fx_future" {
		t.Fatalf("contract type = %q", presented[0].ContractType)
	}
	if presented[0].SettlementType != "physical_delivery" {
		t.Fatalf("settlement type = %q", presented[0].SettlementType)
	}
}
