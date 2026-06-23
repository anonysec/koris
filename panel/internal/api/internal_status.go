package api

import (
	"net/http"
	"time"
)

// internalStatus returns the panel's internal status for CLI consumption.
// It does not require authentication since it is only exposed on the
// Unix socket or localhost internal listener.
// GET /internal/status
func (s *Server) internalStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}

	// Uptime in seconds since process start.
	uptimeSeconds := time.Since(processStartTime).Seconds()

	// DB pool stats from sql.DB.
	dbStats := s.DB.Stats()

	// Node counts by status.
	var online, stale, offline int
	_ = s.DB.QueryRow(`SELECT COUNT(*) FROM nodes WHERE status='online'`).Scan(&online)
	_ = s.DB.QueryRow(`SELECT COUNT(*) FROM nodes WHERE status='stale'`).Scan(&stale)
	_ = s.DB.QueryRow(`SELECT COUNT(*) FROM nodes WHERE status='offline'`).Scan(&offline)

	writeJSON(w, map[string]any{
		"ok":             true,
		"version":        s.Config.Version,
		"uptime_seconds": uptimeSeconds,
		"workers":        1, // Single worker by default; multi-worker will override this
		"db_pool": map[string]any{
			"max_open": dbStats.MaxOpenConnections,
			"open":     dbStats.OpenConnections,
			"in_use":   dbStats.InUse,
			"idle":     dbStats.Idle,
		},
		"nodes": map[string]any{
			"online":  online,
			"stale":   stale,
			"offline": offline,
		},
	})
}
