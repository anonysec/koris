package protocols

import (
	"database/sql"
	"encoding/json"
	"log"
)

// PlanAllowedProtocols retrieves the list of allowed protocols for a given plan.
// Returns nil if all protocols are allowed (no restrictions).
func PlanAllowedProtocols(db *sql.DB, planID int64) []string {
	var raw []byte
	err := db.QueryRow(`SELECT plan_protocols FROM plans WHERE id = ?`, planID).Scan(&raw)
	if err != nil || len(raw) == 0 || string(raw) == "null" {
		return nil // no restriction
	}

	var allowed []string
	if err := json.Unmarshal(raw, &allowed); err != nil {
		log.Printf("[protocols] failed to parse plan_protocols for plan %d: %v", planID, err)
		return nil
	}
	return allowed
}

// IsProtocolAllowedByPlan checks whether a specific protocol is allowed for a plan.
// If the plan has no restrictions (plan_protocols is NULL), all protocols are allowed.
func IsProtocolAllowedByPlan(db *sql.DB, planID int64, protocolName string) bool {
	allowed := PlanAllowedProtocols(db, planID)
	if allowed == nil {
		return true // no restriction
	}
	for _, p := range allowed {
		if p == protocolName {
			return true
		}
	}
	return false
}

// FilterProtocolsByPlan takes a list of protocol names and returns only those
// allowed by the given plan. If the plan has no restrictions, all are returned.
func FilterProtocolsByPlan(db *sql.DB, planID int64, protocols []string) []string {
	allowed := PlanAllowedProtocols(db, planID)
	if allowed == nil {
		return protocols
	}
	allowedSet := make(map[string]bool, len(allowed))
	for _, p := range allowed {
		allowedSet[p] = true
	}
	var filtered []string
	for _, p := range protocols {
		if allowedSet[p] {
			filtered = append(filtered, p)
		}
	}
	return filtered
}
