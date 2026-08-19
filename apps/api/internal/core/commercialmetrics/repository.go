package commercialmetrics

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (repository *Repository) Report(ctx context.Context, tenantID string, window Window) (Report, error) {
	result := Report{
		GeneratedAt: window.EndAt,
		Window:      window,
		Sources:     make([]SourceCount, 0, 4),
		Monthly:     make([]MonthlyActivity, 0, 6),
	}

	if err := repository.pool.QueryRow(ctx, `
		SELECT tenant.currency,
		  (SELECT COUNT(*)::int FROM quote_requests item
		    WHERE item.tenant_id = $1 AND item.created_at >= $2 AND item.created_at < $3),
		  (SELECT COUNT(*)::int FROM assistant_conversations item
		    WHERE item.tenant_id = $1 AND item.created_at >= $2 AND item.created_at < $3),
		  (SELECT COUNT(*)::int FROM customers item
		    WHERE item.tenant_id = $1 AND item.created_at >= $2 AND item.created_at < $3),
		  (SELECT COUNT(*)::int FROM quotes item
		    WHERE item.tenant_id = $1 AND item.created_at >= $2 AND item.created_at < $3),
		  (SELECT COUNT(*)::int FROM quotes item
		    WHERE item.tenant_id = $1 AND item.created_at >= $2 AND item.created_at < $3
		      AND item.status IN ('SENT', 'ACCEPTED', 'REJECTED', 'EXPIRED')),
		  (SELECT COUNT(*)::int FROM quotes item
		    WHERE item.tenant_id = $1 AND item.created_at >= $2 AND item.created_at < $3
		      AND item.status = 'ACCEPTED'),
		  (SELECT COUNT(*)::int FROM quotes item
		    WHERE item.tenant_id = $1 AND item.created_at >= $2 AND item.created_at < $3
		      AND item.status = 'REJECTED'),
		  (SELECT COUNT(*)::int FROM reservations item
		    WHERE item.tenant_id = $1 AND item.created_at >= $2 AND item.created_at < $3),
		  (SELECT COUNT(*)::int FROM reservations item
		    WHERE item.tenant_id = $1 AND item.created_at >= $2 AND item.created_at < $3
		      AND item.quote_id IS NOT NULL),
		  (SELECT COUNT(*)::int FROM invoices item
		    WHERE item.tenant_id = $1 AND item.issued_at >= $2 AND item.issued_at < $3
		      AND item.status IN ('ISSUED', 'PARTIALLY_PAID', 'PAID')),
		  (SELECT COALESCE(SUM(item.total), 0)::float8 FROM quotes item
		    WHERE item.tenant_id = $1 AND item.status IN ('DRAFT', 'SENT')),
		  (SELECT COALESCE(SUM(item.total), 0)::float8 FROM quotes item
		    WHERE item.tenant_id = $1 AND item.created_at >= $2 AND item.created_at < $3
		      AND item.status = 'ACCEPTED'),
		  (SELECT COALESCE(SUM(item.total), 0)::float8 FROM reservations item
		    WHERE item.tenant_id = $1 AND item.created_at >= $2 AND item.created_at < $3
		      AND item.status <> 'CANCELLED'),
		  (SELECT COALESCE(SUM(item.total_amount), 0)::float8 FROM invoices item
		    WHERE item.tenant_id = $1 AND item.issued_at >= $2 AND item.issued_at < $3
		      AND item.status IN ('ISSUED', 'PARTIALLY_PAID', 'PAID')),
		  (SELECT COALESCE(SUM(item.amount), 0)::float8 FROM payments item
		    WHERE item.tenant_id = $1 AND item.received_at >= $2 AND item.received_at < $3
		      AND item.status = 'CONFIRMED'),
		  (SELECT COALESCE(SUM(item.balance_due), 0)::float8 FROM invoices item
		    WHERE item.tenant_id = $1 AND item.status IN ('ISSUED', 'PARTIALLY_PAID')),
		  (SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (reply.created_at - inbound.created_at))) / 60, 0)::float8
		     FROM assistant_messages inbound
		     JOIN LATERAL (
		       SELECT outbound.created_at
		       FROM assistant_messages outbound
		       WHERE outbound.tenant_id = inbound.tenant_id
		         AND outbound.conversation_id = inbound.conversation_id
		         AND outbound.direction = 'OUTBOUND'
		         AND outbound.status = 'SENT'
		         AND outbound.created_at >= inbound.created_at
		       ORDER BY outbound.created_at, outbound.id
		       LIMIT 1
		     ) reply ON TRUE
		    WHERE inbound.tenant_id = $1 AND inbound.direction = 'INBOUND'
		      AND inbound.created_at >= $2 AND inbound.created_at < $3),
		  (SELECT COUNT(*)::int
		     FROM assistant_messages inbound
		     WHERE inbound.tenant_id = $1 AND inbound.direction = 'INBOUND'
		       AND inbound.created_at >= $2 AND inbound.created_at < $3
		       AND EXISTS (
		         SELECT 1 FROM assistant_messages outbound
		         WHERE outbound.tenant_id = inbound.tenant_id
		           AND outbound.conversation_id = inbound.conversation_id
		           AND outbound.direction = 'OUTBOUND' AND outbound.status = 'SENT'
		           AND outbound.created_at >= inbound.created_at
		       )),
		  (SELECT COUNT(*)::int FROM audit_events item
		    WHERE item.tenant_id = $1 AND item.created_at >= $2 AND item.created_at < $3),
		  (SELECT COUNT(*)::int FROM assistant_messages item
		    WHERE item.tenant_id = $1 AND item.direction = 'OUTBOUND' AND item.status = 'SENT'
		      AND item.approved_at IS NOT NULL AND item.created_at >= $2 AND item.created_at < $3),
		  (SELECT COUNT(*)::int FROM quote_portal_events item
		    WHERE item.tenant_id = $1 AND item.actor_type = 'CUSTOMER'
		      AND item.event_type IN ('ACCEPTED', 'REJECTED')
		      AND item.created_at >= $2 AND item.created_at < $3)
		FROM tenants tenant
		WHERE tenant.id = $1
	`, tenantID, window.StartAt, window.EndAt).Scan(
		&result.Currency,
		&result.Overview.PublicRequests,
		&result.Overview.AssistantConversations,
		&result.Overview.NewCustomers,
		&result.Overview.QuotesCreated,
		&result.Overview.QuotesPresented,
		&result.Overview.QuotesAccepted,
		&result.Overview.QuotesRejected,
		&result.Overview.ReservationsCreated,
		&result.Overview.QuoteReservationsCreated,
		&result.Overview.InvoicesIssued,
		&result.Overview.QuotePipelineValue,
		&result.Overview.AcceptedQuoteValue,
		&result.Overview.ReservationValue,
		&result.Overview.IssuedValue,
		&result.Overview.CollectedValue,
		&result.Overview.OutstandingValue,
		&result.Overview.AverageResponseMinutes,
		&result.Overview.ResponseSamples,
		&result.Overview.AuditEvents,
		&result.Overview.HumanApprovedMessages,
		&result.Overview.CustomerPortalDecisions,
	); err != nil {
		return Report{}, fmt.Errorf("load commercial metrics overview: %w", err)
	}

	if err := repository.loadOutcomes(ctx, tenantID, window, &result); err != nil {
		return Report{}, err
	}
	if err := repository.loadSources(ctx, tenantID, window, &result); err != nil {
		return Report{}, err
	}
	if err := repository.loadMonthly(ctx, tenantID, window.EndAt, &result); err != nil {
		return Report{}, err
	}
	result.finalize()
	return result, nil
}

func (repository *Repository) loadOutcomes(ctx context.Context, tenantID string, window Window, result *Report) error {
	if err := repository.pool.QueryRow(ctx, `
		SELECT
		  COUNT(*) FILTER (WHERE status IN ('PENDING', 'CONFIRMED', 'PREPARING', 'READY', 'CHECKED_OUT'))::int,
		  COUNT(*) FILTER (WHERE status IN ('RETURNED', 'COMPLETED'))::int,
		  COUNT(*) FILTER (WHERE status = 'CANCELLED')::int
		FROM reservations
		WHERE tenant_id = $1 AND created_at >= $2 AND created_at < $3
	`, tenantID, window.StartAt, window.EndAt).Scan(
		&result.Outcomes.Active,
		&result.Outcomes.Completed,
		&result.Outcomes.Cancelled,
	); err != nil {
		return fmt.Errorf("load commercial metric outcomes: %w", err)
	}
	return nil
}

func (repository *Repository) loadSources(ctx context.Context, tenantID string, window Window, result *Report) error {
	rows, err := repository.pool.Query(ctx, `
		WITH sources(source, sort_order) AS (
		  VALUES ('WEB', 1), ('WHATSAPP', 2), ('MANUAL', 3), ('IMPORT', 4)
		)
		SELECT sources.source, COUNT(customer.id)::int
		FROM sources
		LEFT JOIN customers customer
		  ON customer.tenant_id = $1 AND customer.source = sources.source
		 AND customer.created_at >= $2 AND customer.created_at < $3
		GROUP BY sources.source, sources.sort_order
		ORDER BY sources.sort_order
	`, tenantID, window.StartAt, window.EndAt)
	if err != nil {
		return fmt.Errorf("load commercial metric sources: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item SourceCount
		if err := rows.Scan(&item.Source, &item.Count); err != nil {
			return fmt.Errorf("scan commercial metric source: %w", err)
		}
		result.Sources = append(result.Sources, item)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate commercial metric sources: %w", err)
	}
	return nil
}

func (repository *Repository) loadMonthly(ctx context.Context, tenantID string, endAt time.Time, result *Report) error {
	rows, err := repository.pool.Query(ctx, `
		SELECT TO_CHAR(months.month, 'YYYY-MM'),
		  COALESCE((SELECT SUM(item.total) FROM quotes item
		    WHERE item.tenant_id = $1 AND item.status <> 'CANCELLED'
		      AND item.created_at >= months.month AND item.created_at < months.month + INTERVAL '1 month'), 0)::float8,
		  COALESCE((SELECT SUM(item.total) FROM reservations item
		    WHERE item.tenant_id = $1 AND item.status <> 'CANCELLED'
		      AND item.created_at >= months.month AND item.created_at < months.month + INTERVAL '1 month'), 0)::float8,
		  COALESCE((SELECT SUM(item.amount) FROM payments item
		    WHERE item.tenant_id = $1 AND item.status = 'CONFIRMED'
		      AND item.received_at >= months.month AND item.received_at < months.month + INTERVAL '1 month'), 0)::float8
		FROM generate_series(
		  date_trunc('month', $2::timestamptz) - INTERVAL '5 months',
		  date_trunc('month', $2::timestamptz),
		  INTERVAL '1 month'
		) months(month)
		ORDER BY months.month
	`, tenantID, endAt)
	if err != nil {
		return fmt.Errorf("load monthly commercial metrics: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item MonthlyActivity
		if err := rows.Scan(&item.Month, &item.QuoteValue, &item.ReservationValue, &item.CollectedValue); err != nil {
			return fmt.Errorf("scan monthly commercial metrics: %w", err)
		}
		result.Monthly = append(result.Monthly, item)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate monthly commercial metrics: %w", err)
	}
	return nil
}
