package reservation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rentstage/rentstage/apps/api/internal/core/availability"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) List(ctx context.Context, tenantID, search, status string) ([]Summary, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			r.id,
			r.reservation_number,
			r.customer_id,
			TRIM(c.first_name || ' ' || c.last_name) AS customer_name,
			c.phone,
			r.quote_id,
			q.quote_number,
			r.source,
			r.status,
			r.block_start_at,
			r.block_end_at,
			r.event_start_at,
			r.event_end_at,
			r.event_type,
			r.event_location,
			r.subtotal::float8,
			r.discount_amount::float8,
			r.extra_charges::float8,
			r.total::float8,
			COUNT(ri.id)::int AS item_count,
			r.created_at,
			r.updated_at,
			CASE WHEN r.status = 'COMPLETED' THEN r.updated_at ELSE NULL END AS completed_at,
			r.checked_out_at,
			r.checked_out_by,
			r.returned_at,
			r.returned_by
		FROM reservations r
		JOIN customers c ON c.tenant_id = r.tenant_id AND c.id = r.customer_id
		LEFT JOIN quotes q ON q.tenant_id = r.tenant_id AND q.id = r.quote_id
		LEFT JOIN reservation_items ri ON ri.tenant_id = r.tenant_id AND ri.reservation_id = r.id
		WHERE r.tenant_id = $1
		  AND (
			$2 = '' OR
			r.reservation_number::text ILIKE '%' || $2 || '%' OR
			('RS-' || LPAD(r.reservation_number::text, 6, '0')) ILIKE '%' || $2 || '%' OR
			c.first_name ILIKE '%' || $2 || '%' OR
			c.last_name ILIKE '%' || $2 || '%' OR
			COALESCE(c.company_name, '') ILIKE '%' || $2 || '%' OR
			COALESCE(r.event_type, '') ILIKE '%' || $2 || '%' OR
			COALESCE(r.event_location, '') ILIKE '%' || $2 || '%'
		  )
		  AND ($3 = '' OR r.status = $3)
		GROUP BY r.id, c.id, q.id
		ORDER BY r.block_start_at, r.created_at DESC
	`, tenantID, search, status)
	if err != nil {
		return nil, fmt.Errorf("list reservations: %w", err)
	}
	defer rows.Close()

	items := make([]Summary, 0)
	for rows.Next() {
		item, scanErr := scanSummary(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan reservation summary: %w", scanErr)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) Get(ctx context.Context, tenantID, reservationID string) (Detail, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT
			r.id,
			r.reservation_number,
			r.customer_id,
			TRIM(c.first_name || ' ' || c.last_name) AS customer_name,
			c.phone,
			r.quote_id,
			q.quote_number,
			r.source,
			r.status,
			r.block_start_at,
			r.block_end_at,
			r.event_start_at,
			r.event_end_at,
			r.event_type,
			r.event_location,
			r.subtotal::float8,
			r.discount_amount::float8,
			r.extra_charges::float8,
			r.total::float8,
			COUNT(ri.id)::int AS item_count,
			r.created_at,
			r.updated_at,
			CASE WHEN r.status = 'COMPLETED' THEN r.updated_at ELSE NULL END AS completed_at,
			r.checked_out_at,
			r.checked_out_by,
			r.returned_at,
			r.returned_by,
			r.notes,
			r.checkout_notes,
			r.return_notes
		FROM reservations r
		JOIN customers c ON c.tenant_id = r.tenant_id AND c.id = r.customer_id
		LEFT JOIN quotes q ON q.tenant_id = r.tenant_id AND q.id = r.quote_id
		LEFT JOIN reservation_items ri ON ri.tenant_id = r.tenant_id AND ri.reservation_id = r.id
		WHERE r.tenant_id = $1 AND r.id = $2
		GROUP BY r.id, c.id, q.id
	`, tenantID, reservationID)

	var item Detail
	if err := row.Scan(
		&item.ID,
		&item.ReservationNumber,
		&item.CustomerID,
		&item.CustomerName,
		&item.CustomerPhone,
		&item.QuoteID,
		&item.QuoteNumber,
		&item.Source,
		&item.Status,
		&item.BlockStartAt,
		&item.BlockEndAt,
		&item.EventStartAt,
		&item.EventEndAt,
		&item.EventType,
		&item.EventLocation,
		&item.Subtotal,
		&item.DiscountAmount,
		&item.ExtraCharges,
		&item.Total,
		&item.ItemCount,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.CompletedAt,
		&item.CheckedOutAt,
		&item.CheckedOutBy,
		&item.ReturnedAt,
		&item.ReturnedBy,
		&item.Notes,
		&item.CheckoutNotes,
		&item.ReturnNotes,
	); errors.Is(err, pgx.ErrNoRows) {
		return Detail{}, ErrNotFound
	} else if err != nil {
		return Detail{}, fmt.Errorf("get reservation: %w", err)
	}

	itemRows, err := r.pool.Query(ctx, `
		SELECT
			ri.id,
			ri.resource_id,
			res.name,
			ri.description,
			res.track_individual_assets,
			ri.quantity,
			ri.unit_price::float8,
			ri.discount_amount::float8,
			ri.line_total::float8
		FROM reservation_items ri
		JOIN resources res ON res.tenant_id = ri.tenant_id AND res.id = ri.resource_id
		WHERE ri.tenant_id = $1 AND ri.reservation_id = $2
		ORDER BY ri.created_at, ri.id
	`, tenantID, reservationID)
	if err != nil {
		return Detail{}, fmt.Errorf("list reservation items: %w", err)
	}
	defer itemRows.Close()
	item.Items = make([]Item, 0)
	for itemRows.Next() {
		var line Item
		if err := itemRows.Scan(
			&line.ID,
			&line.ResourceID,
			&line.ResourceName,
			&line.Description,
			&line.TrackIndividualAssets,
			&line.Quantity,
			&line.UnitPrice,
			&line.DiscountAmount,
			&line.LineTotal,
		); err != nil {
			return Detail{}, fmt.Errorf("scan reservation item: %w", err)
		}
		item.Items = append(item.Items, line)
	}
	if err := itemRows.Err(); err != nil {
		return Detail{}, fmt.Errorf("iterate reservation items: %w", err)
	}

	historyRows, err := r.pool.Query(ctx, `
		SELECT id, from_status, to_status, actor_id, note, created_at
		FROM reservation_status_history
		WHERE tenant_id = $1 AND reservation_id = $2
		ORDER BY created_at, id
	`, tenantID, reservationID)
	if err != nil {
		return Detail{}, fmt.Errorf("list reservation history: %w", err)
	}
	defer historyRows.Close()
	item.StatusHistory = make([]StatusHistory, 0)
	for historyRows.Next() {
		var event StatusHistory
		if err := historyRows.Scan(
			&event.ID,
			&event.FromStatus,
			&event.ToStatus,
			&event.ActorID,
			&event.Note,
			&event.CreatedAt,
		); err != nil {
			return Detail{}, fmt.Errorf("scan reservation history: %w", err)
		}
		item.StatusHistory = append(item.StatusHistory, event)
	}
	if err := historyRows.Err(); err != nil {
		return Detail{}, fmt.Errorf("iterate reservation history: %w", err)
	}

	scheduleRows, err := r.pool.Query(ctx, `
		SELECT id, previous_block_start_at, previous_block_end_at,
		       previous_event_start_at, previous_event_end_at,
		       new_block_start_at, new_block_end_at,
		       new_event_start_at, new_event_end_at,
		       reason, actor_id, created_at
		FROM reservation_schedule_history
		WHERE tenant_id = $1 AND reservation_id = $2
		ORDER BY created_at DESC, id DESC
	`, tenantID, reservationID)
	if err != nil {
		return Detail{}, fmt.Errorf("list reservation schedule history: %w", err)
	}
	defer scheduleRows.Close()
	item.ScheduleHistory = make([]ScheduleHistory, 0)
	for scheduleRows.Next() {
		var event ScheduleHistory
		if err := scheduleRows.Scan(
			&event.ID,
			&event.PreviousBlockStartAt,
			&event.PreviousBlockEndAt,
			&event.PreviousEventStartAt,
			&event.PreviousEventEndAt,
			&event.NewBlockStartAt,
			&event.NewBlockEndAt,
			&event.NewEventStartAt,
			&event.NewEventEndAt,
			&event.Reason,
			&event.ActorID,
			&event.CreatedAt,
		); err != nil {
			return Detail{}, fmt.Errorf("scan reservation schedule history: %w", err)
		}
		item.ScheduleHistory = append(item.ScheduleHistory, event)
	}
	if err := scheduleRows.Err(); err != nil {
		return Detail{}, fmt.Errorf("iterate reservation schedule history: %w", err)
	}

	if err := r.populateWarehouseDetails(ctx, tenantID, &item); err != nil {
		return Detail{}, err
	}
	return item, nil
}

func (r *Repository) CreateFromQuote(
	ctx context.Context,
	tenantID string,
	quoteID string,
	actorID string,
) (Detail, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return Detail{}, fmt.Errorf("begin quote conversion: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	result, err := r.CreateFromQuoteTx(ctx, tx, tenantID, quoteID, actorID, "ACCEPTED")
	if errors.Is(err, ErrQuoteStatus) {
		return Detail{}, ErrQuoteNotAccepted
	}
	if err != nil {
		return Detail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Detail{}, fmt.Errorf("commit quote conversion: %w", err)
	}
	return r.Get(ctx, tenantID, result.ReservationID)
}

// CreateFromQuoteTx creates a reservation using an existing transaction. The
// caller chooses the exact quote status that is allowed. Back-office conversion
// requires ACCEPTED, while the public portal requires SENT and changes the quote
// to ACCEPTED only after the reservation has been created successfully.
func (r *Repository) CreateFromQuoteTx(
	ctx context.Context,
	tx pgx.Tx,
	tenantID string,
	quoteID string,
	actorID string,
	requiredStatus string,
) (QuoteConversionResult, error) {
	var sourceQuote struct {
		CustomerID     string
		QuoteNumber    int64
		Status         string
		StartAt        time.Time
		EndAt          time.Time
		EventType      *string
		EventLocation  *string
		Subtotal       float64
		DiscountAmount float64
		ExtraCharges   float64
		Total          float64
		Notes          string
	}
	if err := tx.QueryRow(ctx, `
		SELECT customer_id, quote_number, status, start_at, end_at, event_type,
		       event_location, subtotal::float8, discount_amount::float8,
		       extra_charges::float8, total::float8, notes
		FROM quotes
		WHERE tenant_id = $1 AND id = $2
		FOR UPDATE
	`, tenantID, quoteID).Scan(
		&sourceQuote.CustomerID,
		&sourceQuote.QuoteNumber,
		&sourceQuote.Status,
		&sourceQuote.StartAt,
		&sourceQuote.EndAt,
		&sourceQuote.EventType,
		&sourceQuote.EventLocation,
		&sourceQuote.Subtotal,
		&sourceQuote.DiscountAmount,
		&sourceQuote.ExtraCharges,
		&sourceQuote.Total,
		&sourceQuote.Notes,
	); errors.Is(err, pgx.ErrNoRows) {
		return QuoteConversionResult{}, ErrQuoteNotFound
	} else if err != nil {
		return QuoteConversionResult{}, fmt.Errorf("lock quote for conversion: %w", err)
	}

	var existingID string
	var existingNumber int64
	if err := tx.QueryRow(ctx, `
		SELECT id, reservation_number
		FROM reservations
		WHERE tenant_id = $1 AND quote_id = $2
	`, tenantID, quoteID).Scan(&existingID, &existingNumber); err == nil {
		return QuoteConversionResult{}, &QuoteAlreadyConvertedError{ReservationID: existingID}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return QuoteConversionResult{}, fmt.Errorf("check converted quote: %w", err)
	}
	if sourceQuote.Status != requiredStatus {
		return QuoteConversionResult{}, ErrQuoteStatus
	}

	rows, err := tx.Query(ctx, `
		SELECT id, resource_id, description, quantity, unit_price::float8,
		       discount_amount::float8, line_total::float8
		FROM quote_items
		WHERE tenant_id = $1 AND quote_id = $2
		ORDER BY created_at, id
	`, tenantID, quoteID)
	if err != nil {
		return QuoteConversionResult{}, fmt.Errorf("load quote items for conversion: %w", err)
	}
	type quoteLine struct {
		ID             string
		ResourceID     string
		Description    string
		Quantity       int
		UnitPrice      float64
		DiscountAmount float64
		LineTotal      float64
	}
	lines := make([]quoteLine, 0)
	availabilityItems := make([]availability.NormalizedItem, 0)
	for rows.Next() {
		var line quoteLine
		if err := rows.Scan(
			&line.ID,
			&line.ResourceID,
			&line.Description,
			&line.Quantity,
			&line.UnitPrice,
			&line.DiscountAmount,
			&line.LineTotal,
		); err != nil {
			rows.Close()
			return QuoteConversionResult{}, fmt.Errorf("scan quote item for conversion: %w", err)
		}
		lines = append(lines, line)
		availabilityItems = append(availabilityItems, availability.NormalizedItem{
			ResourceID: line.ResourceID,
			Quantity:   line.Quantity,
		})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return QuoteConversionResult{}, fmt.Errorf("iterate quote items for conversion: %w", err)
	}
	rows.Close()
	if len(lines) == 0 {
		return QuoteConversionResult{}, fmt.Errorf("quote has no items")
	}

	availabilityInput := availability.NormalizedInput{
		StartAt: sourceQuote.StartAt,
		EndAt:   sourceQuote.EndAt,
		Items:   availabilityItems,
	}
	if err := availability.LockResources(ctx, tx, tenantID, availabilityItems); err != nil {
		return QuoteConversionResult{}, err
	}
	availabilityResult, err := availability.CheckWithQuerier(ctx, tx, tenantID, availabilityInput)
	if err != nil {
		return QuoteConversionResult{}, err
	}
	if !availabilityResult.Available {
		return QuoteConversionResult{}, &AvailabilityConflictError{Result: availabilityResult}
	}

	var result QuoteConversionResult
	if err := tx.QueryRow(ctx, `
		INSERT INTO reservations (
			tenant_id, customer_id, quote_id, source, block_start_at, block_end_at,
			event_start_at, event_end_at, status, event_type, event_location,
			subtotal, discount_amount, extra_charges, total, notes
		) VALUES (
			$1, $2, $3, 'QUOTE', $4, $5, $4, $5, 'PENDING', $6, $7,
			$8, $9, $10, $11, $12
		)
		RETURNING id, reservation_number
	`,
		tenantID,
		sourceQuote.CustomerID,
		quoteID,
		sourceQuote.StartAt,
		sourceQuote.EndAt,
		sourceQuote.EventType,
		sourceQuote.EventLocation,
		sourceQuote.Subtotal,
		sourceQuote.DiscountAmount,
		sourceQuote.ExtraCharges,
		sourceQuote.Total,
		sourceQuote.Notes,
	).Scan(&result.ReservationID, &result.ReservationNumber); err != nil {
		return QuoteConversionResult{}, fmt.Errorf("insert reservation from quote: %w", err)
	}

	for _, line := range lines {
		if _, err := tx.Exec(ctx, `
			INSERT INTO reservation_items (
				tenant_id, reservation_id, resource_id, quote_item_id,
				description, quantity, unit_price, discount_amount, line_total
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`,
			tenantID,
			result.ReservationID,
			line.ResourceID,
			line.ID,
			line.Description,
			line.Quantity,
			line.UnitPrice,
			line.DiscountAmount,
			line.LineTotal,
		); err != nil {
			return QuoteConversionResult{}, fmt.Errorf("insert reservation item from quote: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO reservation_status_history (
			tenant_id, reservation_id, from_status, to_status, actor_id, note
		) VALUES ($1, $2, NULL, 'PENDING', $3, $4)
	`, tenantID, result.ReservationID, actorID, "Created from quote QT-"+formatNumber(sourceQuote.QuoteNumber)); err != nil {
		return QuoteConversionResult{}, fmt.Errorf("insert initial reservation status: %w", err)
	}
	return result, nil
}

func (r *Repository) Transition(
	ctx context.Context,
	tenantID string,
	reservationID string,
	allowed []string,
	target string,
	actorID string,
) (Detail, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Detail{}, fmt.Errorf("begin reservation transition: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var current string
	if err := tx.QueryRow(ctx, `
		SELECT status
		FROM reservations
		WHERE tenant_id = $1 AND id = $2
		FOR UPDATE
	`, tenantID, reservationID).Scan(&current); errors.Is(err, pgx.ErrNoRows) {
		return Detail{}, ErrNotFound
	} else if err != nil {
		return Detail{}, fmt.Errorf("lock reservation: %w", err)
	}
	if !contains(allowed, current) {
		return Detail{}, ErrInvalidTransition
	}
	if _, err := tx.Exec(ctx, `
		UPDATE reservations
		SET status = $3
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, reservationID, target); err != nil {
		return Detail{}, fmt.Errorf("update reservation status: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO reservation_status_history (
			tenant_id, reservation_id, from_status, to_status, actor_id
		) VALUES ($1, $2, $3, $4, $5)
	`, tenantID, reservationID, current, target, actorID); err != nil {
		return Detail{}, fmt.Errorf("insert reservation status history: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Detail{}, fmt.Errorf("commit reservation transition: %w", err)
	}
	return r.Get(ctx, tenantID, reservationID)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSummary(row rowScanner) (Summary, error) {
	var item Summary
	if err := row.Scan(
		&item.ID,
		&item.ReservationNumber,
		&item.CustomerID,
		&item.CustomerName,
		&item.CustomerPhone,
		&item.QuoteID,
		&item.QuoteNumber,
		&item.Source,
		&item.Status,
		&item.BlockStartAt,
		&item.BlockEndAt,
		&item.EventStartAt,
		&item.EventEndAt,
		&item.EventType,
		&item.EventLocation,
		&item.Subtotal,
		&item.DiscountAmount,
		&item.ExtraCharges,
		&item.Total,
		&item.ItemCount,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.CompletedAt,
		&item.CheckedOutAt,
		&item.CheckedOutBy,
		&item.ReturnedAt,
		&item.ReturnedBy,
	); err != nil {
		return Summary{}, err
	}
	return item, nil
}

func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func formatNumber(value int64) string {
	result := fmt.Sprintf("%06d", value)
	return result
}
