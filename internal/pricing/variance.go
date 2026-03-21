package pricing

import (
	"fmt"
	"math"
	"strconv"

	"github.com/numofx/matching-backend/internal/instruments"
)

// Variance-native markets settle and mark in variance space. Any vol percentage
// representation is presentation-only and must never be used for core economics.
type VarianceDisplay struct {
	VariancePrice float64 `json:"variance_price"`
	VolPercent    float64 `json:"vol_percent"`
}

func VarianceDisplayFromTicks(instrument instruments.Metadata, ticks string) (VarianceDisplay, error) {
	variancePrice, err := TicksToVarianceFloat64(instrument, ticks)
	if err != nil {
		return VarianceDisplay{}, err
	}

	return VarianceDisplay{
		VariancePrice: variancePrice,
		VolPercent:    RoundVolPercent(VarianceToVolPercent(variancePrice)),
	}, nil
}

func TicksToVarianceFloat64(instrument instruments.Metadata, ticks string) (float64, error) {
	converter, err := NewConverter(instrument)
	if err != nil {
		return 0, err
	}
	value, err := converter.FormatTicks(ticks)
	if err != nil {
		return 0, err
	}
	return ParseVarianceFloat64(value)
}

func ParseVarianceFloat64(variance string) (float64, error) {
	value, err := strconv.ParseFloat(variance, 64)
	if err != nil {
		return 0, fmt.Errorf("parse variance %q: %w", variance, err)
	}
	if value < 0 {
		return 0, fmt.Errorf("variance must be non-negative")
	}
	return value, nil
}

func VarianceToVolPercent(variance float64) float64 {
	if variance < 0 {
		return 0
	}
	return math.Sqrt(variance) * 100.0
}

func VolPercentToVariance(volPercent float64) float64 {
	value := volPercent / 100.0
	return value * value
}

func RoundVolPercent(value float64) float64 {
	return roundTo(value, 2)
}

func roundTo(value float64, decimals int) float64 {
	if decimals < 0 {
		return value
	}
	factor := math.Pow10(decimals)
	return math.Round(value*factor) / factor
}

func CalculateVariancePnL(entryVariance float64, exitVariance float64, notional float64) float64 {
	return (exitVariance - entryVariance) * notional
}

func CalculateVariancePnLFromTicks(instrument instruments.Metadata, entryTicks string, exitTicks string, notional float64) (float64, error) {
	entryVariance, err := TicksToVarianceFloat64(instrument, entryTicks)
	if err != nil {
		return 0, err
	}
	exitVariance, err := TicksToVarianceFloat64(instrument, exitTicks)
	if err != nil {
		return 0, err
	}
	return CalculateVariancePnL(entryVariance, exitVariance, notional), nil
}
