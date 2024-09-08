package control

type Controller interface {
	// Get Reference counter volume
	Get() int64
	// Add Reference counter increment
	Add(delta int64)
	// Repeat Pointing to duplicate memory space
	Repeat() []byte
	// Done Notification processing completed
	Done() <-chan struct{}
}
