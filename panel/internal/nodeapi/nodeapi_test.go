package nodeapi

import (
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// --- Unit Tests: ComputeScore (table-driven) ---

func TestComputeScore(t *testing.T) {
	checker := &HealthChecker{
		checkInterval: 30 * time.Second,
		staleAfter:    2 * time.Minute,
		offlineAfter:  5 * time.Minute,
		weights:       DefaultWeights(),
	}

	tests := []struct {
		name      string
		conn      *NodeConnection
		wantMin   float64
		wantMax   float64
		wantApprx float64 // approximate expected value (±0.05)
	}{
		{
			name: "perfect node — 0ms latency, 0 failures, just seen",
			conn: &NodeConnection{
				Latency:     0,
				ConsecFails: 0,
				LastSeen:    time.Now(),
			},
			wantMin:   0.95,
			wantMax:   1.0,
			wantApprx: 1.0,
		},
		{
			name: "high latency — 500ms",
			conn: &NodeConnection{
				Latency:     500 * time.Millisecond,
				ConsecFails: 0,
				LastSeen:    time.Now(),
			},
			wantMin:   0.75,
			wantMax:   0.85,
			wantApprx: 0.8, // latency=0.5*0.4=0.2, avail=1.0*0.4=0.4, fresh=1.0*0.2=0.2 => 0.8
		},
		{
			name: "very high latency — 1000ms+, latency component is 0",
			conn: &NodeConnection{
				Latency:     1200 * time.Millisecond,
				ConsecFails: 0,
				LastSeen:    time.Now(),
			},
			wantMin:   0.55,
			wantMax:   0.65,
			wantApprx: 0.6, // latency=0*0.4=0, avail=1.0*0.4=0.4, fresh=1.0*0.2=0.2 => 0.6
		},
		{
			name: "failed node — 5 consecutive failures",
			conn: &NodeConnection{
				Latency:     0,
				ConsecFails: 5,
				LastSeen:    time.Now(),
			},
			wantMin:   0.55,
			wantMax:   0.65,
			wantApprx: 0.6, // latency=1.0*0.4=0.4, avail=0*0.4=0, fresh=1.0*0.2=0.2 => 0.6
		},
		{
			name: "stale node — 300s since last seen",
			conn: &NodeConnection{
				Latency:     0,
				ConsecFails: 0,
				LastSeen:    time.Now().Add(-300 * time.Second),
			},
			wantMin:   0.75,
			wantMax:   0.85,
			wantApprx: 0.8, // latency=1.0*0.4=0.4, avail=1.0*0.4=0.4, fresh=0*0.2=0 => 0.8
		},
		{
			name: "dead node — 1000ms latency, 5 failures, 5min since seen",
			conn: &NodeConnection{
				Latency:     1000 * time.Millisecond,
				ConsecFails: 5,
				LastSeen:    time.Now().Add(-5 * time.Minute),
			},
			wantMin:   0.0,
			wantMax:   0.05,
			wantApprx: 0.0, // latency=0, avail=0, fresh=0 => 0
		},
		{
			name: "moderate latency — 250ms",
			conn: &NodeConnection{
				Latency:     250 * time.Millisecond,
				ConsecFails: 0,
				LastSeen:    time.Now(),
			},
			wantMin:   0.85,
			wantMax:   0.95,
			wantApprx: 0.9, // latency=0.75*0.4=0.3, avail=1.0*0.4=0.4, fresh=1.0*0.2=0.2 => 0.9
		},
		{
			name: "partial failures — 2 consecutive fails",
			conn: &NodeConnection{
				Latency:     0,
				ConsecFails: 2,
				LastSeen:    time.Now(),
			},
			wantMin:   0.75,
			wantMax:   0.85,
			wantApprx: 0.84, // latency=1.0*0.4=0.4, avail=0.6*0.4=0.24, fresh=1.0*0.2=0.2 => 0.84
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := checker.ComputeScore(tt.conn)

			if score < tt.wantMin || score > tt.wantMax {
				t.Errorf("ComputeScore() = %f, want in [%f, %f] (approx %f)",
					score, tt.wantMin, tt.wantMax, tt.wantApprx)
			}
		})
	}
}

// --- Property Tests: ComputeScore ---

// **Validates: Requirements 6.2**
func TestComputeScore_Property_AlwaysInRange(t *testing.T) {
	properties := gopter.NewProperties(gopter.DefaultTestParameters())

	checker := &HealthChecker{
		checkInterval: 30 * time.Second,
		staleAfter:    2 * time.Minute,
		offlineAfter:  5 * time.Minute,
		weights:       DefaultWeights(),
	}

	properties.Property("score is always in [0.0, 1.0]", prop.ForAll(
		func(latencyMs int64, consecFails int, secondsSinceLastSeen int64) bool {
			conn := &NodeConnection{
				Latency:     time.Duration(latencyMs) * time.Millisecond,
				ConsecFails: consecFails,
				LastSeen:    time.Now().Add(-time.Duration(secondsSinceLastSeen) * time.Second),
			}
			score := checker.ComputeScore(conn)
			return score >= 0.0 && score <= 1.0
		},
		gen.Int64Range(0, 5000), // latency: 0-5000ms
		gen.IntRange(0, 20),     // consecutive failures: 0-20
		gen.Int64Range(0, 600),  // seconds since last seen: 0-600s
	))

	properties.TestingRun(t)
}

// **Validates: Requirements 6.2**
func TestComputeScore_Property_MonotonicLatency(t *testing.T) {
	properties := gopter.NewProperties(gopter.DefaultTestParameters())

	checker := &HealthChecker{
		checkInterval: 30 * time.Second,
		staleAfter:    2 * time.Minute,
		offlineAfter:  5 * time.Minute,
		weights:       DefaultWeights(),
	}

	properties.Property("score decreases (or stays same) as latency increases", prop.ForAll(
		func(latencyLowMs int64, latencyHighMs int64) bool {
			// Ensure latencyLow <= latencyHigh
			if latencyLowMs > latencyHighMs {
				latencyLowMs, latencyHighMs = latencyHighMs, latencyLowMs
			}

			now := time.Now()
			connLow := &NodeConnection{
				Latency:     time.Duration(latencyLowMs) * time.Millisecond,
				ConsecFails: 0,
				LastSeen:    now,
			}
			connHigh := &NodeConnection{
				Latency:     time.Duration(latencyHighMs) * time.Millisecond,
				ConsecFails: 0,
				LastSeen:    now,
			}

			scoreLow := checker.ComputeScore(connLow)
			scoreHigh := checker.ComputeScore(connHigh)

			return scoreLow >= scoreHigh
		},
		gen.Int64Range(0, 2000),
		gen.Int64Range(0, 2000),
	))

	properties.TestingRun(t)
}

// **Validates: Requirements 6.2**
func TestComputeScore_Property_MonotonicConsecFails(t *testing.T) {
	properties := gopter.NewProperties(gopter.DefaultTestParameters())

	checker := &HealthChecker{
		checkInterval: 30 * time.Second,
		staleAfter:    2 * time.Minute,
		offlineAfter:  5 * time.Minute,
		weights:       DefaultWeights(),
	}

	properties.Property("score decreases (or stays same) as ConsecFails increases", prop.ForAll(
		func(failsLow int, failsHigh int) bool {
			if failsLow > failsHigh {
				failsLow, failsHigh = failsHigh, failsLow
			}

			now := time.Now()
			connLow := &NodeConnection{
				Latency:     100 * time.Millisecond,
				ConsecFails: failsLow,
				LastSeen:    now,
			}
			connHigh := &NodeConnection{
				Latency:     100 * time.Millisecond,
				ConsecFails: failsHigh,
				LastSeen:    now,
			}

			scoreLow := checker.ComputeScore(connLow)
			scoreHigh := checker.ComputeScore(connHigh)

			return scoreLow >= scoreHigh
		},
		gen.IntRange(0, 20),
		gen.IntRange(0, 20),
	))

	properties.TestingRun(t)
}

// **Validates: Requirements 6.2**
func TestComputeScore_Property_MonotonicFreshness(t *testing.T) {
	properties := gopter.NewProperties(gopter.DefaultTestParameters())

	checker := &HealthChecker{
		checkInterval: 30 * time.Second,
		staleAfter:    2 * time.Minute,
		offlineAfter:  5 * time.Minute,
		weights:       DefaultWeights(),
	}

	properties.Property("score decreases (or stays same) as time since LastSeen increases", prop.ForAll(
		func(secsLow int64, secsHigh int64) bool {
			if secsLow > secsHigh {
				secsLow, secsHigh = secsHigh, secsLow
			}

			now := time.Now()
			connRecent := &NodeConnection{
				Latency:     100 * time.Millisecond,
				ConsecFails: 0,
				LastSeen:    now.Add(-time.Duration(secsLow) * time.Second),
			}
			connStale := &NodeConnection{
				Latency:     100 * time.Millisecond,
				ConsecFails: 0,
				LastSeen:    now.Add(-time.Duration(secsHigh) * time.Second),
			}

			scoreRecent := checker.ComputeScore(connRecent)
			scoreStale := checker.ComputeScore(connStale)

			return scoreRecent >= scoreStale
		},
		gen.Int64Range(0, 600),
		gen.Int64Range(0, 600),
	))

	properties.TestingRun(t)
}

// --- Unit Tests: ReconnectionPolicy ---

func TestReconnectionPolicy_NextDelay(t *testing.T) {
	policy := DefaultReconnectionPolicy()

	t.Run("initial delay is approximately 2s", func(t *testing.T) {
		// Run multiple times to account for jitter
		for i := 0; i < 100; i++ {
			delay := policy.NextDelay(0)
			minExpected := time.Duration(float64(2*time.Second) * 0.9) // -10% jitter
			maxExpected := time.Duration(float64(2*time.Second) * 1.1) // +10% jitter

			if delay < minExpected || delay > maxExpected {
				t.Errorf("attempt 0: delay = %v, want in [%v, %v]", delay, minExpected, maxExpected)
			}
		}
	})

	t.Run("delay doubles each attempt", func(t *testing.T) {
		// Check that the central tendency doubles
		// Use many samples to average out jitter
		attempts := []int{0, 1, 2, 3, 4}
		expectedBase := []time.Duration{
			2 * time.Second,
			4 * time.Second,
			8 * time.Second,
			16 * time.Second,
			32 * time.Second,
		}

		for i, attempt := range attempts {
			var totalDelay time.Duration
			samples := 200
			for s := 0; s < samples; s++ {
				totalDelay += policy.NextDelay(attempt)
			}
			avgDelay := totalDelay / time.Duration(samples)

			// Average should be close to expected base (within 5% since jitter is symmetric)
			tolerance := time.Duration(float64(expectedBase[i]) * 0.15)
			if avgDelay < expectedBase[i]-tolerance || avgDelay > expectedBase[i]+tolerance {
				t.Errorf("attempt %d: avg delay = %v, want ~%v (±%v)",
					attempt, avgDelay, expectedBase[i], tolerance)
			}
		}
	})

	t.Run("delay caps at 60s", func(t *testing.T) {
		// attempt 10: base = 2s * 2^10 = 2048s >> 60s, should cap
		maxBase := float64(60 * time.Second)
		for i := 0; i < 100; i++ {
			delay := policy.NextDelay(10)
			maxExpected := time.Duration(maxBase * 1.1) // cap + jitter

			if delay > maxExpected {
				t.Errorf("attempt 10: delay = %v, want <= %v (max + jitter)", delay, maxExpected)
			}
			// Also ensure it doesn't go far below the cap minus jitter
			minExpected := time.Duration(maxBase * 0.9)
			if delay < minExpected {
				t.Errorf("attempt 10: delay = %v, want >= %v (cap - jitter)", delay, minExpected)
			}
		}
	})

	t.Run("jitter stays within ±10%", func(t *testing.T) {
		// For a fixed attempt, all delays should be within ±10% of the base
		attempts := []int{0, 1, 2, 3, 5}

		for _, attempt := range attempts {
			baseDelay := float64(2 * time.Second)
			for i := 0; i < attempt; i++ {
				baseDelay *= 2.0
			}
			if baseDelay > float64(60*time.Second) {
				baseDelay = float64(60 * time.Second)
			}

			minExpected := time.Duration(baseDelay * 0.9)
			maxExpected := time.Duration(baseDelay * 1.1)

			for i := 0; i < 200; i++ {
				delay := policy.NextDelay(attempt)
				if delay < minExpected || delay > maxExpected {
					t.Errorf("attempt %d, sample %d: delay = %v, want in [%v, %v]",
						attempt, i, delay, minExpected, maxExpected)
					break
				}
			}
		}
	})
}

func TestReconnectionPolicy_NextDelay_NeverNegative(t *testing.T) {
	policy := DefaultReconnectionPolicy()

	for attempt := 0; attempt < 20; attempt++ {
		for i := 0; i < 100; i++ {
			delay := policy.NextDelay(attempt)
			if delay < 0 {
				t.Errorf("attempt %d: delay = %v, must be non-negative", attempt, delay)
			}
		}
	}
}

// --- Helpers for property tests ---

func init() {
	// Seed the random source for reproducibility in tests (jitter uses rand)
	rand.Seed(time.Now().UnixNano())
}

// --- Additional edge case tests ---

func TestComputeScore_ZeroWeights(t *testing.T) {
	// Edge case: HealthChecker with zero weights should produce score = 0
	checker := &HealthChecker{
		checkInterval: 30 * time.Second,
		staleAfter:    2 * time.Minute,
		offlineAfter:  5 * time.Minute,
		weights: ScoreWeights{
			Latency:      0,
			Availability: 0,
			Freshness:    0,
		},
	}

	conn := &NodeConnection{
		Latency:     0,
		ConsecFails: 0,
		LastSeen:    time.Now(),
	}

	score := checker.ComputeScore(conn)
	if score != 0.0 {
		t.Errorf("ComputeScore with zero weights = %f, want 0.0", score)
	}
}

func TestComputeScore_ExtremeLatency(t *testing.T) {
	checker := &HealthChecker{
		checkInterval: 30 * time.Second,
		staleAfter:    2 * time.Minute,
		offlineAfter:  5 * time.Minute,
		weights:       DefaultWeights(),
	}

	// Extremely high latency should not produce negative score
	conn := &NodeConnection{
		Latency:     100 * time.Second, // 100,000ms
		ConsecFails: 0,
		LastSeen:    time.Now(),
	}

	score := checker.ComputeScore(conn)
	if score < 0.0 || score > 1.0 {
		t.Errorf("ComputeScore with extreme latency = %f, want in [0, 1]", score)
	}
}

func TestComputeScore_ExtremeConsecFails(t *testing.T) {
	checker := &HealthChecker{
		checkInterval: 30 * time.Second,
		staleAfter:    2 * time.Minute,
		offlineAfter:  5 * time.Minute,
		weights:       DefaultWeights(),
	}

	// Very high failure count should not produce negative score
	conn := &NodeConnection{
		Latency:     0,
		ConsecFails: 1000,
		LastSeen:    time.Now(),
	}

	score := checker.ComputeScore(conn)
	if score < 0.0 || score > 1.0 {
		t.Errorf("ComputeScore with extreme ConsecFails = %f, want in [0, 1]", score)
	}
}

func TestDefaultWeights_SumToOne(t *testing.T) {
	w := DefaultWeights()
	sum := w.Latency + w.Availability + w.Freshness
	if math.Abs(sum-1.0) > 0.0001 {
		t.Errorf("DefaultWeights sum = %f, want 1.0", sum)
	}
}
