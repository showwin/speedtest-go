package main

import (
	"fmt"
	"math"
)

// Welford Fast standard deviation calculation with moving window
// ref Welford, B. P. (1962). Note on a Method for Calculating Corrected Sums of Squares and Products. Technometrics, 4(3), 419–420. https://doi.org/10.1080/00401706.1962.10490022
type Welford struct {
	n                              int       // data size
	mean                           float64   // mean
	sum                            float64   // sum
	vector                         []float64 // data set
	eraseIndex                     int       // the value will be erased next time
	cap                            int
	currentStdDev                  float64
	tolerance                      float64
	factor                         float64
	prevStdDev                     float64
	consecutiveStableIterations    int
	maxConsecutiveStableIterations int
}

// NewWelford recommended windowSize = moving time window / sampling frequency
func NewWelford(windowSize int) *Welford {
	return &Welford{
		eraseIndex:                     0,
		vector:                         make([]float64, windowSize),
		cap:                            windowSize,
		factor:                         1e-3,
		tolerance:                      1e-6,
		maxConsecutiveStableIterations: 3,
	}
}

func (w *Welford) Add(value float64) {
	if w.n == w.cap {
		delta := w.vector[w.eraseIndex] - w.mean
		w.mean -= delta / float64(w.n-1)
		w.sum -= delta * (w.vector[w.eraseIndex] - w.mean)
		// the calc error is approximated to zero
		if w.sum < 0 {
			w.sum = 0
		}
		w.vector[w.eraseIndex] = value
		w.eraseIndex++
		if w.eraseIndex == w.cap {
			w.eraseIndex = 0
		}
	} else {
		w.vector[w.n] = value
		w.n++
	}
	delta := value - w.mean
	w.mean += delta / float64(w.n)
	w.sum += delta * (value - w.mean)
}

func (w *Welford) Mean() float64 {
	return w.mean
}

func (w *Welford) Variance() float64 {
	if w.n < 2 {
		return 0
	}
	return w.sum / float64(w.n-1)
}

func (w *Welford) StandardDeviation() float64 {
	w.currentStdDev = math.Sqrt(w.Variance())
	return w.currentStdDev
}

func (w *Welford) Convergence() bool {
	if w.n > 1 {
		w.tolerance = w.currentStdDev * w.factor // dynamic tolerance
		conv := math.Abs(w.currentStdDev - w.prevStdDev)
		fmt.Printf("conv: %f, tolerance: %f\n", conv, w.tolerance)
		if conv < w.tolerance && w.cap/2 < w.n {
			w.consecutiveStableIterations++
		}
	}
	w.prevStdDev = w.currentStdDev
	return w.consecutiveStableIterations >= w.maxConsecutiveStableIterations
}
