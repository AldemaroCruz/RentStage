package reservation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rentstage/rentstage/apps/api/internal/core/availability"
)

func (r *Repository) CreateManual(
	ctx context.Context,
	tenantID string,
	input normalizedCreateInput,
	actorID string,
) (Detail, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return Detail{}, fmt.Errorf("begin manual reservation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var customerExists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM customers WHERE tenant_id = $1 AND id = $2
		)
	`, tenantID, input.CustomerID).Scan(&customerExists); err != nil {
		return Detail{}, fmt.Errorf("check reservation customer: %w", err)
	}
	if !customerExists {
		return Detail{}, ErrCustomerNotFound
	}

	availabilityItems := make([]availability.NormalizedItem, 0, len(input.Items))
	for _, line := range input.Items {
		availabilityItems = append(availabilityItems, availability.NormalizedItem{
			ResourceID: line.ResourceID,
			Quantity:   line.Quantity,
		})
	}
	if err := availability.LockResources(ctx, tx, tenantID, availabilityItems); err != nil {
		return Detail{}, err
	}
	availabilityResult, err := availability.CheckWithQuerier(ctx, tx, tenantID, availability.NormalizedInput{
		StartAt: input.BlockStartAt,
		EndAt:   input.BlockEndAt,
		Items:   availabilityItems,
	})
	if err != nil {
		return Detail{}, err
	}
	if !availabilityResult.Available {
		return Detail{}, &AvailabilityConflictError{Result: availabilityResult}
	}

	resourceNames := make(map[string]string, len(availabilityResult.Items))
	for _, item := range availabilityResult.Items {
		resourceNames[item.ResourceID] = item.ResourceName
	}

	var reservationID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO reservations (
			tenant_id, customer_id, quote_id, source,
			block_start_at, block_end_at, event_start_at, event_end_at,
			status, event_type, event_location,
			subtotal, discount_amount, extra_charges, total, notes
		) VALUES (
			$1, $2, NULL, 'MANUAL',
			$3, $4, $5, $6,
			'PENDING', $7, $8,
			$9, $10, $11, $12, $13
		)
		RETURNING id
	`,
		tenantID,
		input.CustomerID,
		input.BlockStartAt,
		input.BlockEndAt,
		input.EventStartAt,
		input.EventEndAt,
		input.EventType,
		input.EventLocation,
		input.Subtotal,
		input.DiscountAmount,
		input.ExtraCharges,
		input.Total,
		input.Notes,
	).Scan(&reservationID); err != nil {
		return Detail{}, fmt.Errorf("insert manual reservation: %w", err)
	}

	for _, line := range input.Items {
		description := line.Description
		if description == "" {
			description = resourceNames[line.ResourceID]
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO reservation_items (
				tenant_id, reservation_id, resource_id, quote_item_id,
				description, quantity, unit_price, discount_amount, line_total
			) VALUES ($1, $2, $3, NULL, $4, $5, $6, $7, $8)
		`,
			tenantID,
			reservationID,
			line.ResourceID,
			description,
			line.Quantity,
			line.UnitPrice,
			line.DiscountAmount,
			line.LineTotal,
		); err != nil {
			return Detail{}, fmt.Errorf("insert manual reservation item: %w", err)
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO reservation_status_history (
			tenant_id, reservation_id, from_status, to_status, actor_id, note
		) VALUES ($1, $2, NULL, 'PENDING', $3, 'Created manually')
	`, tenantID, reservationID, actorID); err != nil {
		return Detail{}, fmt.Errorf("insert manual reservation status: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Detail{}, fmt.Errorf("commit manual reservation: %w", err)
	}
	return r.Get(ctx, tenantID, reservationID)
}

func (r *Repository) Reschedule(
	ctx context.Context,
	tenantID string,
	reservationID string,
	input normalizedRescheduleInput,
	actorID string,
) (Detail, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return Detail{}, fmt.Errorf("begin reservation reschedule: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var current struct {
		Status       string
		BlockStartAt time.Time
		BlockEndAt   time.Time
		EventStartAt time.Time
		EventEndAt   time.Time
	}
	if err := tx.QueryRow(ctx, `
		SELECT status, block_start_at, block_end_at, event_start_at, event_end_at
		FROM reservations
		WHERE tenant_id = $1 AND id = $2
		FOR UPDATE
	`, tenantID, reservationID).Scan(
		&current.Status,
		&current.BlockStartAt,
		&current.BlockEndAt,
		&current.EventStartAt,
		&current.EventEndAt,
	); errors.Is(err, pgx.ErrNoRows) {
		return Detail{}, ErrNotFound
	} else if err != nil {
		return Detail{}, fmt.Errorf("lock reservation for reschedule: %w", err)
	}
	if !contains([]string{"PENDING", "CONFIRMED", "PREPARING", "READY"}, current.Status) {
		return Detail{}, ErrRescheduleState
	}

	rows, err := tx.Query(ctx, `
		SELECT resource_id, quantity
		FROM reservation_items
		WHERE tenant_id = $1 AND reservation_id = $2
		ORDER BY resource_id
	`, tenantID, reservationID)
	if err != nil {
		return Detail{}, fmt.Errorf("load reservation items for reschedule: %w", err)
	}
	availabilityItems := make([]availability.NormalizedItem, 0)
	for rows.Next() {
		var item availability.NormalizedItem
		if err := rows.Scan(&item.ResourceID, &item.Quantity); err != nil {
			rows.Close()
			return Detail{}, fmt.Errorf("scan reservation item for reschedule: %w", err)
		}
		availabilityItems = append(availabilityItems, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return Detail{}, fmt.Errorf("iterate reservation items for reschedule: %w", err)
	}
	rows.Close()

	if err := availability.LockResources(ctx, tx, tenantID, availabilityItems); err != nil {
		return Detail{}, err
	}
	excludeID := reservationID
	availabilityResult, err := availability.CheckWithQuerier(ctx, tx, tenantID, availability.NormalizedInput{
		StartAt:              input.BlockStartAt,
		EndAt:                input.BlockEndAt,
		Items:                availabilityItems,
		ExcludeReservationID: &excludeID,
	})
	if err != nil {
		return Detail{}, err
	}
	if !availabilityResult.Available {
		return Detail{}, &AvailabilityConflictError{Result: availabilityResult}
	}

	var exactConflict AssetScheduleConflictError
	err = tx.QueryRow(ctx, `
		SELECT a.id, a.asset_code, other.id, other.reservation_number
		FROM reservation_assets current_assignment
		JOIN assets a
		  ON a.tenant_id = current_assignment.tenant_id
		 AND a.id = current_assignment.asset_id
		JOIN reservation_assets other_assignment
		  ON other_assignment.tenant_id = current_assignment.tenant_id
		 AND other_assignment.asset_id = current_assignment.asset_id
		 AND other_assignment.reservation_id <> current_assignment.reservation_id
		 AND other_assignment.released_at IS NULL
		JOIN reservations other
		  ON other.tenant_id = other_assignment.tenant_id
		 AND other.id = other_assignment.reservation_id
		WHERE current_assignment.tenant_id = $1
		  AND current_assignment.reservation_id = $2
		  AND current_assignment.released_at IS NULL
		  AND other.status = ANY($5::text[])
		  AND other.block_start_at < $4
		  AND other.block_end_at > $3
		ORDER BY other.block_start_at, a.asset_code
		LIMIT 1
	`, tenantID, reservationID, input.BlockStartAt, input.BlockEndAt,
		[]string{"PENDING", "CONFIRMED", "PREPARING", "READY", "CHECKED_OUT"},
	).Scan(
		&exactConflict.AssetID,
		&exactConflict.AssetCode,
		&exactConflict.ReservationID,
		&exactConflict.ReservationNumber,
	)
	if err == nil {
		return Detail{}, &exactConflict
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Detail{}, fmt.Errorf("check assigned asset schedule conflicts: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE reservations
		SET block_start_at = $3,
		    block_end_at = $4,
		    event_start_at = $5,
		    event_end_at = $6
		WHERE tenant_id = $1 AND id = $2
	`,
		tenantID,
		reservationID,
		input.BlockStartAt,
		input.BlockEndAt,
		input.EventStartAt,
		input.EventEndAt,
	); err != nil {
		return Detail{}, fmt.Errorf("update reservation schedule: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO reservation_schedule_history (
			tenant_id, reservation_id,
			previous_block_start_at, previous_block_end_at,
			previous_event_start_at, previous_event_end_at,
			new_block_start_at, new_block_end_at,
			new_event_start_at, new_event_end_at,
			reason, actor_id
		) VALUES (
			$1, $2,
			$3, $4, $5, $6,
			$7, $8, $9, $10,
			$11, $12
		)
	`,
		tenantID,
		reservationID,
		current.BlockStartAt,
		current.BlockEndAt,
		current.EventStartAt,
		current.EventEndAt,
		input.BlockStartAt,
		input.BlockEndAt,
		input.EventStartAt,
		input.EventEndAt,
		input.Reason,
		actorID,
	); err != nil {
		return Detail{}, fmt.Errorf("insert reservation schedule history: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Detail{}, fmt.Errorf("commit reservation reschedule: %w", err)
	}
	return r.Get(ctx, tenantID, reservationID)
}
