package api

import (
	"database/sql"
	"net/http"
	"strings"
	"time"
)

// getCustomerConnections handles GET /api/customers/:id/connections.
// Returns currently active VPN sessions for the given customer by querying
// the radacct table for sessions where acctstoptime IS NULL.
func (s *Server) getCustomerConnections(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}

	// Look up the customer's username from their ID
	var username string
	err := s.DB.QueryRow(`SELECT username FROM customers WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&username)
	if err == sql.ErrNoRows {
		writeJSONCode(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not_found"})
		return
	}
	if err != nil {
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error"})
		return
	}

	// Query active sessions from radacct
	rows, err := s.DB.Query(`
		SELECT
			COALESCE(r.framedipaddress, '') AS ip,
			COALESCE(r.callingstationid, '') AS calling_station_id,
			COALESCE(r.calledstationid, '') AS called_station_id,
			COALESCE(r.connectinfo_start, '') AS connect_info,
			r.acctstarttime
		FROM radacct r
		WHERE r.username = $1 AND r.acctstoptime IS NULL
		ORDER BY r.acctstarttime DESC
	`, username)
	if err != nil {
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": "db_error"})
		return
	}
	defer rows.Close()

	type connectionResp struct {
		IP          string `json:"ip"`
		Device      string `json:"device"`
		UserAgent   string `json:"user_agent"`
		ConnectedAt string `json:"connected_at"`
		Protocol    string `json:"protocol"`
	}

	connections := []connectionResp{}
	for rows.Next() {
		var ip, callingStationID, calledStationID, connectInfo string
		var startTime sql.NullTime
		if err := rows.Scan(&ip, &callingStationID, &calledStationID, &connectInfo, &startTime); err != nil {
			continue
		}

		connectedAt := ""
		if startTime.Valid {
			connectedAt = startTime.Time.UTC().Format(time.RFC3339)
		}

		protocol := normalizeProtocol(calledStationID)
		device := enrichDevice(callingStationID)
		userAgent := connectInfo

		connections = append(connections, connectionResp{
			IP:          ip,
			Device:      device,
			UserAgent:   userAgent,
			ConnectedAt: connectedAt,
			Protocol:    protocol,
		})
	}

	writeJSON(w, map[string]any{
		"ok":          true,
		"connections": connections,
	})
}

// enrichDevice attempts to derive a human-friendly device name from the
// calling_station_id value in RADIUS accounting. This field typically
// contains a MAC address or client identifier string.
func enrichDevice(callingStationID string) string {
	if callingStationID == "" {
		return ""
	}

	lower := strings.ToLower(callingStationID)

	// Check for known device identifiers embedded in the calling station ID
	switch {
	case strings.Contains(lower, "iphone"):
		return "iPhone"
	case strings.Contains(lower, "ipad"):
		return "iPad"
	case strings.Contains(lower, "android"):
		return "Android"
	case strings.Contains(lower, "mac") || strings.Contains(lower, "macos"):
		return "macOS"
	case strings.Contains(lower, "windows"):
		return "Windows"
	case strings.Contains(lower, "linux"):
		return "Linux"
	}

	// If it looks like a MAC address (XX:XX:XX:XX:XX:XX or XX-XX-XX-XX-XX-XX),
	// attempt OUI-based vendor identification
	normalized := strings.ReplaceAll(callingStationID, "-", ":")
	parts := strings.Split(normalized, ":")
	if len(parts) == 6 && len(parts[0]) == 2 {
		vendor := macVendorHint(strings.ToUpper(strings.Join(parts[:3], ":")))
		if vendor != "" {
			return vendor
		}
		// Return the MAC address itself as the device identifier
		return normalized
	}

	// Return the raw value if we can't determine the device
	return callingStationID
}

// macVendorHint returns a human-friendly device name based on the MAC OUI prefix.
// This is a simplified lookup of common device manufacturers.
func macVendorHint(oui string) string {
	// Common OUI prefixes for popular device manufacturers
	vendors := map[string]string{
		"00:50:E4": "Apple",
		"3C:06:30": "Apple",
		"A4:83:E7": "Apple",
		"F0:18:98": "Apple",
		"AC:DE:48": "Apple",
		"DC:A9:04": "Apple",
		"70:3E:AC": "Apple",
		"BC:52:B7": "Apple",
		"C8:69:CD": "Apple",
		"14:7D:DA": "Apple",
		"00:1A:11": "Google",
		"F4:F5:D8": "Google",
		"54:60:09": "Google",
		"30:FD:38": "Google",
		"94:EB:2C": "Google",
		"48:2C:A0": "Huawei",
		"88:66:A5": "Huawei",
		"CC:46:D6": "Samsung",
		"8C:F5:A3": "Samsung",
		"B4:3A:28": "Samsung",
		"00:26:5A": "D-Link",
		"84:C9:B2": "D-Link",
		"B0:BE:76": "TP-Link",
		"50:C7:BF": "TP-Link",
	}

	if v, ok := vendors[oui]; ok {
		return v
	}
	return ""
}
