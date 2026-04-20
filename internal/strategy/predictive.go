package strategy

import (
	"fmt"
	"math"
)

const (
	minPredictTrainWindow = 5
)

// Predictive computes a linear model over recent candle returns to forecast next return.
//
// Features:
// - lag1 return (r[t-1])
// - lag2 return (r[t-2])
// - 3-period momentum (price[t]/price[t-3]-1)
//
// The model is fit each tick with ordinary least squares over the most recent
// trainWindow samples and converts forecast to BUY/SELL/HOLD via minDelta.
func Predictive(prices []float64, trainWindow int, minDelta float64) (Signal, error) {
	if trainWindow < minPredictTrainWindow {
		return Signal{}, fmt.Errorf("trainWindow must be >= %d", minPredictTrainWindow)
	}
	if minDelta < 0 {
		return Signal{}, fmt.Errorf("minDelta must be non-negative")
	}
	if len(prices) < trainWindow+4 {
		return Signal{}, fmt.Errorf("insufficient prices: need at least %d, have %d", trainWindow+4, len(prices))
	}

	returns := make([]float64, 0, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		prev := prices[i-1]
		if prev == 0 {
			return Signal{}, fmt.Errorf("price at index %d is zero; cannot compute return", i-1)
		}
		r := (prices[i] - prev) / prev
		if math.IsNaN(r) || math.IsInf(r, 0) {
			return Signal{}, fmt.Errorf("invalid return at index %d", i-1)
		}
		returns = append(returns, r)
	}

	X, y := buildTrainingSet(prices, returns)
	if len(y) < trainWindow {
		return Signal{}, fmt.Errorf("not enough training samples: need %d, have %d", trainWindow, len(y))
	}

	start := len(y) - trainWindow
	lastIdx := len(prices) - 1
	lastPrice := prices[lastIdx]
	if prices[lastIdx-3] == 0 {
		return Signal{}, fmt.Errorf("price at index %d is zero; cannot compute momentum feature", lastIdx-3)
	}

	lastFeatures := []float64{
		1.0,
		returns[lastIdx-1], // r[t-1]
		returns[lastIdx-2], // r[t-2]
		prices[lastIdx]/prices[lastIdx-3] - 1.0,
	}

	predicted := 0.0
	mae := 0.0
	confidence := 0.0
	weights, err := fitOLS(X[start:], y[start:])

	if err != nil {
		// Degenerate training windows (e.g. very flat returns) can make
		// X^T*X singular. Instead of failing the whole tick, fallback to the
		// latest observed return as a conservative proxy and low confidence.
		predicted = returns[lastIdx-1]
		mae = 0
		confidence = 0.15
	} else {
		predicted = dot(weights, lastFeatures)
		mae = meanAbsError(X[start:], y[start:], weights)
		confidence = 1.0 / (1.0 + mae*200.0)
		if confidence < 0 {
			confidence = 0
		}
		if confidence > 1 {
			confidence = 1
		}
	}

	decision := DecisionHold
	switch {
	case predicted >= minDelta:
		decision = DecisionBuy
	case predicted <= -minDelta:
		decision = DecisionSell
	}

	return Signal{
		Decision:         decision,
		Delta:            predicted,
		LastPrice:        lastPrice,
		PredictedReturn:  predicted,
		ModelConfidence:  confidence,
		TrainingSamples:  trainWindow,
		TrainingMeanAbsE: mae,
	}, nil
}

func buildTrainingSet(prices, returns []float64) ([][]float64, []float64) {
	X := make([][]float64, 0, len(returns))
	y := make([]float64, 0, len(returns))

	// We predict return r[t] using info available at t-1.
	// Minimum t is 3 so we can access r[t-1], r[t-2], and price[t-1]/price[t-3]-1.
	for t := 3; t < len(returns); t++ {
		f := []float64{
			1.0,
			returns[t-1],
			returns[t-2],
			(prices[t]/prices[t-3] - 1.0), // prices index aligns with return time.
		}
		X = append(X, f)
		y = append(y, returns[t])
	}
	return X, y
}

func fitOLS(X [][]float64, y []float64) ([]float64, error) {
	if len(X) == 0 || len(y) == 0 {
		return nil, fmt.Errorf("empty training data")
	}
	if len(X) != len(y) {
		return nil, fmt.Errorf("shape mismatch between X and y")
	}

	features := len(X[0])
	if features == 0 {
		return nil, fmt.Errorf("no features")
	}
	for i := range X {
		if len(X[i]) != features {
			return nil, fmt.Errorf("inconsistent feature size at row %d", i)
		}
	}

	xtx := make([][]float64, features)
	for i := range xtx {
		xtx[i] = make([]float64, features)
	}
	xty := make([]float64, features)

	for row := range X {
		for i := 0; i < features; i++ {
			xty[i] += X[row][i] * y[row]
			for j := 0; j < features; j++ {
				xtx[i][j] += X[row][i] * X[row][j]
			}
		}
	}

	weights, ok := solveLinearSystem(xtx, xty)
	if !ok {
		return nil, fmt.Errorf("normal equation is singular")
	}
	return weights, nil
}

func solveLinearSystem(A [][]float64, b []float64) ([]float64, bool) {
	n := len(A)
	if n == 0 || len(b) != n {
		return nil, false
	}
	for i := range A {
		if len(A[i]) != n {
			return nil, false
		}
	}

	// Augmented matrix [A|b]
	aug := make([][]float64, n)
	for i := 0; i < n; i++ {
		aug[i] = make([]float64, n+1)
		copy(aug[i], A[i])
		aug[i][n] = b[i]
	}

	for col := 0; col < n; col++ {
		pivot := col
		maxAbs := math.Abs(aug[col][col])
		for row := col + 1; row < n; row++ {
			v := math.Abs(aug[row][col])
			if v > maxAbs {
				maxAbs = v
				pivot = row
			}
		}
		if maxAbs < 1e-12 {
			return nil, false
		}
		if pivot != col {
			aug[col], aug[pivot] = aug[pivot], aug[col]
		}

		diag := aug[col][col]
		for j := col; j <= n; j++ {
			aug[col][j] /= diag
		}
		for row := 0; row < n; row++ {
			if row == col {
				continue
			}
			factor := aug[row][col]
			if factor == 0 {
				continue
			}
			for j := col; j <= n; j++ {
				aug[row][j] -= factor * aug[col][j]
			}
		}
	}

	x := make([]float64, n)
	for i := 0; i < n; i++ {
		x[i] = aug[i][n]
	}
	return x, true
}

func dot(weights, features []float64) float64 {
	n := len(weights)
	if len(features) < n {
		n = len(features)
	}
	out := 0.0
	for i := 0; i < n; i++ {
		out += weights[i] * features[i]
	}
	return out
}

func meanAbsError(X [][]float64, y, w []float64) float64 {
	if len(X) == 0 || len(y) == 0 {
		return 0
	}
	n := len(y)
	if len(X) < n {
		n = len(X)
	}
	total := 0.0
	for i := 0; i < n; i++ {
		p := dot(w, X[i])
		total += math.Abs(y[i] - p)
	}
	return total / float64(n)
}
