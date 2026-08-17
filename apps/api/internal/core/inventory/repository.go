package inventory

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) ListAssets(ctx context.Context, tenantID, resourceID string) ([]Asset, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT a.id, a.resource_id, r.name, a.asset_code, a.serial_number,
		       a.physical_status, a.purchase_date, a.purchase_price::float8,
		       a.notes, a.created_at, a.updated_at
		FROM assets a
		JOIN resources r ON r.id = a.resource_id AND r.tenant_id = a.tenant_id
		WHERE a.tenant_id = $1 AND a.resource_id = $2
		ORDER BY
		  CASE a.physical_status
		    WHEN 'AVAILABLE' THEN 1
		    WHEN 'MAINTENANCE' THEN 2
		    WHEN 'DAMAGED' THEN 3
		    WHEN 'LOST' THEN 4
		    ELSE 5
		  END,
		  a.asset_code
	`, tenantID, resourceID)
	if err != nil {
		return nil, fmt.Errorf("list assets: %w", err)
	}
	defer rows.Close()

	items := make([]Asset, 0)
	for rows.Next() {
		item, err := scanAsset(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) GetAsset(ctx context.Context, tenantID, assetID string) (Asset, error) {
	item, err := scanAsset(r.pool.QueryRow(ctx, `
		SELECT a.id, a.resource_id, r.name, a.asset_code, a.serial_number,
		       a.physical_status, a.purchase_date, a.purchase_price::float8,
		       a.notes, a.created_at, a.updated_at
		FROM assets a
		JOIN resources r ON r.id = a.resource_id AND r.tenant_id = a.tenant_id
		WHERE a.tenant_id = $1 AND a.id = $2
	`, tenantID, assetID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Asset{}, ErrNotFound
	}
	if err != nil {
		return Asset{}, fmt.Errorf("get asset: %w", err)
	}
	return item, nil
}

func (r *Repository) CreateAsset(
	ctx context.Context,
	tenantID string,
	resourceID string,
	input normalizedAssetInput,
) (Asset, error) {
	var assetID string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO assets (
			tenant_id, resource_id, asset_code, serial_number, physical_status,
			purchase_date, purchase_price, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`,
		tenantID,
		resourceID,
		input.AssetCode,
		input.SerialNumber,
		input.PhysicalStatus,
		input.PurchaseDate,
		input.PurchasePrice,
		input.Notes,
	).Scan(&assetID)
	if isPgCode(err, "23505") {
		return Asset{}, ErrConflict
	}
	if isPgCode(err, "23503") {
		return Asset{}, ErrNotFound
	}
	if err != nil {
		return Asset{}, fmt.Errorf("create asset: %w", err)
	}
	return r.GetAsset(ctx, tenantID, assetID)
}

func (r *Repository) UpdateAsset(
	ctx context.Context,
	tenantID string,
	assetID string,
	input normalizedAssetInput,
) (Asset, error) {
	command, err := r.pool.Exec(ctx, `
		UPDATE assets SET
			asset_code = $3,
			serial_number = $4,
			physical_status = $5,
			purchase_date = $6,
			purchase_price = $7,
			notes = $8
		WHERE tenant_id = $1 AND id = $2
	`,
		tenantID,
		assetID,
		input.AssetCode,
		input.SerialNumber,
		input.PhysicalStatus,
		input.PurchaseDate,
		input.PurchasePrice,
		input.Notes,
	)
	if isPgCode(err, "23505") {
		return Asset{}, ErrConflict
	}
	if err != nil {
		return Asset{}, fmt.Errorf("update asset: %w", err)
	}
	if command.RowsAffected() == 0 {
		return Asset{}, ErrNotFound
	}
	return r.GetAsset(ctx, tenantID, assetID)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAsset(row rowScanner) (Asset, error) {
	var item Asset
	var purchaseDate *time.Time
	if err := row.Scan(
		&item.ID,
		&item.ResourceID,
		&item.ResourceName,
		&item.AssetCode,
		&item.SerialNumber,
		&item.PhysicalStatus,
		&purchaseDate,
		&item.PurchasePrice,
		&item.Notes,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return Asset{}, err
	}
	item.PurchaseDate = purchaseDate
	return item, nil
}

func isPgCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}
