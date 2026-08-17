package reservation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
)

var blockingReservationStatuses = []string{"PENDING", "CONFIRMED", "PREPARING", "READY", "CHECKED_OUT"}

func (r *Repository) populateWarehouseDetails(ctx context.Context, tenantID string, item *Detail) error {
	itemIndex := make(map[string]int, len(item.Items))
	for index := range item.Items {
		item.Items[index].Assignments = make([]AssignedAsset, 0)
		itemIndex[item.Items[index].ID] = index
		if item.Items[index].TrackIndividualAssets {
			item.RequiredAssetCount += item.Items[index].Quantity
		}
	}

	rows, err := r.pool.Query(ctx, `
		SELECT
			ra.id,
			ra.reservation_item_id,
			ra.asset_id,
			a.asset_code,
			a.serial_number,
			a.physical_status,
			ra.assigned_at,
			ra.assigned_by,
			ra.checked_out_at,
			ra.checked_out_by,
			ra.returned_at,
			ra.returned_by,
			ra.return_condition,
			ra.return_notes,
			ra.released_at,
			ra.released_by,
			ra.release_reason
		FROM reservation_assets ra
		JOIN assets a ON a.tenant_id = ra.tenant_id AND a.id = ra.asset_id
		WHERE ra.tenant_id = $1 AND ra.reservation_id = $2
		ORDER BY ra.assigned_at, ra.id
	`, tenantID, item.ID)
	if err != nil {
		return fmt.Errorf("list reservation asset assignments: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var reservationItemID string
		var assignment AssignedAsset
		if err := rows.Scan(
			&assignment.AssignmentID,
			&reservationItemID,
			&assignment.AssetID,
			&assignment.AssetCode,
			&assignment.SerialNumber,
			&assignment.PhysicalStatus,
			&assignment.AssignedAt,
			&assignment.AssignedBy,
			&assignment.CheckedOutAt,
			&assignment.CheckedOutBy,
			&assignment.ReturnedAt,
			&assignment.ReturnedBy,
			&assignment.ReturnCondition,
			&assignment.ReturnNotes,
			&assignment.ReleasedAt,
			&assignment.ReleasedBy,
			&assignment.ReleaseReason,
		); err != nil {
			return fmt.Errorf("scan reservation asset assignment: %w", err)
		}
		assignment.State = assignmentState(assignment)
		index, ok := itemIndex[reservationItemID]
		if !ok {
			continue
		}
		item.Items[index].Assignments = append(item.Items[index].Assignments, assignment)
		if assignment.ReleasedAt == nil {
			item.Items[index].AssignedQuantity++
			item.AssignedAssetCount++
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate reservation asset assignments: %w", err)
	}

	item.WarehouseComplete = true
	for index := range item.Items {
		if !item.Items[index].TrackIndividualAssets {
			continue
		}
		missing := item.Items[index].Quantity - item.Items[index].AssignedQuantity
		if missing < 0 {
			missing = 0
		}
		item.Items[index].MissingQuantity = missing
		if missing > 0 {
			item.WarehouseComplete = false
		}
	}

	activityRows, err := r.pool.Query(ctx, `
		SELECT
			e.id,
			e.event_type,
			e.asset_id,
			a.asset_code,
			res.name,
			e.actor_id,
			e.note,
			e.metadata,
			e.created_at
		FROM reservation_activity_events e
		LEFT JOIN assets a ON a.tenant_id = e.tenant_id AND a.id = e.asset_id
		LEFT JOIN resources res ON res.tenant_id = a.tenant_id AND res.id = a.resource_id
		WHERE e.tenant_id = $1 AND e.reservation_id = $2
		ORDER BY e.created_at, e.id
	`, tenantID, item.ID)
	if err != nil {
		return fmt.Errorf("list reservation activity history: %w", err)
	}
	defer activityRows.Close()

	item.ActivityHistory = make([]ActivityEvent, 0)
	for activityRows.Next() {
		var event ActivityEvent
		var metadataBytes []byte
		if err := activityRows.Scan(
			&event.ID,
			&event.EventType,
			&event.AssetID,
			&event.AssetCode,
			&event.ResourceName,
			&event.ActorID,
			&event.Note,
			&metadataBytes,
			&event.CreatedAt,
		); err != nil {
			return fmt.Errorf("scan reservation activity event: %w", err)
		}
		event.Metadata = map[string]any{}
		if len(metadataBytes) > 0 {
			_ = json.Unmarshal(metadataBytes, &event.Metadata)
		}
		item.ActivityHistory = append(item.ActivityHistory, event)
	}
	if err := activityRows.Err(); err != nil {
		return fmt.Errorf("iterate reservation activity history: %w", err)
	}
	return nil
}

func (r *Repository) GetWarehouseInventory(ctx context.Context, tenantID, reservationID string) (WarehouseInventory, error) {
	detail, err := r.Get(ctx, tenantID, reservationID)
	if err != nil {
		return WarehouseInventory{}, err
	}

	result := WarehouseInventory{
		ReservationID:        detail.ID,
		Status:               detail.Status,
		CanManageAssignments: detail.Status == "PREPARING",
		Complete:             detail.WarehouseComplete,
		RequiredAssetCount:   detail.RequiredAssetCount,
		AssignedAssetCount:   detail.AssignedAssetCount,
		Items:                make([]WarehouseItem, 0, len(detail.Items)),
	}

	for _, item := range detail.Items {
		warehouseItem := WarehouseItem{
			ReservationItemID:     item.ID,
			ResourceID:            item.ResourceID,
			ResourceName:          item.ResourceName,
			TrackIndividualAssets: item.TrackIndividualAssets,
			RequiredQuantity:      item.Quantity,
			AssignedQuantity:      item.AssignedQuantity,
			MissingQuantity:       item.MissingQuantity,
			Assignments:           activeAssignments(item.Assignments),
			AvailableAssets:       make([]AssignableAsset, 0),
		}
		if result.CanManageAssignments && item.TrackIndividualAssets && item.MissingQuantity > 0 {
			available, listErr := r.listAssignableAssets(
				ctx,
				tenantID,
				reservationID,
				item.ResourceID,
				detail.BlockStartAt,
				detail.BlockEndAt,
			)
			if listErr != nil {
				return WarehouseInventory{}, listErr
			}
			warehouseItem.AvailableAssets = available
		}
		result.Items = append(result.Items, warehouseItem)
	}
	return result, nil
}

func (r *Repository) listAssignableAssets(
	ctx context.Context,
	tenantID string,
	reservationID string,
	resourceID string,
	blockStartAt time.Time,
	blockEndAt time.Time,
) ([]AssignableAsset, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT a.id, a.resource_id, a.asset_code, a.serial_number,
		       a.physical_status, a.notes
		FROM assets a
		WHERE a.tenant_id = $1
		  AND a.resource_id = $2
		  AND a.physical_status = 'AVAILABLE'
		  AND NOT EXISTS (
			SELECT 1
			FROM reservation_assets own_ra
			WHERE own_ra.tenant_id = a.tenant_id
			  AND own_ra.reservation_id = $3
			  AND own_ra.asset_id = a.id
			  AND own_ra.released_at IS NULL
		  )
		  AND NOT EXISTS (
			SELECT 1
			FROM reservation_assets other_ra
			JOIN reservations other_r
			  ON other_r.tenant_id = other_ra.tenant_id
			 AND other_r.id = other_ra.reservation_id
			WHERE other_ra.tenant_id = a.tenant_id
			  AND other_ra.asset_id = a.id
			  AND other_ra.released_at IS NULL
			  AND other_r.id <> $3
			  AND other_r.status = ANY($4::text[])
			  AND other_r.block_start_at < $6
			  AND other_r.block_end_at > $5
		  )
		ORDER BY a.asset_code
	`, tenantID, resourceID, reservationID, blockingReservationStatuses, blockStartAt, blockEndAt)
	if err != nil {
		return nil, fmt.Errorf("list assignable assets: %w", err)
	}
	defer rows.Close()

	items := make([]AssignableAsset, 0)
	for rows.Next() {
		var item AssignableAsset
		if err := rows.Scan(
			&item.ID,
			&item.ResourceID,
			&item.AssetCode,
			&item.SerialNumber,
			&item.PhysicalStatus,
			&item.Notes,
		); err != nil {
			return nil, fmt.Errorf("scan assignable asset: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) AssignAsset(
	ctx context.Context,
	tenantID string,
	reservationID string,
	assetID string,
	actorID string,
) (Detail, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Detail{}, fmt.Errorf("begin asset assignment: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var status string
	var blockStartAt, blockEndAt time.Time
	if err := tx.QueryRow(ctx, `
		SELECT status, block_start_at, block_end_at
		FROM reservations
		WHERE tenant_id = $1 AND id = $2
		FOR UPDATE
	`, tenantID, reservationID).Scan(&status, &blockStartAt, &blockEndAt); errors.Is(err, pgx.ErrNoRows) {
		return Detail{}, ErrNotFound
	} else if err != nil {
		return Detail{}, fmt.Errorf("lock reservation for assignment: %w", err)
	}
	if status != "PREPARING" {
		return Detail{}, ErrWarehouseState
	}

	var resourceID, assetCode, physicalStatus string
	if err := tx.QueryRow(ctx, `
		SELECT resource_id, asset_code, physical_status
		FROM assets
		WHERE tenant_id = $1 AND id = $2
		FOR UPDATE
	`, tenantID, assetID).Scan(&resourceID, &assetCode, &physicalStatus); errors.Is(err, pgx.ErrNoRows) {
		return Detail{}, ErrAssetNotFound
	} else if err != nil {
		return Detail{}, fmt.Errorf("lock asset for assignment: %w", err)
	}
	if physicalStatus != "AVAILABLE" {
		return Detail{}, ErrAssetUnavailable
	}

	var reservationItemID string
	var requiredQuantity int
	var trackIndividualAssets bool
	if err := tx.QueryRow(ctx, `
		SELECT ri.id, ri.quantity, res.track_individual_assets
		FROM reservation_items ri
		JOIN resources res ON res.tenant_id = ri.tenant_id AND res.id = ri.resource_id
		WHERE ri.tenant_id = $1
		  AND ri.reservation_id = $2
		  AND ri.resource_id = $3
		FOR UPDATE OF ri
	`, tenantID, reservationID, resourceID).Scan(
		&reservationItemID,
		&requiredQuantity,
		&trackIndividualAssets,
	); errors.Is(err, pgx.ErrNoRows) {
		return Detail{}, ErrAssetResourceMismatch
	} else if err != nil {
		return Detail{}, fmt.Errorf("lock reservation item for assignment: %w", err)
	}
	if !trackIndividualAssets {
		return Detail{}, ErrAssetResourceMismatch
	}

	var alreadyAssigned bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM reservation_assets
			WHERE tenant_id = $1
			  AND reservation_id = $2
			  AND asset_id = $3
			  AND released_at IS NULL
		)
	`, tenantID, reservationID, assetID).Scan(&alreadyAssigned); err != nil {
		return Detail{}, fmt.Errorf("check existing asset assignment: %w", err)
	}
	if alreadyAssigned {
		return Detail{}, ErrAssetAlreadyAssigned
	}

	var assignedQuantity int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM reservation_assets
		WHERE tenant_id = $1
		  AND reservation_item_id = $2
		  AND released_at IS NULL
	`, tenantID, reservationItemID).Scan(&assignedQuantity); err != nil {
		return Detail{}, fmt.Errorf("count reservation item assignments: %w", err)
	}
	if assignedQuantity >= requiredQuantity {
		return Detail{}, ErrAssignmentCapacity
	}

	var conflict AssetConflictError
	conflict.AssetID = assetID
	conflict.AssetCode = assetCode
	conflictErr := tx.QueryRow(ctx, `
		SELECT other_r.id, other_r.reservation_number
		FROM reservation_assets other_ra
		JOIN reservations other_r
		  ON other_r.tenant_id = other_ra.tenant_id
		 AND other_r.id = other_ra.reservation_id
		WHERE other_ra.tenant_id = $1
		  AND other_ra.asset_id = $2
		  AND other_ra.released_at IS NULL
		  AND other_r.id <> $3
		  AND other_r.status = ANY($4::text[])
		  AND other_r.block_start_at < $6
		  AND other_r.block_end_at > $5
		ORDER BY other_r.block_start_at
		LIMIT 1
	`, tenantID, assetID, reservationID, blockingReservationStatuses, blockStartAt, blockEndAt).Scan(
		&conflict.ReservationID,
		&conflict.ReservationNumber,
	)
	if conflictErr == nil {
		return Detail{}, &conflict
	}
	if !errors.Is(conflictErr, pgx.ErrNoRows) {
		return Detail{}, fmt.Errorf("check overlapping asset assignment: %w", conflictErr)
	}

	var assignmentID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO reservation_assets (
			tenant_id, reservation_id, reservation_item_id, asset_id, assigned_by
		) VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, tenantID, reservationID, reservationItemID, assetID, actorID).Scan(&assignmentID); err != nil {
		return Detail{}, fmt.Errorf("insert asset assignment: %w", err)
	}
	if err := insertActivityEvent(ctx, tx, tenantID, reservationID, "ASSET_ASSIGNED", &assetID, actorID,
		"Unidad física asignada a la reserva.", map[string]any{
			"assignment_id": assignmentID,
			"asset_code":    assetCode,
			"resource_id":   resourceID,
		}); err != nil {
		return Detail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Detail{}, fmt.Errorf("commit asset assignment: %w", err)
	}
	return r.Get(ctx, tenantID, reservationID)
}

func (r *Repository) UnassignAsset(
	ctx context.Context,
	tenantID string,
	reservationID string,
	assetID string,
	actorID string,
) (Detail, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Detail{}, fmt.Errorf("begin asset unassignment: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var status string
	if err := tx.QueryRow(ctx, `
		SELECT status
		FROM reservations
		WHERE tenant_id = $1 AND id = $2
		FOR UPDATE
	`, tenantID, reservationID).Scan(&status); errors.Is(err, pgx.ErrNoRows) {
		return Detail{}, ErrNotFound
	} else if err != nil {
		return Detail{}, fmt.Errorf("lock reservation for unassignment: %w", err)
	}
	if status != "PREPARING" {
		return Detail{}, ErrWarehouseState
	}

	var assignmentID, assetCode string
	if err := tx.QueryRow(ctx, `
		SELECT ra.id, a.asset_code
		FROM reservation_assets ra
		JOIN assets a ON a.tenant_id = ra.tenant_id AND a.id = ra.asset_id
		WHERE ra.tenant_id = $1
		  AND ra.reservation_id = $2
		  AND ra.asset_id = $3
		  AND ra.released_at IS NULL
		FOR UPDATE OF ra, a
	`, tenantID, reservationID, assetID).Scan(&assignmentID, &assetCode); errors.Is(err, pgx.ErrNoRows) {
		return Detail{}, ErrAssignmentNotFound
	} else if err != nil {
		return Detail{}, fmt.Errorf("lock asset assignment for release: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE reservation_assets
		SET released_at = NOW(), released_by = $4, release_reason = 'MANUAL_UNASSIGN'
		WHERE tenant_id = $1 AND reservation_id = $2 AND asset_id = $3 AND released_at IS NULL
	`, tenantID, reservationID, assetID, actorID); err != nil {
		return Detail{}, fmt.Errorf("release asset assignment: %w", err)
	}
	if err := insertActivityEvent(ctx, tx, tenantID, reservationID, "ASSET_UNASSIGNED", &assetID, actorID,
		"Unidad física retirada de la preparación.", map[string]any{
			"assignment_id": assignmentID,
			"asset_code":    assetCode,
		}); err != nil {
		return Detail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Detail{}, fmt.Errorf("commit asset unassignment: %w", err)
	}
	return r.Get(ctx, tenantID, reservationID)
}

func (r *Repository) MarkReadyWithInventory(
	ctx context.Context,
	tenantID string,
	reservationID string,
	actorID string,
) (Detail, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Detail{}, fmt.Errorf("begin mark-ready operation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	current, err := lockReservationStatus(ctx, tx, tenantID, reservationID)
	if err != nil {
		return Detail{}, err
	}
	if current != "PREPARING" {
		return Detail{}, ErrInvalidTransition
	}
	gaps, err := inventoryGaps(ctx, tx, tenantID, reservationID)
	if err != nil {
		return Detail{}, err
	}
	if len(gaps) > 0 {
		return Detail{}, &InventoryIncompleteError{Items: gaps}
	}
	if err := updateReservationStatus(ctx, tx, tenantID, reservationID, current, "READY", actorID,
		"Todas las unidades físicas requeridas están asignadas."); err != nil {
		return Detail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Detail{}, fmt.Errorf("commit mark-ready operation: %w", err)
	}
	return r.Get(ctx, tenantID, reservationID)
}

func (r *Repository) CheckOutWithAssets(
	ctx context.Context,
	tenantID string,
	reservationID string,
	actorID string,
	notes string,
) (Detail, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Detail{}, fmt.Errorf("begin checkout operation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	current, err := lockReservationStatus(ctx, tx, tenantID, reservationID)
	if err != nil {
		return Detail{}, err
	}
	if current != "READY" {
		return Detail{}, ErrInvalidTransition
	}
	gaps, err := inventoryGaps(ctx, tx, tenantID, reservationID)
	if err != nil {
		return Detail{}, err
	}
	if len(gaps) > 0 {
		return Detail{}, &InventoryIncompleteError{Items: gaps}
	}

	assignments, err := lockActiveAssignments(ctx, tx, tenantID, reservationID)
	if err != nil {
		return Detail{}, err
	}
	for _, assignment := range assignments {
		if _, err := tx.Exec(ctx, `
			UPDATE reservation_assets
			SET checked_out_at = NOW(), checked_out_by = $4, checkout_notes = $5
			WHERE tenant_id = $1 AND reservation_id = $2 AND id = $3
		`, tenantID, reservationID, assignment.AssignmentID, actorID, notes); err != nil {
			return Detail{}, fmt.Errorf("mark asset checked out: %w", err)
		}
		assetID := assignment.AssetID
		if err := insertActivityEvent(ctx, tx, tenantID, reservationID, "ASSET_CHECKED_OUT", &assetID, actorID,
			"Unidad física entregada al cliente.", map[string]any{
				"assignment_id": assignment.AssignmentID,
				"asset_code":    assignment.AssetCode,
			}); err != nil {
			return Detail{}, err
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE reservations
		SET status = 'CHECKED_OUT', checked_out_at = NOW(), checked_out_by = $3, checkout_notes = $4
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, reservationID, actorID, notes); err != nil {
		return Detail{}, fmt.Errorf("update reservation checkout: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO reservation_status_history (
			tenant_id, reservation_id, from_status, to_status, actor_id, note
		) VALUES ($1, $2, 'READY', 'CHECKED_OUT', $3, $4)
	`, tenantID, reservationID, actorID, fmt.Sprintf("%d unidades entregadas.", len(assignments))); err != nil {
		return Detail{}, fmt.Errorf("insert checkout status history: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Detail{}, fmt.Errorf("commit checkout operation: %w", err)
	}
	return r.Get(ctx, tenantID, reservationID)
}

func (r *Repository) ReturnWithInspection(
	ctx context.Context,
	tenantID string,
	reservationID string,
	actorID string,
	input ReturnInput,
) (Detail, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Detail{}, fmt.Errorf("begin return operation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	current, err := lockReservationStatus(ctx, tx, tenantID, reservationID)
	if err != nil {
		return Detail{}, err
	}
	if current != "CHECKED_OUT" {
		return Detail{}, ErrInvalidTransition
	}

	assignments, err := lockActiveAssignments(ctx, tx, tenantID, reservationID)
	if err != nil {
		return Detail{}, err
	}
	for _, assignment := range assignments {
		if assignment.CheckedOutAt == nil {
			return Detail{}, ErrWarehouseState
		}
	}

	inputByAsset := make(map[string]ReturnAssetInput, len(input.Assets))
	for _, item := range input.Assets {
		inputByAsset[item.AssetID] = item
	}
	expected := make([]string, 0, len(assignments))
	missing := make([]string, 0)
	for _, assignment := range assignments {
		expected = append(expected, assignment.AssetID)
		if _, ok := inputByAsset[assignment.AssetID]; !ok {
			missing = append(missing, assignment.AssetID)
		}
	}
	unexpected := make([]string, 0)
	expectedSet := make(map[string]struct{}, len(expected))
	for _, assetID := range expected {
		expectedSet[assetID] = struct{}{}
	}
	for assetID := range inputByAsset {
		if _, ok := expectedSet[assetID]; !ok {
			unexpected = append(unexpected, assetID)
		}
	}
	if len(missing) > 0 || len(unexpected) > 0 {
		sort.Strings(expected)
		sort.Strings(missing)
		sort.Strings(unexpected)
		return Detail{}, &ReturnMismatchError{
			ExpectedAssetIDs:   expected,
			MissingAssetIDs:    missing,
			UnexpectedAssetIDs: unexpected,
		}
	}

	for _, assignment := range assignments {
		inspection := inputByAsset[assignment.AssetID]
		newPhysicalStatus := physicalStatusForReturn(inspection.Condition)
		if _, err := tx.Exec(ctx, `
			UPDATE assets
			SET physical_status = $3
			WHERE tenant_id = $1 AND id = $2
		`, tenantID, assignment.AssetID, newPhysicalStatus); err != nil {
			return Detail{}, fmt.Errorf("update returned asset status: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE reservation_assets
			SET returned_at = NOW(), returned_by = $4,
			    return_condition = $5, return_notes = $6
			WHERE tenant_id = $1 AND reservation_id = $2 AND id = $3
		`, tenantID, reservationID, assignment.AssignmentID, actorID, inspection.Condition, inspection.Notes); err != nil {
			return Detail{}, fmt.Errorf("record returned asset: %w", err)
		}
		assetID := assignment.AssetID
		if err := insertActivityEvent(ctx, tx, tenantID, reservationID, "ASSET_RETURNED", &assetID, actorID,
			"Unidad inspeccionada al regresar al almacén.", map[string]any{
				"assignment_id":       assignment.AssignmentID,
				"asset_code":          assignment.AssetCode,
				"condition":           inspection.Condition,
				"previous_status":     assignment.PhysicalStatus,
				"new_physical_status": newPhysicalStatus,
				"notes":               inspection.Notes,
			}); err != nil {
			return Detail{}, err
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE reservations
		SET status = 'RETURNED', returned_at = NOW(), returned_by = $3, return_notes = $4
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, reservationID, actorID, input.Notes); err != nil {
		return Detail{}, fmt.Errorf("update reservation return: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO reservation_status_history (
			tenant_id, reservation_id, from_status, to_status, actor_id, note
		) VALUES ($1, $2, 'CHECKED_OUT', 'RETURNED', $3, $4)
	`, tenantID, reservationID, actorID, fmt.Sprintf("%d unidades devueltas e inspeccionadas.", len(assignments))); err != nil {
		return Detail{}, fmt.Errorf("insert return status history: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Detail{}, fmt.Errorf("commit return operation: %w", err)
	}
	return r.Get(ctx, tenantID, reservationID)
}

func (r *Repository) CompleteAfterReturn(
	ctx context.Context,
	tenantID string,
	reservationID string,
	actorID string,
) (Detail, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Detail{}, fmt.Errorf("begin complete operation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	current, err := lockReservationStatus(ctx, tx, tenantID, reservationID)
	if err != nil {
		return Detail{}, err
	}
	if current != "RETURNED" {
		return Detail{}, ErrInvalidTransition
	}
	var outstanding int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM reservation_assets
		WHERE tenant_id = $1
		  AND reservation_id = $2
		  AND released_at IS NULL
		  AND returned_at IS NULL
	`, tenantID, reservationID).Scan(&outstanding); err != nil {
		return Detail{}, fmt.Errorf("count outstanding returned assets: %w", err)
	}
	if outstanding > 0 {
		return Detail{}, ErrAssetsNotReturned
	}
	if err := updateReservationStatus(ctx, tx, tenantID, reservationID, current, "COMPLETED", actorID,
		"Operación de almacén cerrada."); err != nil {
		return Detail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Detail{}, fmt.Errorf("commit complete operation: %w", err)
	}
	return r.Get(ctx, tenantID, reservationID)
}

func (r *Repository) CancelAndRelease(
	ctx context.Context,
	tenantID string,
	reservationID string,
	actorID string,
) (Detail, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Detail{}, fmt.Errorf("begin cancellation operation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	current, err := lockReservationStatus(ctx, tx, tenantID, reservationID)
	if err != nil {
		return Detail{}, err
	}
	if !contains([]string{"PENDING", "CONFIRMED", "PREPARING", "READY"}, current) {
		return Detail{}, ErrInvalidTransition
	}
	assignments, err := lockActiveAssignments(ctx, tx, tenantID, reservationID)
	if err != nil {
		return Detail{}, err
	}
	for _, assignment := range assignments {
		if _, err := tx.Exec(ctx, `
			UPDATE reservation_assets
			SET released_at = NOW(), released_by = $4, release_reason = 'RESERVATION_CANCELLED'
			WHERE tenant_id = $1 AND reservation_id = $2 AND id = $3
		`, tenantID, reservationID, assignment.AssignmentID, actorID); err != nil {
			return Detail{}, fmt.Errorf("release cancelled reservation assignment: %w", err)
		}
		assetID := assignment.AssetID
		if err := insertActivityEvent(ctx, tx, tenantID, reservationID, "ASSIGNMENTS_RELEASED", &assetID, actorID,
			"Asignación liberada por cancelación de la reserva.", map[string]any{
				"assignment_id": assignment.AssignmentID,
				"asset_code":    assignment.AssetCode,
			}); err != nil {
			return Detail{}, err
		}
	}
	if err := updateReservationStatus(ctx, tx, tenantID, reservationID, current, "CANCELLED", actorID,
		fmt.Sprintf("Reserva cancelada; %d asignaciones liberadas.", len(assignments))); err != nil {
		return Detail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Detail{}, fmt.Errorf("commit cancellation operation: %w", err)
	}
	return r.Get(ctx, tenantID, reservationID)
}

type lockedAssignment struct {
	AssignmentID   string
	AssetID        string
	AssetCode      string
	PhysicalStatus string
	CheckedOutAt   *time.Time
}

func lockActiveAssignments(
	ctx context.Context,
	tx pgx.Tx,
	tenantID string,
	reservationID string,
) ([]lockedAssignment, error) {
	rows, err := tx.Query(ctx, `
		SELECT ra.id, ra.asset_id, a.asset_code, a.physical_status, ra.checked_out_at
		FROM reservation_assets ra
		JOIN assets a ON a.tenant_id = ra.tenant_id AND a.id = ra.asset_id
		WHERE ra.tenant_id = $1
		  AND ra.reservation_id = $2
		  AND ra.released_at IS NULL
		ORDER BY ra.asset_id
		FOR UPDATE OF ra, a
	`, tenantID, reservationID)
	if err != nil {
		return nil, fmt.Errorf("lock active reservation assignments: %w", err)
	}
	defer rows.Close()
	items := make([]lockedAssignment, 0)
	for rows.Next() {
		var item lockedAssignment
		if err := rows.Scan(
			&item.AssignmentID,
			&item.AssetID,
			&item.AssetCode,
			&item.PhysicalStatus,
			&item.CheckedOutAt,
		); err != nil {
			return nil, fmt.Errorf("scan active reservation assignment: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func inventoryGaps(
	ctx context.Context,
	tx pgx.Tx,
	tenantID string,
	reservationID string,
) ([]InventoryGap, error) {
	rows, err := tx.Query(ctx, `
		SELECT
			ri.id,
			ri.resource_id,
			res.name,
			ri.quantity,
			COUNT(ra.id) FILTER (WHERE ra.released_at IS NULL)::int AS assigned_quantity
		FROM reservation_items ri
		JOIN resources res
		  ON res.tenant_id = ri.tenant_id
		 AND res.id = ri.resource_id
		LEFT JOIN reservation_assets ra
		  ON ra.tenant_id = ri.tenant_id
		 AND ra.reservation_item_id = ri.id
		WHERE ri.tenant_id = $1
		  AND ri.reservation_id = $2
		  AND res.track_individual_assets = TRUE
		GROUP BY ri.id, res.id
		HAVING COUNT(ra.id) FILTER (WHERE ra.released_at IS NULL) < ri.quantity
		ORDER BY res.name
	`, tenantID, reservationID)
	if err != nil {
		return nil, fmt.Errorf("check reservation inventory completeness: %w", err)
	}
	defer rows.Close()
	gaps := make([]InventoryGap, 0)
	for rows.Next() {
		var gap InventoryGap
		if err := rows.Scan(
			&gap.ReservationItemID,
			&gap.ResourceID,
			&gap.ResourceName,
			&gap.RequiredQuantity,
			&gap.AssignedQuantity,
		); err != nil {
			return nil, fmt.Errorf("scan reservation inventory gap: %w", err)
		}
		gap.MissingQuantity = gap.RequiredQuantity - gap.AssignedQuantity
		gaps = append(gaps, gap)
	}
	return gaps, rows.Err()
}

func lockReservationStatus(
	ctx context.Context,
	tx pgx.Tx,
	tenantID string,
	reservationID string,
) (string, error) {
	var current string
	if err := tx.QueryRow(ctx, `
		SELECT status
		FROM reservations
		WHERE tenant_id = $1 AND id = $2
		FOR UPDATE
	`, tenantID, reservationID).Scan(&current); errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	} else if err != nil {
		return "", fmt.Errorf("lock reservation status: %w", err)
	}
	return current, nil
}

func updateReservationStatus(
	ctx context.Context,
	tx pgx.Tx,
	tenantID string,
	reservationID string,
	fromStatus string,
	toStatus string,
	actorID string,
	note string,
) error {
	if _, err := tx.Exec(ctx, `
		UPDATE reservations
		SET status = $3
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, reservationID, toStatus); err != nil {
		return fmt.Errorf("update reservation status: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO reservation_status_history (
			tenant_id, reservation_id, from_status, to_status, actor_id, note
		) VALUES ($1, $2, $3, $4, $5, $6)
	`, tenantID, reservationID, fromStatus, toStatus, actorID, note); err != nil {
		return fmt.Errorf("insert reservation status history: %w", err)
	}
	return nil
}

func insertActivityEvent(
	ctx context.Context,
	tx pgx.Tx,
	tenantID string,
	reservationID string,
	eventType string,
	assetID *string,
	actorID string,
	note string,
	metadata map[string]any,
) error {
	if metadata == nil {
		metadata = map[string]any{}
	}
	payload, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal reservation activity metadata: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO reservation_activity_events (
			tenant_id, reservation_id, event_type, asset_id, actor_id, note, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
	`, tenantID, reservationID, eventType, assetID, actorID, note, string(payload)); err != nil {
		return fmt.Errorf("insert reservation activity event: %w", err)
	}
	return nil
}

func assignmentState(item AssignedAsset) string {
	switch {
	case item.ReleasedAt != nil:
		return "RELEASED"
	case item.ReturnedAt != nil:
		return "RETURNED"
	case item.CheckedOutAt != nil:
		return "CHECKED_OUT"
	default:
		return "ASSIGNED"
	}
}

func activeAssignments(items []AssignedAsset) []AssignedAsset {
	result := make([]AssignedAsset, 0, len(items))
	for _, item := range items {
		if item.ReleasedAt == nil {
			result = append(result, item)
		}
	}
	return result
}

func physicalStatusForReturn(condition string) string {
	switch condition {
	case "MAINTENANCE_REQUIRED":
		return "MAINTENANCE"
	case "DAMAGED":
		return "DAMAGED"
	case "LOST":
		return "LOST"
	default:
		return "AVAILABLE"
	}
}
