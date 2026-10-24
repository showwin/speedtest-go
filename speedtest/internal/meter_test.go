package internal

import (
	"math"
	"testing"
)

func TestMeterRateLeavesOutTheFirstThird(t *testing.T) {
	var m Meter
	// 10 MB/s for the first second, then 1 MB/s until 3 s.
	for i := 1; i <= 60; i++ {
		seconds := float64(i) * 0.05
		bytes := 10e6 * math.Min(seconds, 1)
		if seconds > 1 {
			bytes += 1e6 * (seconds - 1)
		}
		m.Add(seconds, int64(bytes))
	}
	if got := m.Rate(); math.Abs(got-1e6) > 1 {
		t.Fatalf("rate is %v, want 1e6 from the last two thirds of the test", got)
	}
}

func TestMeterSettlesOnASteadyRate(t *testing.T) {
	var m Meter
	for i := 1; i <= 60; i++ {
		jitter := 1000.0 * float64(i%3-1) // up to 1 kB around 50 kB a sample
		m.Add(float64(i)*0.05, int64(float64(i)*50e3+jitter))
	}
	if !m.Settled(0.015) {
		t.Fatal("a steady 1 MB/s rate did not settle")
	}
}

func TestMeterDoesNotSettleOnATrend(t *testing.T) {
	var m Meter
	for i := 1; i <= 60; i++ {
		seconds := float64(i) * 0.05
		m.Add(seconds, int64(1e6*seconds*seconds)) // rate rising linearly
	}
	if m.Settled(0.015) {
		t.Fatal("a rising rate settled")
	}
}

func TestMeterDoesNotSettleOnNoise(t *testing.T) {
	var m Meter
	total := 0.0
	for i := 1; i <= 60; i++ {
		if i%10 == 0 {
			total += 500e3 // 500 kB lumps once every half second, as confirmed uploads arrive
		}
		m.Add(float64(i)*0.05, int64(total))
	}
	if m.Settled(0.015) {
		t.Fatal("a lumpy rate settled")
	}
}

func TestMeterNeverSettlesAtZero(t *testing.T) {
	var m Meter
	for i := 1; i <= 300; i++ {
		m.Add(float64(i)*0.05, 0)
	}
	if m.Settled(0.015) {
		t.Fatal("a zero rate settled")
	}
	if got := m.Rate(); got != 0 {
		t.Fatalf("rate is %v, want 0", got)
	}
}
