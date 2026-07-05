package api

import (
	"database/sql"
	"log"
	"strings"
)

// domainEndpoint represents a single active domain binding for a protocol on a node.
type domainEndpoint struct {
	DomainName string
	Position   int
}

// protocolDomainEndpoints queries vpn_protocol_bindings + vpn_domains to get active domains
// for a given node and protocol, ordered by position ASC.
// Returns only domains with status 'active' — blocked/retired domains are skipped.
// When the primary (position 1) domain is blocked, the next active domain is promoted.
// Returns nil if no active domain bindings exist (caller should fall back to node IP).
func (s *Server) protocolDomainEndpoints(nodeID int64, protocol string) []domainEndpoint {
	if nodeID <= 0 {
		return nil
	}

	rows, err := s.DB.Query(`
		SELECT d.name, pb.position
		FROM vpn_protocol_bindings pb
		JOIN vpn_domains d ON d.id = pb.domain_id
		WHERE pb.node_id = $1 AND pb.protocol = $2 AND d.status = 'active'
		ORDER BY pb.position ASC
	`, nodeID, protocol)
	if err != nil {
		// Table may not exist yet (pre-migration) — silently return nil for fallback
		if !isTableNotExistError(err) {
			log.Printf("[config-domain-resolver] query error for node=%d protocol=%s: %v", nodeID, protocol, err)
		}
		return nil
	}
	defer rows.Close()

	var endpoints []domainEndpoint
	for rows.Next() {
		var ep domainEndpoint
		if err := rows.Scan(&ep.DomainName, &ep.Position); err != nil {
			log.Printf("[config-domain-resolver] scan error: %v", err)
			continue
		}
		endpoints = append(endpoints, ep)
	}
	return endpoints
}

// protocolPrimaryDomain returns the primary (first active) domain for a protocol on a node.
// Returns empty string if no active domain bindings exist.
func (s *Server) protocolPrimaryDomain(nodeID int64, protocol string) string {
	if nodeID <= 0 {
		return ""
	}

	var name string
	err := s.DB.QueryRow(`
		SELECT d.name
		FROM vpn_protocol_bindings pb
		JOIN vpn_domains d ON d.id = pb.domain_id
		WHERE pb.node_id = $1 AND pb.protocol = $2 AND d.status = 'active'
		ORDER BY pb.position ASC
		LIMIT 1
	`, nodeID, protocol).Scan(&name)
	if err != nil {
		if err != sql.ErrNoRows && !isTableNotExistError(err) {
			log.Printf("[config-domain-resolver] primary domain query error for node=%d protocol=%s: %v", nodeID, protocol, err)
		}
		return ""
	}
	return name
}

// isTableNotExistError checks if a database error indicates a missing table.
// This handles the case where migration 074 hasn't been applied yet.
func isTableNotExistError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "does not exist") || strings.Contains(msg, "relation") && strings.Contains(msg, "does not exist")
}
