package strategy

import "testing"

func TestSMA(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		prices     []float64
		fast       int
		slow       int
		minDelta   float64
		want       Decision
		expectFail bool
	}{
		{
			name:     "buy signal when fast above slow by threshold",
			prices:   []float64{10, 10, 10, 11, 12, 13, 14, 15},
			fast:     3,
			slow:     6,
			minDelta: 0.01,
			want:     DecisionBuy,
		},
		{
			name:     "sell signal when fast below slow by threshold",
			prices:   []float64{15, 14, 13, 12, 11, 10, 9, 8},
			fast:     3,
			slow:     6,
			minDelta: 0.01,
			want:     DecisionSell,
		},
		{
			name:     "hold when delta below threshold",
			prices:   []float64{10, 10.1, 10.0, 10.1, 10.0, 10.1, 10.0, 10.1},
			fast:     3,
			slow:     6,
			minDelta: 0.05,
			want:     DecisionHold,
		},
		{
			name:       "fail on insufficient price history",
			prices:     []float64{1, 2, 3},
			fast:       2,
			slow:       6,
			minDelta:   0.01,
			expectFail: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			signal, err := SMA(tc.prices, tc.fast, tc.slow, tc.minDelta)
			if tc.expectFail {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if signal.Decision != tc.want {
				t.Fatalf("expected decision %s, got %s", tc.want, signal.Decision)
			}
		})
	}
}
