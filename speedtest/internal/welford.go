package internal

import (
	"fmt"
	"math"
)

// Welford Fast standard deviation calculation with moving window
// ref Welford, B. P. (1962). Note on a Method for Calculating Corrected Sums of Squares and Products. Technometrics, 4(3), 419–420. https://doi.org/10.1080/00401706.1962.10490022
type Welford struct {
	n                                    int       // data size
	mean                                 float64   // mean
	sum                                  float64   // sum
	vector                               []float64 // data set
	eraseIndex                           int       // the value will be erased next time
	cap                                  int
	currentStdDev                        float64
	consecutiveStableIterations          int
	consecutiveStableIterationsThreshold int
	cv                                   float64
	ewmaMean                             float64
}

// NewWelford recommended windowSize = moving time window / sampling frequency
func NewWelford(windowSize int) *Welford {
	return &Welford{
		vector:                               make([]float64, windowSize),
		cap:                                  windowSize,
		consecutiveStableIterationsThreshold: 10,
	}
}

// Update Enter the given value into the measuring system.
// return bool stability evaluation
func (w *Welford) Update(value float64) bool {
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
	w.currentStdDev = math.Sqrt(w.Variance())
	// update C.V
	if w.mean == 0 {
		w.cv = 1
	} else {
		w.cv = w.currentStdDev / w.mean
		if w.cv > 1 {
			w.cv = 1
		}
	}
	// ewma beta ratio
	// TODO: w.cv needs normalization
	beta := w.cv*0.381 + 0.618
	w.ewmaMean = w.mean*beta + w.ewmaMean*(1-beta)
	// acc consecutiveStableIterations
	if w.cap/2 < w.n && w.cv < 0.03 {
		w.consecutiveStableIterations++
	}
	return w.consecutiveStableIterations >= w.consecutiveStableIterationsThreshold
}

func (w *Welford) Mean() float64 {
	return w.mean
}

func (w *Welford) CV() float64 {
	return w.cv
}

func (w *Welford) Variance() float64 {
	if w.n < 2 {
		return 0
	}
	return w.sum / float64(w.n-1)
}

func (w *Welford) StandardDeviation() float64 {
	return w.currentStdDev
}

func (w *Welford) EWMA() float64 {
	return w.ewmaMean
}

func (w *Welford) String() string {
	return fmt.Sprintf("Mean: %.2f, Standard Deviation: %.2f, C.V: %.2f, EWMA: %.2f", w.Mean(), w.StandardDeviation(), w.CV(), w.EWMA())
}
