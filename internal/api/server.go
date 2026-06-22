package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"KorisLite/internal/auth"
	"KorisLite/internal/config"
)

type Server struct {
	DB     *sql.DB
	Config config.Config
	Auth   *auth.Service
}

func New(db *sql.DB, cfg config.Config) *Server {
	return &Server{
		DB:     db,
		Config: cfg,
		Auth:   &auth.Service{DB: db, Secret: cfg.SessionSecret},
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// Public
	mux.HandleFunc("/api/health", s.health)
	mux.HandleFunc("/api/auth/login", s.login)

	// Admin (protected)
	mux.HandleFunc("/api/admin/logout", s.requireAdmin(s.logout))
	mux.HandleFunc("/api/admin/me", s.requireAdmin(s.adminMe))
	mux.HandleFunc("/api/admin/settings", s.requireAdmin(s.settings))
	mux.HandleFunc("/api/admin/customers", s.requireAdmin(s.customers))
	mux.HandleFunc("/api/admin/customers/", s.requireAdmin(s.customerByID))
	mux.HandleFunc("/api/admin/nodes", s.requireAdmin(s.nodes))
	mux.HandleFunc("/api/admin/nodes/", s.requireAdmin(s.nodeByID))
	mux.HandleFunc("/api/admin/protocols", s.requireAdmin(s.protocols))

	// Node agent API (token auth)
	mux.HandleFunc("/api/node/push", s.nodePush)
	mux.HandleFunc("/api/node/tasks/poll", s.nodeTaskPoll)

	// SPA handlers
	mux.Handle("/dashboard/", http.StripPrefix("/dashboard/", http.FileServer(http.Dir(s.Config.AdminWebDir))))
	mux.Handle("/portal/", http.StripPrefix("/portal/", http.FileServer(http.Dir(s.Config.PortalWebDir))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/dashboard/", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	})

	return mux
}

// ── Middleware ──

func (s *Server) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := auth.GetSessionFromRequest(r)
		if token == "" {
			writeJSON(w, http.StatusUnauthorized, M{"ok": false, "error": "unauthorized"})
			return
		}
		sess, err := s.Auth.ValidateSession(token)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, M{"ok": false, "error": "session_expired"})
			return
		}
		r.Header.Set("X-Admin-Username", sess.Username)
		r.Header.Set("X-Admin-Role", sess.Role)
		next(w, r)
	}
}

// ── Helpers ──

type M map[string]any

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func ok(w http.ResponseWriter, v any) {
	writeJSON(w, http.StatusOK, v)
}

func limitBody(r *http.Request) {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20) // 1MB
}

func (s *Server) currentAdmin(r *http.Request) string {
	return r.Header.Get("X-Admin-Username")
}

func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}

func init() {
	_ = log.Printf
}
