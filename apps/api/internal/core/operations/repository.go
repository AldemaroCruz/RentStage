package operations

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Calendar(ctx context.Context, tenantID string, filter CalendarFilter) (CalendarResult, error) {
	timezone, err := r.timezone(ctx, tenantID)
	if err != nil {
		return CalendarResult{}, err
	}

	var statuses any
	if len(filter.Statuses) > 0 {
		statuses = filter.Statuses
	}
	var customerID any
	if filter.CustomerID != nil {
		customerID = *filter.CustomerID
	}
	var resourceID any
	if filter.ResourceID != nil {
		resourceID = *filter.ResourceID
	}

	rows, err := r.pool.Query(ctx, `
		SELECT
			r.id,
			r.reservation_number,
			r.customer_id,
			TRIM(c.first_name || ' ' || c.last_name) AS customer_name,
			c.phone,
			r.source,
			r.status,
			r.block_start_at,
			r.block_end_at,
			r.event_start_at,
			r.event_end_at,
			r.event_type,
			r.event_location,
			r.total::float8,
			(SELECT COUNT(*)::int
			   FROM reservation_items count_items
			  WHERE count_items.tenant_id = r.tenant_id
			    AND count_items.reservation_id = r.id) AS item_count,
			COALESCE((SELECT SUM(CASE WHEN resource.track_individual_assets THEN item.quantity ELSE 0 END)::int
			   FROM reservation_items item
			   JOIN resources resource
			     ON resource.tenant_id = item.tenant_id
			    AND resource.id = item.resource_id
			  WHERE item.tenant_id = r.tenant_id
			    AND item.reservation_id = r.id), 0) AS required_asset_count,
			(SELECT COUNT(*)::int
			   FROM reservation_assets assignment
			  WHERE assignment.tenant_id = r.tenant_id
			    AND assignment.reservation_id = r.id
			    AND assignment.released_at IS NULL) AS assigned_asset_count,
			COALESCE((SELECT STRING_AGG(item.quantity::text || '× ' || resource.name, ' · ' ORDER BY resource.name)
			   FROM reservation_items item
			   JOIN resources resource
			     ON resource.tenant_id = item.tenant_id
			    AND resource.id = item.resource_id
			  WHERE item.tenant_id = r.tenant_id
			    AND item.reservation_id = r.id), '') AS resource_summary,
			r.checked_out_at,
			r.returned_at
		FROM reservations r
		JOIN customers c
		  ON c.tenant_id = r.tenant_id
		 AND c.id = r.customer_id
		WHERE r.tenant_id = $1
		  AND r.block_start_at < $3
		  AND r.block_end_at > $2
		  AND ($4::text[] IS NULL OR r.status = ANY($4::text[]))
		  AND ($5::uuid IS NULL OR r.customer_id = $5::uuid)
		  AND ($6::uuid IS NULL OR EXISTS (
			SELECT 1
			FROM reservation_items filtered_item
			WHERE filtered_item.tenant_id = r.tenant_id
			  AND filtered_item.reservation_id = r.id
			  AND filtered_item.resource_id = $6::uuid
		  ))
		ORDER BY r.block_start_at, r.reservation_number
	`, tenantID, filter.From, filter.To, statuses, customerID, resourceID)
	if err != nil {
		return CalendarResult{}, fmt.Errorf("query calendar reservations: %w", err)
	}
	defer rows.Close()

	items := make([]CalendarEvent, 0)
	for rows.Next() {
		var item CalendarEvent
		if err := rows.Scan(
			&item.ID,
			&item.ReservationNumber,
			&item.CustomerID,
			&item.CustomerName,
			&item.CustomerPhone,
			&item.Source,
			&item.Status,
			&item.BlockStartAt,
			&item.BlockEndAt,
			&item.EventStartAt,
			&item.EventEndAt,
			&item.EventType,
			&item.EventLocation,
			&item.Total,
			&item.ItemCount,
			&item.RequiredAssetCount,
			&item.AssignedAssetCount,
			&item.ResourceSummary,
			&item.CheckedOutAt,
			&item.ReturnedAt,
		); err != nil {
			return CalendarResult{}, fmt.Errorf("scan calendar reservation: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return CalendarResult{}, fmt.Errorf("iterate calendar reservations: %w", err)
	}
	return CalendarResult{From: filter.From, To: filter.To, Timezone: timezone, Items: items}, nil
}

func (r *Repository) Agenda(ctx context.Context, tenantID, date string) (AgendaResult, error) {
	resolvedDate, timezone, dayStart, dayEnd, err := r.dayBounds(ctx, tenantID, date)
	if err != nil {
		return AgendaResult{}, err
	}
	calendar, err := r.Calendar(ctx, tenantID, CalendarFilter{From: dayStart, To: dayEnd})
	if err != nil {
		return AgendaResult{}, err
	}
	result := AgendaResult{
		Date:         resolvedDate,
		Timezone:     timezone,
		DayStart:     dayStart,
		DayEnd:       dayEnd,
		Departures:   make([]CalendarEvent, 0),
		Events:       make([]CalendarEvent, 0),
		Returns:      make([]CalendarEvent, 0),
		PendingClose: make([]CalendarEvent, 0),
	}
	blockingBeforeCheckout := map[string]bool{
		"PENDING": true, "CONFIRMED": true, "PREPARING": true, "READY": true,
	}
	for _, item := range calendar.Items {
		if !item.BlockStartAt.Before(dayStart) && item.BlockStartAt.Before(dayEnd) && blockingBeforeCheckout[item.Status] {
			result.Departures = append(result.Departures, item)
		}
		if !item.EventStartAt.Before(dayStart) && item.EventStartAt.Before(dayEnd) && item.Status != "CANCELLED" {
			result.Events = append(result.Events, item)
		}
		if !item.BlockEndAt.Before(dayStart) && item.BlockEndAt.Before(dayEnd) && item.Status == "CHECKED_OUT" {
			result.Returns = append(result.Returns, item)
		}
		if item.Status == "RETURNED" {
			result.PendingClose = append(result.PendingClose, item)
		}
	}
	alerts, err := r.Alerts(ctx, tenantID)
	if err != nil {
		return AgendaResult{}, err
	}
	result.OverdueReturns = make([]Alert, 0)
	for _, alert := range alerts.Items {
		if alert.Type == "OVERDUE_RETURN" {
			result.OverdueReturns = append(result.OverdueReturns, alert)
		}
	}
	return result, nil
}

func (r *Repository) Alerts(ctx context.Context, tenantID string) (AlertResult, error) {
	now := time.Now().UTC()
	windowEnd := now.Add(24 * time.Hour)
	rows, err := r.pool.Query(ctx, `
		SELECT
			r.id,
			r.reservation_number,
			TRIM(c.first_name || ' ' || c.last_name) AS customer_name,
			r.event_type,
			r.status,
			r.block_start_at,
			r.block_end_at,
			COALESCE((SELECT SUM(CASE WHEN resource.track_individual_assets THEN item.quantity ELSE 0 END)::int
			   FROM reservation_items item
			   JOIN resources resource
			     ON resource.tenant_id = item.tenant_id
			    AND resource.id = item.resource_id
			  WHERE item.tenant_id = r.tenant_id
			    AND item.reservation_id = r.id), 0) AS required_asset_count,
			(SELECT COUNT(*)::int
			   FROM reservation_assets assignment
			  WHERE assignment.tenant_id = r.tenant_id
			    AND assignment.reservation_id = r.id
			    AND assignment.released_at IS NULL) AS assigned_asset_count
		FROM reservations r
		JOIN customers c
		  ON c.tenant_id = r.tenant_id
		 AND c.id = r.customer_id
		WHERE r.tenant_id = $1
		  AND (
			(r.status = 'CHECKED_OUT' AND r.block_end_at < $2)
			OR (r.status IN ('CONFIRMED', 'PREPARING') AND r.block_start_at >= $2 AND r.block_start_at <= $3)
			OR (r.status = 'READY' AND r.block_start_at < $2)
			OR r.status = 'RETURNED'
		  )
		ORDER BY
		  CASE
		    WHEN r.status = 'CHECKED_OUT' AND r.block_end_at < $2 THEN 1
		    WHEN r.status IN ('CONFIRMED', 'PREPARING') THEN 2
		    WHEN r.status = 'READY' THEN 3
		    ELSE 4
		  END,
		  r.block_start_at
		LIMIT 50
	`, tenantID, now, windowEnd)
	if err != nil {
		return AlertResult{}, fmt.Errorf("query operational alerts: %w", err)
	}
	defer rows.Close()

	result := AlertResult{GeneratedAt: now, Items: make([]Alert, 0)}
	for rows.Next() {
		var reservationID, customerName, status string
		var reservationNumber int64
		var eventType *string
		var blockStart, blockEnd time.Time
		var required, assigned int
		if err := rows.Scan(
			&reservationID,
			&reservationNumber,
			&customerName,
			&eventType,
			&status,
			&blockStart,
			&blockEnd,
			&required,
			&assigned,
		); err != nil {
			return AlertResult{}, fmt.Errorf("scan operational alert: %w", err)
		}
		missing := required - assigned
		if missing < 0 {
			missing = 0
		}
		alert := Alert{
			ReservationID:      reservationID,
			ReservationNumber:  reservationNumber,
			CustomerName:       customerName,
			EventType:          eventType,
			Status:             status,
			MissingAssetCount:  missing,
			RequiredAssetCount: required,
			AssignedAssetCount: assigned,
		}
		switch {
		case status == "CHECKED_OUT" && blockEnd.Before(now):
			alert.ID = "overdue-return:" + reservationID
			alert.Type = "OVERDUE_RETURN"
			alert.Severity = "CRITICAL"
			alert.Title = "Retorno atrasado"
			alert.Message = "El equipo sigue afuera después de la hora esperada de retorno."
			alert.DueAt = &blockEnd
			alert.MinutesOverdue = int64(now.Sub(blockEnd).Minutes())
		case status == "CONFIRMED":
			alert.ID = "preparation-not-started:" + reservationID
			alert.Type = "PREPARATION_NOT_STARTED"
			alert.Severity = "WARNING"
			alert.Title = "Preparación pendiente"
			alert.Message = "La reserva inicia en menos de 24 horas y todavía no ha entrado a preparación."
			alert.DueAt = &blockStart
		case status == "PREPARING" && missing > 0:
			alert.ID = "preparation-incomplete:" + reservationID
			alert.Type = "PREPARATION_INCOMPLETE"
			alert.Severity = "WARNING"
			alert.Title = "Inventario incompleto"
			alert.Message = "Faltan unidades físicas por asignar antes de marcar el pedido como listo."
			alert.DueAt = &blockStart
		case status == "READY" && blockStart.Before(now):
			alert.ID = "checkout-pending:" + reservationID
			alert.Type = "CHECKOUT_PENDING"
			alert.Severity = "WARNING"
			alert.Title = "Entrega pendiente"
			alert.Message = "El período bloqueado ya comenzó y la reserva sigue marcada como lista."
			alert.DueAt = &blockStart
		case status == "RETURNED":
			alert.ID = "completion-pending:" + reservationID
			alert.Type = "COMPLETION_PENDING"
			alert.Severity = "INFO"
			alert.Title = "Cierre administrativo pendiente"
			alert.Message = "El equipo ya fue devuelto; falta completar la reserva."
		default:
			continue
		}
		result.Items = append(result.Items, alert)
		switch alert.Severity {
		case "CRITICAL":
			result.Counts.Critical++
		case "WARNING":
			result.Counts.Warning++
		default:
			result.Counts.Info++
		}
	}
	if err := rows.Err(); err != nil {
		return AlertResult{}, fmt.Errorf("iterate operational alerts: %w", err)
	}
	result.Counts.Total = len(result.Items)
	sort.SliceStable(result.Items, func(i, j int) bool {
		order := map[string]int{"CRITICAL": 0, "WARNING": 1, "INFO": 2}
		return order[result.Items[i].Severity] < order[result.Items[j].Severity]
	})
	return result, nil
}

func (r *Repository) timezone(ctx context.Context, tenantID string) (string, error) {
	var timezone string
	if err := r.pool.QueryRow(ctx, `SELECT timezone FROM tenants WHERE id = $1`, tenantID).Scan(&timezone); err != nil {
		return "", fmt.Errorf("load tenant timezone: %w", err)
	}
	return timezone, nil
}

func (r *Repository) dayBounds(ctx context.Context, tenantID, date string) (string, string, time.Time, time.Time, error) {
	var resolvedDate, timezone string
	var dayStart, dayEnd time.Time
	if err := r.pool.QueryRow(ctx, `
		SELECT
			COALESCE(NULLIF($2, ''), (CURRENT_TIMESTAMP AT TIME ZONE timezone)::date::text) AS local_date,
			timezone,
			(COALESCE(NULLIF($2, ''), (CURRENT_TIMESTAMP AT TIME ZONE timezone)::date::text)::date::timestamp AT TIME ZONE timezone) AS day_start,
			((COALESCE(NULLIF($2, ''), (CURRENT_TIMESTAMP AT TIME ZONE timezone)::date::text)::date + 1)::timestamp AT TIME ZONE timezone) AS day_end
		FROM tenants
		WHERE id = $1
	`, tenantID, date).Scan(&resolvedDate, &timezone, &dayStart, &dayEnd); err != nil {
		return "", "", time.Time{}, time.Time{}, fmt.Errorf("resolve operations day: %w", err)
	}
	return resolvedDate, timezone, dayStart, dayEnd, nil
}
