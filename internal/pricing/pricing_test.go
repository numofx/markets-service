package pricing

import (
	"math"
	"testing"

	"github.com/numofx/matching-backend/internal/instruments"
)

func TestParseAndFormatBTCVar30Price(t *testing.T) {
	converter, err := NewConverter(instruments.Metadata{
		Symbol:         "BTCVAR30-PERP",
		TickSize:       "0.0001",
		QuotePrecision: 6,
	})
	if err != nil {
		t.Fatalf("NewConverter returned error: %v", err)
	}

	ticks, normalized, err := converter.Parse("0.2724")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if ticks != "2724" {
		t.Fatalf("ticks = %s", ticks)
	}
	if normalized != "0.2724" {
		t.Fatalf("normalized = %s", normalized)
	}

	display, err := converter.FormatTicks("2725")
	if err != nil {
		t.Fatalf("FormatTicks returned error: %v", err)
	}
	if display != "0.2725" {
		t.Fatalf("display = %s", display)
	}
}

func TestBTCVar30VarianceDisplayFromTicks(t *testing.T) {
	instrument := instruments.Metadata{
		Symbol:         "BTCVAR30-PERP",
		TickSize:       "0.0001",
		QuotePrecision: 6,
	}

	display, err := VarianceDisplayFromTicks(instrument, "2728")
	if err != nil {
		t.Fatalf("VarianceDisplayFromTicks returned error: %v", err)
	}
	if math.Abs(display.VariancePrice-0.2728) > 1e-12 {
		t.Fatalf("variance price = %v", display.VariancePrice)
	}
	if math.Abs(display.VolPercent-52.23) > 0.01 {
		t.Fatalf("vol percent = %v", display.VolPercent)
	}
}

func TestVarianceDisplayIsNotDirectTickOrVolPointScaling(t *testing.T) {
	instrument := instruments.Metadata{
		Symbol:         "BTCVAR30-PERP",
		TickSize:       "0.0001",
		QuotePrecision: 6,
	}

	display, err := VarianceDisplayFromTicks(instrument, "2728")
	if err != nil {
		t.Fatalf("VarianceDisplayFromTicks returned error: %v", err)
	}
	if math.Abs(display.VolPercent-27.28) < 0.01 || math.Abs(display.VolPercent-272.8) < 0.01 {
		t.Fatalf("vol percent was treated as direct price scaling: %v", display.VolPercent)
	}
}

func TestVarianceVolRoundTrip(t *testing.T) {
	variance := 0.2728
	vol := VarianceToVolPercent(variance)
	back := VolPercentToVariance(vol)
	if math.Abs(back-variance) > 1e-9 {
		t.Fatalf("roundtrip variance=%v vol=%v back=%v", variance, vol, back)
	}
}

func TestCalculateVariancePnLIsLinearInVariance(t *testing.T) {
	got := CalculateVariancePnL(0.2500, 0.2728, 100)
	if math.Abs(got-2.28) > 1e-12 {
		t.Fatalf("CalculateVariancePnL() = %v", got)
	}
}

func TestCalculateVariancePnLFromTicks(t *testing.T) {
	instrument := instruments.Metadata{
		Symbol:         "BTCVAR30-PERP",
		TickSize:       "0.0001",
		QuotePrecision: 6,
	}

	got, err := CalculateVariancePnLFromTicks(instrument, "2500", "2728", 100)
	if err != nil {
		t.Fatalf("CalculateVariancePnLFromTicks returned error: %v", err)
	}
	if math.Abs(got-2.28) > 1e-12 {
		t.Fatalf("CalculateVariancePnLFromTicks() = %v", got)
	}
}

func TestCalculateVariancePnLIsNotLinearInVol(t *testing.T) {
	entryVariance := 0.25
	exitVariance := 0.36
	notional := 100.0

	got := CalculateVariancePnL(entryVariance, exitVariance, notional)
	volDeltaPnL := (VarianceToVolPercent(exitVariance) - VarianceToVolPercent(entryVariance)) * notional

	if math.Abs(got-11.0) > 1e-12 {
		t.Fatalf("variance pnl = %v", got)
	}
	if math.Abs(got-volDeltaPnL) < 1e-6 {
		t.Fatalf("pnl unexpectedly matched vol-delta semantics: variance=%v vol=%v", got, volDeltaPnL)
	}
}

func TestParseRejectsOffTickPrice(t *testing.T) {
	converter, err := NewConverter(instruments.Metadata{
		Symbol:         "BTCVAR30-PERP",
		TickSize:       "0.0001",
		QuotePrecision: 6,
	})
	if err != nil {
		t.Fatalf("NewConverter returned error: %v", err)
	}

	if _, _, err := converter.Parse("0.27245"); err == nil {
		t.Fatal("expected off-tick price to fail")
	}
}

func TestParsePreservesIntegerInstrument(t *testing.T) {
	converter, err := NewConverter(instruments.Metadata{
		Symbol:         "BTCUSDC-CVXPERP",
		TickSize:       "1",
		QuotePrecision: 8,
	})
	if err != nil {
		t.Fatalf("NewConverter returned error: %v", err)
	}

	ticks, normalized, err := converter.Parse("40")
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if ticks != "40" || normalized != "40" {
		t.Fatalf("unexpected result ticks=%s normalized=%s", ticks, normalized)
	}
}
