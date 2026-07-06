package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// revenueTrendResponse is the payload for GET /api/admin/revenue-trend.
type revenueTrendResponse struct {
	OK          bool                `json:"ok"`
	Period      string              `json:"period"`
	TotalCents  int64               `json:"total_cents"`  // sum over the window, in cents
	NewCents    int64               `json:"new_cents"`    // sum of "new subscriptions" only
	RenewCents  int64               `json:"renew_cents"`  // sum of renewals only
	CurrentMRR  int64               `json:"current_mrr"`  // last complete month, cents
	PriorMRR    int64               `json:"prior_mrr"`    // month before that, for delta
	Points      []revenueTrendPoint `json:"points"`
}

// revenueTrendPoint is one bucket in the trend series.
type revenueTrendPoint struct {
	Label      string `json:"label"`       // e.g. "Mon 12"
	Timestamp  string `json:"timestamp"`   // RFC3339 bucket start
	TotalCents int64  `json:"total_cents"` // total revenue in the bucket, cents
	NewCents   int64  `json:"new_cents"`
	RenewCents int64  `json:"renew_cents"`
}

// revenueTrend responds with approved-payment revenue bucketed over a window.
// GET /api/admin/revenue-trend?period=7d|30d|90d|365d
//
// Uses only the payments table (present in migration 001), so it works on
// every deployment without requiring TimescaleDB hyperfunctions.
func (s *Server) revenueTrend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "30d"
	}

	var windowDays int
	var bucket time.Duration
	var labelFmt string
	switch period {
	case "7d":
		windowDays, bucket, labelFmt = 7, 24*time.Hour, "Mon 02"
	case "30d":
		windowDays, bucket, labelFmt = 30, 24*time.Hour, "Jan 02"
	case "90d":
		windowDays, bucket, labelFmt = 90, 7*24*time.Hour, "Jan 02"
	case "365d":
		windowDays, bucket, labelFmt = 365, 30*24*time.Hour, "Jan 2006"
	default:
		http.Error(w, "invalid period (7d|30d|90d|365d)", http.StatusBadRequest)
		return
	}

	end := time.Now().UTC().Truncate(bucket).Add(bucket)
	start := end.Add(-time.Duration(windowDays) * 24 * time.Hour)

	// Read approved payments in the window.
	// intent_type: 'wallet_topup' | 'plan_purchase' | 'plan_renewal' — we
	// count anything that isn't a bare wallet topup as revenue-generating.
	rows, err := s.DB.Query(`
		SELECT
		  created_at,
		  CAST(amount * 100 AS BIGINT) AS cents,
		  COALESCE(intent_type, '') AS intent
		FROM payments
		WHERE status = 'approved'
		  AND created_at >= $1
		  AND created_at <  $2
	`, start, end)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", "revenue-trend: "+err.Error())
		return
	}
	defer rows.Close()

	// Bucket into a map[bucketStart]*revenueTrendPoint.
	series := make(map[time.Time]*revenueTrendPoint)
	for t := start; t.Before(end); t = t.Add(bucket) {
		key := t
		series[key] = &revenueTrendPoint{
			Label:     t.Format(labelFmt),
			Timestamp: t.Format(time.RFC3339),
		}
	}

	var totalCents, newCents, renewCents int64
	for rows.Next() {
		var createdAt time.Time
		var cents int64
		var intent string
		if err := rows.Scan(&createdAt, &cents, &intent); err != nil {
			writeError(w, http.StatusInternalServerError, "db_error", "revenue-trend: "+err.Error())
			return
		}
		key := createdAt.UTC().Truncate(bucket)
		pt, ok := series[key]
		if !ok {
			// Shouldn't happen given the WHERE clause, but be defensive.
			continue
		}
		pt.TotalCents += cents
		totalCents += cents
		switch intent {
		case "plan_purchase":
			pt.NewCents += cents
			newCents += cents
		case "plan_renewal":
			pt.RenewCents += cents
			renewCents += cents
		}
	}

	// Compute MRR (current full month vs prior full month).
	currentMRR, priorMRR := s.computeMRR()

	// Order buckets by time for the response.
	points := make([]revenueTrendPoint, 0, len(series))
	for t := start; t.Before(end); t = t.Add(bucket) {
		if p := series[t]; p != nil {
			points = append(points, *p)
		}
	}

	resp := revenueTrendResponse{
		OK:         true,
		Period:     period,
		TotalCents: totalCents,
		NewCents:   newCents,
		RenewCents: renewCents,
		CurrentMRR: currentMRR,
		PriorMRR:   priorMRR,
		Points:     points,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// computeMRR returns approved-payment revenue for the last full calendar
// month and the one before it (cents). "Full" = complete months, not the
// current in-progress month.
func (s *Server) computeMRR() (current int64, prior int64) {
	now := time.Now().UTC()
	// Start of current month
	curMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	// Start of previous month
	priorMonthStart := curMonthStart.AddDate(0, -1, 0)
	// Start of two months ago
	twoBackStart := curMonthStart.AddDate(0, -2, 0)

	// last complete month = [priorMonthStart, curMonthStart)
	_ = s.DB.QueryRow(`
		SELECT COALESCE(CAST(SUM(amount * 100) AS BIGINT), 0)
		FROM payments
		WHERE status = 'approved'
		  AND created_at >= $1 AND created_at < $2
	`, priorMonthStart, curMonthStart).Scan(&current)

	// month before that = [twoBackStart, priorMonthStart)
	_ = s.DB.QueryRow(`
		SELECT COALESCE(CAST(SUM(amount * 100) AS BIGINT), 0)
		FROM payments
		WHERE status = 'approved'
		  AND created_at >= $1 AND created_at < $2
	`, twoBackStart, priorMonthStart).Scan(&prior)

	return current, prior
}

