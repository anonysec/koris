package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"KorisPanel/panel/internal/grpcclient"
	"KorisPanel/panel/internal/noderegistry"
)

// dispatchKnodeCores handles routing for /api/admin/knode/nodes/{nodeID}/cores[/{coreType}/{action}].
// Called from handleKnodeNodeByID when the path segment after the ID starts with "cores".
func (s *Server) dispatchKnodeCores(w http.ResponseWriter, r *http.Request, nodeID int64) {
	// Parse remaining path: /api/admin/knode/nodes/{id}/cores/{coreType}/{action}
	rest := strings.TrimPrefix(r.URL.Path, "/api/admin/knode/nodes/")
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	// parts: ["{id}", "cores", ...]

	// GET /api/admin/knode/nodes/{id}/cores — list all cores with status
	if len(parts) <= 2 {
		if r.Method == http.MethodGet {
			s.listKnodeCores(w, r, nodeID)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	coreType := parts[2]
	action := ""
	if len(parts) >= 4 {
		action = parts[3]
	}

	switch {
	case action == "config" && r.Method == http.MethodGet:
		s.getKnodeCoreConfig(w, r, nodeID, coreType)
	case action == "enable" && r.Method == http.MethodPost:
		s.enableKnodeCore(w, r, nodeID, coreType)
	case action == "disable" && r.Method == http.MethodPost:
		s.disableKnodeCore(w, r, nodeID, coreType)
	case action == "restart" && r.Method == http.MethodPost:
		s.restartKnodeCore(w, r, nodeID, coreType)
	default:
		writeJSONCode(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not_found"})
	}
}

// listKnodeCores handles GET /api/admin/knode/nodes/{nodeID}/cores.
// Always returns all 5 default protocols. Enriched with live gRPC data when available.
func (s *Server) listKnodeCores(w http.ResponseWriter, r *http.Request, nodeID int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	type coreResponse struct {
		Type           string `json:"type"`
		State          string `json:"state"`
		ActiveSessions int    `json:"active_sessions"`
		PID            int    `json:"pid"`
		Port           int    `json:"port"`
		LastError      string `json:"last_error,omitempty"`
	}

	// Default protocols — always show all 5
	defaultProtocols := []struct {
		protocol string
		port     int
	}{
		{"openvpn", 1194},
		{"wireguard", 51820},
		{"l2tp", 1701},
		{"ikev2", 500},
		{"ssh", 2222},
	}

	// Try to get live statuses from knode via gRPC
	liveMap := make(map[string]grpcclient.CoreStatus)
	if s.CoreMgr != nil {
		statuses, err := s.CoreMgr.AllCoreStatuses(ctx, nodeID)
		if err != nil {
			log.Printf("[knode-cores] AllCoreStatuses failed for node %d: %v", nodeID, err)
		} else {
			for _, cs := range statuses {
				liveMap[cs.Type] = cs
			}
		}
	}

	// Build response: merge live data with defaults
	cores := make([]coreResponse, 0, 5)
	for _, dp := range defaultProtocols {
		cr := coreResponse{
			Type:  dp.protocol,
			State: "stopped",
			Port:  dp.port,
		}

		if live, ok := liveMap[dp.protocol]; ok {
			cr.State = live.State
			cr.ActiveSessions = live.ActiveSessions
			cr.PID = live.PID
		}

		// Check DB for saved status (fallback when no live data)
		if _, ok := liveMap[dp.protocol]; !ok {
			var dbStatus string
			if err := s.DB.QueryRowContext(ctx, `SELECT status FROM node_services WHERE node_id=$1 AND service=$2`, nodeID, dp.protocol).Scan(&dbStatus); err == nil && dbStatus != "" {
				cr.State = dbStatus
			}
		}

		// Check DB for saved port override
		var savedPort int
		if err := s.DB.QueryRowContext(ctx, `SELECT port FROM node_vpn_configs WHERE node_id=$1 AND protocol=$2`, nodeID, dp.protocol).Scan(&savedPort); err == nil && savedPort > 0 {
			cr.Port = savedPort
		}

		// Enrich with last_error if crashed/error
		if cr.State == "crashed" || cr.State == "error" {
			var lastErr string
			if err := s.DB.QueryRowContext(ctx, `SELECT COALESCE(last_error, '') FROM node_services WHERE node_id=$1 AND service=$2`, nodeID, dp.protocol).Scan(&lastErr); err == nil {
				cr.LastError = lastErr
			}
		}

		cores = append(cores, cr)
	}

	writeJSON(w, map[string]any{"ok": true, "cores": cores})
}

// enableCoreRequest is the JSON body for enabling a core.
type enableCoreRequest struct {
	ListenPort int             `json:"listen_port"`
	Extra      json.RawMessage `json:"extra"`
}

// enableKnodeCore handles POST /api/admin/knode/nodes/{nodeID}/cores/{coreType}/enable.
func (s *Server) enableKnodeCore(w http.ResponseWriter, r *http.Request, nodeID int64, coreType string) {
	if s.CoreMgr == nil {
		writeJSONCode(w, http.StatusServiceUnavailable, map[string]any{"ok": false, "error": "grpc_not_configured"})
		return
	}

	limitBody(w, r, maxJSONBody)
	var in enableCoreRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "bad_json"})
		return
	}

	if in.ListenPort <= 0 || in.ListenPort > 65535 {
		writeJSONCode(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid_port"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	// For IKEv2, inject the node's stored domain into Extra_Config
	extra := in.Extra
	if coreType == "ikev2" {
		extra = s.injectDomainForIKEv2(ctx, nodeID, extra)
	}

	if err := s.CoreMgr.EnableCore(ctx, nodeID, coreType, in.ListenPort, extra); err != nil {
		log.Printf("[knode-cores] EnableCore failed for node %d core %s: %v", nodeID, coreType, err)
		writeJSONCode(w, http.StatusBadGateway, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	// Update local node_vpn_configs to keep panel state in sync
	extraStr := ""
	if len(in.Extra) > 0 {
		extraStr = string(in.Extra)
	}
	_, _ = s.DB.ExecContext(ctx, `INSERT INTO node_vpn_configs(node_id, protocol, enabled, port, extra_json)
		VALUES($1, $2, 1, $3, $4)
		ON CONFLICT (node_id, protocol) DO UPDATE SET enabled=1, port=EXCLUDED.port, extra_json=EXCLUDED.extra_json`,
		nodeID, coreType, in.ListenPort, nullString(extraStr))

	writeJSON(w, map[string]any{"ok": true})
}

// disableKnodeCore handles POST /api/admin/knode/nodes/{nodeID}/cores/{coreType}/disable.
func (s *Server) disableKnodeCore(w http.ResponseWriter, r *http.Request, nodeID int64, coreType string) {
	if s.CoreMgr == nil {
		writeJSONCode(w, http.StatusServiceUnavailable, map[string]any{"ok": false, "error": "grpc_not_configured"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := s.CoreMgr.DisableCore(ctx, nodeID, coreType); err != nil {
		log.Printf("[knode-cores] DisableCore failed for node %d core %s: %v", nodeID, coreType, err)
		writeJSONCode(w, http.StatusBadGateway, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	_, _ = s.DB.ExecContext(ctx, `UPDATE node_vpn_configs SET enabled=0 WHERE node_id=$1 AND protocol=$2`, nodeID, coreType)
	writeJSON(w, map[string]any{"ok": true})
}

// restartKnodeCore handles POST /api/admin/knode/nodes/{nodeID}/cores/{coreType}/restart.
func (s *Server) restartKnodeCore(w http.ResponseWriter, r *http.Request, nodeID int64, coreType string) {
	if s.CoreMgr == nil {
		writeJSONCode(w, http.StatusServiceUnavailable, map[string]any{"ok": false, "error": "grpc_not_configured"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	if err := s.CoreMgr.RestartCore(ctx, nodeID, coreType); err != nil {
		log.Printf("[knode-cores] RestartCore failed for node %d core %s: %v", nodeID, coreType, err)
		writeJSONCode(w, http.StatusBadGateway, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	writeJSON(w, map[string]any{"ok": true})
}

// getKnodeCoreConfig handles GET /api/admin/knode/nodes/{nodeID}/cores/{coreType}/config.
func (s *Server) getKnodeCoreConfig(w http.ResponseWriter, r *http.Request, nodeID int64, coreType string) {
	ctx := r.Context()

	var port int
	var network string
	var extraJSON []byte
	var enabled bool

	err := s.DB.QueryRowContext(ctx,
		`SELECT enabled, port, COALESCE(network,''), COALESCE(extra_json,'') FROM node_vpn_configs WHERE node_id=$1 AND protocol=$2`,
		nodeID, coreType,
	).Scan(&enabled, &port, &network, &extraJSON)
	if err != nil {
		writeJSONCode(w, http.StatusNotFound, map[string]any{"ok": false, "error": "config_not_found"})
		return
	}

	var extra map[string]any
	if len(extraJSON) > 0 {
		_ = json.Unmarshal(extraJSON, &extra)
	}
	if extra == nil {
		extra = map[string]any{}
	}

	writeJSON(w, map[string]any{
		"ok": true,
		"config": map[string]any{
			"type":        coreType,
			"enabled":     enabled,
			"listen_port": port,
			"network":     network,
			"extra":       extra,
		},
	})
}

// injectDomainForIKEv2 reads the node's stored domain from knode_connections and
// injects it into the Extra_Config JSON for the IKEv2 core.
func (s *Server) injectDomainForIKEv2(ctx context.Context, nodeID int64, extra json.RawMessage) json.RawMessage {
	if s.NodeRegistry == nil {
		return extra
	}

	reg, ok := s.NodeRegistry.(*noderegistry.DBRegistry)
	if !ok {
		return extra
	}

	domain, err := reg.GetDomain(ctx, nodeID)
	if err != nil || domain == "" {
		return extra
	}

	var cfg map[string]any
	if len(extra) > 0 {
		if err := json.Unmarshal(extra, &cfg); err != nil {
			cfg = make(map[string]any)
		}
	} else {
		cfg = make(map[string]any)
	}

	if _, exists := cfg["domain"]; !exists {
		cfg["domain"] = domain
	}

	result, err := json.Marshal(cfg)
	if err != nil {
		return extra
	}
	return result
}
