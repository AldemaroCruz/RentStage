package quote

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) List(
	ctx context.Context,
	tenantID string,
	search string,
	status string,
	customerID string,
) ([]Summary, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			q.id,
			q.quote_number,
			q.customer_id,
			TRIM(c.first_name || ' ' || c.last_name) AS customer_name,
			c.phone,
			reservation.id,
			reservation.reservation_number,
			q.status,
			q.start_at,
			q.end_at,
			q.event_type,
			q.event_location,
			q.subtotal::float8,
			q.discount_amount::float8,
			q.extra_charges::float8,
			q.total::float8,
			COUNT(qi.id)::int AS item_count,
			q.expires_at,
			q.created_at,
			q.updated_at
		FROM quotes q
		JOIN customers c ON c.tenant_id = q.tenant_id AND c.id = q.customer_id
		LEFT JOIN quote_items qi ON qi.tenant_id = q.tenant_id AND qi.quote_id = q.id
		LEFT JOIN reservations reservation ON reservation.tenant_id = q.tenant_id AND reservation.quote_id = q.id
		WHERE q.tenant_id = $1
		  AND (
			$2 = '' OR
			q.quote_number::text ILIKE '%' || $2 || '%' OR
			('QT-' || LPAD(q.quote_number::text, 6, '0')) ILIKE '%' || $2 || '%' OR
			c.first_name ILIKE '%' || $2 || '%' OR
			c.last_name ILIKE '%' || $2 || '%' OR
			COALESCE(c.company_name, '') ILIKE '%' || $2 || '%' OR
			COALESCE(q.event_type, '') ILIKE '%' || $2 || '%' OR
			COALESCE(q.event_location, '') ILIKE '%' || $2 || '%'
		  )
		  AND ($3 = '' OR q.status = $3)
		  AND (NULLIF($4, '') IS NULL OR q.customer_id = NULLIF($4, '')::uuid)
		GROUP BY q.id, c.id, reservation.id
		ORDER BY q.created_at DESC
	`, tenantID, search, status, customerID)
	if err != nil {
		return nil, fmt.Errorf("list quotes: %w", err)
	}
	defer rows.Close()

	items := make([]Summary, 0)
	for rows.Next() {
		item, err := scanSummary(rows)
		if err != nil {
			return nil, fmt.Errorf("scan quote summary: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) Get(ctx context.Context, tenantID, quoteID string) (Detail, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT
			q.id,
			q.quote_number,
			q.customer_id,
			TRIM(c.first_name || ' ' || c.last_name) AS customer_name,
			c.phone,
			reservation.id,
			reservation.reservation_number,
			q.status,
			q.start_at,
			q.end_at,
			q.event_type,
			q.event_location,
			q.subtotal::float8,
			q.discount_amount::float8,
			q.extra_charges::float8,
			q.total::float8,
			COUNT(qi.id)::int AS item_count,
			q.expires_at,
			q.created_at,
			q.updated_at,
			q.notes
		FROM quotes q
		JOIN customers c ON c.tenant_id = q.tenant_id AND c.id = q.customer_id
		LEFT JOIN quote_items qi ON qi.tenant_id = q.tenant_id AND qi.quote_id = q.id
		LEFT JOIN reservations reservation ON reservation.tenant_id = q.tenant_id AND reservation.quote_id = q.id
		WHERE q.tenant_id = $1 AND q.id = $2
		GROUP BY q.id, c.id, reservation.id
	`, tenantID, quoteID)

	var item Detail
	if err := row.Scan(
		&item.ID,
		&item.QuoteNumber,
		&item.CustomerID,
		&item.CustomerName,
		&item.CustomerPhone,
		&item.ReservationID,
		&item.ReservationNumber,
		&item.Status,
		&item.StartAt,
		&item.EndAt,
		&item.EventType,
		&item.EventLocation,
		&item.Subtotal,
		&item.DiscountAmount,
		&item.ExtraCharges,
		&item.Total,
		&item.ItemCount,
		&item.ExpiresAt,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.Notes,
	); errors.Is(err, pgx.ErrNoRows) {
		return Detail{}, ErrNotFound
	} else if err != nil {
		return Detail{}, fmt.Errorf("get quote: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT
			qi.id,
			qi.resource_id,
			r.name,
			qi.description,
			qi.quantity,
			qi.unit_price::float8,
			qi.discount_amount::float8,
			qi.line_total::float8,
			qi.created_at
		FROM quote_items qi
		JOIN resources r ON r.tenant_id = qi.tenant_id AND r.id = qi.resource_id
		WHERE qi.tenant_id = $1 AND qi.quote_id = $2
		ORDER BY qi.created_at, qi.id
	`, tenantID, quoteID)
	if err != nil {
		return Detail{}, fmt.Errorf("list quote items: %w", err)
	}
	defer rows.Close()

	item.Items = make([]Item, 0)
	for rows.Next() {
		var quoteItem Item
		if err := rows.Scan(
			&quoteItem.ID,
			&quoteItem.ResourceID,
			&quoteItem.ResourceName,
			&quoteItem.Description,
			&quoteItem.Quantity,
			&quoteItem.UnitPrice,
			&quoteItem.DiscountAmount,
			&quoteItem.LineTotal,
			&quoteItem.CreatedAt,
		); err != nil {
			return Detail{}, fmt.Errorf("scan quote item: %w", err)
		}
		item.Items = append(item.Items, quoteItem)
	}
	if err := rows.Err(); err != nil {
		return Detail{}, fmt.Errorf("iterate quote items: %w", err)
	}
	portal, err := r.portalSummary(ctx, tenantID, quoteID)
	if err != nil {
		return Detail{}, err
	}
	item.Portal = portal
	return item, nil
}

func (r *Repository) portalSummary(ctx context.Context, tenantID, quoteID string) (*PortalSummary, error) {
	var item PortalSummary
	err := r.pool.QueryRow(ctx, `
		SELECT id, status, revision, expires_at, last_viewed_at, view_count,
		       decision_at, decision_source, response_name, response_email,
		       rejection_reason, terms_version, created_at, updated_at
		FROM quote_portals
		WHERE tenant_id = $1 AND quote_id = $2
	`, tenantID, quoteID).Scan(
		&item.ID,
		&item.Status,
		&item.Revision,
		&item.ExpiresAt,
		&item.LastViewedAt,
		&item.ViewCount,
		&item.DecisionAt,
		&item.DecisionSource,
		&item.ResponseName,
		&item.ResponseEmail,
		&item.RejectionReason,
		&item.TermsVersion,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get quote portal summary: %w", err)
	}
	return &item, nil
}

func (r *Repository) Create(ctx context.Context, tenantID string, input normalizedInput) (Detail, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Detail{}, fmt.Errorf("begin create quote: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := requireCustomer(ctx, tx, tenantID, input.CustomerID); err != nil {
		return Detail{}, err
	}

	var quoteID string
	err = tx.QueryRow(ctx, `
		INSERT INTO quotes (
			tenant_id, customer_id, start_at, end_at, status, event_type,
			event_location, subtotal, discount_amount, extra_charges, total,
			notes, expires_at
		) VALUES ($1, $2, $3, $4, 'DRAFT', $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`,
		tenantID,
		input.CustomerID,
		input.StartAt,
		input.EndAt,
		input.EventType,
		input.EventLocation,
		input.Subtotal,
		input.DiscountAmount,
		input.ExtraCharges,
		input.Total,
		input.Notes,
		input.ExpiresAt,
	).Scan(&quoteID)
	if err != nil {
		return Detail{}, fmt.Errorf("insert quote: %w", err)
	}
	if err := replaceItems(ctx, tx, tenantID, quoteID, input.Items); err != nil {
		return Detail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Detail{}, fmt.Errorf("commit create quote: %w", err)
	}
	return r.Get(ctx, tenantID, quoteID)
}

func (r *Repository) Update(ctx context.Context, tenantID, quoteID string, input normalizedInput) (Detail, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Detail{}, fmt.Errorf("begin update quote: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var currentStatus string
	if err := tx.QueryRow(ctx, `
		SELECT status
		FROM quotes
		WHERE tenant_id = $1 AND id = $2
		FOR UPDATE
	`, tenantID, quoteID).Scan(&currentStatus); errors.Is(err, pgx.ErrNoRows) {
		return Detail{}, ErrNotFound
	} else if err != nil {
		return Detail{}, fmt.Errorf("lock quote: %w", err)
	}
	if currentStatus != "DRAFT" {
		return Detail{}, ErrImmutable
	}
	if err := requireCustomer(ctx, tx, tenantID, input.CustomerID); err != nil {
		return Detail{}, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE quotes SET
			customer_id = $3,
			start_at = $4,
			end_at = $5,
			event_type = $6,
			event_location = $7,
			subtotal = $8,
			discount_amount = $9,
			extra_charges = $10,
			total = $11,
			notes = $12,
			expires_at = $13
		WHERE tenant_id = $1 AND id = $2
	`,
		tenantID,
		quoteID,
		input.CustomerID,
		input.StartAt,
		input.EndAt,
		input.EventType,
		input.EventLocation,
		input.Subtotal,
		input.DiscountAmount,
		input.ExtraCharges,
		input.Total,
		input.Notes,
		input.ExpiresAt,
	)
	if err != nil {
		return Detail{}, fmt.Errorf("update quote: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM quote_items WHERE tenant_id = $1 AND quote_id = $2`, tenantID, quoteID); err != nil {
		return Detail{}, fmt.Errorf("delete quote items: %w", err)
	}
	if err := replaceItems(ctx, tx, tenantID, quoteID, input.Items); err != nil {
		return Detail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Detail{}, fmt.Errorf("commit update quote: %w", err)
	}
	return r.Get(ctx, tenantID, quoteID)
}

func (r *Repository) Transition(
	ctx context.Context,
	tenantID string,
	quoteID string,
	allowed []string,
	target string,
) (Detail, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		UPDATE quotes
		SET status = $3
		WHERE tenant_id = $1
		  AND id = $2
		  AND status = ANY($4::text[])
		RETURNING id
	`, tenantID, quoteID, target, allowed).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		var exists bool
		if checkErr := r.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM quotes WHERE tenant_id = $1 AND id = $2)`,
			tenantID,
			quoteID,
		).Scan(&exists); checkErr != nil {
			return Detail{}, fmt.Errorf("check quote transition: %w", checkErr)
		}
		if !exists {
			return Detail{}, ErrNotFound
		}
		return Detail{}, ErrInvalidTransition
	}
	if err != nil {
		return Detail{}, fmt.Errorf("transition quote: %w", err)
	}
	return r.Get(ctx, tenantID, id)
}

func requireCustomer(ctx context.Context, tx pgx.Tx, tenantID, customerID string) error {
	var exists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM customers WHERE tenant_id = $1 AND id = $2
		)
	`, tenantID, customerID).Scan(&exists); err != nil {
		return fmt.Errorf("check quote customer: %w", err)
	}
	if !exists {
		return ErrCustomerNotFound
	}
	return nil
}

func replaceItems(ctx context.Context, tx pgx.Tx, tenantID, quoteID string, items []normalizedItem) error {
	for _, item := range items {
		command, err := tx.Exec(ctx, `
			INSERT INTO quote_items (
				tenant_id, quote_id, resource_id, description, quantity,
				unit_price, discount_amount, line_total
			)
			SELECT
				$1, $2, r.id, COALESCE(NULLIF($4, ''), r.name), $5, $6, $7, $8
			FROM resources r
			WHERE r.tenant_id = $1 AND r.id = $3 AND r.active = TRUE
		`,
			tenantID,
			quoteID,
			item.ResourceID,
			item.Description,
			item.Quantity,
			item.UnitPrice,
			item.DiscountAmount,
			item.LineTotal,
		)
		if err != nil {
			return fmt.Errorf("insert quote item: %w", err)
		}
		if command.RowsAffected() == 0 {
			return ErrResourceNotFound
		}
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSummary(row rowScanner) (Summary, error) {
	var item Summary
	if err := row.Scan(
		&item.ID,
		&item.QuoteNumber,
		&item.CustomerID,
		&item.CustomerName,
		&item.CustomerPhone,
		&item.ReservationID,
		&item.ReservationNumber,
		&item.Status,
		&item.StartAt,
		&item.EndAt,
		&item.EventType,
		&item.EventLocation,
		&item.Subtotal,
		&item.DiscountAmount,
		&item.ExtraCharges,
		&item.Total,
		&item.ItemCount,
		&item.ExpiresAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return Summary{}, err
	}
	return item, nil
}
