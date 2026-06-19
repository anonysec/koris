package api

import (
	"net/http"
	"time"
)

// ========== Revenue Reports ==========

func (s *Server) revenueReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}

	period := r.URL.Query().Get("period") // daily, weekly, monthly
	if period == "" {
		period = "daily"
	}

	var groupBy, dateFormat string
	switch period {
	case "weekly":
		groupBy = "YEARWEEK(created_at)"
		dateFormat = "%Y-W%u"
	case "monthly":
		groupBy = "DATE_FORMAT(created_at, '%Y-%m')"
		dateFormat = "%Y-%m"
	default:
		groupBy = "DATE(created_at)"
		dateFormat = "%Y-%m-%d"
	}

	// Revenue by period
	rows, err := s.DB.Query(`
		SELECT DATE_FORMAT(created_at, '` + dateFormat + `') as period,
		       COUNT(*) as count,
		       COALESCE(SUM(amount), 0) as total
		FROM payments
		WHERE status = 'approved'
		AND created_at >= NOW() - INTERVAL 90 DAY
		GROUP BY ` + groupBy + `
		ORDER BY period DESC
		LIMIT 90`)
	if err != nil {
		writeJSONCode(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	defer rows.Close()

	type RevenuePoint struct {
		Period string  `json:"period"`
		Count  int     `json:"count"`
		Total  float64 `json:"total"`
	}
	points := []RevenuePoint{}
	for rows.Next() {
		var p RevenuePoint
		if rows.Scan(&p.Period, &p.Count, &p.Total) == nil {
			points = append(points, p)
		}
	}

	// Revenue by plan
	planRows, _ := s.DB.Query(`
		SELECT COALESCE(p.name, 'Unknown') as plan_name, COUNT(*) as count, COALESCE(SUM(pay.amount), 0) as total
		FROM payments pay
		LEFT JOIN subscriptions sub ON sub.id = pay.intent_id AND pay.intent_type = 'plan'
		LEFT JOIN plans p ON p.id = sub.plan_id
		WHERE pay.status = 'approved' AND pay.created_at >= NOW() - INTERVAL 30 DAY
		GROUP BY p.name
		ORDER BY total DESC`)
	type PlanRevenue struct {
		Plan  string  `json:"plan"`
		Count int     `json:"count"`
		Total float64 `json:"total"`
	}
	byPlan := []PlanRevenue{}
	if planRows != nil {
		defer planRows.Close()
		for planRows.Next() {
			var p PlanRevenue
			if planRows.Scan(&p.Plan, &p.Count, &p.Total) == nil {
				byPlan = append(byPlan, p)
			}
		}
	}

	// Summary stats
	var totalRevenue, todayRevenue float64
	var totalPayments, pendingPayments int
	s.DB.QueryRow(`SELECT COALESCE(SUM(amount),0), COUNT(*) FROM payments WHERE status='approved'`).Scan(&totalRevenue, &totalPayments)
	s.DB.QueryRow(`SELECT COALESCE(SUM(amount),0) FROM payments WHERE status='approved' AND DATE(created_at)=CURDATE()`).Scan(&todayRevenue)
	s.DB.QueryRow(`SELECT COUNT(*) FROM payments WHERE status='pending'`).Scan(&pendingPayments)

	writeJSON(w, map[string]any{
		"ok":               true,
		"period":           period,
		"revenue":          points,
		"by_plan":          byPlan,
		"total_revenue":    totalRevenue,
		"today_revenue":    todayRevenue,
		"total_payments":   totalPayments,
		"pending_payments": pendingPayments,
	})
}

// ========== User Reports ==========

func (s *Server) userReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}

	// New registrations per day (last 30 days)
	regRows, _ := s.DB.Query(`
		SELECT DATE(created_at) as day, COUNT(*) as count
		FROM customers
		WHERE deleted_at IS NULL AND created_at >= NOW() - INTERVAL 30 DAY
		GROUP BY DATE(created_at)
		ORDER BY day DESC`)
	type DayCount struct {
		Day   string `json:"day"`
		Count int    `json:"count"`
	}
	registrations := []DayCount{}
	if regRows != nil {
		defer regRows.Close()
		for regRows.Next() {
			var d DayCount
			var t time.Time
			if regRows.Scan(&t, &d.Count) == nil {
				d.Day = t.Format("2006-01-02")
				registrations = append(registrations, d)
			}
		}
	}

	// Status breakdown
	var active, limited, disabled, expired, total int
	s.DB.QueryRow(`SELECT COUNT(*) FROM customers WHERE deleted_at IS NULL`).Scan(&total)
	s.DB.QueryRow(`SELECT COUNT(*) FROM customers WHERE deleted_at IS NULL AND status='active'`).Scan(&active)
	s.DB.QueryRow(`SELECT COUNT(*) FROM customers WHERE deleted_at IS NULL AND status='limited'`).Scan(&limited)
	s.DB.QueryRow(`SELECT COUNT(*) FROM customers WHERE deleted_at IS NULL AND status='disabled'`).Scan(&disabled)
	s.DB.QueryRow(`SELECT COUNT(*) FROM customers WHERE deleted_at IS NULL AND status='expired'`).Scan(&expired)

	writeJSON(w, map[string]any{
		"ok":            true,
		"registrations": registrations,
		"total":         total,
		"active":        active,
		"limited":       limited,
		"disabled":      disabled,
		"expired":       expired,
	})
}

// ========== Bandwidth Reports ==========

func (s *Server) bandwidthReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}

	// Per-node bandwidth (last 24h)
	nodeRows, _ := s.DB.Query(`
		SELECT n.name, COALESCE(SUM(s.rx_bytes),0) as rx, COALESCE(SUM(s.tx_bytes),0) as tx
		FROM node_usage_snapshots s
		JOIN nodes n ON n.id = s.node_id
		WHERE s.created_at >= NOW() - INTERVAL 24 HOUR
		GROUP BY n.id, n.name
		ORDER BY rx + tx DESC`)
	type NodeBandwidth struct {
		Node    string `json:"node"`
		RxBytes int64  `json:"rx_bytes"`
		TxBytes int64  `json:"tx_bytes"`
	}
	byNode := []NodeBandwidth{}
	if nodeRows != nil {
		defer nodeRows.Close()
		for nodeRows.Next() {
			var nb NodeBandwidth
			if nodeRows.Scan(&nb.Node, &nb.RxBytes, &nb.TxBytes) == nil {
				byNode = append(byNode, nb)
			}
		}
	}

	// Top users by bandwidth (last 24h)
	userRows, _ := s.DB.Query(`
		SELECT username, COALESCE(SUM(acctinputoctets),0) as rx, COALESCE(SUM(acctoutputoctets),0) as tx
		FROM radacct
		WHERE acctstarttime >= NOW() - INTERVAL 24 HOUR
		GROUP BY username
		ORDER BY rx + tx DESC
		LIMIT 20`)
	type UserBandwidth struct {
		Username string `json:"username"`
		RxBytes  int64  `json:"rx_bytes"`
		TxBytes  int64  `json:"tx_bytes"`
	}
	topUsers := []UserBandwidth{}
	if userRows != nil {
		defer userRows.Close()
		for userRows.Next() {
			var ub UserBandwidth
			if userRows.Scan(&ub.Username, &ub.RxBytes, &ub.TxBytes) == nil {
				topUsers = append(topUsers, ub)
			}
		}
	}

	// Total bandwidth today
	var todayRx, todayTx int64
	s.DB.QueryRow(`SELECT COALESCE(SUM(acctinputoctets),0), COALESCE(SUM(acctoutputoctets),0) FROM radacct WHERE acctstarttime >= CURDATE()`).Scan(&todayRx, &todayTx)

	writeJSON(w, map[string]any{
		"ok":        true,
		"by_node":   byNode,
		"top_users": topUsers,
		"today_rx":  todayRx,
		"today_tx":  todayTx,
	})
}
