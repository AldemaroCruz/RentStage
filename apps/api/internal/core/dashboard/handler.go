package dashboard

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rentstage/rentstage/apps/api/internal/webutil"
)

type Handler struct {
	pool *pgxpool.Pool
}

type Metrics struct {
	ActiveResources     int     `json:"active_resources"`
	TotalAssets         int     `json:"total_assets"`
	AvailableAssets     int     `json:"available_assets"`
	AttentionAssets     int     `json:"attention_assets"`
	ActiveReservations  int     `json:"active_reservations"`
	InventoryInvestment float64 `json:"inventory_investment"`
	TodayDepartures     int     `json:"today_departures"`
	TodayReturns        int     `json:"today_returns"`
	OverdueReturns      int     `json:"overdue_returns"`
	ActiveValue         float64 `json:"active_value"`
}

type CategorySummary struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ResourceCount int    `json:"resource_count"`
	AssetCount    int    `json:"asset_count"`
}

type RecentResource struct {
	ID                  string  `json:"id"`
	Name                string  `json:"name"`
	CategoryName        *string `json:"category_name,omitempty"`
	BasePrice           float64 `json:"base_price"`
	PricingUnit         string  `json:"pricing_unit"`
	AssetCount          int     `json:"asset_count"`
	AvailableAssetCount int     `json:"available_asset_count"`
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := webutil.TenantID(r.Context())
	var metrics Metrics
	if err := h.pool.QueryRow(r.Context(), `
		WITH tenant_context AS (
		  SELECT
		    ((CURRENT_TIMESTAMP AT TIME ZONE timezone)::date::timestamp AT TIME ZONE timezone) AS day_start,
		    (((CURRENT_TIMESTAMP AT TIME ZONE timezone)::date + 1)::timestamp AT TIME ZONE timezone) AS day_end
		  FROM tenants
		  WHERE id = $1
		)
		SELECT
		  (SELECT COUNT(*)::int FROM resources WHERE tenant_id = $1 AND active = TRUE),
		  (SELECT COUNT(*)::int FROM assets WHERE tenant_id = $1 AND physical_status <> 'RETIRED'),
		  (SELECT COUNT(*)::int FROM assets WHERE tenant_id = $1 AND physical_status = 'AVAILABLE'),
		  (SELECT COUNT(*)::int FROM assets WHERE tenant_id = $1 AND physical_status IN ('MAINTENANCE', 'DAMAGED', 'LOST')),
		  (SELECT COUNT(*)::int FROM reservations WHERE tenant_id = $1 AND status IN ('PENDING', 'CONFIRMED', 'PREPARING', 'READY', 'CHECKED_OUT')),
		  (SELECT COALESCE(SUM(purchase_price), 0)::float8 FROM assets WHERE tenant_id = $1 AND physical_status <> 'RETIRED'),
		  (SELECT COUNT(*)::int
		     FROM reservations reservation, tenant_context context
		    WHERE reservation.tenant_id = $1
		      AND reservation.status IN ('PENDING', 'CONFIRMED', 'PREPARING', 'READY')
		      AND reservation.block_start_at >= context.day_start
		      AND reservation.block_start_at < context.day_end),
		  (SELECT COUNT(*)::int
		     FROM reservations reservation, tenant_context context
		    WHERE reservation.tenant_id = $1
		      AND reservation.status = 'CHECKED_OUT'
		      AND reservation.block_end_at >= context.day_start
		      AND reservation.block_end_at < context.day_end),
		  (SELECT COUNT(*)::int
		     FROM reservations reservation
		    WHERE reservation.tenant_id = $1
		      AND reservation.status = 'CHECKED_OUT'
		      AND reservation.block_end_at < CURRENT_TIMESTAMP),
		  (SELECT COALESCE(SUM(total), 0)::float8
		     FROM reservations
		    WHERE tenant_id = $1
		      AND status IN ('PENDING', 'CONFIRMED', 'PREPARING', 'READY', 'CHECKED_OUT'))
	`, tenantID).Scan(
		&metrics.ActiveResources,
		&metrics.TotalAssets,
		&metrics.AvailableAssets,
		&metrics.AttentionAssets,
		&metrics.ActiveReservations,
		&metrics.InventoryInvestment,
		&metrics.TodayDepartures,
		&metrics.TodayReturns,
		&metrics.OverdueReturns,
		&metrics.ActiveValue,
	); err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "dashboard_metrics_failed", "Could not load dashboard metrics.")
		return
	}

	categoryRows, err := h.pool.Query(r.Context(), `
		SELECT c.id, c.name, COUNT(DISTINCT res.id)::int, COUNT(a.id)::int
		FROM categories c
		LEFT JOIN resources res ON res.category_id = c.id AND res.tenant_id = c.tenant_id AND res.active = TRUE
		LEFT JOIN assets a ON a.resource_id = res.id AND a.tenant_id = c.tenant_id AND a.physical_status <> 'RETIRED'
		WHERE c.tenant_id = $1
		GROUP BY c.id
		ORDER BY COUNT(a.id) DESC, c.name
		LIMIT 6
	`, tenantID)
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "dashboard_categories_failed", "Could not load category summary.")
		return
	}
	categories := make([]CategorySummary, 0)
	for categoryRows.Next() {
		var item CategorySummary
		if err := categoryRows.Scan(&item.ID, &item.Name, &item.ResourceCount, &item.AssetCount); err != nil {
			categoryRows.Close()
			webutil.WriteError(w, r, http.StatusInternalServerError, "dashboard_categories_failed", "Could not load category summary.")
			return
		}
		categories = append(categories, item)
	}
	if err := categoryRows.Err(); err != nil {
		categoryRows.Close()
		webutil.WriteError(w, r, http.StatusInternalServerError, "dashboard_categories_failed", "Could not load category summary.")
		return
	}
	categoryRows.Close()

	resourceRows, err := h.pool.Query(r.Context(), `
		SELECT r.id, r.name, c.name, r.base_price::float8, r.pricing_unit,
		       COUNT(a.id)::int,
		       COUNT(a.id) FILTER (WHERE a.physical_status = 'AVAILABLE')::int
		FROM resources r
		LEFT JOIN categories c ON c.id = r.category_id AND c.tenant_id = r.tenant_id
		LEFT JOIN assets a ON a.resource_id = r.id AND a.tenant_id = r.tenant_id AND a.physical_status <> 'RETIRED'
		WHERE r.tenant_id = $1 AND r.active = TRUE
		GROUP BY r.id, c.name
		ORDER BY r.updated_at DESC
		LIMIT 5
	`, tenantID)
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "dashboard_resources_failed", "Could not load recent resources.")
		return
	}
	resources := make([]RecentResource, 0)
	for resourceRows.Next() {
		var item RecentResource
		if err := resourceRows.Scan(
			&item.ID,
			&item.Name,
			&item.CategoryName,
			&item.BasePrice,
			&item.PricingUnit,
			&item.AssetCount,
			&item.AvailableAssetCount,
		); err != nil {
			resourceRows.Close()
			webutil.WriteError(w, r, http.StatusInternalServerError, "dashboard_resources_failed", "Could not load recent resources.")
			return
		}
		resources = append(resources, item)
	}
	if err := resourceRows.Err(); err != nil {
		resourceRows.Close()
		webutil.WriteError(w, r, http.StatusInternalServerError, "dashboard_resources_failed", "Could not load recent resources.")
		return
	}
	resourceRows.Close()

	webutil.WriteJSON(w, http.StatusOK, map[string]any{
		"metrics":          metrics,
		"categories":       categories,
		"recent_resources": resources,
	})
}
