package control

import (
	"time"
)

type Manager interface {
	SetSamplingPeriod(duration time.Duration) Manager
	SetSamplingDuration(duration time.Duration) Manager

	History() *Tracer

	// SetNThread This function name is confusing.
	// Deprecated: Replaced by [DataManager.SetMaxConnections].
	SetNThread(n int) Manager
	SetMaxConnections(n int) Manager

	GetSamplingPeriod() time.Duration
	GetSamplingDuration() time.Duration

	GetMaxConnections() int
}
