package api

import (
	"testing"

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
