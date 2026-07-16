package api

import "strings"

// domainEndpoint describes a single connection endpoint (host + port) for a
// protocol on a node. The panel addresses nodes by their node IP/address
// directly, so these resolvers return empty and callers fall back to the
// node's own address. Kept as no-ops to preserve the fallback chain in the
// portal/profile generators.
type domainEndpoint struct {
	Host string
	Port int
}

// protocolDomainEndpoints returns the ordered endpoints for a protocol on a
// node. Domains were removed; always returns nil so callers use the node
// address instead.
func (s *Server) protocolDomainEndpoints(nodeID int64, protocol string) []domainEndpoint {
	return nil
}

// protocolPrimaryDomain returns the primary domain for a protocol on a node,
// or "" if none. Domains were removed, so this always returns "".
func (s *Server) protocolPrimaryDomain(nodeID int64, protocol string) string {
	return ""
}

// isTableNotExistError reports whether err indicates a missing SQL relation.
// Retained as a small shared helper.
func isTableNotExistError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "does not exist") ||
		strings.Contains(err.Error(), "undefined_table") ||
		strings.Contains(err.Error(), "42P01")
}
