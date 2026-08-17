package tenant

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rentstage/rentstage/apps/api/internal/webutil"
)

type Handler struct {
	pool *pgxpool.Pool
}

type Tenant struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	LegalName   *string   `json:"legal_name,omitempty"`
	Email       *string   `json:"email,omitempty"`
	Phone       *string   `json:"phone,omitempty"`
	LogoURL     *string   `json:"logo_url,omitempty"`
	Address     *string   `json:"address,omitempty"`
	CountryCode string    `json:"country_code"`
	Timezone    string    `json:"timezone"`
	Currency    string    `json:"currency"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	var item Tenant
	err := h.pool.QueryRow(r.Context(), `
		SELECT id, name, slug, legal_name, email, phone, logo_url, address,
		       country_code, timezone, currency, status, created_at, updated_at
		FROM tenants
		WHERE id = $1
	`, webutil.TenantID(r.Context())).Scan(
		&item.ID,
		&item.Name,
		&item.Slug,
		&item.LegalName,
		&item.Email,
		&item.Phone,
		&item.LogoURL,
		&item.Address,
		&item.CountryCode,
		&item.Timezone,
		&item.Currency,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		webutil.WriteError(w, r, http.StatusNotFound, "tenant_not_found", "Tenant not found.")
		return
	}
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "tenant_load_failed", "Could not load the tenant.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}
