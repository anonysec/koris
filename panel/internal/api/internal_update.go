package api

import (
	"log"
	"net/http"

	"KorisPanel/panel/internal/updater"
)

// internalUpdateCheck checks for available panel updates.
// It does not require authentication since it is only exposed on the
// Unix socket or localhost internal listener.
// GET /internal/update/check
func (s *Server) internalUpdateCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.Config.ReleaseURL == "" {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{
			"ok":    false,
			"error": "updates_disabled",
		})
		return
	}

	u := updater.New(s.Config.Version, s.Config.ReleaseURL, "")
	info, err := u.Check()
	if err != nil {
		log.Printf("[update] internal check failed: %v", err)
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": "check_failed",
		})
		return
	}

	writeJSON(w, map[string]any{
		"ok":     true,
		"update": info,
	})
}

// internalUpdateApply applies an available panel update.
// It does not require authentication since it is only exposed on the
// Unix socket or localhost internal listener.
// POST /internal/update/apply
func (s *Server) internalUpdateApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.Config.ReleaseURL == "" {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{
			"ok":    false,
			"error": "updates_disabled",
		})
		return
	}

	u := updater.New(s.Config.Version, s.Config.ReleaseURL, "")

	info, err := u.Check()
	if err != nil {
		log.Printf("[update] internal check before apply failed: %v", err)
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": "check_failed",
		})
		return
	}

	if !info.Available {
		writeJSON(w, map[string]any{
			"ok":    false,
			"error": "no_update_available",
		})
		return
	}

	progressFn := func(stage string, pct float64) {
		log.Printf("[update] %s: %.0f%%", stage, pct*100)
	}

	if err := u.Apply(info, progressFn); err != nil {
		log.Printf("[update] internal apply failed: %v", err)
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": "apply_failed",
		})
		return
	}

	writeJSON(w, map[string]any{
		"ok":      true,
		"message": "update applied, restarting...",
	})
}
