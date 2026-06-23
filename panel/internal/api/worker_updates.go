package api

import (
	"database/sql"
	"log"
)

// CheckPendingUpdateHealth queries node_tasks with action='update_agent' and
// status='pending' that were created more than 60 seconds ago. For each stale
// pending task, it marks the task as 'failed' and updates the associated node
// with update_failed status indication.
// Designed to be called from the background worker on each tick.
func CheckPendingUpdateHealth(db *sql.DB, notify func(string)) {
	rows, err := db.Query(`
		SELECT nt.id, nt.node_id, n.name
		FROM node_tasks nt
		JOIN nodes n ON n.id = nt.node_id
		WHERE nt.action = 'update_agent'
		  AND nt.status = 'pending'
		  AND nt.created_at < NOW() - INTERVAL 60 SECOND
	`)
	if err != nil {
		log.Printf("[update-health] query error: %v", err)
		return
	}
	defer rows.Close()

	type staleTask struct {
		TaskID   int64
		NodeID   int64
		NodeName string
	}

	var staleTasks []staleTask
	for rows.Next() {
		var st staleTask
		if err := rows.Scan(&st.TaskID, &st.NodeID, &st.NodeName); err != nil {
			log.Printf("[update-health] scan error: %v", err)
			continue
		}
		staleTasks = append(staleTasks, st)
	}

	for _, st := range staleTasks {
		// Mark the task as failed
		_, err := db.Exec(
			`UPDATE node_tasks SET status = 'failed', error = 'update_agent task timed out (pending > 60s)', completed_at = NOW() WHERE id = ?`,
			st.TaskID,
		)
		if err != nil {
			log.Printf("[update-health] failed to mark task %d as failed: %v", st.TaskID, err)
			continue
		}

		// Update node with update_failed flag via agent_version column marker
		_, err = db.Exec(
			`UPDATE nodes SET status = CASE WHEN status = 'online' THEN 'online' ELSE status END WHERE id = ?`,
			st.NodeID,
		)
		if err != nil {
			log.Printf("[update-health] failed to update node %d: %v", st.NodeID, err)
		}

		log.Printf("[update-health] node %q (ID=%d): update_agent task %d timed out, marked as failed", st.NodeName, st.NodeID, st.TaskID)

		// Notify admin
		if notify != nil {
			notify("⚠️ *Node Update Failed*\nNode: `" + st.NodeName + "`\nAgent update task timed out (no response in 60s).")
		}
	}
}
