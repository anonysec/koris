package worker

import (
	"runtime"
	"time"
)

// Config holds the configuration for the multi-worker manager.
type Config struct {
	// NumWorkers is the number of worker processes to spawn.
	// 0 means auto-detect via runtime.NumCPU(), capped at 4.
	NumWorkers int

	// Addr is the address to listen on (e.g. ":8080").
	Addr string

	// GracefulWait is how long to wait for workers to drain before SIGKILL.
	GracefulWait time.Duration

	// MaxRestarts is the maximum number of times a dead worker will be restarted
	// within the monitor interval before giving up.
	MaxRestarts int
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		NumWorkers:   0,
		Addr:         ":8080",
		GracefulWait: 30 * time.Second,
		MaxRestarts:  5,
	}
}

// ResolvedWorkers returns the effective number of workers.
// If NumWorkers is 0 (auto), it returns min(runtime.NumCPU(), 4).
func (c Config) ResolvedWorkers() int {
	if c.NumWorkers > 0 {
		return c.NumWorkers
	}
	n := runtime.NumCPU()
	if n > 4 {
		n = 4
	}
	if n < 1 {
		n = 1
	}
	return n
}
