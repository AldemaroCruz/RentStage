package customer

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) List(ctx context.Context, tenantID, search, source string) ([]Customer, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			c.id,
			c.first_name,
			c.last_name,
			TRIM(c.first_name || ' ' || c.last_name) AS display_name,
			c.phone,
			c.email,
			c.company_name,
			c.tax_id,
			c.tax_registration_number,
			c.billing_address,
			c.document_type_code, c.trade_name, c.economic_activity,
			c.economic_activity_code, c.department_code, c.municipality_code, c.district_code,
			c.preferred_language,
			c.source,
			c.notes,
			COUNT(q.id)::int AS quote_count,
			COUNT(q.id) FILTER (WHERE q.status = 'ACCEPTED')::int AS accepted_quote_count,
			COALESCE(SUM(q.total) FILTER (WHERE q.status = 'ACCEPTED'), 0)::float8 AS accepted_quote_revenue,
			MAX(q.created_at) AS last_quote_at,
			c.created_at,
			c.updated_at
		FROM customers c
		LEFT JOIN quotes q ON q.tenant_id = c.tenant_id AND q.customer_id = c.id
		WHERE c.tenant_id = $1
		  AND (
			$2 = '' OR
			c.first_name ILIKE '%' || $2 || '%' OR
			c.last_name ILIKE '%' || $2 || '%' OR
			COALESCE(c.company_name, '') ILIKE '%' || $2 || '%' OR
			COALESCE(c.phone, '') ILIKE '%' || $2 || '%' OR
			COALESCE(c.email, '') ILIKE '%' || $2 || '%'
		  )
		  AND ($3 = '' OR c.source = $3)
		GROUP BY c.id
		ORDER BY c.updated_at DESC, c.first_name, c.last_name
	`, tenantID, search, source)
	if err != nil {
		return nil, fmt.Errorf("list customers: %w", err)
	}
	defer rows.Close()

	items := make([]Customer, 0)
	for rows.Next() {
		item, err := scanCustomer(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) Get(ctx context.Context, tenantID, customerID string) (Customer, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT
			c.id,
			c.first_name,
			c.last_name,
			TRIM(c.first_name || ' ' || c.last_name) AS display_name,
			c.phone,
			c.email,
			c.company_name,
			c.tax_id,
			c.tax_registration_number,
			c.billing_address,
			c.document_type_code, c.trade_name, c.economic_activity,
			c.economic_activity_code, c.department_code, c.municipality_code, c.district_code,
			c.preferred_language,
			c.source,
			c.notes,
			COUNT(q.id)::int AS quote_count,
			COUNT(q.id) FILTER (WHERE q.status = 'ACCEPTED')::int AS accepted_quote_count,
			COALESCE(SUM(q.total) FILTER (WHERE q.status = 'ACCEPTED'), 0)::float8 AS accepted_quote_revenue,
			MAX(q.created_at) AS last_quote_at,
			c.created_at,
			c.updated_at
		FROM customers c
		LEFT JOIN quotes q ON q.tenant_id = c.tenant_id AND q.customer_id = c.id
		WHERE c.tenant_id = $1 AND c.id = $2
		GROUP BY c.id
	`, tenantID, customerID)
	item, err := scanCustomer(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Customer{}, ErrNotFound
	}
	if err != nil {
		return Customer{}, fmt.Errorf("get customer: %w", err)
	}
	return item, nil
}

func (r *Repository) Create(ctx context.Context, tenantID string, input normalizedInput) (Customer, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO customers (
			tenant_id, first_name, last_name, phone, email, company_name,
			tax_id, tax_registration_number, billing_address,
			document_type_code, trade_name, economic_activity, economic_activity_code,
			department_code, municipality_code, district_code,
			preferred_language, source, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		RETURNING id
	`,
		tenantID,
		input.FirstName,
		input.LastName,
		input.Phone,
		input.Email,
		input.CompanyName,
		input.TaxID,
		input.TaxRegistrationNumber,
		input.BillingAddress,
		input.DocumentTypeCode,
		input.TradeName,
		input.EconomicActivity,
		input.EconomicActivityCode,
		input.DepartmentCode,
		input.MunicipalityCode,
		input.DistrictCode,
		input.PreferredLanguage,
		input.Source,
		input.Notes,
	).Scan(&id)
	if err != nil {
		return Customer{}, fmt.Errorf("create customer: %w", err)
	}
	return r.Get(ctx, tenantID, id)
}

func (r *Repository) Update(ctx context.Context, tenantID, customerID string, input normalizedInput) (Customer, error) {
	command, err := r.pool.Exec(ctx, `
		UPDATE customers SET
			first_name = $3,
			last_name = $4,
			phone = $5,
			email = $6,
			company_name = $7,
			tax_id = $8,
			tax_registration_number = $9,
			billing_address = $10,
			document_type_code = $11,
			trade_name = $12,
			economic_activity = $13,
			economic_activity_code = $14,
			department_code = $15,
			municipality_code = $16,
			district_code = $17,
			preferred_language = $18,
			source = $19,
			notes = $20
		WHERE tenant_id = $1 AND id = $2
	`,
		tenantID,
		customerID,
		input.FirstName,
		input.LastName,
		input.Phone,
		input.Email,
		input.CompanyName,
		input.TaxID,
		input.TaxRegistrationNumber,
		input.BillingAddress,
		input.DocumentTypeCode,
		input.TradeName,
		input.EconomicActivity,
		input.EconomicActivityCode,
		input.DepartmentCode,
		input.MunicipalityCode,
		input.DistrictCode,
		input.PreferredLanguage,
		input.Source,
		input.Notes,
	)
	if err != nil {
		return Customer{}, fmt.Errorf("update customer: %w", err)
	}
	if command.RowsAffected() == 0 {
		return Customer{}, ErrNotFound
	}
	return r.Get(ctx, tenantID, customerID)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanCustomer(row rowScanner) (Customer, error) {
	var item Customer
	if err := row.Scan(
		&item.ID,
		&item.FirstName,
		&item.LastName,
		&item.DisplayName,
		&item.Phone,
		&item.Email,
		&item.CompanyName,
		&item.TaxID,
		&item.TaxRegistrationNumber,
		&item.BillingAddress,
		&item.DocumentTypeCode,
		&item.TradeName,
		&item.EconomicActivity,
		&item.EconomicActivityCode,
		&item.DepartmentCode,
		&item.MunicipalityCode,
		&item.DistrictCode,
		&item.PreferredLanguage,
		&item.Source,
		&item.Notes,
		&item.QuoteCount,
		&item.AcceptedQuoteCount,
		&item.AcceptedQuoteRevenue,
		&item.LastQuoteAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return Customer{}, err
	}
	return item, nil
}
