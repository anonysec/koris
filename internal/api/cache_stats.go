package api

import "net/http"

// cacheStats returns current cache performance metrics.
// GET /api/admin/cache-stats
func (s *Server) cacheStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var hits, misses, evictions int64
	var size int
	var hitRate float64

	if s.Cache != nil {
		stats := s.Cache.Stats()
		hits = stats.Hits
		misses = stats.Misses
		size = stats.Size
		evictions = stats.Evictions
		hitRate = stats.HitRate
	}

	writeJSON(w, map[string]any{
		"ok": true,
		"cache": map[string]any{
			"hits":      hits,
			"misses":    misses,
			"size":      size,
			"evictions": evictions,
			"hit_rate":  hitRate,
		},
	})
}
