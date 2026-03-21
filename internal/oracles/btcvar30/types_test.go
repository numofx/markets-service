package btcvar30

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestDerive(t *testing.T) {
	derived := Derive(64)
	if derived.Vol30D != 64 {
		t.Fatalf("vol = %v", derived.Vol30D)
	}
	if derived.Variance30D != 0.4096 {
		t.Fatalf("variance = %v", derived.Variance30D)
	}
}

func TestDeriveRoundTripsVarianceAndVol(t *testing.T) {
	derived := Derive(52.23)
	roundtripVol := math.Sqrt(derived.Variance30D) * 100
	if math.Abs(roundtripVol-52.23) > 0.01 {
		t.Fatalf("roundtrip vol = %v", roundtripVol)
	}
}

func TestPayloadCanonicalBytes(t *testing.T) {
	payload := Payload{
		Symbol:             Symbol,
		Source:             Source,
		Timestamp:          time.Unix(1_700_000_000, 0).UTC(),
		Vol30D:             64.25,
		Variance30D:        0.41280625,
		MethodologyVersion: MethodologyVersion,
	}

	canonical, err := payload.CanonicalBytes()
	if err != nil {
		t.Fatalf("CanonicalBytes returned error: %v", err)
	}

	text := string(canonical)
	if !strings.Contains(text, `"symbol":"BTCVAR30"`) {
		t.Fatalf("canonical bytes missing symbol: %s", text)
	}
	if strings.Contains(text, "signature") {
		t.Fatalf("canonical bytes unexpectedly include signature: %s", text)
	}
}
