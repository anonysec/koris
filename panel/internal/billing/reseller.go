//go:build !lite

package billing

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

// ResellerMarginInfo holds margin/profit data for a reseller over a time period.
type ResellerMarginInfo struct {
	ResellerID     int64   `json:"reseller_id"`
	TotalPurchased float64 `json:"total_purchased"` // wholesale credit bought (allocations)
	TotalSold      float64 `json:"total_sold"`      // retail revenue from customers (deductions)
	TotalMargin    float64 `json:"total_margin"`    // profit (sold - purchased cost)
	MarginPercent  float64 `json:"margin_percent"`  // margin / sold * 100
	CustomerCount  int     `json:"customer_count"`  // active customers under this reseller
	Period         string  `json:"period"`
}

// PurchaseResellerCredit adds credit to a reseller's balance at wholesale rate.
// The gatewayRef is recorded for audit trail (e.g., a payment reference or admin note).
func (b *BillingEngine) PurchaseResellerCredit(ctx context.Context, resellerID int64, amount float64, gatewayRef string) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	// Verify reseller exists and get username
	var username string
	err := b.db.QueryRowContext(ctx, `
		SELECT username FROM admins WHERE id = $1 AND role = 'reseller'`, resellerID,
	).Scan(&username)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("reseller %d not found", resellerID)
		}
		return fmt.Errorf("fetch reseller: %w", err)
	}

	// Add credit to reseller's balance
	_, err = b.db.ExecContext(ctx, `
		UPDATE admins SET credit = credit + $1 WHERE id = $2`, amount, resellerID,
	)
	if err != nil {
		return fmt.Errorf("update reseller credit: %w", err)
	}

	// Record the transaction
	desc := fmt.Sprintf("Credit purchase (ref: %s)", gatewayRef)
	_, err = b.db.ExecContext(ctx, `
		INSERT INTO reseller_transactions (reseller_username, amount, type, description, actor)
		VALUES ($1, $2, 'allocation', $3, 'system')`,
		username, amount, desc,
	)
	if err != nil {
		return fmt.Errorf("record reseller transaction: %w", err)
	}

	log.Printf("[billing] reseller %s (id=%d) purchased credit: %.2f, ref=%s",
		username, resellerID, amount, gatewayRef)
	b.notify(fmt.Sprintf("reseller %s purchased %.2f credit (ref: %s)", username, amount, gatewayRef))
	return nil
}

// GetResellerMargin calculates the margin/profit for a reseller over a given period.
// It compares wholesale credit purchased (allocations) vs retail revenue (deductions from subscriptions).
func (b *BillingEngine) GetResellerMargin(ctx context.Context, resellerID int64, from, to time.Time) (*ResellerMarginInfo, error) {
	// Verify reseller exists and get username
	var username string
	err := b.db.QueryRowContext(ctx, `
		SELECT username FROM admins WHERE id = $1 AND role = 'reseller'`, resellerID,
	).Scan(&username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("reseller %d not found", resellerID)
		}
		return nil, fmt.Errorf("fetch reseller: %w", err)
	}

	// Total purchased: sum of allocation transactions in the period
	var totalPurchased float64
	err = b.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM reseller_transactions
		WHERE reseller_username = $1 AND type = 'allocation'
		AND created_at >= $1 AND created_at < $2`,
		username, from, to,
	).Scan(&totalPurchased)
	if err != nil {
		return nil, fmt.Errorf("query purchased: %w", err)
	}

	// Total sold (retail): sum of absolute deduction amounts in the period
	// Deductions are stored as negative amounts, so we negate the sum
	var totalSold float64
	err = b.db.QueryRowContext(ctx, `
		SELECT COALESCE(-SUM(amount), 0) FROM reseller_transactions
		WHERE reseller_username = $1 AND type = 'deduction'
		AND created_at >= $1 AND created_at < $2`,
		username, from, to,
	).Scan(&totalSold)
	if err != nil {
		return nil, fmt.Errorf("query sold: %w", err)
	}

	// Active customer count under this reseller
	var customerCount int
	err = b.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM customers
		WHERE created_by = $1 AND deleted_at IS NULL AND status = 'active'`,
		username,
	).Scan(&customerCount)
	if err != nil {
		return nil, fmt.Errorf("query customer count: %w", err)
	}

	// Calculate margin
	totalMargin := totalSold - totalPurchased
	var marginPercent float64
	if totalSold > 0 {
		marginPercent = (totalMargin / totalSold) * 100
	}

	info := &ResellerMarginInfo{
		ResellerID:     resellerID,
		TotalPurchased: totalPurchased,
		TotalSold:      totalSold,
		TotalMargin:    totalMargin,
		MarginPercent:  marginPercent,
		CustomerCount:  customerCount,
		Period:         fmt.Sprintf("%s to %s", from.Format(time.RFC3339), to.Format(time.RFC3339)),
	}

	return info, nil
}

// ResellerCreateSubscription creates a subscription for a customer using the reseller's credit.
// The plan's full retail price is deducted from the reseller's credit balance.
// The reseller's margin is the difference between their sell price and the wholesale cost they paid.
func (b *BillingEngine) ResellerCreateSubscription(ctx context.Context, resellerID int64, customerID int64, planID int64) error {
	// Verify reseller exists and get username + current credit
	var username string
	var credit float64
	err := b.db.QueryRowContext(ctx, `
		SELECT username, COALESCE(credit, 0) FROM admins WHERE id = $1 AND role = 'reseller'`,
		resellerID,
	).Scan(&username, &credit)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("reseller %d not found", resellerID)
		}
		return fmt.Errorf("fetch reseller: %w", err)
	}

	// Verify customer belongs to this reseller
	var custUsername string
	err = b.db.QueryRowContext(ctx, `
		SELECT username FROM customers WHERE id = $1 AND created_by = $2 AND deleted_at IS NULL`,
		customerID, username,
	).Scan(&custUsername)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("customer %d not found or not owned by reseller", customerID)
		}
		return fmt.Errorf("fetch customer: %w", err)
	}

	// Verify plan is allowed for this reseller
	var allowed int
	err = b.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM reseller_allowed_plans WHERE reseller_id = $1 AND plan_id = $2`,
		resellerID, planID,
	).Scan(&allowed)
	if err != nil {
		return fmt.Errorf("check plan access: %w", err)
	}
	if allowed == 0 {
		return fmt.Errorf("plan %d not allowed for reseller %d", planID, resellerID)
	}

	// Fetch plan details (retail price)
	var planName string
	var planPrice float64
	var planDataGB float64
	var planDays int
	err = b.db.QueryRowContext(ctx, `
		SELECT name, price, data_gb, duration_days FROM plans WHERE id = $1 AND is_active = TRUE`,
		planID,
	).Scan(&planName, &planPrice, &planDataGB, &planDays)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("plan %d not found or inactive", planID)
		}
		return fmt.Errorf("fetch plan: %w", err)
	}

	// Check reseller has sufficient credit (plan's full retail price is deducted)
	if planPrice > 0 && credit < planPrice {
		return fmt.Errorf("insufficient reseller credit: %.2f < %.2f", credit, planPrice)
	}

	// Deduct from reseller credit
	if planPrice > 0 {
		_, err = b.db.ExecContext(ctx, `
			UPDATE admins SET credit = credit - $2 WHERE id = $3`, planPrice, resellerID,
		)
		if err != nil {
			return fmt.Errorf("deduct reseller credit: %w", err)
		}

		// Record the deduction transaction
		desc := fmt.Sprintf("Subscription for %s: %s", custUsername, planName)
		_, err = b.db.ExecContext(ctx, `
			INSERT INTO reseller_transactions (reseller_username, amount, type, description, actor)
			VALUES ($2, $3, 'deduction', $4, $5)`,
			username, -planPrice, desc, username,
		)
		if err != nil {
			return fmt.Errorf("record deduction transaction: %w", err)
		}
	}

	// Assign plan to customer
	_, err = b.db.ExecContext(ctx, `
		UPDATE customers SET plan_id = $2, data_limit_gb = $3, status = 'active' WHERE id = $4`,
		planID, planDataGB, customerID,
	)
	if err != nil {
		return fmt.Errorf("assign plan to customer: %w", err)
	}

	log.Printf("[billing] reseller %s created subscription for customer %s (plan=%s, price=%.2f)",
		username, custUsername, planName, planPrice)
	b.notify(fmt.Sprintf("reseller %s assigned plan %s to customer %s (cost: %.2f)",
		username, planName, custUsername, planPrice))
	return nil
}
