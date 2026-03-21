package pricing

import (
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
