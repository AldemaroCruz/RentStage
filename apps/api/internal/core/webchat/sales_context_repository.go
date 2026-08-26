package webchat

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

const draftSalesSettingsQuery = `
	SELECT
		tenant.currency,
		settings.show_prices,
		settings.show_resources,
		settings.quote_requests_enabled
	FROM tenants tenant
	JOIN public_catalog_settings settings
	  ON settings.tenant_id = tenant.id
	WHERE tenant.id = $1
	  AND tenant.status = 'ACTIVE'
	  AND settings.enabled = TRUE
	  AND settings.web_chat_enabled = TRUE
`

const draftSalesPackagesQuery = `
	WITH package_totals AS (
		SELECT
			pkg.id,
			pkg.name,
			LEFT(
				COALESCE(
					NULLIF(BTRIM(pkg.description), ''),
					pkg.name
				),
				$3
			) AS description,
			pkg.guest_capacity,
			pkg.public_featured,
			pkg.public_sort_order,
			pkg.pricing_mode,
			pkg.fixed_price,
			COUNT(package_item.id)::int AS item_count,
			COUNT(package_item.id) FILTER (
				WHERE resource.active = FALSE
			)::int AS inactive_item_count,
			COALESCE(
				SUM(
					package_item.quantity * COALESCE(
						package_item.unit_price_override,
						resource.base_price
					)
				),
				0
			) AS calculated_price
		FROM packages pkg
		LEFT JOIN package_items package_item
		  ON package_item.tenant_id = pkg.tenant_id
		 AND package_item.package_id = pkg.id
		LEFT JOIN resources resource
		  ON resource.tenant_id = package_item.tenant_id
		 AND resource.id = package_item.resource_id
		WHERE pkg.tenant_id = $1
		  AND pkg.active = TRUE
		  AND pkg.public_visible = TRUE
		GROUP BY pkg.id
	)
	SELECT
		name,
		description,
		guest_capacity,
		CASE
			WHEN $2::boolean = FALSE THEN NULL
			WHEN pricing_mode = 'FIXED' THEN
				COALESCE(fixed_price, 0)::float8
			ELSE calculated_price::float8
		END AS public_price
	FROM package_totals
	WHERE item_count > 0
	  AND inactive_item_count = 0
	ORDER BY public_featured DESC, public_sort_order, name
	LIMIT $4
`

const draftSalesResourcesQuery = `
	SELECT
		LEFT(BTRIM(resource.name), $3),
		LEFT(
			COALESCE(
				NULLIF(BTRIM(resource.public_description), ''),
				NULLIF(BTRIM(resource.description), ''),
				resource.name
			),
			$4
		) AS description,
		LEFT(COALESCE(BTRIM(category.name), ''), $5),
		resource.resource_type,
		resource.pricing_unit,
		CASE
			WHEN $2::boolean THEN resource.base_price::float8
			ELSE NULL
		END AS public_price
	FROM resources resource
	LEFT JOIN categories category
	  ON category.tenant_id = resource.tenant_id
	 AND category.id = resource.category_id
	WHERE resource.tenant_id = $1
	  AND resource.active = TRUE
	  AND resource.public_visible = TRUE
	  AND resource.public_slug IS NOT NULL
	ORDER BY
		resource.public_featured DESC,
		resource.public_sort_order,
		resource.name
	LIMIT $6
`

func (r *Repository) LoadDraftSalesContext(
	ctx context.Context,
	tenantID string,
) (DraftSalesContext, error) {
	var salesContext DraftSalesContext

	err := r.pool.QueryRow(
		ctx,
		draftSalesSettingsQuery,
		tenantID,
	).Scan(
		&salesContext.Currency,
		&salesContext.ShowPrices,
		&salesContext.ShowResources,
		&salesContext.QuoteRequestsEnabled,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return DraftSalesContext{}, ErrDisabled
	}
	if err != nil {
		return DraftSalesContext{}, fmt.Errorf(
			"load web chat sales settings: %w",
			err,
		)
	}

	salesContext.Packages, err = r.loadDraftSalesPackages(
		ctx,
		tenantID,
		salesContext.ShowPrices,
	)
	if err != nil {
		return DraftSalesContext{}, err
	}

	if salesContext.ShowResources {
		salesContext.Resources, err = r.loadDraftSalesResources(
			ctx,
			tenantID,
			salesContext.ShowPrices,
		)
		if err != nil {
			return DraftSalesContext{}, err
		}
	} else {
		salesContext.Resources = make([]DraftSalesResource, 0)
	}

	salesContext, err = NormalizeDraftSalesContext(salesContext)
	if err != nil {
		return DraftSalesContext{}, fmt.Errorf(
			"normalize web chat sales context: %w",
			err,
		)
	}

	return salesContext, nil
}

func (r *Repository) loadDraftSalesPackages(
	ctx context.Context,
	tenantID string,
	showPrices bool,
) ([]DraftSalesPackage, error) {
	rows, err := r.pool.Query(
		ctx,
		draftSalesPackagesQuery,
		tenantID,
		showPrices,
		MaximumDraftSalesDescriptionRunes,
		MaximumDraftSalesPackages,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"load published packages for web chat draft: %w",
			err,
		)
	}
	defer rows.Close()

	items := make([]DraftSalesPackage, 0)
	for rows.Next() {
		var item DraftSalesPackage
		if err := rows.Scan(
			&item.Name,
			&item.Description,
			&item.GuestCapacity,
			&item.Price,
		); err != nil {
			return nil, fmt.Errorf(
				"scan published package for web chat draft: %w",
				err,
			)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate published packages for web chat draft: %w",
			err,
		)
	}

	return items, nil
}

func (r *Repository) loadDraftSalesResources(
	ctx context.Context,
	tenantID string,
	showPrices bool,
) ([]DraftSalesResource, error) {
	rows, err := r.pool.Query(
		ctx,
		draftSalesResourcesQuery,
		tenantID,
		showPrices,
		MaximumDraftSalesNameRunes,
		MaximumDraftSalesDescriptionRunes,
		MaximumDraftSalesMetadataRunes,
		MaximumDraftSalesResources,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"load published resources for web chat draft: %w",
			err,
		)
	}
	defer rows.Close()

	items := make([]DraftSalesResource, 0)
	for rows.Next() {
		var item DraftSalesResource
		if err := rows.Scan(
			&item.Name,
			&item.Description,
			&item.Category,
			&item.Type,
			&item.PricingUnit,
			&item.Price,
		); err != nil {
			return nil, fmt.Errorf(
				"scan published resource for web chat draft: %w",
				err,
			)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate published resources for web chat draft: %w",
			err,
		)
	}

	return items, nil
}
