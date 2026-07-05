package api

import (
	"net/http"
	"os"
	"runtime"
	"time"
)

// internalWorkers returns the worker process list for CLI consumption.
// Requires admin authentication because it is exposed on the public HTTP listener.
// GET /internal/workers
//
// Reports the main process, goroutine count, memory stats, and Go runtime version
// as a single worker entry. Multi-worker support (Task Group 3) will extend this
// to report all spawned worker processes.
func (s *Server) internalWorkers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	uptimeSeconds := int64(time.Since(processStartTime).Seconds())

	type workerEntry struct {
		ID             int    `json:"id"`
		PID            int    `json:"pid"`
		Status         string `json:"status"`
		UptimeSeconds  int64  `json:"uptime_seconds"`
		Restarts       int    `json:"restarts"`
		Goroutines     int    `json:"goroutines"`
		MemAllocMiB    int    `json:"mem_alloc_mib"`
		MemSysMiB      int    `json:"mem_sys_mib"`
		GoVersion      string `json:"go_version"`
		NumCPU         int    `json:"num_cpu"`
	}

	workers := []workerEntry{
		{
			ID:            1,
			PID:           os.Getpid(),
			Status:        "running",
			UptimeSeconds: uptimeSeconds,
			Restarts:      0,
			Goroutines:    runtime.NumGoroutine(),
			MemAllocMiB:   int(m.Alloc / 1024 / 1024),
			MemSysMiB:     int(m.Sys / 1024 / 1024),
			GoVersion:     runtime.Version(),
			NumCPU:        runtime.NumCPU(),
		},
	}

	writeJSON(w, map[string]any{
		"ok":       true,
		"workers":  workers,
		"hostname": func() string { h, _ := os.Hostname(); return h }(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

