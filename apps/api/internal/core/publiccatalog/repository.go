package publiccatalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rentstage/rentstage/apps/api/internal/core/availability"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) EnsureSettings(ctx context.Context, tenantID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO public_catalog_settings (tenant_id)
		VALUES ($1)
		ON CONFLICT (tenant_id) DO NOTHING
	`, tenantID)
	if err != nil {
		return fmt.Errorf("ensure public catalog settings: %w", err)
	}
	return nil
}

func (r *Repository) GetSettings(ctx context.Context, tenantID string) (Settings, error) {
	if err := r.EnsureSettings(ctx, tenantID); err != nil {
		return Settings{}, err
	}
	return scanSettings(r.pool.QueryRow(ctx, `
		SELECT tenant_id, enabled, headline, description, cover_image_url,
		       accent_color, show_prices, show_resources, quote_requests_enabled,
		       contact_email, contact_phone, contact_address, terms_text, terms_version,
		       created_at, updated_at
		FROM public_catalog_settings
		WHERE tenant_id = $1
	`, tenantID))
}

func (r *Repository) UpsertSettings(ctx context.Context, tenantID string, input normalizedSettings) (Settings, error) {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO public_catalog_settings (
			tenant_id, enabled, headline, description, cover_image_url, accent_color,
			show_prices, show_resources, quote_requests_enabled,
			contact_email, contact_phone, contact_address, terms_text, terms_version
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (tenant_id) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			headline = EXCLUDED.headline,
			description = EXCLUDED.description,
			cover_image_url = EXCLUDED.cover_image_url,
			accent_color = EXCLUDED.accent_color,
			show_prices = EXCLUDED.show_prices,
			show_resources = EXCLUDED.show_resources,
			quote_requests_enabled = EXCLUDED.quote_requests_enabled,
			contact_email = EXCLUDED.contact_email,
			contact_phone = EXCLUDED.contact_phone,
			contact_address = EXCLUDED.contact_address,
			terms_text = EXCLUDED.terms_text,
			terms_version = EXCLUDED.terms_version
	`, tenantID, input.Enabled, input.Headline, input.Description, input.CoverImageURL,
		input.AccentColor, input.ShowPrices, input.ShowResources, input.QuoteRequestsEnabled,
		input.ContactEmail, input.ContactPhone, input.ContactAddress, input.TermsText, input.TermsVersion)
	if err != nil {
		return Settings{}, fmt.Errorf("upsert public catalog settings: %w", err)
	}
	return r.GetSettings(ctx, tenantID)
}

func (r *Repository) GetTenant(ctx context.Context, tenantID string) (PublicTenant, error) {
	var item PublicTenant
	err := r.pool.QueryRow(ctx, `
		SELECT name, slug, logo_url, email, phone, address, currency, timezone
		FROM tenants
		WHERE id = $1 AND status = 'ACTIVE'
	`, tenantID).Scan(&item.Name, &item.Slug, &item.LogoURL, &item.Email, &item.Phone, &item.Address, &item.Currency, &item.Timezone)
	if errors.Is(err, pgx.ErrNoRows) {
		return PublicTenant{}, ErrCatalogNotFound
	}
	if err != nil {
		return PublicTenant{}, fmt.Errorf("get public catalog tenant: %w", err)
	}
	return item, nil
}

func (r *Repository) ResolvePublicCatalog(ctx context.Context, tenantSlug string) (PublicTenant, Settings, error) {
	var tenant PublicTenant
	var settings Settings
	err := r.pool.QueryRow(ctx, `
		SELECT
			t.name, t.slug, t.logo_url, t.email, t.phone, t.address, t.currency, t.timezone,
			s.tenant_id, s.enabled, s.headline, s.description, s.cover_image_url,
			s.accent_color, s.show_prices, s.show_resources, s.quote_requests_enabled,
			s.contact_email, s.contact_phone, s.contact_address, s.terms_text, s.terms_version,
			s.created_at, s.updated_at
		FROM tenants t
		JOIN public_catalog_settings s ON s.tenant_id = t.id
		WHERE t.slug = $1 AND t.status = 'ACTIVE' AND s.enabled = TRUE
	`, tenantSlug).Scan(
		&tenant.Name, &tenant.Slug, &tenant.LogoURL, &tenant.Email, &tenant.Phone, &tenant.Address, &tenant.Currency, &tenant.Timezone,
		&settings.TenantID, &settings.Enabled, &settings.Headline, &settings.Description, &settings.CoverImageURL,
		&settings.AccentColor, &settings.ShowPrices, &settings.ShowResources, &settings.QuoteRequestsEnabled,
		&settings.ContactEmail, &settings.ContactPhone, &settings.ContactAddress, &settings.TermsText, &settings.TermsVersion,
		&settings.CreatedAt, &settings.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return PublicTenant{}, Settings{}, ErrCatalogNotFound
	}
	if err != nil {
		return PublicTenant{}, Settings{}, fmt.Errorf("resolve public catalog: %w", err)
	}
	return tenant, settings, nil
}

const adminPackageSelect = `
	WITH totals AS (
		SELECT p.id, p.tenant_id, p.name, p.slug, p.description, p.guest_capacity,
		       p.image_url, p.active, p.public_visible, p.public_featured, p.public_sort_order,
		       COUNT(pi.id)::int AS item_count,
		       COALESCE(SUM(pi.quantity), 0)::int AS total_quantity,
		       COUNT(pi.id) FILTER (WHERE resource.active = FALSE)::int AS unavailable_item_count,
		       COALESCE(SUM(pi.quantity * COALESCE(pi.unit_price_override, resource.base_price)), 0)::float8 AS calculated_price,
		       p.pricing_mode, p.fixed_price::float8 AS fixed_price
		FROM packages p
		LEFT JOIN package_items pi ON pi.tenant_id = p.tenant_id AND pi.package_id = p.id
		LEFT JOIN resources resource ON resource.tenant_id = pi.tenant_id AND resource.id = pi.resource_id
		GROUP BY p.id
	)
	SELECT id, name, slug, description, guest_capacity, image_url,
	       CASE WHEN pricing_mode = 'FIXED' THEN COALESCE(fixed_price, 0) ELSE calculated_price END::float8,
	       item_count, total_quantity,
	       (active AND item_count > 0 AND unavailable_item_count = 0) AS ready,
	       active, public_visible, public_featured, public_sort_order
	FROM totals
`

func (r *Repository) ListAdminPackages(ctx context.Context, tenantID string) ([]AdminPackage, error) {
	rows, err := r.pool.Query(ctx, adminPackageSelect+`
		WHERE tenant_id = $1
		ORDER BY public_visible DESC, public_featured DESC, public_sort_order, name
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list admin public packages: %w", err)
	}
	defer rows.Close()
	items := make([]AdminPackage, 0)
	for rows.Next() {
		var item AdminPackage
		if err := rows.Scan(&item.ID, &item.Name, &item.Slug, &item.Description, &item.GuestCapacity,
			&item.ImageURL, &item.EffectivePrice, &item.ItemCount, &item.TotalQuantity, &item.Ready,
			&item.Active, &item.PublicVisible, &item.PublicFeatured, &item.PublicSortOrder); err != nil {
			return nil, fmt.Errorf("scan admin public package: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) ListAdminResources(ctx context.Context, tenantID string) ([]AdminResource, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT r.id, c.name, r.resource_type, r.name, r.description, r.base_price::float8,
		       r.pricing_unit, r.active, r.public_slug, r.public_description, r.public_image_url,
		       r.public_visible, r.public_featured, r.public_sort_order
		FROM resources r
		LEFT JOIN categories c ON c.tenant_id = r.tenant_id AND c.id = r.category_id
		WHERE r.tenant_id = $1
		ORDER BY r.public_visible DESC, r.public_featured DESC, r.public_sort_order, r.name
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list admin public resources: %w", err)
	}
	defer rows.Close()
	items := make([]AdminResource, 0)
	for rows.Next() {
		var item AdminResource
		if err := rows.Scan(&item.ID, &item.CategoryName, &item.ResourceType, &item.Name, &item.Description,
			&item.BasePrice, &item.PricingUnit, &item.Active, &item.PublicSlug, &item.PublicDescription,
			&item.PublicImageURL, &item.PublicVisible, &item.PublicFeatured, &item.PublicSortOrder); err != nil {
			return nil, fmt.Errorf("scan admin public resource: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) UpdatePackagePublication(ctx context.Context, tenantID, packageID string, input PackagePublicationInput) error {
	command, err := r.pool.Exec(ctx, `
		UPDATE packages SET
			public_visible = $3,
			public_featured = $4,
			public_sort_order = $5
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, packageID, input.Visible, input.Featured, input.SortOrder)
	if err != nil {
		return fmt.Errorf("update package publication: %w", err)
	}
	if command.RowsAffected() == 0 {
		return ErrPackageNotPublic
	}
	return nil
}

func (r *Repository) UpdateResourcePublication(ctx context.Context, tenantID, resourceID string, input ResourcePublicationInput) error {
	command, err := r.pool.Exec(ctx, `
		UPDATE resources SET
			public_slug = NULLIF($3, ''),
			public_description = $4,
			public_image_url = $5,
			public_visible = $6,
			public_featured = $7,
			public_sort_order = $8
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, resourceID, input.PublicSlug, input.PublicDescription, input.PublicImageURL,
		input.Visible, input.Featured, input.SortOrder)
	if err != nil {
		if isPgCode(err, "23505") {
			return ErrPublicationConflict
		}
		return fmt.Errorf("update resource publication: %w", err)
	}
	if command.RowsAffected() == 0 {
		return ErrResourceNotPublic
	}
	return nil
}

func (r *Repository) ListPublicPackages(ctx context.Context, tenantID string, showPrices bool) ([]PublicPackageSummary, error) {
	rows, err := r.pool.Query(ctx, adminPackageSelect+`
		WHERE tenant_id = $1 AND active = TRUE AND item_count > 0 AND unavailable_item_count = 0 AND public_visible = TRUE
		ORDER BY public_featured DESC, public_sort_order, name
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list public packages: %w", err)
	}
	defer rows.Close()
	items := make([]PublicPackageSummary, 0)
	for rows.Next() {
		var admin AdminPackage
		if err := rows.Scan(&admin.ID, &admin.Name, &admin.Slug, &admin.Description, &admin.GuestCapacity,
			&admin.ImageURL, &admin.EffectivePrice, &admin.ItemCount, &admin.TotalQuantity, &admin.Ready,
			&admin.Active, &admin.PublicVisible, &admin.PublicFeatured, &admin.PublicSortOrder); err != nil {
			return nil, fmt.Errorf("scan public package: %w", err)
		}
		item := PublicPackageSummary{
			Name: admin.Name, Slug: admin.Slug, Description: admin.Description,
			GuestCapacity: admin.GuestCapacity, ImageURL: admin.ImageURL,
			ItemCount: admin.ItemCount, TotalQuantity: admin.TotalQuantity, Featured: admin.PublicFeatured,
		}
		if showPrices {
			price := admin.EffectivePrice
			item.EffectivePrice = &price
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) GetPublicPackage(ctx context.Context, tenantID, packageSlug string, showPrices bool) (PublicPackageDetail, error) {
	row := r.pool.QueryRow(ctx, adminPackageSelect+`
		WHERE tenant_id = $1 AND slug = $2 AND active = TRUE AND item_count > 0 AND unavailable_item_count = 0 AND public_visible = TRUE
	`, tenantID, packageSlug)
	var admin AdminPackage
	if err := row.Scan(&admin.ID, &admin.Name, &admin.Slug, &admin.Description, &admin.GuestCapacity,
		&admin.ImageURL, &admin.EffectivePrice, &admin.ItemCount, &admin.TotalQuantity, &admin.Ready,
		&admin.Active, &admin.PublicVisible, &admin.PublicFeatured, &admin.PublicSortOrder); errors.Is(err, pgx.ErrNoRows) {
		return PublicPackageDetail{}, ErrPackageNotPublic
	} else if err != nil {
		return PublicPackageDetail{}, fmt.Errorf("get public package: %w", err)
	}
	item := PublicPackageDetail{PublicPackageSummary: PublicPackageSummary{
		Name: admin.Name, Slug: admin.Slug, Description: admin.Description,
		GuestCapacity: admin.GuestCapacity, ImageURL: admin.ImageURL,
		ItemCount: admin.ItemCount, TotalQuantity: admin.TotalQuantity, Featured: admin.PublicFeatured,
	}}
	if showPrices {
		price := admin.EffectivePrice
		item.EffectivePrice = &price
	}
	rows, err := r.pool.Query(ctx, `
		SELECT resource.name, COALESCE(NULLIF(BTRIM(pi.description), ''), resource.name), pi.quantity
		FROM package_items pi
		JOIN packages p ON p.tenant_id = pi.tenant_id AND p.id = pi.package_id
		JOIN resources resource ON resource.tenant_id = pi.tenant_id AND resource.id = pi.resource_id
		WHERE pi.tenant_id = $1 AND p.slug = $2 AND p.public_visible = TRUE AND p.active = TRUE
		ORDER BY pi.sort_order, pi.created_at, pi.id
	`, tenantID, packageSlug)
	if err != nil {
		return PublicPackageDetail{}, fmt.Errorf("list public package items: %w", err)
	}
	defer rows.Close()
	item.Items = make([]PublicPackageItem, 0)
	for rows.Next() {
		var component PublicPackageItem
		if err := rows.Scan(&component.ResourceName, &component.Description, &component.Quantity); err != nil {
			return PublicPackageDetail{}, fmt.Errorf("scan public package item: %w", err)
		}
		item.Items = append(item.Items, component)
	}
	return item, rows.Err()
}

func (r *Repository) ListPublicResources(ctx context.Context, tenantID string, showPrices bool) ([]PublicResource, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT r.public_slug, c.name, r.resource_type, r.name,
		       COALESCE(NULLIF(BTRIM(r.public_description), ''), r.description),
		       r.public_image_url, r.base_price::float8, r.pricing_unit, r.public_featured
		FROM resources r
		LEFT JOIN categories c ON c.tenant_id = r.tenant_id AND c.id = r.category_id
		WHERE r.tenant_id = $1 AND r.active = TRUE AND r.public_visible = TRUE AND r.public_slug IS NOT NULL
		ORDER BY r.public_featured DESC, r.public_sort_order, r.name
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list public resources: %w", err)
	}
	defer rows.Close()
	items := make([]PublicResource, 0)
	for rows.Next() {
		var item PublicResource
		var price float64
		if err := rows.Scan(&item.Slug, &item.CategoryName, &item.ResourceType, &item.Name,
			&item.Description, &item.ImageURL, &price, &item.PricingUnit, &item.Featured); err != nil {
			return nil, fmt.Errorf("scan public resource: %w", err)
		}
		if showPrices {
			item.BasePrice = &price
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) GetPublicResource(ctx context.Context, tenantID, resourceSlug string, showPrices bool) (PublicResource, error) {
	var item PublicResource
	var price float64
	err := r.pool.QueryRow(ctx, `
		SELECT r.public_slug, c.name, r.resource_type, r.name,
		       COALESCE(NULLIF(BTRIM(r.public_description), ''), r.description),
		       r.public_image_url, r.base_price::float8, r.pricing_unit, r.public_featured
		FROM resources r
		LEFT JOIN categories c ON c.tenant_id = r.tenant_id AND c.id = r.category_id
		WHERE r.tenant_id = $1 AND r.public_slug = $2 AND r.active = TRUE AND r.public_visible = TRUE
	`, tenantID, resourceSlug).Scan(&item.Slug, &item.CategoryName, &item.ResourceType, &item.Name,
		&item.Description, &item.ImageURL, &price, &item.PricingUnit, &item.Featured)
	if errors.Is(err, pgx.ErrNoRows) {
		return PublicResource{}, ErrResourceNotPublic
	}
	if err != nil {
		return PublicResource{}, fmt.Errorf("get public resource: %w", err)
	}
	if showPrices {
		item.BasePrice = &price
	}
	return item, nil
}

func (r *Repository) CountRecentRequests(ctx context.Context, tenantID, submitterHash string, since time.Time) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM quote_requests
		WHERE tenant_id = $1 AND submitter_hash = $2 AND created_at >= $3
	`, tenantID, submitterHash, since).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count recent quote requests: %w", err)
	}
	return count, nil
}

func (r *Repository) CreateQuoteRequest(ctx context.Context, tenantID, currency string, input preparedRequest) (QuoteRequestReceipt, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return QuoteRequestReceipt{}, fmt.Errorf("begin create quote request: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Serialize submissions for the same tenant/fingerprint so concurrent requests
	// cannot race past the application-level rate-limit check. The advisory lock
	// lives only for this transaction and stores no raw client address.
	lockKey := tenantID + "|" + input.SubmitterHash
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, lockKey); err != nil {
		return QuoteRequestReceipt{}, fmt.Errorf("lock quote request submitter: %w", err)
	}
	var recentCount int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM quote_requests
		WHERE tenant_id = $1 AND submitter_hash = $2
		  AND created_at >= NOW() - INTERVAL '1 hour'
	`, tenantID, input.SubmitterHash).Scan(&recentCount); err != nil {
		return QuoteRequestReceipt{}, fmt.Errorf("count locked quote requests: %w", err)
	}
	if recentCount >= 5 {
		return QuoteRequestReceipt{}, ErrQuoteRequestRateLimited
	}

	availabilityJSON, err := json.Marshal(input.Availability)
	if err != nil {
		return QuoteRequestReceipt{}, fmt.Errorf("marshal quote request availability: %w", err)
	}
	var receipt QuoteRequestReceipt
	var estimatedTotal float64
	err = tx.QueryRow(ctx, `
		INSERT INTO quote_requests (
			tenant_id, first_name, last_name, phone, email, company_name, preferred_language,
			event_type, event_location, start_at, end_at, notes,
			estimated_subtotal, estimated_discount_amount, estimated_extra_charges, estimated_total, currency,
			availability_available, availability_snapshot, terms_text, terms_version, consent_accepted,
			submitter_hash, user_agent
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,
		          $13, $14, $15, $16, $17, $18, $19::jsonb, $20, $21, $22, $23, $24)
		RETURNING id, reference_code, status, estimated_total::float8, availability_available, created_at
	`, tenantID, input.Input.FirstName, input.Input.LastName, input.Input.Phone, input.Input.Email,
		input.Input.CompanyName, input.Input.PreferredLanguage, input.Input.EventType, input.Input.EventLocation,
		input.Input.StartAt, input.Input.EndAt, input.Input.Notes, input.EstimatedSubtotal,
		input.EstimatedDiscount, input.EstimatedExtraCharges, input.EstimatedTotal, currency, input.Availability.Available,
		string(availabilityJSON), input.TermsText, input.TermsVersion, input.Input.ConsentAccepted, input.SubmitterHash,
		input.UserAgent).Scan(&receipt.RequestID, &receipt.ReferenceCode, &receipt.Status, &estimatedTotal,
		&receipt.AvailabilityAvailable, &receipt.CreatedAt)
	if err != nil {
		return QuoteRequestReceipt{}, fmt.Errorf("insert quote request: %w", err)
	}
	receipt.EstimatedTotal = &estimatedTotal

	for index, item := range input.Packages {
		templateJSON, marshalErr := json.Marshal(item.Template)
		if marshalErr != nil {
			return QuoteRequestReceipt{}, fmt.Errorf("marshal quote request package template: %w", marshalErr)
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO quote_request_packages (
				tenant_id, quote_request_id, package_id, package_name, package_slug,
				quantity, unit_price, line_total, template, sort_order
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10)
		`, tenantID, receipt.RequestID, item.PackageID, item.PackageName, item.PackageSlug,
			item.Quantity, item.UnitPrice, item.LineTotal, string(templateJSON), index)
		if err != nil {
			return QuoteRequestReceipt{}, fmt.Errorf("insert quote request package: %w", err)
		}
	}
	for index, item := range input.Items {
		_, err = tx.Exec(ctx, `
			INSERT INTO quote_request_items (
				tenant_id, quote_request_id, resource_id, resource_name, description,
				quantity, unit_price, discount_amount, line_total, sort_order
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`, tenantID, receipt.RequestID, item.ResourceID, item.ResourceName, item.Description,
			item.Quantity, item.UnitPrice, item.DiscountAmount, item.LineTotal, index)
		if err != nil {
			return QuoteRequestReceipt{}, fmt.Errorf("insert quote request item: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return QuoteRequestReceipt{}, fmt.Errorf("commit quote request: %w", err)
	}
	return receipt, nil
}

func (r *Repository) ListQuoteRequests(ctx context.Context, tenantID, search, status string) (QuoteRequestList, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT qr.id, qr.reference_code, qr.status,
		       TRIM(qr.first_name || ' ' || qr.last_name), qr.phone, qr.email,
		       qr.event_type, qr.event_location, qr.start_at, qr.end_at,
		       qr.estimated_total::float8, qr.currency, qr.availability_available,
		       COUNT(qrp.id)::int, qr.converted_quote_id, qr.handled_at,
		       qr.created_at, qr.updated_at
		FROM quote_requests qr
		LEFT JOIN quote_request_packages qrp ON qrp.tenant_id = qr.tenant_id AND qrp.quote_request_id = qr.id
		WHERE qr.tenant_id = $1
		  AND ($2 = '' OR qr.reference_code ILIKE '%' || $2 || '%' OR qr.first_name ILIKE '%' || $2 || '%'
		       OR qr.last_name ILIKE '%' || $2 || '%' OR COALESCE(qr.email, '') ILIKE '%' || $2 || '%'
		       OR COALESCE(qr.phone, '') ILIKE '%' || $2 || '%' OR COALESCE(qr.event_location, '') ILIKE '%' || $2 || '%')
		  AND ($3 = '' OR qr.status = $3)
		GROUP BY qr.id
		ORDER BY CASE qr.status WHEN 'NEW' THEN 0 WHEN 'IN_REVIEW' THEN 1 ELSE 2 END, qr.created_at DESC
	`, tenantID, search, status)
	if err != nil {
		return QuoteRequestList{}, fmt.Errorf("list quote requests: %w", err)
	}
	defer rows.Close()
	result := QuoteRequestList{Items: make([]QuoteRequestSummary, 0), Counts: map[string]int{}}
	for rows.Next() {
		var item QuoteRequestSummary
		if err := rows.Scan(&item.ID, &item.ReferenceCode, &item.Status, &item.CustomerName,
			&item.Phone, &item.Email, &item.EventType, &item.EventLocation, &item.StartAt, &item.EndAt,
			&item.EstimatedTotal, &item.Currency, &item.AvailabilityAvailable, &item.PackageCount, &item.ConvertedQuoteID,
			&item.HandledAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return QuoteRequestList{}, fmt.Errorf("scan quote request summary: %w", err)
		}
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		return QuoteRequestList{}, fmt.Errorf("iterate quote requests: %w", err)
	}
	countRows, err := r.pool.Query(ctx, `
		SELECT status, COUNT(*)::int FROM quote_requests WHERE tenant_id = $1 GROUP BY status
	`, tenantID)
	if err != nil {
		return QuoteRequestList{}, fmt.Errorf("count quote request statuses: %w", err)
	}
	defer countRows.Close()
	for countRows.Next() {
		var status string
		var count int
		if err := countRows.Scan(&status, &count); err != nil {
			return QuoteRequestList{}, fmt.Errorf("scan quote request count: %w", err)
		}
		result.Counts[status] = count
	}
	return result, countRows.Err()
}

func (r *Repository) GetQuoteRequest(ctx context.Context, tenantID, requestID string) (QuoteRequestDetail, error) {
	var item QuoteRequestDetail
	var availabilityJSON []byte
	err := r.pool.QueryRow(ctx, `
		SELECT qr.id, qr.reference_code, qr.status,
		       TRIM(qr.first_name || ' ' || qr.last_name), qr.phone, qr.email,
		       qr.event_type, qr.event_location, qr.start_at, qr.end_at,
		       qr.estimated_total::float8, qr.currency, qr.availability_available,
		       (SELECT COUNT(*)::int FROM quote_request_packages qrp WHERE qrp.tenant_id = qr.tenant_id AND qrp.quote_request_id = qr.id),
		       qr.converted_quote_id, qr.handled_at, qr.created_at, qr.updated_at,
		       qr.first_name, qr.last_name, qr.company_name, qr.preferred_language, qr.notes,
		       qr.estimated_subtotal::float8, qr.estimated_discount_amount::float8, qr.estimated_extra_charges::float8,
		       qr.availability_snapshot, qr.terms_text, qr.terms_version, qr.consent_accepted, qr.converted_customer_id
		FROM quote_requests qr
		WHERE qr.tenant_id = $1 AND qr.id = $2
	`, tenantID, requestID).Scan(&item.ID, &item.ReferenceCode, &item.Status, &item.CustomerName,
		&item.Phone, &item.Email, &item.EventType, &item.EventLocation, &item.StartAt, &item.EndAt,
		&item.EstimatedTotal, &item.Currency, &item.AvailabilityAvailable, &item.PackageCount, &item.ConvertedQuoteID,
		&item.HandledAt, &item.CreatedAt, &item.UpdatedAt, &item.FirstName, &item.LastName,
		&item.CompanyName, &item.PreferredLanguage, &item.Notes, &item.EstimatedSubtotal,
		&item.EstimatedDiscount, &item.EstimatedExtraCharges, &availabilityJSON, &item.TermsText, &item.TermsVersion, &item.ConsentAccepted,
		&item.ConvertedCustomerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return QuoteRequestDetail{}, ErrQuoteRequestNotFound
	}
	if err != nil {
		return QuoteRequestDetail{}, fmt.Errorf("get quote request: %w", err)
	}
	if err := json.Unmarshal(availabilityJSON, &item.Availability); err != nil {
		item.Availability = availability.Result{}
	}

	packageRows, err := r.pool.Query(ctx, `
		SELECT id, package_id, package_name, package_slug, quantity,
		       unit_price::float8, line_total::float8, template, created_at
		FROM quote_request_packages
		WHERE tenant_id = $1 AND quote_request_id = $2
		ORDER BY sort_order, id
	`, tenantID, requestID)
	if err != nil {
		return QuoteRequestDetail{}, fmt.Errorf("list quote request packages: %w", err)
	}
	defer packageRows.Close()
	item.Packages = make([]RequestPackage, 0)
	for packageRows.Next() {
		var entry RequestPackage
		var templateJSON []byte
		if err := packageRows.Scan(&entry.ID, &entry.PackageID, &entry.PackageName, &entry.PackageSlug,
			&entry.Quantity, &entry.UnitPrice, &entry.LineTotal, &templateJSON, &entry.CreatedAt); err != nil {
			return QuoteRequestDetail{}, fmt.Errorf("scan quote request package: %w", err)
		}
		_ = json.Unmarshal(templateJSON, &entry.Template)
		item.Packages = append(item.Packages, entry)
	}
	if err := packageRows.Err(); err != nil {
		return QuoteRequestDetail{}, fmt.Errorf("iterate quote request packages: %w", err)
	}
	packageRows.Close()

	itemRows, err := r.pool.Query(ctx, `
		SELECT id, resource_id, resource_name, description, quantity,
		       unit_price::float8, discount_amount::float8, line_total::float8, created_at
		FROM quote_request_items
		WHERE tenant_id = $1 AND quote_request_id = $2
		ORDER BY sort_order, id
	`, tenantID, requestID)
	if err != nil {
		return QuoteRequestDetail{}, fmt.Errorf("list quote request items: %w", err)
	}
	defer itemRows.Close()
	item.Items = make([]RequestItem, 0)
	for itemRows.Next() {
		var entry RequestItem
		if err := itemRows.Scan(&entry.ID, &entry.ResourceID, &entry.ResourceName, &entry.Description,
			&entry.Quantity, &entry.UnitPrice, &entry.DiscountAmount, &entry.LineTotal, &entry.CreatedAt); err != nil {
			return QuoteRequestDetail{}, fmt.Errorf("scan quote request item: %w", err)
		}
		item.Items = append(item.Items, entry)
	}
	if err := itemRows.Err(); err != nil {
		return QuoteRequestDetail{}, fmt.Errorf("iterate quote request items: %w", err)
	}
	return item, nil
}

func (r *Repository) UpdateQuoteRequestStatus(ctx context.Context, tenantID, requestID, actorID, status string) (QuoteRequestDetail, error) {
	command, err := r.pool.Exec(ctx, `
		UPDATE quote_requests SET
			status = $4,
			handled_at = CASE WHEN $4 IN ('CLOSED', 'SPAM') THEN NOW() ELSE NULL END,
			handled_by = CASE WHEN $4 = 'NEW' THEN NULL ELSE $3::uuid END
		WHERE tenant_id = $1 AND id = $2 AND status <> 'CONVERTED'
	`, tenantID, requestID, actorID, status)
	if err != nil {
		return QuoteRequestDetail{}, fmt.Errorf("update quote request status: %w", err)
	}
	if command.RowsAffected() == 0 {
		return QuoteRequestDetail{}, ErrQuoteRequestConflict
	}
	return r.GetQuoteRequest(ctx, tenantID, requestID)
}

func (r *Repository) ConvertQuoteRequest(ctx context.Context, tenantID, requestID, actorID string) (ConversionResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ConversionResult{}, fmt.Errorf("begin quote request conversion: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var referenceCode, status, firstName, lastName, preferredLanguage, notes string
	var phone, email, companyName, eventType, eventLocation *string
	var startAt, endAt time.Time
	var subtotal, discountAmount, extraCharges, total float64
	var existingCustomerID, existingQuoteID *string
	err = tx.QueryRow(ctx, `
		SELECT reference_code, status, first_name, last_name, phone, email, company_name,
		       preferred_language, event_type, event_location, start_at, end_at, notes,
		       estimated_subtotal::float8, estimated_discount_amount::float8, estimated_extra_charges::float8, estimated_total::float8,
		       converted_customer_id, converted_quote_id
		FROM quote_requests
		WHERE tenant_id = $1 AND id = $2
		FOR UPDATE
	`, tenantID, requestID).Scan(&referenceCode, &status, &firstName, &lastName, &phone, &email,
		&companyName, &preferredLanguage, &eventType, &eventLocation, &startAt, &endAt, &notes,
		&subtotal, &discountAmount, &extraCharges, &total, &existingCustomerID, &existingQuoteID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ConversionResult{}, ErrQuoteRequestNotFound
	}
	if err != nil {
		return ConversionResult{}, fmt.Errorf("lock quote request: %w", err)
	}
	if status == QuoteRequestStatusConverted || existingQuoteID != nil {
		return ConversionResult{}, ErrConversionConflict
	}
	if status != QuoteRequestStatusNew && status != QuoteRequestStatusInReview {
		return ConversionResult{}, ErrQuoteRequestConflict
	}

	var customerID string
	err = tx.QueryRow(ctx, `
		SELECT id
		FROM customers
		WHERE tenant_id = $1
		  AND ((NULLIF($2, '') IS NOT NULL AND LOWER(COALESCE(email, '')) = LOWER($2))
		       OR (NULLIF($3, '') IS NOT NULL AND COALESCE(phone, '') = $3))
		ORDER BY CASE WHEN NULLIF($2, '') IS NOT NULL AND LOWER(COALESCE(email, '')) = LOWER($2) THEN 0 ELSE 1 END,
		         updated_at DESC
		LIMIT 1
	`, tenantID, nullableString(email), nullableString(phone)).Scan(&customerID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `
			INSERT INTO customers (
				tenant_id, first_name, last_name, phone, email, company_name,
				preferred_language, source, notes
			) VALUES ($1, $2, $3, $4, $5, $6, $7, 'WEB', $8)
			RETURNING id
		`, tenantID, firstName, lastName, phone, email, companyName, preferredLanguage,
			"Creado desde solicitud pública "+referenceCode+".").Scan(&customerID)
	}
	if err != nil {
		return ConversionResult{}, fmt.Errorf("resolve quote request customer: %w", err)
	}

	var quoteID string
	var quoteNumber int64
	err = tx.QueryRow(ctx, `
		INSERT INTO quotes (
			tenant_id, customer_id, start_at, end_at, status, event_type, event_location,
			subtotal, discount_amount, extra_charges, total, notes, expires_at
		) VALUES ($1, $2, $3, $4, 'DRAFT', $5, $6, $7, $8, $9, $10, $11, NOW() + INTERVAL '7 days')
		RETURNING id, quote_number
	`, tenantID, customerID, startAt, endAt, eventType, eventLocation, subtotal,
		discountAmount, extraCharges, total,
		"Solicitud pública "+referenceCode+". "+notes).Scan(&quoteID, &quoteNumber)
	if err != nil {
		return ConversionResult{}, fmt.Errorf("insert quote from request: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO quote_items (
			tenant_id, quote_id, resource_id, description, quantity,
			unit_price, discount_amount, line_total
		)
		SELECT tenant_id, $3, resource_id, description, quantity,
		       unit_price, discount_amount, line_total
		FROM quote_request_items
		WHERE tenant_id = $1 AND quote_request_id = $2
		ORDER BY sort_order, id
	`, tenantID, requestID, quoteID)
	if err != nil {
		return ConversionResult{}, fmt.Errorf("copy quote request items: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE quote_requests SET
			status = 'CONVERTED',
			converted_customer_id = $3,
			converted_quote_id = $4,
			handled_at = NOW(),
			handled_by = $5
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, requestID, customerID, quoteID, actorID)
	if err != nil {
		return ConversionResult{}, fmt.Errorf("mark quote request converted: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return ConversionResult{}, fmt.Errorf("commit quote request conversion: %w", err)
	}
	return ConversionResult{RequestID: requestID, ReferenceCode: referenceCode,
		CustomerID: customerID, QuoteID: quoteID, QuoteNumber: quoteNumber}, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSettings(row rowScanner) (Settings, error) {
	var item Settings
	err := row.Scan(&item.TenantID, &item.Enabled, &item.Headline, &item.Description,
		&item.CoverImageURL, &item.AccentColor, &item.ShowPrices, &item.ShowResources,
		&item.QuoteRequestsEnabled, &item.ContactEmail, &item.ContactPhone,
		&item.ContactAddress, &item.TermsText, &item.TermsVersion, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return Settings{}, fmt.Errorf("scan public catalog settings: %w", err)
	}
	return item, nil
}

func nullableString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func isPgCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}
