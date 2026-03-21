package btcvar30

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/numofx/matching-backend/internal/pricing"
)

const (
	Symbol             = "BTCVAR30"
	Source             = "deribit"
	MethodologyVersion = "deribit-vol-index-v1"
)

type Payload struct {
	Symbol             string    `json:"symbol"`
	Source             string    `json:"source"`
	Timestamp          time.Time `json:"timestamp"`
	Vol30D             float64   `json:"vol_30d"`
	Variance30D        float64   `json:"variance_30d"`
	MethodologyVersion string    `json:"methodology_version"`
	Signature          string    `json:"signature,omitempty"`
	Stale              bool      `json:"stale,omitempty"`
}

type Derivation struct {
	Vol30D      float64
	Variance30D float64
}

func Derive(vol30D float64) Derivation {
	return Derivation{
		Vol30D:      vol30D,
		Variance30D: pricing.VolPercentToVariance(vol30D),
	}
}

func (p Payload) CanonicalBytes() ([]byte, error) {
	type canonical struct {
		Symbol             string  `json:"symbol"`
		Source             string  `json:"source"`
		Timestamp          string  `json:"timestamp"`
		Vol30D             float64 `json:"vol_30d"`
		Variance30D        float64 `json:"variance_30d"`
		MethodologyVersion string  `json:"methodology_version"`
	}

	encoded, err := json.Marshal(canonical{
		Symbol:             p.Symbol,
		Source:             p.Source,
		Timestamp:          p.Timestamp.UTC().Format(time.RFC3339Nano),
		Vol30D:             p.Vol30D,
		Variance30D:        p.Variance30D,
		MethodologyVersion: p.MethodologyVersion,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal canonical oracle payload: %w", err)
	}

	return bytes.TrimSpace(encoded), nil
}
