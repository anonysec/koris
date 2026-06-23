package api

import (
	"database/sql"
	"log"
	"sync"
)

// PreparedStmts holds prepared statements for high-frequency queries.
// Statements are prepared lazily on first use via sync.Once to avoid
// startup dependency ordering issues.
type PreparedStmts struct {
	once sync.Once
	err  error

	// Node authentication (used by both push and poll endpoints)
	nodeAuth *sql.Stmt

	// Node push queries
	nodePushUpdateWithIP *sql.Stmt
	nodePushUpdateNoIP   *sql.Stmt
	nodePushUpsertStatus *sql.Stmt
	nodePushSnapshot     *sql.Stmt

	// Task poll queries
	taskPollSelect *sql.Stmt
	taskPollUpdate *sql.Stmt
}

// prepareAll prepares all cached statements. Called via sync.Once.
func (p *PreparedStmts) prepareAll(db *sql.DB) {
	p.once.Do(func() {
		var err error

		p.nodeAuth, err = db.Prepare(`SELECT id,status FROM nodes WHERE api_token_hash=? LIMIT 1`)
		if err != nil {
			log.Printf("[prepared] failed to prepare nodeAuth: %v", err)
			p.err = err
			return
		}

		p.nodePushUpdateWithIP, err = db.Prepare(`UPDATE nodes SET status='online',last_seen_at=NOW(),public_ip=? WHERE id=?`)
		if err != nil {
			log.Printf("[prepared] failed to prepare nodePushUpdateWithIP: %v", err)
			p.err = err
			return
		}

		p.nodePushUpdateNoIP, err = db.Prepare(`UPDATE nodes SET status='online',last_seen_at=NOW() WHERE id=?`)
		if err != nil {
			log.Printf("[prepared] failed to prepare nodePushUpdateNoIP: %v", err)
			p.err = err
			return
		}

		p.nodePushUpsertStatus, err = db.Prepare(`INSERT INTO node_status(node_id,cpu_percent,ram_percent,disk_percent,rx_bps,tx_bps,openvpn_status,l2tp_status,ikev2_status,payload_json)
		VALUES(?,?,?,?,?,?,?,?,?,?)
		ON DUPLICATE KEY UPDATE cpu_percent=VALUES(cpu_percent),ram_percent=VALUES(ram_percent),disk_percent=VALUES(disk_percent),rx_bps=VALUES(rx_bps),tx_bps=VALUES(tx_bps),openvpn_status=VALUES(openvpn_status),l2tp_status=VALUES(l2tp_status),ikev2_status=VALUES(ikev2_status),payload_json=VALUES(payload_json)`)
		if err != nil {
			log.Printf("[prepared] failed to prepare nodePushUpsertStatus: %v", err)
			p.err = err
			return
		}

		p.nodePushSnapshot, err = db.Prepare(`INSERT INTO node_usage_snapshots(node_id,rx_bytes,tx_bytes,online_users) VALUES(?,?,?,?)`)
		if err != nil {
			log.Printf("[prepared] failed to prepare nodePushSnapshot: %v", err)
			p.err = err
			return
		}

		p.taskPollSelect, err = db.Prepare(`SELECT t.id,t.node_id,n.name,t.action,COALESCE(t.payload_json,JSON_OBJECT()),t.status,COALESCE(t.result_json,JSON_OBJECT()),COALESCE(t.error,''),COALESCE(t.created_by,''),t.claimed_at,t.completed_at,t.created_at,t.updated_at FROM node_tasks t LEFT JOIN nodes n ON n.id=t.node_id WHERE t.node_id=? AND t.status='pending' ORDER BY t.id ASC LIMIT 5 FOR UPDATE`)
		if err != nil {
			log.Printf("[prepared] failed to prepare taskPollSelect: %v", err)
			p.err = err
			return
		}

		p.taskPollUpdate, err = db.Prepare(`UPDATE node_tasks SET status='running',claimed_at=NOW() WHERE id=? AND status='pending'`)
		if err != nil {
			log.Printf("[prepared] failed to prepare taskPollUpdate: %v", err)
			p.err = err
			return
		}

		log.Printf("[prepared] all statements prepared successfully")
	})
}

// initStmts ensures the prepared statements are initialized and returns
// true if they are ready to use. Falls back to direct queries on failure.
func (s *Server) initStmts() bool {
	s.stmts.prepareAll(s.DB)
	return s.stmts.err == nil
}
