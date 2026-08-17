package packages

import (
	"context"
	"errors"
	"fmt"

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

const summarySelect = `
	WITH package_totals AS (
		SELECT
			p.id,
			p.tenant_id,
			p.name,
			p.slug,
			p.description,
			p.guest_capacity,
			p.pricing_mode,
			p.fixed_price::float8 AS fixed_price,
			p.image_url,
			p.public_visible,
			p.public_featured,
			p.public_sort_order,
			p.active,
			COUNT(pi.id)::int AS item_count,
			COALESCE(SUM(pi.quantity), 0)::int AS total_quantity,
			COUNT(pi.id) FILTER (WHERE resource.active = FALSE)::int AS unavailable_item_count,
			COALESCE(SUM(pi.quantity * COALESCE(pi.unit_price_override, resource.base_price)), 0)::float8 AS calculated_price,
			p.created_at,
			p.updated_at
		FROM packages p
		LEFT JOIN package_items pi
			ON pi.tenant_id = p.tenant_id AND pi.package_id = p.id
		LEFT JOIN resources resource
			ON resource.tenant_id = pi.tenant_id AND resource.id = pi.resource_id
		GROUP BY p.id
	)
	SELECT
		id,
		name,
		slug,
		description,
		guest_capacity,
		pricing_mode,
		fixed_price,
		image_url,
		public_visible,
		public_featured,
		public_sort_order,
		calculated_price,
		CASE WHEN pricing_mode = 'FIXED' THEN COALESCE(fixed_price, 0) ELSE calculated_price END::float8 AS effective_price,
		GREATEST(calculated_price - CASE WHEN pricing_mode = 'FIXED' THEN COALESCE(fixed_price, 0) ELSE calculated_price END, 0)::float8 AS discount_value,
		GREATEST(CASE WHEN pricing_mode = 'FIXED' THEN COALESCE(fixed_price, 0) ELSE calculated_price END - calculated_price, 0)::float8 AS surcharge_value,
		item_count,
		total_quantity,
		unavailable_item_count,
		(active AND item_count > 0 AND unavailable_item_count = 0) AS ready,
		active,
		created_at,
		updated_at
	FROM package_totals
`

func (r *Repository) List(ctx context.Context, tenantID, search, activeFilter string) ([]Summary, error) {
	query := summarySelect + `
	WHERE tenant_id = $1
	  AND (
		$2 = '' OR
		name ILIKE '%' || $2 || '%' OR
		slug ILIKE '%' || $2 || '%' OR
		description ILIKE '%' || $2 || '%' OR
		EXISTS (
			SELECT 1
			FROM package_items search_item
			JOIN resources search_resource
			  ON search_resource.tenant_id = search_item.tenant_id
			 AND search_resource.id = search_item.resource_id
			WHERE search_item.tenant_id = package_totals.tenant_id
			  AND search_item.package_id = package_totals.id
			  AND search_resource.name ILIKE '%' || $2 || '%'
		)
	  )
	  AND (
		$3 = '' OR $3 = 'all' OR
		($3 = 'true' AND active = TRUE) OR
		($3 = 'false' AND active = FALSE)
	  )
	ORDER BY active DESC, updated_at DESC, name
	`
	rows, err := r.pool.Query(ctx, query, tenantID, search, activeFilter)
	if err != nil {
		return nil, fmt.Errorf("list packages: %w", err)
	}
	defer rows.Close()

	items := make([]Summary, 0)
	for rows.Next() {
		item, scanErr := scanSummary(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan package summary: %w", scanErr)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate packages: %w", err)
	}
	return items, nil
}

func (r *Repository) Get(ctx context.Context, tenantID, packageID string) (Detail, error) {
	// Package totals and their ordered item composition must come from one
	// snapshot. Otherwise a concurrent package or catalog update could produce a
	// quote template whose summary price does not match its item lines.
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.RepeatableRead,
		AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		return Detail{}, fmt.Errorf("begin package read: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	query := summarySelect + ` WHERE tenant_id = $1 AND id = $2`
	summary, err := scanSummary(tx.QueryRow(ctx, query, tenantID, packageID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Detail{}, ErrNotFound
	}
	if err != nil {
		return Detail{}, fmt.Errorf("get package: %w", err)
	}

	rows, err := tx.Query(ctx, `
		SELECT
			pi.id,
			pi.resource_id,
			resource.name,
			resource.resource_type,
			resource.pricing_unit,
			resource.active,
			COALESCE(NULLIF(BTRIM(pi.description), ''), resource.name) AS description,
			pi.quantity,
			resource.base_price::float8,
			pi.unit_price_override::float8,
			COALESCE(pi.unit_price_override, resource.base_price)::float8 AS unit_price,
			(pi.quantity * COALESCE(pi.unit_price_override, resource.base_price))::float8 AS line_total,
			(
				SELECT COUNT(*)::int FROM assets asset
				WHERE asset.tenant_id = resource.tenant_id
				  AND asset.resource_id = resource.id
				  AND asset.physical_status <> 'RETIRED'
			) AS asset_count,
			(
				SELECT COUNT(*)::int FROM assets asset
				WHERE asset.tenant_id = resource.tenant_id
				  AND asset.resource_id = resource.id
				  AND asset.physical_status = 'AVAILABLE'
			) AS available_asset_count,
			(
				SELECT COUNT(*)::int FROM assets asset
				WHERE asset.tenant_id = resource.tenant_id
				  AND asset.resource_id = resource.id
				  AND asset.physical_status IN ('MAINTENANCE', 'DAMAGED', 'LOST')
			) AS attention_asset_count,
			pi.sort_order,
			pi.created_at,
			pi.updated_at
		FROM package_items pi
		JOIN resources resource
		  ON resource.tenant_id = pi.tenant_id AND resource.id = pi.resource_id
		WHERE pi.tenant_id = $1 AND pi.package_id = $2
		ORDER BY pi.sort_order, pi.created_at, pi.id
	`, tenantID, packageID)
	if err != nil {
		return Detail{}, fmt.Errorf("list package items: %w", err)
	}
	defer rows.Close()

	detail := Detail{Summary: summary, Items: make([]Item, 0, summary.ItemCount)}
	for rows.Next() {
		var item Item
		if err := rows.Scan(
			&item.ID,
			&item.ResourceID,
			&item.ResourceName,
			&item.ResourceType,
			&item.PricingUnit,
			&item.ResourceActive,
			&item.Description,
			&item.Quantity,
			&item.BasePrice,
			&item.UnitPriceOverride,
			&item.UnitPrice,
			&item.LineTotal,
			&item.AssetCount,
			&item.AvailableAssetCount,
			&item.AttentionAssetCount,
			&item.SortOrder,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return Detail{}, fmt.Errorf("scan package item: %w", err)
		}
		detail.Items = append(detail.Items, item)
	}
	if err := rows.Err(); err != nil {
		return Detail{}, fmt.Errorf("iterate package items: %w", err)
	}
	rows.Close()
	if err := tx.Commit(ctx); err != nil {
		return Detail{}, fmt.Errorf("commit package read: %w", err)
	}
	return detail, nil
}

func (r *Repository) GetBySlug(ctx context.Context, tenantID, slug string) (Detail, error) {
	var packageID string
	err := r.pool.QueryRow(ctx, `
		SELECT id FROM packages WHERE tenant_id = $1 AND slug = $2
	`, tenantID, slug).Scan(&packageID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Detail{}, ErrNotFound
	}
	if err != nil {
		return Detail{}, fmt.Errorf("get package by slug: %w", err)
	}
	return r.Get(ctx, tenantID, packageID)
}

func (r *Repository) Create(ctx context.Context, tenantID string, input normalizedInput) (Detail, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Detail{}, fmt.Errorf("begin create package: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var packageID string
	err = tx.QueryRow(ctx, `
		INSERT INTO packages (
			tenant_id, name, slug, description, guest_capacity,
			pricing_mode, fixed_price, image_url, active
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`, tenantID, input.Name, input.Slug, input.Description, input.GuestCapacity,
		input.PricingMode, input.FixedPrice, input.ImageURL, input.Active).Scan(&packageID)
	if isPgCode(err, "23505") {
		return Detail{}, ErrConflict
	}
	if err != nil {
		return Detail{}, fmt.Errorf("insert package: %w", err)
	}
	if err := replaceItems(ctx, tx, tenantID, packageID, input.Items); err != nil {
		return Detail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Detail{}, fmt.Errorf("commit create package: %w", err)
	}
	return r.Get(ctx, tenantID, packageID)
}

func (r *Repository) Update(ctx context.Context, tenantID, packageID string, input normalizedInput) (Detail, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Detail{}, fmt.Errorf("begin update package: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	command, err := tx.Exec(ctx, `
		UPDATE packages SET
			name = $3,
			slug = $4,
			description = $5,
			guest_capacity = $6,
			pricing_mode = $7,
			fixed_price = $8,
			image_url = $9,
			active = $10,
			public_visible = CASE WHEN $10 THEN public_visible ELSE FALSE END,
			public_featured = CASE WHEN $10 THEN public_featured ELSE FALSE END
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, packageID, input.Name, input.Slug, input.Description, input.GuestCapacity,
		input.PricingMode, input.FixedPrice, input.ImageURL, input.Active)
	if isPgCode(err, "23505") {
		return Detail{}, ErrConflict
	}
	if err != nil {
		return Detail{}, fmt.Errorf("update package: %w", err)
	}
	if command.RowsAffected() == 0 {
		return Detail{}, ErrNotFound
	}
	if _, err := tx.Exec(ctx, `DELETE FROM package_items WHERE tenant_id = $1 AND package_id = $2`, tenantID, packageID); err != nil {
		return Detail{}, fmt.Errorf("delete package items: %w", err)
	}
	if err := replaceItems(ctx, tx, tenantID, packageID, input.Items); err != nil {
		return Detail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Detail{}, fmt.Errorf("commit update package: %w", err)
	}
	return r.Get(ctx, tenantID, packageID)
}

func (r *Repository) Archive(ctx context.Context, tenantID, packageID string) (Detail, error) {
	command, err := r.pool.Exec(ctx, `
		UPDATE packages SET active = FALSE, public_visible = FALSE, public_featured = FALSE WHERE tenant_id = $1 AND id = $2
	`, tenantID, packageID)
	if err != nil {
		return Detail{}, fmt.Errorf("archive package: %w", err)
	}
	if command.RowsAffected() == 0 {
		return Detail{}, ErrNotFound
	}
	return r.Get(ctx, tenantID, packageID)
}

func replaceItems(ctx context.Context, tx pgx.Tx, tenantID, packageID string, items []normalizedItem) error {
	for _, item := range items {
		command, err := tx.Exec(ctx, `
			INSERT INTO package_items (
				tenant_id, package_id, resource_id, description,
				quantity, unit_price_override, sort_order
			)
			SELECT $1, $2, resource.id, $4, $5, $6, $7
			FROM resources resource
			WHERE resource.tenant_id = $1
			  AND resource.id = $3
			  AND resource.active = TRUE
		`, tenantID, packageID, item.ResourceID, item.Description,
			item.Quantity, item.UnitPriceOverride, item.SortOrder)
		if isPgCode(err, "23505") {
			return ErrConflict
		}
		if err != nil {
			return fmt.Errorf("insert package item: %w", err)
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
		&item.Name,
		&item.Slug,
		&item.Description,
		&item.GuestCapacity,
		&item.PricingMode,
		&item.FixedPrice,
		&item.ImageURL,
		&item.PublicVisible,
		&item.PublicFeatured,
		&item.PublicSortOrder,
		&item.CalculatedPrice,
		&item.EffectivePrice,
		&item.DiscountValue,
		&item.SurchargeValue,
		&item.ItemCount,
		&item.TotalQuantity,
		&item.UnavailableItemCount,
		&item.Ready,
		&item.Active,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return Summary{}, err
	}
	return item, nil
}

func isPgCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}
