package api

import (
	"net/http"
	"os"
	"time"
)

// internalWorkers returns the worker process list for CLI consumption.
// It does not require authentication since it is only exposed on the
// Unix socket or localhost internal listener.
// GET /internal/workers
//
// Currently returns the main process as a single worker entry.
// Multi-worker support (Task Group 3) will extend this to report
// all spawned worker processes.
func (s *Server) internalWorkers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}

	uptimeSeconds := int64(time.Since(processStartTime).Seconds())

	type workerEntry struct {
		ID            int    `json:"id"`
		PID           int    `json:"pid"`
		Status        string `json:"status"`
		UptimeSeconds int64  `json:"uptime_seconds"`
		Restarts      int    `json:"restarts"`
	}

	workers := []workerEntry{
		{
			ID:            1,
			PID:           os.Getpid(),
			Status:        "running",
			UptimeSeconds: uptimeSeconds,
			Restarts:      0,
		},
	}

	writeJSON(w, map[string]any{
		"ok":      true,
		"workers": workers,
	})
}
