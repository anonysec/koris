//go:build lite

package api

import "net/http"

// registerSettingsRoutes is a no-op in the lite build.
func (s *Server) registerSettingsRoutes(mux *http.ServeMux) {}
