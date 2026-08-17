package availability

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var blockingStatuses = []string{"PENDING", "CONFIRMED", "PREPARING", "READY", "CHECKED_OUT"}

type queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type executor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Check(ctx context.Context, tenantID string, input NormalizedInput) (Result, error) {
	return CheckWithQuerier(ctx, r.pool, tenantID, input)
}

func CheckWithQuerier(ctx context.Context, q queryer, tenantID string, input NormalizedInput) (Result, error) {
	result := Result{
		StartAt:   input.StartAt,
		EndAt:     input.EndAt,
		Available: true,
		Items:     make([]ItemResult, 0, len(input.Items)),
	}

	for _, requested := range input.Items {
		item := ItemResult{
			ResourceID:        requested.ResourceID,
			RequestedQuantity: requested.Quantity,
		}
		err := q.QueryRow(ctx, `
			SELECT
			  r.name,
			  (SELECT COUNT(*)::int
			     FROM assets a
			    WHERE a.tenant_id = r.tenant_id
			      AND a.resource_id = r.id
			      AND a.physical_status = 'AVAILABLE') AS eligible_assets,
			  COALESCE((SELECT SUM(ri.quantity)::int
			     FROM reservation_items ri
			     JOIN reservations reservation
			       ON reservation.id = ri.reservation_id
			      AND reservation.tenant_id = ri.tenant_id
			    WHERE ri.tenant_id = r.tenant_id
			      AND ri.resource_id = r.id
			      AND reservation.status = ANY($5::text[])
			      AND reservation.block_start_at < $4
			      AND reservation.block_end_at > $3
			      AND ($6::uuid IS NULL OR reservation.id <> $6::uuid)), 0) AS reserved_quantity
			FROM resources r
			WHERE r.tenant_id = $1 AND r.id = $2 AND r.active = TRUE
		`, tenantID, requested.ResourceID, input.StartAt, input.EndAt, blockingStatuses, input.ExcludeReservationID).Scan(
			&item.ResourceName,
			&item.EligibleAssets,
			&item.ReservedQuantity,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return Result{}, &ResourceNotFoundError{ResourceID: requested.ResourceID}
		}
		if err != nil {
			return Result{}, fmt.Errorf("calculate availability for %s: %w", requested.ResourceID, err)
		}
		item.AvailableQuantity = item.EligibleAssets - item.ReservedQuantity
		if item.AvailableQuantity < 0 {
			item.AvailableQuantity = 0
		}
		item.CanFulfill = item.AvailableQuantity >= item.RequestedQuantity
		if !item.CanFulfill {
			result.Available = false
		}
		result.Items = append(result.Items, item)
	}
	return result, nil
}

func LockResources(ctx context.Context, tx executor, tenantID string, items []NormalizedItem) error {
	resourceIDs := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if _, exists := seen[item.ResourceID]; exists {
			continue
		}
		seen[item.ResourceID] = struct{}{}
		resourceIDs = append(resourceIDs, item.ResourceID)
	}
	sort.Strings(resourceIDs)
	for _, resourceID := range resourceIDs {
		key := tenantID + ":" + resourceID
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, key); err != nil {
			return fmt.Errorf("lock resource %s: %w", resourceID, err)
		}
	}
	return nil
}
