package funding

import (
	"math"
	"testing"
)

func TestCalculateFunding(t *testing.T) {
	tests := []struct {
		name           string
		markPrice      float64
		oracleVariance float64
		coefficient    float64
		cap            float64
		want           float64
	}{
		{
			name:           "basic calculation",
			markPrice:      0.45,
			oracleVariance: 0.40,
			coefficient:    0.10,
			cap:            0.05,
			want:           0.005,
		},
		{
			name:           "positive clamp",
			markPrice:      1.20,
			oracleVariance: 0.20,
			coefficient:    0.10,
			cap:            0.05,
			want:           0.05,
		},
		{
			name:           "negative clamp",
			markPrice:      0.10,
			oracleVariance: 1.10,
			coefficient:    0.10,
			cap:            0.05,
			want:           -0.05,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateFunding(tt.markPrice, tt.oracleVariance, tt.coefficient, tt.cap)
			if math.Abs(got-tt.want) > 1e-12 {
				t.Fatalf("CalculateFunding() = %v, want %v", got, tt.want)
			}
		})
	}
}
