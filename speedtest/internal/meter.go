package internal

import (
	"math"
	"sort"
)

// Meter estimates a transfer rate from cumulative byte counts sampled at a fixed interval.
//
// The rate is measured over the last two thirds of the elapsed time, so TCP ramp-up drops out
// of the result as the test runs, without a ramp-up detector. The rate has settled when two
// standard errors and the difference between the window's halves are both within a tolerance
// of it. The standard error is Newey-West over the per-sample byte increments (Bartlett weights,
// sqrt(n) lags), because throughput samples follow each other.
type Meter struct {
	seconds []float64 // elapsed time of each sample
	bytes   []float64 // cumulative bytes at each sample
}

// windowStart is the fraction of the elapsed time a rate estimate leaves out.
const windowStart = 1.0 / 3

// Add records the cumulative byte count at an elapsed time. Samples must be added in time order.
func (m *Meter) Add(seconds float64, bytes int64) {
	if len(m.seconds) == 0 {
		m.seconds, m.bytes = append(m.seconds, 0), append(m.bytes, 0)
	}
	if seconds <= m.seconds[len(m.seconds)-1] {
		return
	}
	m.seconds = append(m.seconds, seconds)
	m.bytes = append(m.bytes, float64(bytes))
}

// Rate returns bytes per second over the last two thirds of the elapsed time.
func (m *Meter) Rate() float64 {
	i, j := m.window()
	return m.rate(i, j)
}

// Settled reports whether the rate is known within the relative tolerance eps.
// A zero rate never settles.
func (m *Meter) Settled(eps float64) bool {
	i, j := m.window()
	rate := m.rate(i, j)
	if rate <= 0 || j-i < 4 {
		return false
	}
	mid := m.index((m.seconds[i] + m.seconds[j]) / 2)
	if math.Abs(m.rate(mid, j)-m.rate(i, mid)) > eps*rate {
		return false
	}
	return 2*m.standardError(i, j, rate) <= eps*rate
}

func (m *Meter) window() (int, int) {
	j := len(m.seconds) - 1
	if j < 1 {
		return 0, 0
	}
	return m.index(m.seconds[j] * windowStart), j
}

// index returns the sample nearest to an elapsed time.
func (m *Meter) index(seconds float64) int {
	i := sort.SearchFloat64s(m.seconds, seconds)
	if i == len(m.seconds) || (i > 0 && seconds-m.seconds[i-1] < m.seconds[i]-seconds) {
		i--
	}
	return i
}

func (m *Meter) rate(i, j int) float64 {
	if j <= i {
		return 0
	}
	return (m.bytes[j] - m.bytes[i]) / (m.seconds[j] - m.seconds[i])
}

func (m *Meter) standardError(i, j int, rate float64) float64 {
	n := j - i
	residuals := make([]float64, n)
	for k := range residuals {
		residuals[k] = m.bytes[i+k+1] - m.bytes[i+k] - rate*(m.seconds[i+k+1]-m.seconds[i+k])
	}
	lags := int(math.Sqrt(float64(n)))
	longRun := 0.0
	for lag := 0; lag <= lags; lag++ {
		c := 0.0
		for k := lag; k < n; k++ {
			c += residuals[k] * residuals[k-lag]
		}
		weight := 1.0
		if lag > 0 {
			weight = 2 * (1 - float64(lag)/float64(lags+1))
		}
		longRun += weight * c / float64(n)
	}
	return math.Sqrt(float64(n)*math.Max(longRun, 0)) / (m.seconds[j] - m.seconds[i])
}
