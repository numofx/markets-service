package matching

import (
	"testing"
	"time"

	"github.com/numofx/matching-backend/internal/orders"
)

func TestCrosses(t *testing.T) {
	now := time.Unix(1700000000, 0)

	tests := []struct {
		name    string
		taker   orders.Order
		maker   orders.Order
		want    bool
		wantErr bool
	}{
		{
			name:  "buy taker crosses lower sell maker",
			taker: orders.Order{Side: orders.SideBuy, LimitPrice: "100", CreatedAt: now},
			maker: orders.Order{Side: orders.SideSell, LimitPrice: "90", CreatedAt: now.Add(-time.Second)},
			want:  true,
		},
		{
			name:  "sell taker crosses higher buy maker",
			taker: orders.Order{Side: orders.SideSell, LimitPrice: "90", CreatedAt: now},
			maker: orders.Order{Side: orders.SideBuy, LimitPrice: "100", CreatedAt: now.Add(-time.Second)},
			want:  true,
		},
		{
			name:  "buy taker does not cross",
			taker: orders.Order{Side: orders.SideBuy, LimitPrice: "80", CreatedAt: now},
			maker: orders.Order{Side: orders.SideSell, LimitPrice: "90", CreatedAt: now.Add(-time.Second)},
			want:  false,
		},
		{
			name:    "invalid price",
			taker:   orders.Order{Side: orders.SideBuy, LimitPrice: "bad", CreatedAt: now},
			maker:   orders.Order{Side: orders.SideSell, LimitPrice: "90", CreatedAt: now.Add(-time.Second)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := crosses(tt.taker, tt.maker)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("crosses returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("crosses = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldFinalizeAfterExecutorError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "decoded contract error name",
			err:  errString("executor returned status 500: TM_FillLimitCrossed()"),
			want: true,
		},
		{
			name: "raw contract selector",
			err:  errString("executor returned status 500: 0xfea8fa6f"),
			want: true,
		},
		{
			name: "unrelated executor error",
			err:  errString("executor returned status 500: connection reset"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldFinalizeAfterExecutorError(tt.err); got != tt.want {
				t.Fatalf("shouldFinalizeAfterExecutorError() = %v, want %v", got, tt.want)
			}
		})
	}
}

type errString string

func (e errString) Error() string {
	return string(e)
}
