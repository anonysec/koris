// Package nodeapi provides enhanced node-to-panel communication with
// connection health monitoring, heartbeat scoring, and automatic reconnection logic.
package nodeapi

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"
)

// ScoreWeights defines the weighting factors for health score computation.
type ScoreWeights struct {
	Latency      float64 // weight for response latency (default 0.4)
	Availability float64 // weight for uptime ratio (default 0.4)
	Freshness    float64 // weight for data freshness (default 0.2)
}

// DefaultWeights returns the default health score weights.
func DefaultWeights() ScoreWeights {
	return ScoreWeights{
		Latency:      0.4,
		Availability: 0.4,
		Freshness:    0.2,
	}
}

// NodeConnection represents a node's connection state and health metrics.
type NodeConnection struct {
	NodeID        int64
	LastSeen      time.Time
	LastPushOK    bool
	HealthScore   float64 // 0.0 (dead) to 1.0 (healthy)
	Latency       time.Duration
	ConsecFails   int
	ConsecSuccess int
	AgentUptime   int64
	QueueDepth    int
	RetryCount    int
}

// HealthChecker computes node health scores and manages state transitions.
type HealthChecker struct {
	checkInterval time.Duration
	staleAfter    time.Duration // mark stale after this
	offlineAfter  time.Duration // mark offline after this
	weights       ScoreWeights
}

// DefaultHealthThreshold is the health score below which a notification is sent.
const DefaultHealthThreshold = 0.4

// NodeConnectionManager manages all node connections and health state.
type NodeConnectionManager struct {
	mu              sync.RWMutex
	nodes           map[int64]*NodeConnection
	db              *sql.DB
	checker         *HealthChecker
	NotifyFn        func(msg string) // callback for sending health alerts (e.g. Telegram)
	HealthThreshold float64          // score below which an alert is triggered (default 0.4)
}

// PushPayload extends the standard node push with health metadata.
type PushPayload struct {
	NodeID        int64 `json:"node_id"`
	AgentUptime   int64 `json:"agent_uptime"`
	PushLatencyMs int64 `json:"push_latency_ms"`
	QueueDepth    int   `json:"queue_depth"`
	RetryCount    int   `json:"retry_count"`
}

// NewNodeConnectionManager creates a new manager.
func NewNodeConnectionManager(db *sql.DB) *NodeConnectionManager {
	return &NodeConnectionManager{
		nodes: make(map[int64]*NodeConnection),
		db:    db,
		checker: &HealthChecker{
			checkInterval: 30 * time.Second,
			staleAfter:    2 * time.Minute,
			offlineAfter:  5 * time.Minute,
			weights:       DefaultWeights(),
		},
		HealthThreshold: DefaultHealthThreshold,
	}
}

// HandlePush processes a push from a node agent, updating its health metrics.
func (m *NodeConnectionManager) HandlePush(nodeID int64, payload PushPayload) {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, exists := m.nodes[nodeID]
	if !exists {
		conn = &NodeConnection{NodeID: nodeID}
		m.nodes[nodeID] = conn
	}

	prevScore := conn.HealthScore

	conn.LastSeen = time.Now()
	conn.LastPushOK = true
	conn.Latency = time.Duration(payload.PushLatencyMs) * time.Millisecond
	conn.AgentUptime = payload.AgentUptime
	conn.QueueDepth = payload.QueueDepth
	conn.RetryCount = payload.RetryCount
	conn.ConsecSuccess++
	conn.ConsecFails = 0

	// Recompute health score
	conn.HealthScore = m.checker.ComputeScore(conn)

	// Notify on transition from healthy → unhealthy (score drops below threshold)
	threshold := m.HealthThreshold
	if threshold <= 0 {
		threshold = DefaultHealthThreshold
	}
	if conn.HealthScore < threshold && prevScore >= threshold && m.NotifyFn != nil {
		msg := fmt.Sprintf("⚠️ Node %d health score dropped to %.2f (threshold: %.2f)", nodeID, conn.HealthScore, threshold)
		go m.NotifyFn(msg)
	}
}

// GetHealth returns the health state for a node.
func (m *NodeConnectionManager) GetHealth(nodeID int64) (*NodeConnection, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conn, exists := m.nodes[nodeID]
	if !exists {
		return nil, false
	}
	// Return a copy
	copy := *conn
	return &copy, true
}

// GetAllHealth returns health state for all tracked nodes.
func (m *NodeConnectionManager) GetAllHealth() []NodeConnection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]NodeConnection, 0, len(m.nodes))
	for _, conn := range m.nodes {
		result = append(result, *conn)
	}
	return result
}

// StartMonitor starts the background goroutine that checks for stale/offline nodes.
func (m *NodeConnectionManager) StartMonitor(ctx <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(m.checker.checkInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx:
				return
			case <-ticker.C:
				m.checkStaleNodes()
			}
		}
	}()
}

// checkStaleNodes transitions nodes: online→stale (2min) →offline (5min).
func (m *NodeConnectionManager) checkStaleNodes() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for _, conn := range m.nodes {
		elapsed := now.Sub(conn.LastSeen)

		if elapsed > m.checker.offlineAfter {
			conn.HealthScore = 0.0
			conn.LastPushOK = false
		} else if elapsed > m.checker.staleAfter {
			// Degrade health score linearly between stale and offline
			fraction := float64(elapsed-m.checker.staleAfter) / float64(m.checker.offlineAfter-m.checker.staleAfter)
			conn.HealthScore = math.Max(0, conn.HealthScore*(1.0-fraction*0.5))
		}
	}
}

// ComputeScore calculates a health score [0.0, 1.0] based on weighted metrics.
func (h *HealthChecker) ComputeScore(conn *NodeConnection) float64 {
	weights := h.weights

	// Latency score: 0ms=1.0, 500ms=0.5, 1000ms+=0.0
	latencyMs := float64(conn.Latency.Milliseconds())
	latencyScore := math.Max(0, 1.0-latencyMs/1000.0)

	// Availability score: based on consecutive successes vs failures
	availScore := 1.0
	if conn.ConsecFails > 0 {
		availScore = math.Max(0, 1.0-float64(conn.ConsecFails)*0.2)
	}

	// Freshness score: how recently the node pushed
	elapsed := time.Since(conn.LastSeen)
	freshnessScore := 1.0
	if elapsed > 30*time.Second {
		freshnessScore = math.Max(0, 1.0-elapsed.Seconds()/300.0)
	}

	score := latencyScore*weights.Latency +
		availScore*weights.Availability +
		freshnessScore*weights.Freshness

	// Clamp to [0, 1]
	if score > 1.0 {
		score = 1.0
	}
	if score < 0.0 {
		score = 0.0
	}

	return score
}

// ReconnectionPolicy defines exponential backoff for node agents.
type ReconnectionPolicy struct {
	InitialDelay time.Duration // 2s
	MaxDelay     time.Duration // 60s
	Multiplier   float64       // 2.0
	Jitter       float64       // 0.1 (10% jitter)
}

// DefaultReconnectionPolicy returns the standard backoff configuration.
func DefaultReconnectionPolicy() ReconnectionPolicy {
	return ReconnectionPolicy{
		InitialDelay: 2 * time.Second,
		MaxDelay:     60 * time.Second,
		Multiplier:   2.0,
		Jitter:       0.1,
	}
}

// NextDelay computes the delay for the given attempt number (0-based).
func (p *ReconnectionPolicy) NextDelay(attempt int) time.Duration {
	delay := float64(p.InitialDelay)
	for i := 0; i < attempt; i++ {
		delay *= p.Multiplier
	}

	if delay > float64(p.MaxDelay) {
		delay = float64(p.MaxDelay)
	}

	// Apply jitter: ±jitter%
	jitterRange := delay * p.Jitter
	jitterOffset := (rand.Float64()*2 - 1) * jitterRange
	delay += jitterOffset

	if delay < 0 {
		delay = float64(p.InitialDelay)
	}

	return time.Duration(delay)
}

func init() {
	_ = log.Printf // Avoid unused import if no logging statements
}
