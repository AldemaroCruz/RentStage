package catalog

import (
	"context"
	"encoding/json"
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

func (r *Repository) ListCategories(ctx context.Context, tenantID string) ([]Category, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, c.name, c.description, COUNT(r.id)::int, c.created_at, c.updated_at
		FROM categories c
		LEFT JOIN resources r ON r.category_id = c.id AND r.tenant_id = c.tenant_id AND r.active = TRUE
		WHERE c.tenant_id = $1
		GROUP BY c.id
		ORDER BY c.name
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	items := make([]Category, 0)
	for rows.Next() {
		var item Category
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Description,
			&item.ResourceCount,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) CreateCategory(ctx context.Context, tenantID string, input CreateCategoryInput) (Category, error) {
	var item Category
	err := r.pool.QueryRow(ctx, `
		INSERT INTO categories (tenant_id, name, description)
		VALUES ($1, $2, $3)
		RETURNING id, name, description, 0, created_at, updated_at
	`, tenantID, input.Name, input.Description).Scan(
		&item.ID,
		&item.Name,
		&item.Description,
		&item.ResourceCount,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if isPgCode(err, "23505") {
		return Category{}, ErrConflict
	}
	if err != nil {
		return Category{}, fmt.Errorf("create category: %w", err)
	}
	return item, nil
}

func (r *Repository) DeleteCategory(ctx context.Context, tenantID, categoryID string) error {
	command, err := r.pool.Exec(ctx, `DELETE FROM categories WHERE tenant_id = $1 AND id = $2`, tenantID, categoryID)
	if isPgCode(err, "23503") {
		return ErrConflict
	}
	if err != nil {
		return fmt.Errorf("delete category: %w", err)
	}
	if command.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) ListResources(
	ctx context.Context,
	tenantID string,
	search string,
	categoryID string,
	activeOnly bool,
) ([]Resource, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			r.id,
			r.category_id,
			c.name,
			r.resource_type,
			r.name,
			r.description,
			r.sku,
			r.base_price::float8,
			r.pricing_unit,
			r.deposit_amount::float8,
			r.track_individual_assets,
			r.active,
			r.metadata,
			COUNT(a.id)::int AS asset_count,
			COUNT(a.id) FILTER (WHERE a.physical_status = 'AVAILABLE')::int AS available_asset_count,
			COUNT(a.id) FILTER (WHERE a.physical_status IN ('MAINTENANCE', 'DAMAGED', 'LOST'))::int AS attention_asset_count,
			r.created_at,
			r.updated_at
		FROM resources r
		LEFT JOIN categories c ON c.id = r.category_id AND c.tenant_id = r.tenant_id
		LEFT JOIN assets a ON a.resource_id = r.id AND a.tenant_id = r.tenant_id AND a.physical_status <> 'RETIRED'
		WHERE r.tenant_id = $1
		  AND ($2 = '' OR r.name ILIKE '%' || $2 || '%' OR COALESCE(r.sku, '') ILIKE '%' || $2 || '%')
		  AND (NULLIF($3, '') IS NULL OR r.category_id = NULLIF($3, '')::uuid)
		  AND ($4 = FALSE OR r.active = TRUE)
		GROUP BY r.id, c.name
		ORDER BY r.active DESC, r.name
	`, tenantID, search, categoryID, activeOnly)
	if err != nil {
		return nil, fmt.Errorf("list resources: %w", err)
	}
	defer rows.Close()

	items := make([]Resource, 0)
	for rows.Next() {
		item, err := scanResource(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) GetResource(ctx context.Context, tenantID, resourceID string) (Resource, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT
			r.id,
			r.category_id,
			c.name,
			r.resource_type,
			r.name,
			r.description,
			r.sku,
			r.base_price::float8,
			r.pricing_unit,
			r.deposit_amount::float8,
			r.track_individual_assets,
			r.active,
			r.metadata,
			COUNT(a.id)::int,
			COUNT(a.id) FILTER (WHERE a.physical_status = 'AVAILABLE')::int,
			COUNT(a.id) FILTER (WHERE a.physical_status IN ('MAINTENANCE', 'DAMAGED', 'LOST'))::int,
			r.created_at,
			r.updated_at
		FROM resources r
		LEFT JOIN categories c ON c.id = r.category_id AND c.tenant_id = r.tenant_id
		LEFT JOIN assets a ON a.resource_id = r.id AND a.tenant_id = r.tenant_id AND a.physical_status <> 'RETIRED'
		WHERE r.tenant_id = $1 AND r.id = $2
		GROUP BY r.id, c.name
	`, tenantID, resourceID)
	item, err := scanResource(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Resource{}, ErrNotFound
	}
	if err != nil {
		return Resource{}, fmt.Errorf("get resource: %w", err)
	}
	return item, nil
}

func (r *Repository) CreateResource(ctx context.Context, tenantID string, input CreateResourceInput) (Resource, error) {
	metadata, err := json.Marshal(input.Metadata)
	if err != nil {
		return Resource{}, fmt.Errorf("marshal metadata: %w", err)
	}

	var resourceID string
	err = r.pool.QueryRow(ctx, `
		INSERT INTO resources (
			tenant_id, category_id, resource_type, name, description, sku,
			base_price, pricing_unit, deposit_amount, track_individual_assets, active, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12::jsonb)
		RETURNING id
	`,
		tenantID,
		input.CategoryID,
		input.ResourceType,
		input.Name,
		input.Description,
		input.SKU,
		input.BasePrice,
		input.PricingUnit,
		input.DepositAmount,
		*input.TrackIndividualAssets,
		*input.Active,
		string(metadata),
	).Scan(&resourceID)
	if isPgCode(err, "23505") {
		return Resource{}, ErrConflict
	}
	if isPgCode(err, "23503") {
		return Resource{}, ErrNotFound
	}
	if err != nil {
		return Resource{}, fmt.Errorf("create resource: %w", err)
	}
	return r.GetResource(ctx, tenantID, resourceID)
}

func (r *Repository) UpdateResource(ctx context.Context, tenantID, resourceID string, input CreateResourceInput) (Resource, error) {
	metadata, err := json.Marshal(input.Metadata)
	if err != nil {
		return Resource{}, fmt.Errorf("marshal metadata: %w", err)
	}

	command, err := r.pool.Exec(ctx, `
		UPDATE resources SET
			category_id = $3,
			resource_type = $4,
			name = $5,
			description = $6,
			sku = $7,
			base_price = $8,
			pricing_unit = $9,
			deposit_amount = $10,
			track_individual_assets = $11,
			active = $12,
			public_visible = CASE WHEN $12 THEN public_visible ELSE FALSE END,
			public_featured = CASE WHEN $12 THEN public_featured ELSE FALSE END,
			metadata = $13::jsonb
		WHERE tenant_id = $1 AND id = $2
	`,
		tenantID,
		resourceID,
		input.CategoryID,
		input.ResourceType,
		input.Name,
		input.Description,
		input.SKU,
		input.BasePrice,
		input.PricingUnit,
		input.DepositAmount,
		*input.TrackIndividualAssets,
		*input.Active,
		string(metadata),
	)
	if isPgCode(err, "23505") {
		return Resource{}, ErrConflict
	}
	if isPgCode(err, "23503") {
		return Resource{}, ErrNotFound
	}
	if err != nil {
		return Resource{}, fmt.Errorf("update resource: %w", err)
	}
	if command.RowsAffected() == 0 {
		return Resource{}, ErrNotFound
	}
	return r.GetResource(ctx, tenantID, resourceID)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanResource(row rowScanner) (Resource, error) {
	var item Resource
	var metadataBytes []byte
	if err := row.Scan(
		&item.ID,
		&item.CategoryID,
		&item.CategoryName,
		&item.ResourceType,
		&item.Name,
		&item.Description,
		&item.SKU,
		&item.BasePrice,
		&item.PricingUnit,
		&item.DepositAmount,
		&item.TrackIndividualAssets,
		&item.Active,
		&metadataBytes,
		&item.AssetCount,
		&item.AvailableAssetCount,
		&item.AttentionAssetCount,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return Resource{}, err
	}
	if err := json.Unmarshal(metadataBytes, &item.Metadata); err != nil {
		item.Metadata = map[string]any{}
	}
	return item, nil
}

func isPgCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}
