package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/anonysec/koris/internal/notify"
	"github.com/anonysec/koris/proto/korispb"
)

// Processor handles job execution for the worker service
type Processor struct {
	db    *sql.DB
	email *notify.EmailSender
	tg    *notify.TelegramSender
}

// NewProcessor creates a new job processor
func NewProcessor(db *sql.DB, emailCfg notify.EmailConfig, tgCfg notify.TelegramConfig) *Processor {
	return &Processor{
		db:    db,
		email: notify.NewEmailSender(emailCfg),
		tg:    notify.NewTelegramSender(tgCfg),
	}
}

// Process handles job execution based on job type
func (p *Processor) Process(job Job) (string, error) {
	return p.ProcessWithContext(context.Background(), job)
}

func (p *Processor) processBilling(ctx context.Context, req *korispb.TriggerBillingRequest) (string, error) {
	periodStart, _ := time.Parse("2006-01-02", req.PeriodStart)
	periodEnd, _ := time.Parse("2006-01-02", req.PeriodEnd)

	query := `
		SELECT username, SUM(bytes_in + bytes_out) as total_bytes
		FROM traffic_log
		WHERE timestamp >= $1 AND timestamp < $2 AND username = $3
		GROUP BY username
	`

	var totalBytes int64
	err := p.db.QueryRowContext(ctx, query, periodStart, periodEnd, req.Username).Scan(&totalBytes)
	if err != nil && err != sql.ErrNoRows {
		return "", fmt.Errorf("failed to query traffic: %w", err)
	}

	// Generate billing record
	insertQuery := `
		INSERT INTO billing_records (username, period_start, period_end, bytes_used, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (username, period_start) DO UPDATE SET bytes_used = $4, updated_at = NOW()
	`

	_, err = p.db.ExecContext(ctx, insertQuery, req.Username, periodStart, periodEnd, totalBytes)
	if err != nil {
		return "", fmt.Errorf("failed to insert billing record: %w", err)
	}

	log.Printf("Billing calculated for %s: %d bytes used", req.Username, totalBytes)
	return fmt.Sprintf("Billing calculated for %s: %d bytes used", req.Username, totalBytes), nil
}

func (p *Processor) processInvoice(ctx context.Context, req *korispb.TriggerInvoiceRequest) (string, error) {
	// Get customer info
	var email, name string
	custQuery := `SELECT email, name FROM customers WHERE id = $1`
	err := p.db.QueryRowContext(ctx, custQuery, req.CustomerId).Scan(&email, &name)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("customer not found: %d", req.CustomerId)
		}
		return "", fmt.Errorf("failed to get customer: %w", err)
	}

	// Generate invoice number
	var invoiceNum int64
	invoiceQuery := `SELECT COALESCE(MAX(CAST(SUBSTRING(invoice_number, 5) AS INTEGER)), 0) + 1 FROM invoices`
	p.db.QueryRowContext(ctx, invoiceQuery).Scan(&invoiceNum)

	// Get pending charges
	var totalAmount float64
	chargesQuery := `
		SELECT COALESCE(SUM(amount), 0) FROM charges
		WHERE customer_id = $1 AND status = 'pending' AND period_start >= $2 AND period_end <= $3
	`
	periodStart, _ := time.Parse("2006-01-02", req.PeriodStart)
	periodEnd, _ := time.Parse("2006-01-02", req.PeriodEnd)
	p.db.QueryRowContext(ctx, chargesQuery, req.CustomerId, periodStart, periodEnd).Scan(&totalAmount)

	// Create invoice
	insertInvoice := `
		INSERT INTO invoices (customer_id, invoice_number, amount, status, period_start, period_end, created_at)
		VALUES ($1, $2, $3, 'draft', $4, $5, NOW())
	`
	_, err = p.db.ExecContext(ctx, insertInvoice, req.CustomerId, fmt.Sprintf("INV-%d", invoiceNum), totalAmount, periodStart, periodEnd)
	if err != nil {
		return "", fmt.Errorf("failed to create invoice: %w", err)
	}

	// Mark charges as invoiced
	updateCharges := `
		UPDATE charges SET status = 'invoiced', invoice_number = $1
		WHERE customer_id = $2 AND status = 'pending'
	`
	p.db.ExecContext(ctx, updateCharges, fmt.Sprintf("INV-%d", invoiceNum), req.CustomerId)

	log.Printf("Invoice INV-%d created for customer %d: %.2f", invoiceNum, req.CustomerId, totalAmount)

	return fmt.Sprintf("Invoice INV-%d created: %.2f", invoiceNum, totalAmount), nil
}

func (p *Processor) processEmail(ctx context.Context, req *korispb.TriggerEmailRequest) (string, error) {
	// Apply template variables
	subject := req.Subject
	body := req.Body
	for k, v := range req.Vars {
		subject = strings.ReplaceAll(subject, "{"+k+"}", v)
		body = strings.ReplaceAll(body, "{"+k+"}", v)
	}

	// Send email
	err := p.email.Send(req.To, subject, body)
	if err != nil {
		return "", fmt.Errorf("failed to send email: %w", err)
	}

	// Log email
	insertLog := `INSERT INTO email_logs (recipient, subject, status, created_at) VALUES ($1, $2, 'sent', NOW())`
	p.db.ExecContext(ctx, insertLog, req.To, req.Subject)

	// Send Telegram notification if enabled
	if p.tg != nil && p.tg.IsEnabled() {
		p.tg.Send(fmt.Sprintf("📧 Email sent to %s: %s", req.To, req.Subject))
	}

	return fmt.Sprintf("Email sent to %s: %s", req.To, req.Subject), nil
}

func (p *Processor) processReport(ctx context.Context, req *korispb.TriggerReportRequest) (string, error) {
	periodStart, _ := time.Parse("2006-01-02", req.PeriodStart)
	periodEnd, _ := time.Parse("2006-01-02", req.PeriodEnd)

	var reportData string
	switch req.ReportType {
	case "usage":
		reportData = p.generateUsageReport(ctx, periodStart, periodEnd)
	case "revenue":
		reportData = p.generateRevenueReport(ctx, periodStart, periodEnd)
	case "customers":
		reportData = p.generateCustomerReport(ctx, periodStart, periodEnd)
	default:
		return "", fmt.Errorf("unknown report type: %s", req.ReportType)
	}

	// Store report
	insertReport := `
		INSERT INTO reports (admin_id, report_type, period_start, period_end, data, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`
	_, err := p.db.ExecContext(ctx, insertReport, req.AdminId, req.ReportType, periodStart, periodEnd, reportData)
	if err != nil {
		return "", fmt.Errorf("failed to store report: %w", err)
	}

	return fmt.Sprintf("Report generated: %s", req.ReportType), nil
}

func (p *Processor) generateUsageReport(ctx context.Context, start, end time.Time) string {
	query := `
		SELECT COUNT(DISTINCT username), COALESCE(SUM(bytes_in), 0), COALESCE(SUM(bytes_out), 0)
		FROM traffic_log
		WHERE timestamp >= $1 AND timestamp < $2
	`
	var users, bytesIn, bytesOut int64
	p.db.QueryRowContext(ctx, query, start, end).Scan(&users, &bytesIn, &bytesOut)
	return fmt.Sprintf(`{"users":%d,"bytes_in":%d,"bytes_out":%d}`, users, bytesIn, bytesOut)
}

func (p *Processor) generateRevenueReport(ctx context.Context, start, end time.Time) string {
	query := `
		SELECT COUNT(*), COALESCE(SUM(amount), 0)
		FROM invoices
		WHERE status = 'paid' AND created_at >= $1 AND created_at < $2
	`
	var count int64
	var amount float64
	p.db.QueryRowContext(ctx, query, start, end).Scan(&count, &amount)
	return fmt.Sprintf(`{"invoices":%d,"total":%.2f}`, count, amount)
}

func (p *Processor) generateCustomerReport(ctx context.Context, start, end time.Time) string {
	query := `SELECT COUNT(*) FROM customers WHERE created_at >= $1 AND created_at < $2`
	var count int64
	p.db.QueryRowContext(ctx, query, start, end).Scan(&count)
	return fmt.Sprintf(`{"new_customers":%d}`, count)
}
