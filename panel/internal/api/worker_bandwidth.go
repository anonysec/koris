package api

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"
)

// workerBWNotifyTracker tracks daily notification state for bandwidth quotas
// on the nodes table (bandwidth_quota_gb / bandwidth_used_bytes columns).
// Prevents duplicate notifications within the same day for each node.
var (
	workerBWNotifyTracker   = make(map[int64]workerBWNotifyState)
	workerBWNotifyTrackerMu sync.Mutex
)

type workerBWNotifyState struct {
	Level int       // 1 = warning (>=80%), 2 = critical (>=100%)
	Date  time.Time // date of last notification
}

// CheckNodeBandwidthQuotas checks all nodes with bandwidth_quota_gb configured
// in the nodes table and sends notifications when thresholds are exceeded.
// Designed to be called from the background worker on each tick.
func CheckNodeBandwidthQuotas(db *sql.DB, notify func(string)) {
	rows, err := db.Query(`
		SELECT id, name, bandwidth_quota_gb, bandwidth_used_bytes
		FROM nodes
		WHERE bandwidth_quota_gb IS NOT NULL AND bandwidth_quota_gb > 0
	`)
	if err != nil {
		log.Printf("[bandwidth] check query error: %v", err)
		return
	}
	defer rows.Close()

	workerBWNotifyTrackerMu.Lock()
	defer workerBWNotifyTrackerMu.Unlock()

	today := time.Now().UTC().Truncate(24 * time.Hour)

	for rows.Next() {
		var nodeID int64
		var nodeName string
		var quotaGB int
		var usedBytes int64

		if err := rows.Scan(&nodeID, &nodeName, &quotaGB, &usedBytes); err != nil {
			log.Printf("[bandwidth] scan error: %v", err)
			continue
		}

		quotaBytes := int64(quotaGB) * 1073741824
		if quotaBytes <= 0 {
			continue
		}
		percentage := int(float64(usedBytes) / float64(quotaBytes) * 100)

		state := workerBWNotifyTracker[nodeID]
		alreadyNotifiedToday := state.Date.Equal(today)

		if percentage >= 100 && (!alreadyNotifiedToday || state.Level < 2) {
			msg := fmt.Sprintf("🚨 *Bandwidth Quota Exceeded*\nNode: `%s`\nUsage: %d%% (%d GB / %d GB)\nMonthly quota reached!",
				nodeName, percentage, usedBytes/1073741824, quotaGB)
			if notify != nil {
				notify(msg)
			}
			workerBWNotifyTracker[nodeID] = workerBWNotifyState{Level: 2, Date: today}
			log.Printf("[bandwidth] critical: node %s at %d%% of quota — notified admin", nodeName, percentage)
		} else if percentage >= 80 && percentage < 100 && (!alreadyNotifiedToday || state.Level < 1) {
			msg := fmt.Sprintf("⚠️ *Bandwidth Quota Warning*\nNode: `%s`\nUsage: %d%% (%d GB / %d GB)\nApproaching monthly quota.",
				nodeName, percentage, usedBytes/1073741824, quotaGB)
			if notify != nil {
				notify(msg)
			}
			workerBWNotifyTracker[nodeID] = workerBWNotifyState{Level: 1, Date: today}
			log.Printf("[bandwidth] warning: node %s at %d%% of quota — notified admin", nodeName, percentage)
		}
	}
}

// ResetMonthlyNodeBandwidth resets bandwidth counters for all nodes with quotas
// on the 1st of each month. Designed to be called from the background worker.
func ResetMonthlyNodeBandwidth(db *sql.DB) {
	now := time.Now().UTC()
	if now.Day() != 1 {
		return
	}

	result, err := db.Exec(`
		UPDATE nodes
		SET bandwidth_used_bytes = 0, bandwidth_reset_at = NOW()
		WHERE bandwidth_quota_gb IS NOT NULL
	`)
	if err != nil {
		log.Printf("[bandwidth] monthly reset error: %v", err)
		return
	}

	affected, _ := result.RowsAffected()
	if affected > 0 {
		log.Printf("[bandwidth] monthly reset: cleared bandwidth counters for %d nodes", affected)

		// Clear notification tracker so alerts can fire again next cycle
		workerBWNotifyTrackerMu.Lock()
		workerBWNotifyTracker = make(map[int64]workerBWNotifyState)
		workerBWNotifyTrackerMu.Unlock()
	}
}
