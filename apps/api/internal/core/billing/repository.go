package billing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rentstage/rentstage/apps/api/internal/webutil"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) TenantCurrency(ctx context.Context, tenantID string) (string, error) {
	var currency string
	err := r.pool.QueryRow(ctx, `SELECT currency FROM tenants WHERE id = $1`, tenantID).Scan(&currency)
	if err != nil {
		return "", fmt.Errorf("load tenant currency: %w", err)
	}
	return strings.ToUpper(strings.TrimSpace(currency)), nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func (r *Repository) ensureSettings(ctx context.Context, tenantID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO billing_settings (
			tenant_id, legal_name, trade_name, fiscal_address, email, phone
		)
		SELECT id, COALESCE(legal_name, name, ''), name, COALESCE(address, ''),
		       COALESCE(email, ''), COALESCE(phone, '')
		FROM tenants
		WHERE id = $1
		ON CONFLICT (tenant_id) DO NOTHING
	`, tenantID)
	if err != nil {
		return fmt.Errorf("ensure billing settings: %w", err)
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO tax_rules (tenant_id, code, name, category, rate, active, is_default)
		SELECT $1::uuid, seeded.code, seeded.name, seeded.category, seeded.rate, TRUE, seeded.is_default
		FROM (VALUES
			('IVA', 'IVA estándar', 'TAXABLE', 13.00::numeric, TRUE),
			('EXEMPT', 'Venta exenta', 'EXEMPT', 0.00::numeric, FALSE),
			('NON_TAXABLE', 'Venta no sujeta', 'NON_TAXABLE', 0.00::numeric, FALSE)
		) AS seeded(code, name, category, rate, is_default)
		WHERE EXISTS (SELECT 1 FROM tenants WHERE id = $1)
		ON CONFLICT (tenant_id, code) DO NOTHING
	`, tenantID)
	if err != nil {
		return fmt.Errorf("ensure tax rules: %w", err)
	}
	return nil
}

func (r *Repository) GetSettings(ctx context.Context, tenantID string) (Settings, error) {
	if err := r.ensureSettings(ctx, tenantID); err != nil {
		return Settings{}, err
	}
	row := r.pool.QueryRow(ctx, `
		SELECT tenant_id, enabled, legal_name, trade_name, tax_id,
		       tax_registration_number, economic_activity, economic_activity_code,
		       fiscal_address, department, municipality, district,
		       department_code, municipality_code, district_code, email, phone,
		       prices_include_tax, default_tax_rate::float8,
		       default_payment_terms_days, invoice_prefix, next_invoice_number,
		       created_at, updated_at
		FROM billing_settings
		WHERE tenant_id = $1
	`, tenantID)
	return scanSettings(row)
}

func scanSettings(row rowScanner) (Settings, error) {
	var item Settings
	if err := row.Scan(
		&item.TenantID,
		&item.Enabled,
		&item.LegalName,
		&item.TradeName,
		&item.TaxID,
		&item.TaxRegistrationNumber,
		&item.EconomicActivity,
		&item.EconomicActivityCode,
		&item.FiscalAddress,
		&item.Department,
		&item.Municipality,
		&item.District,
		&item.DepartmentCode,
		&item.MunicipalityCode,
		&item.DistrictCode,
		&item.Email,
		&item.Phone,
		&item.PricesIncludeTax,
		&item.DefaultTaxRate,
		&item.DefaultPaymentTermsDays,
		&item.InvoicePrefix,
		&item.NextInvoiceNumber,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return Settings{}, fmt.Errorf("scan billing settings: %w", err)
	}
	item.FiscalProfileMissingFields = fiscalProfileMissing(item)
	item.FiscalProfileComplete = len(item.FiscalProfileMissingFields) == 0
	return item, nil
}

func (r *Repository) UpdateSettings(ctx context.Context, tenantID string, input SettingsInput) (Settings, error) {
	if err := r.ensureSettings(ctx, tenantID); err != nil {
		return Settings{}, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Settings{}, fmt.Errorf("begin billing settings update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
		UPDATE billing_settings SET
			enabled = $2,
			legal_name = $3,
			trade_name = $4,
			tax_id = $5,
			tax_registration_number = $6,
			economic_activity = $7,
			economic_activity_code = $8,
			fiscal_address = $9,
			department = $10,
			municipality = $11,
			district = $12,
			department_code = $13,
			municipality_code = $14,
			district_code = $15,
			email = $16,
			phone = $17,
			prices_include_tax = $18,
			default_tax_rate = $19,
			default_payment_terms_days = $20,
			invoice_prefix = $21
		WHERE tenant_id = $1
	`, tenantID, input.Enabled, input.LegalName, input.TradeName, input.TaxID,
		input.TaxRegistrationNumber, input.EconomicActivity, input.EconomicActivityCode,
		input.FiscalAddress, input.Department, input.Municipality, input.District,
		input.DepartmentCode, input.MunicipalityCode, input.DistrictCode,
		input.Email, input.Phone, input.PricesIncludeTax, input.DefaultTaxRate,
		input.DefaultPaymentTermsDays, input.InvoicePrefix)
	if err != nil {
		return Settings{}, fmt.Errorf("update billing settings: %w", err)
	}
	_, err = tx.Exec(ctx, `
		UPDATE tax_rules
		SET rate = $2, name = 'IVA estándar', active = TRUE, is_default = TRUE
		WHERE tenant_id = $1 AND code = 'IVA'
	`, tenantID, input.DefaultTaxRate)
	if err != nil {
		return Settings{}, fmt.Errorf("update default tax rule: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Settings{}, fmt.Errorf("commit billing settings update: %w", err)
	}
	return r.GetSettings(ctx, tenantID)
}

func (r *Repository) ListTaxRules(ctx context.Context, tenantID string) ([]TaxRule, error) {
	if err := r.ensureSettings(ctx, tenantID); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, code, name, category, rate::float8, active, is_default,
		       valid_from::text, valid_until::text, created_at, updated_at
		FROM tax_rules
		WHERE tenant_id = $1
		  AND active = TRUE
		  AND valid_from <= CURRENT_DATE
		  AND (valid_until IS NULL OR valid_until >= CURRENT_DATE)
		ORDER BY is_default DESC, category, code
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list tax rules: %w", err)
	}
	defer rows.Close()
	items := make([]TaxRule, 0)
	for rows.Next() {
		var item TaxRule
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.Category, &item.Rate,
			&item.Active, &item.IsDefault, &item.ValidFrom, &item.ValidUntil,
			&item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan tax rule: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) LoadSource(ctx context.Context, tenantID, sourceType, sourceID string) (sourceDraft, error) {
	switch sourceType {
	case "QUOTE":
		return r.loadQuoteSource(ctx, tenantID, sourceID)
	case "RESERVATION":
		return r.loadReservationSource(ctx, tenantID, sourceID)
	default:
		return sourceDraft{}, ErrSourceNotFound
	}
}

func (r *Repository) loadQuoteSource(ctx context.Context, tenantID, quoteID string) (sourceDraft, error) {
	var item sourceDraft
	var quoteNumber int64
	var reservationID *string
	var reservationNumber *int64
	err := r.pool.QueryRow(ctx, `
		SELECT q.customer_id, tenant.currency, q.notes,
		       q.discount_amount::float8, q.extra_charges::float8,
		       q.quote_number, reservation.id, reservation.reservation_number
		FROM quotes q
		JOIN tenants tenant ON tenant.id = q.tenant_id
		LEFT JOIN reservations reservation
		  ON reservation.tenant_id = q.tenant_id AND reservation.quote_id = q.id
		WHERE q.tenant_id = $1 AND q.id = $2 AND q.status = 'ACCEPTED'
	`, tenantID, quoteID).Scan(
		&item.CustomerID, &item.Currency, &item.Notes,
		&item.HeaderDiscount, &item.ExtraCharges,
		&quoteNumber, &reservationID, &reservationNumber,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return sourceDraft{}, ErrSourceNotFound
	}
	if err != nil {
		return sourceDraft{}, fmt.Errorf("load quote invoice source: %w", err)
	}
	item.SourceType = "QUOTE"
	item.QuoteID = stringPointer(quoteID)
	item.QuoteNumber = &quoteNumber
	item.ReservationID = reservationID
	item.ReservationNumber = reservationNumber
	if err := r.requireSourceAvailable(ctx, tenantID, item.QuoteID, item.ReservationID); err != nil {
		return sourceDraft{}, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT qi.resource_id, qi.description, qi.quantity::float8,
		       qi.unit_price::float8, qi.discount_amount::float8
		FROM quote_items qi
		WHERE qi.tenant_id = $1 AND qi.quote_id = $2
		ORDER BY qi.created_at, qi.id
	`, tenantID, quoteID)
	if err != nil {
		return sourceDraft{}, fmt.Errorf("load quote invoice items: %w", err)
	}
	defer rows.Close()
	item.Items = make([]sourceItem, 0)
	for rows.Next() {
		var line sourceItem
		var resourceID string
		if err := rows.Scan(&resourceID, &line.Description, &line.Quantity, &line.UnitPrice, &line.DiscountAmount); err != nil {
			return sourceDraft{}, fmt.Errorf("scan quote invoice item: %w", err)
		}
		line.ResourceID = &resourceID
		item.Items = append(item.Items, line)
	}
	if len(item.Items) == 0 {
		return sourceDraft{}, ErrSourceNotFound
	}
	return item, rows.Err()
}

func (r *Repository) loadReservationSource(ctx context.Context, tenantID, reservationID string) (sourceDraft, error) {
	var item sourceDraft
	var reservationNumber int64
	var quoteID *string
	var quoteNumber *int64
	err := r.pool.QueryRow(ctx, `
		SELECT reservation.customer_id, tenant.currency, reservation.notes,
		       reservation.discount_amount::float8, reservation.extra_charges::float8,
		       reservation.reservation_number, reservation.quote_id, quote.quote_number
		FROM reservations reservation
		JOIN tenants tenant ON tenant.id = reservation.tenant_id
		LEFT JOIN quotes quote
		  ON quote.tenant_id = reservation.tenant_id AND quote.id = reservation.quote_id
		WHERE reservation.tenant_id = $1
		  AND reservation.id = $2
		  AND reservation.status <> 'CANCELLED'
	`, tenantID, reservationID).Scan(
		&item.CustomerID, &item.Currency, &item.Notes,
		&item.HeaderDiscount, &item.ExtraCharges,
		&reservationNumber, &quoteID, &quoteNumber,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return sourceDraft{}, ErrSourceNotFound
	}
	if err != nil {
		return sourceDraft{}, fmt.Errorf("load reservation invoice source: %w", err)
	}
	item.SourceType = "RESERVATION"
	item.ReservationID = stringPointer(reservationID)
	item.ReservationNumber = &reservationNumber
	item.QuoteID = quoteID
	item.QuoteNumber = quoteNumber
	if err := r.requireSourceAvailable(ctx, tenantID, item.QuoteID, item.ReservationID); err != nil {
		return sourceDraft{}, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT ri.resource_id, ri.description, ri.quantity::float8,
		       ri.unit_price::float8, ri.discount_amount::float8
		FROM reservation_items ri
		WHERE ri.tenant_id = $1 AND ri.reservation_id = $2
		ORDER BY ri.created_at, ri.id
	`, tenantID, reservationID)
	if err != nil {
		return sourceDraft{}, fmt.Errorf("load reservation invoice items: %w", err)
	}
	defer rows.Close()
	item.Items = make([]sourceItem, 0)
	for rows.Next() {
		var line sourceItem
		var resourceID string
		if err := rows.Scan(&resourceID, &line.Description, &line.Quantity, &line.UnitPrice, &line.DiscountAmount); err != nil {
			return sourceDraft{}, fmt.Errorf("scan reservation invoice item: %w", err)
		}
		line.ResourceID = &resourceID
		item.Items = append(item.Items, line)
	}
	if len(item.Items) == 0 {
		return sourceDraft{}, ErrSourceNotFound
	}
	return item, rows.Err()
}

func (r *Repository) requireSourceAvailable(ctx context.Context, tenantID string, quoteID, reservationID *string) error {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM invoices
			WHERE tenant_id = $1 AND status <> 'VOID'
			  AND (($2::uuid IS NOT NULL AND quote_id = $2::uuid)
			       OR ($3::uuid IS NOT NULL AND reservation_id = $3::uuid))
		)
	`, tenantID, quoteID, reservationID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check existing invoice source: %w", err)
	}
	if exists {
		return ErrSourceConflict
	}
	return nil
}

func (r *Repository) customerSnapshot(ctx context.Context, tenantID, customerID string) (name, taxID, email, phone, address string, err error) {
	err = r.pool.QueryRow(ctx, `
		SELECT COALESCE(NULLIF(TRIM(company_name), ''), TRIM(first_name || ' ' || last_name)),
		       tax_id, COALESCE(email, ''), COALESCE(phone, ''), billing_address
		FROM customers
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, customerID).Scan(&name, &taxID, &email, &phone, &address)
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrCustomerNotFound
	} else if err != nil {
		err = fmt.Errorf("load invoice customer: %w", err)
	}
	return
}

func (r *Repository) CreateInvoice(ctx context.Context, tenantID string, input normalizedInvoice, settings Settings) (InvoiceDetail, error) {
	name, taxID, email, phone, address, err := r.customerSnapshot(ctx, tenantID, input.CustomerID)
	if err != nil {
		return InvoiceDetail{}, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return InvoiceDetail{}, fmt.Errorf("begin create invoice: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var invoiceID string
	err = tx.QueryRow(ctx, `
		INSERT INTO invoices (
			tenant_id, customer_id, quote_id, reservation_id, source_type,
			invoice_prefix, status, issue_date, due_date, currency, prices_include_tax,
			customer_name, customer_tax_id, customer_email, customer_phone, customer_address,
			seller_legal_name, seller_trade_name, seller_tax_id, seller_registration_number,
			seller_economic_activity, seller_economic_activity_code, seller_address,
			seller_email, seller_phone,
			taxable_amount, exempt_amount, non_taxable_amount, tax_amount, total_amount,
			fiscal_status, notes, terms
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, 'DRAFT', $7, $8, $9, $10,
			$11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22, $23, $24,
			$25, $26, $27, $28, $29,
			'NOT_READY', $30, $31
		)
		RETURNING id
	`, tenantID, input.CustomerID, input.QuoteID, input.ReservationID, input.SourceType,
		settings.InvoicePrefix, input.IssueDate, input.DueDate, input.Currency, input.PricesIncludeTax,
		name, taxID, email, phone, address,
		settings.LegalName, settings.TradeName, settings.TaxID, settings.TaxRegistrationNumber,
		settings.EconomicActivity, settings.EconomicActivityCode, settings.FiscalAddress,
		settings.Email, settings.Phone,
		input.TaxableAmount, input.ExemptAmount, input.NonTaxableAmount, input.TaxAmount, input.TotalAmount,
		input.Notes, input.Terms,
	).Scan(&invoiceID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return InvoiceDetail{}, ErrSourceConflict
		}
		return InvoiceDetail{}, fmt.Errorf("insert invoice: %w", err)
	}
	if err := insertInvoiceItems(ctx, tx, tenantID, invoiceID, input.Items); err != nil {
		return InvoiceDetail{}, err
	}
	if err := insertInvoiceEvent(ctx, tx, tenantID, invoiceID, "CREATED", webutil.ActorID(ctx), map[string]any{
		"source_type": input.SourceType,
		"total":       input.TotalAmount,
	}); err != nil {
		return InvoiceDetail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return InvoiceDetail{}, fmt.Errorf("commit create invoice: %w", err)
	}
	return r.GetInvoice(ctx, tenantID, invoiceID)
}

func (r *Repository) UpdateInvoice(ctx context.Context, tenantID, invoiceID string, input normalizedInvoice, settings Settings) (InvoiceDetail, error) {
	name, taxID, email, phone, address, err := r.customerSnapshot(ctx, tenantID, input.CustomerID)
	if err != nil {
		return InvoiceDetail{}, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return InvoiceDetail{}, fmt.Errorf("begin update invoice: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status string
	if err := tx.QueryRow(ctx, `
		SELECT status FROM invoices WHERE tenant_id = $1 AND id = $2 FOR UPDATE
	`, tenantID, invoiceID).Scan(&status); errors.Is(err, pgx.ErrNoRows) {
		return InvoiceDetail{}, ErrInvoiceNotFound
	} else if err != nil {
		return InvoiceDetail{}, fmt.Errorf("lock invoice: %w", err)
	}
	if status != "DRAFT" {
		return InvoiceDetail{}, ErrInvoiceImmutable
	}
	_, err = tx.Exec(ctx, `
		UPDATE invoices SET
			customer_id = $3,
			issue_date = $4,
			due_date = $5,
			currency = $6,
			prices_include_tax = $7,
			customer_name = $8,
			customer_tax_id = $9,
			customer_email = $10,
			customer_phone = $11,
			customer_address = $12,
			invoice_prefix = $13,
			seller_legal_name = $14,
			seller_trade_name = $15,
			seller_tax_id = $16,
			seller_registration_number = $17,
			seller_economic_activity = $18,
			seller_economic_activity_code = $19,
			seller_address = $20,
			seller_email = $21,
			seller_phone = $22,
			taxable_amount = $23,
			exempt_amount = $24,
			non_taxable_amount = $25,
			tax_amount = $26,
			total_amount = $27,
			notes = $28,
			terms = $29
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, invoiceID, input.CustomerID, input.IssueDate, input.DueDate,
		input.Currency, input.PricesIncludeTax, name, taxID, email, phone, address,
		settings.InvoicePrefix, settings.LegalName, settings.TradeName, settings.TaxID,
		settings.TaxRegistrationNumber, settings.EconomicActivity, settings.EconomicActivityCode,
		settings.FiscalAddress, settings.Email, settings.Phone,
		input.TaxableAmount, input.ExemptAmount, input.NonTaxableAmount, input.TaxAmount,
		input.TotalAmount, input.Notes, input.Terms)
	if err != nil {
		return InvoiceDetail{}, fmt.Errorf("update invoice: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM invoice_items WHERE tenant_id = $1 AND invoice_id = $2`, tenantID, invoiceID); err != nil {
		return InvoiceDetail{}, fmt.Errorf("delete invoice items: %w", err)
	}
	if err := insertInvoiceItems(ctx, tx, tenantID, invoiceID, input.Items); err != nil {
		return InvoiceDetail{}, err
	}
	if err := insertInvoiceEvent(ctx, tx, tenantID, invoiceID, "UPDATED", webutil.ActorID(ctx), map[string]any{
		"total": input.TotalAmount,
	}); err != nil {
		return InvoiceDetail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return InvoiceDetail{}, fmt.Errorf("commit update invoice: %w", err)
	}
	return r.GetInvoice(ctx, tenantID, invoiceID)
}

func insertInvoiceItems(ctx context.Context, tx pgx.Tx, tenantID, invoiceID string, items []normalizedInvoiceItem) error {
	for _, item := range items {
		_, err := tx.Exec(ctx, `
			INSERT INTO invoice_items (
				tenant_id, invoice_id, resource_id, tax_rule_id, description,
				quantity, unit_price, discount_amount, gross_amount, net_amount,
				tax_code, tax_category, tax_rate, tax_amount, line_total, sort_order
			) VALUES (
				$1, $2, $3, $4, $5,
				$6, $7, $8, $9, $10,
				$11, $12, $13, $14, $15, $16
			)
		`, tenantID, invoiceID, item.ResourceID, item.TaxRuleID, item.Description,
			item.Quantity, item.UnitPrice, item.DiscountAmount, item.GrossAmount,
			item.NetAmount, item.TaxCode, item.TaxCategory, item.TaxRate,
			item.TaxAmount, item.LineTotal, item.SortOrder)
		if err != nil {
			return fmt.Errorf("insert invoice item: %w", err)
		}
	}
	return nil
}

func insertInvoiceEvent(ctx context.Context, tx pgx.Tx, tenantID, invoiceID, eventType, actorID string, metadata map[string]any) error {
	payload, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal invoice event metadata: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO invoice_events (tenant_id, invoice_id, event_type, actor_id, metadata)
		VALUES ($1, $2, $3, $4, $5::jsonb)
	`, tenantID, invoiceID, eventType, actorID, string(payload))
	if err != nil {
		return fmt.Errorf("insert invoice event: %w", err)
	}
	return nil
}

func (r *Repository) ListInvoices(ctx context.Context, tenantID, search, status string, limit int) ([]InvoiceSummary, error) {
	if limit <= 0 || limit > 250 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, `
		WITH invoice_rows AS (
			SELECT invoice.id, invoice.invoice_number, invoice.invoice_prefix,
			       invoice.customer_id, invoice.customer_name,
			       invoice.quote_id, quote.quote_number,
			       invoice.reservation_id, reservation.reservation_number,
			       invoice.source_type, invoice.status,
			       CASE
			         WHEN invoice.status IN ('ISSUED', 'PARTIALLY_PAID')
			          AND invoice.due_date < CURRENT_DATE AND invoice.balance_due > 0
			         THEN 'OVERDUE'
			         ELSE invoice.status
			       END AS display_status,
			       invoice.issue_date::text, invoice.due_date::text, invoice.currency,
			       invoice.prices_include_tax,
			       invoice.taxable_amount::float8, invoice.exempt_amount::float8,
			       invoice.non_taxable_amount::float8, invoice.tax_amount::float8,
			       invoice.total_amount::float8, invoice.paid_amount::float8,
			       invoice.balance_due::float8, invoice.fiscal_status,
			       COUNT(item.id)::int AS item_count,
			       invoice.issued_at, invoice.voided_at, invoice.created_at, invoice.updated_at
			FROM invoices invoice
			LEFT JOIN quotes quote ON quote.tenant_id = invoice.tenant_id AND quote.id = invoice.quote_id
			LEFT JOIN reservations reservation ON reservation.tenant_id = invoice.tenant_id AND reservation.id = invoice.reservation_id
			LEFT JOIN invoice_items item ON item.tenant_id = invoice.tenant_id AND item.invoice_id = invoice.id
			WHERE invoice.tenant_id = $1
			  AND (
			    $2 = '' OR invoice.customer_name ILIKE '%' || $2 || '%'
			    OR COALESCE(invoice.invoice_prefix || '-' || LPAD(invoice.invoice_number::text, 6, '0'), '') ILIKE '%' || $2 || '%'
			    OR COALESCE(quote.quote_number::text, '') ILIKE '%' || $2 || '%'
			    OR COALESCE(reservation.reservation_number::text, '') ILIKE '%' || $2 || '%'
			  )
			GROUP BY invoice.id, quote.id, reservation.id
		)
		SELECT * FROM invoice_rows
		WHERE $3 = '' OR display_status = $3 OR status = $3
		ORDER BY created_at DESC
		LIMIT $4
	`, tenantID, strings.TrimSpace(search), strings.TrimSpace(status), limit)
	if err != nil {
		return nil, fmt.Errorf("list invoices: %w", err)
	}
	defer rows.Close()
	items := make([]InvoiceSummary, 0)
	for rows.Next() {
		item, err := scanInvoiceSummary(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanInvoiceSummary(row rowScanner) (InvoiceSummary, error) {
	var item InvoiceSummary
	if err := row.Scan(
		&item.ID, &item.InvoiceNumber, &item.InvoicePrefix,
		&item.CustomerID, &item.CustomerName,
		&item.QuoteID, &item.QuoteNumber,
		&item.ReservationID, &item.ReservationNumber,
		&item.SourceType, &item.Status, &item.DisplayStatus,
		&item.IssueDate, &item.DueDate, &item.Currency, &item.PricesIncludeTax,
		&item.TaxableAmount, &item.ExemptAmount, &item.NonTaxableAmount,
		&item.TaxAmount, &item.TotalAmount, &item.PaidAmount, &item.BalanceDue,
		&item.FiscalStatus, &item.ItemCount,
		&item.IssuedAt, &item.VoidedAt, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return InvoiceSummary{}, fmt.Errorf("scan invoice summary: %w", err)
	}
	item.DisplayNumber = invoiceDisplayNumber(item.InvoicePrefix, item.InvoiceNumber)
	return item, nil
}

func (r *Repository) GetInvoice(ctx context.Context, tenantID, invoiceID string) (InvoiceDetail, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT invoice.id, invoice.invoice_number, invoice.invoice_prefix,
		       invoice.customer_id, invoice.customer_name,
		       invoice.quote_id, quote.quote_number,
		       invoice.reservation_id, reservation.reservation_number,
		       invoice.source_type, invoice.status,
		       CASE
		         WHEN invoice.status IN ('ISSUED', 'PARTIALLY_PAID')
		          AND invoice.due_date < CURRENT_DATE AND invoice.balance_due > 0
		         THEN 'OVERDUE'
		         ELSE invoice.status
		       END,
		       invoice.issue_date::text, invoice.due_date::text, invoice.currency,
		       invoice.prices_include_tax,
		       invoice.taxable_amount::float8, invoice.exempt_amount::float8,
		       invoice.non_taxable_amount::float8, invoice.tax_amount::float8,
		       invoice.total_amount::float8, invoice.paid_amount::float8,
		       invoice.balance_due::float8, invoice.fiscal_status,
		       (SELECT COUNT(*)::int FROM invoice_items item
		        WHERE item.tenant_id = invoice.tenant_id AND item.invoice_id = invoice.id),
		       invoice.issued_at, invoice.voided_at, invoice.created_at, invoice.updated_at,
		       invoice.customer_tax_id, invoice.customer_email, invoice.customer_phone,
		       invoice.customer_address,
		       invoice.seller_legal_name, invoice.seller_trade_name, invoice.seller_tax_id,
		       invoice.seller_registration_number, invoice.seller_economic_activity,
		       invoice.seller_economic_activity_code, invoice.seller_address,
		       invoice.seller_email, invoice.seller_phone,
		       invoice.notes, invoice.terms, invoice.void_reason
		FROM invoices invoice
		LEFT JOIN quotes quote ON quote.tenant_id = invoice.tenant_id AND quote.id = invoice.quote_id
		LEFT JOIN reservations reservation ON reservation.tenant_id = invoice.tenant_id AND reservation.id = invoice.reservation_id
		WHERE invoice.tenant_id = $1 AND invoice.id = $2
	`, tenantID, invoiceID)
	var detail InvoiceDetail
	var summary InvoiceSummary
	if err := row.Scan(
		&summary.ID, &summary.InvoiceNumber, &summary.InvoicePrefix,
		&summary.CustomerID, &summary.CustomerName,
		&summary.QuoteID, &summary.QuoteNumber,
		&summary.ReservationID, &summary.ReservationNumber,
		&summary.SourceType, &summary.Status, &summary.DisplayStatus,
		&summary.IssueDate, &summary.DueDate, &summary.Currency, &summary.PricesIncludeTax,
		&summary.TaxableAmount, &summary.ExemptAmount, &summary.NonTaxableAmount,
		&summary.TaxAmount, &summary.TotalAmount, &summary.PaidAmount, &summary.BalanceDue,
		&summary.FiscalStatus, &summary.ItemCount,
		&summary.IssuedAt, &summary.VoidedAt, &summary.CreatedAt, &summary.UpdatedAt,
		&detail.CustomerTaxID, &detail.CustomerEmail, &detail.CustomerPhone,
		&detail.CustomerAddress,
		&detail.SellerLegalName, &detail.SellerTradeName, &detail.SellerTaxID,
		&detail.SellerRegistrationNumber, &detail.SellerEconomicActivity,
		&detail.SellerEconomicActivityCode, &detail.SellerAddress,
		&detail.SellerEmail, &detail.SellerPhone,
		&detail.Notes, &detail.Terms, &detail.VoidReason,
	); errors.Is(err, pgx.ErrNoRows) {
		return InvoiceDetail{}, ErrInvoiceNotFound
	} else if err != nil {
		return InvoiceDetail{}, fmt.Errorf("get invoice: %w", err)
	}
	summary.DisplayNumber = invoiceDisplayNumber(summary.InvoicePrefix, summary.InvoiceNumber)
	detail.InvoiceSummary = summary

	rows, err := r.pool.Query(ctx, `
		SELECT id, resource_id, tax_rule_id, description, quantity::float8,
		       unit_price::float8, discount_amount::float8, gross_amount::float8,
		       net_amount::float8, tax_code, tax_category, tax_rate::float8,
		       tax_amount::float8, line_total::float8, sort_order
		FROM invoice_items
		WHERE tenant_id = $1 AND invoice_id = $2
		ORDER BY sort_order, id
	`, tenantID, invoiceID)
	if err != nil {
		return InvoiceDetail{}, fmt.Errorf("list invoice items: %w", err)
	}
	detail.Items = make([]InvoiceItem, 0)
	for rows.Next() {
		var item InvoiceItem
		if err := rows.Scan(&item.ID, &item.ResourceID, &item.TaxRuleID, &item.Description,
			&item.Quantity, &item.UnitPrice, &item.DiscountAmount, &item.GrossAmount,
			&item.NetAmount, &item.TaxCode, &item.TaxCategory, &item.TaxRate,
			&item.TaxAmount, &item.LineTotal, &item.SortOrder); err != nil {
			rows.Close()
			return InvoiceDetail{}, fmt.Errorf("scan invoice item: %w", err)
		}
		detail.Items = append(detail.Items, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return InvoiceDetail{}, err
	}
	rows.Close()

	eventRows, err := r.pool.Query(ctx, `
		SELECT id, event_type, actor_id, metadata, created_at
		FROM invoice_events
		WHERE tenant_id = $1 AND invoice_id = $2
		ORDER BY created_at DESC, id DESC
	`, tenantID, invoiceID)
	if err != nil {
		return InvoiceDetail{}, fmt.Errorf("list invoice events: %w", err)
	}
	detail.Events = make([]InvoiceEvent, 0)
	for eventRows.Next() {
		var item InvoiceEvent
		var payload []byte
		if err := eventRows.Scan(&item.ID, &item.EventType, &item.ActorID, &payload, &item.CreatedAt); err != nil {
			eventRows.Close()
			return InvoiceDetail{}, fmt.Errorf("scan invoice event: %w", err)
		}
		if err := json.Unmarshal(payload, &item.Metadata); err != nil {
			item.Metadata = map[string]any{}
		}
		detail.Events = append(detail.Events, item)
	}
	if err := eventRows.Err(); err != nil {
		eventRows.Close()
		return InvoiceDetail{}, err
	}
	eventRows.Close()

	allocationRows, err := r.pool.Query(ctx, `
		SELECT allocation.id, payment.id, payment.payment_number,
		       allocation.invoice_id, invoice.invoice_number,
		       invoice.invoice_prefix, allocation.amount::float8
		FROM payment_allocations allocation
		JOIN payments payment ON payment.tenant_id = allocation.tenant_id AND payment.id = allocation.payment_id
		JOIN invoices invoice ON invoice.tenant_id = allocation.tenant_id AND invoice.id = allocation.invoice_id
		WHERE allocation.tenant_id = $1 AND allocation.invoice_id = $2
		  AND payment.status = 'CONFIRMED'
		ORDER BY payment.received_at DESC, allocation.created_at DESC
	`, tenantID, invoiceID)
	if err != nil {
		return InvoiceDetail{}, fmt.Errorf("list invoice payment allocations: %w", err)
	}
	defer allocationRows.Close()
	detail.Allocations = make([]PaymentAllocation, 0)
	for allocationRows.Next() {
		var item PaymentAllocation
		if err := allocationRows.Scan(&item.ID, &item.PaymentID, &item.PaymentNumber,
			&item.InvoiceID, &item.InvoiceNumber, &item.InvoicePrefix, &item.Amount); err != nil {
			return InvoiceDetail{}, fmt.Errorf("scan invoice payment allocation: %w", err)
		}
		if item.PaymentNumber != nil {
			item.PaymentDisplayNumber = paymentDisplayNumber(*item.PaymentNumber)
		}
		item.DisplayNumber = invoiceDisplayNumber(item.InvoicePrefix, item.InvoiceNumber)
		detail.Allocations = append(detail.Allocations, item)
	}
	return detail, allocationRows.Err()
}

func (r *Repository) IssueInvoice(ctx context.Context, tenantID, invoiceID string) (InvoiceDetail, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return InvoiceDetail{}, fmt.Errorf("begin issue invoice: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status string
	var itemCount int
	if err := tx.QueryRow(ctx, `
		SELECT status, (SELECT COUNT(*)::int FROM invoice_items item
		                WHERE item.tenant_id = invoice.tenant_id AND item.invoice_id = invoice.id)
		FROM invoices invoice
		WHERE tenant_id = $1 AND id = $2
		FOR UPDATE
	`, tenantID, invoiceID).Scan(&status, &itemCount); errors.Is(err, pgx.ErrNoRows) {
		return InvoiceDetail{}, ErrInvoiceNotFound
	} else if err != nil {
		return InvoiceDetail{}, fmt.Errorf("lock invoice for issue: %w", err)
	}
	if status != "DRAFT" || itemCount == 0 {
		return InvoiceDetail{}, ErrInvoiceState
	}

	var settings Settings
	row := tx.QueryRow(ctx, `
		SELECT tenant_id, enabled, legal_name, trade_name, tax_id,
		       tax_registration_number, economic_activity, economic_activity_code,
		       fiscal_address, department, municipality, district,
		       department_code, municipality_code, district_code, email, phone,
		       prices_include_tax, default_tax_rate::float8,
		       default_payment_terms_days, invoice_prefix, next_invoice_number,
		       created_at, updated_at
		FROM billing_settings
		WHERE tenant_id = $1
		FOR UPDATE
	`, tenantID)
	settings, err = scanSettings(row)
	if err != nil {
		return InvoiceDetail{}, err
	}
	if !settings.Enabled {
		return InvoiceDetail{}, ErrBillingDisabled
	}
	fiscalStatus := "NOT_READY"
	if settings.FiscalProfileComplete {
		fiscalStatus = "READY_FOR_DTE"
	}
	invoiceNumber := settings.NextInvoiceNumber
	_, err = tx.Exec(ctx, `
		UPDATE invoices invoice SET
			invoice_number = $3,
			invoice_prefix = $4,
			status = 'ISSUED',
			issued_at = NOW(),
			issued_by = $5,
			fiscal_status = $6,
			seller_legal_name = $7,
			seller_trade_name = $8,
			seller_tax_id = $9,
			seller_registration_number = $10,
			seller_economic_activity = $11,
			seller_economic_activity_code = $12,
			seller_address = $13,
			seller_email = $14,
			seller_phone = $15,
			seller_department_code = $16,
			seller_municipality_code = $17,
			seller_district_code = $18,
			customer_registration_number = customer.tax_registration_number,
			customer_document_type_code = customer.document_type_code,
			customer_trade_name = customer.trade_name,
			customer_economic_activity = customer.economic_activity,
			customer_economic_activity_code = customer.economic_activity_code,
			customer_department_code = customer.department_code,
			customer_municipality_code = customer.municipality_code,
			customer_district_code = customer.district_code
		FROM customers customer
		WHERE invoice.tenant_id = $1 AND invoice.id = $2
		  AND customer.tenant_id = invoice.tenant_id
		  AND customer.id = invoice.customer_id
	`, tenantID, invoiceID, invoiceNumber, settings.InvoicePrefix, nullableUUID(webutil.UserID(ctx)),
		fiscalStatus, settings.LegalName, settings.TradeName, settings.TaxID,
		settings.TaxRegistrationNumber, settings.EconomicActivity, settings.EconomicActivityCode,
		settings.FiscalAddress, settings.Email, settings.Phone,
		settings.DepartmentCode, settings.MunicipalityCode, settings.DistrictCode)
	if err != nil {
		return InvoiceDetail{}, fmt.Errorf("issue invoice: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE billing_settings SET next_invoice_number = next_invoice_number + 1
		WHERE tenant_id = $1
	`, tenantID); err != nil {
		return InvoiceDetail{}, fmt.Errorf("advance invoice number: %w", err)
	}
	if err := insertInvoiceEvent(ctx, tx, tenantID, invoiceID, "ISSUED", webutil.ActorID(ctx), map[string]any{
		"invoice_number": invoiceNumber,
		"fiscal_status":  fiscalStatus,
	}); err != nil {
		return InvoiceDetail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return InvoiceDetail{}, fmt.Errorf("commit issue invoice: %w", err)
	}
	return r.GetInvoice(ctx, tenantID, invoiceID)
}

func (r *Repository) VoidInvoice(ctx context.Context, tenantID, invoiceID, reason string) (InvoiceDetail, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return InvoiceDetail{}, fmt.Errorf("begin void invoice: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status string
	var fiscalStatus string
	var paidAmount float64
	var hasActiveDTE bool
	if err := tx.QueryRow(ctx, `
		SELECT invoice.status, invoice.fiscal_status, invoice.paid_amount::float8,
		       EXISTS (
		         SELECT 1 FROM dte_documents document
		         WHERE document.tenant_id = invoice.tenant_id
		           AND document.invoice_id = invoice.id
		           AND document.status NOT IN ('REJECTED', 'INVALIDATED', 'CANCELLED')
		       )
		FROM invoices invoice
		WHERE invoice.tenant_id = $1 AND invoice.id = $2
		FOR UPDATE
	`, tenantID, invoiceID).Scan(&status, &fiscalStatus, &paidAmount, &hasActiveDTE); errors.Is(err, pgx.ErrNoRows) {
		return InvoiceDetail{}, ErrInvoiceNotFound
	} else if err != nil {
		return InvoiceDetail{}, fmt.Errorf("lock invoice for void: %w", err)
	}
	if (status != "DRAFT" && status != "ISSUED") || moneyCents(paidAmount) != 0 {
		return InvoiceDetail{}, ErrInvoiceState
	}
	if fiscalStatus == "SUBMITTED" || fiscalStatus == "ACCEPTED" || hasActiveDTE {
		return InvoiceDetail{}, ErrInvoiceState
	}
	_, err = tx.Exec(ctx, `
		UPDATE invoices SET
			status = 'VOID',
			fiscal_status = 'VOIDED',
			voided_at = NOW(),
			voided_by = $3,
			void_reason = $4
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, invoiceID, nullableUUID(webutil.UserID(ctx)), reason)
	if err != nil {
		return InvoiceDetail{}, fmt.Errorf("void invoice: %w", err)
	}
	if err := insertInvoiceEvent(ctx, tx, tenantID, invoiceID, "VOIDED", webutil.ActorID(ctx), map[string]any{
		"reason": reason,
	}); err != nil {
		return InvoiceDetail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return InvoiceDetail{}, fmt.Errorf("commit void invoice: %w", err)
	}
	return r.GetInvoice(ctx, tenantID, invoiceID)
}

func (r *Repository) ListPayments(ctx context.Context, tenantID, search, status string, limit int) ([]PaymentSummary, error) {
	if limit <= 0 || limit > 250 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, `
		SELECT payment.id, payment.payment_number,
		       payment.customer_id,
		       COALESCE(NULLIF(TRIM(customer.company_name), ''), TRIM(customer.first_name || ' ' || customer.last_name)),
		       payment.status, payment.amount::float8, payment.currency, payment.method,
		       payment.reference, payment.received_at,
		       COUNT(allocation.id)::int, payment.voided_at,
		       payment.created_at, payment.updated_at
		FROM payments payment
		JOIN customers customer ON customer.tenant_id = payment.tenant_id AND customer.id = payment.customer_id
		LEFT JOIN payment_allocations allocation
		  ON allocation.tenant_id = payment.tenant_id AND allocation.payment_id = payment.id
		WHERE payment.tenant_id = $1
		  AND ($2 = '' OR customer.first_name ILIKE '%' || $2 || '%'
		       OR customer.last_name ILIKE '%' || $2 || '%'
		       OR COALESCE(customer.company_name, '') ILIKE '%' || $2 || '%'
		       OR payment.reference ILIKE '%' || $2 || '%'
		       OR ('PMT-' || LPAD(payment.payment_number::text, 6, '0')) ILIKE '%' || $2 || '%')
		  AND ($3 = '' OR payment.status = $3)
		GROUP BY payment.id, customer.id
		ORDER BY payment.received_at DESC, payment.created_at DESC
		LIMIT $4
	`, tenantID, strings.TrimSpace(search), strings.TrimSpace(status), limit)
	if err != nil {
		return nil, fmt.Errorf("list payments: %w", err)
	}
	defer rows.Close()
	items := make([]PaymentSummary, 0)
	for rows.Next() {
		item, err := scanPaymentSummary(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanPaymentSummary(row rowScanner) (PaymentSummary, error) {
	var item PaymentSummary
	if err := row.Scan(&item.ID, &item.PaymentNumber, &item.CustomerID, &item.CustomerName,
		&item.Status, &item.Amount, &item.Currency, &item.Method, &item.Reference,
		&item.ReceivedAt, &item.AllocationCount, &item.VoidedAt,
		&item.CreatedAt, &item.UpdatedAt); err != nil {
		return PaymentSummary{}, fmt.Errorf("scan payment summary: %w", err)
	}
	item.DisplayNumber = paymentDisplayNumber(item.PaymentNumber)
	return item, nil
}

func (r *Repository) GetPayment(ctx context.Context, tenantID, paymentID string) (PaymentDetail, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT payment.id, payment.payment_number,
		       payment.customer_id,
		       COALESCE(NULLIF(TRIM(customer.company_name), ''), TRIM(customer.first_name || ' ' || customer.last_name)),
		       payment.status, payment.amount::float8, payment.currency, payment.method,
		       payment.reference, payment.received_at,
		       (SELECT COUNT(*)::int FROM payment_allocations allocation
		        WHERE allocation.tenant_id = payment.tenant_id AND allocation.payment_id = payment.id),
		       payment.voided_at, payment.created_at, payment.updated_at,
		       payment.notes, payment.void_reason
		FROM payments payment
		JOIN customers customer ON customer.tenant_id = payment.tenant_id AND customer.id = payment.customer_id
		WHERE payment.tenant_id = $1 AND payment.id = $2
	`, tenantID, paymentID)
	var detail PaymentDetail
	var summary PaymentSummary
	if err := row.Scan(&summary.ID, &summary.PaymentNumber, &summary.CustomerID, &summary.CustomerName,
		&summary.Status, &summary.Amount, &summary.Currency, &summary.Method, &summary.Reference,
		&summary.ReceivedAt, &summary.AllocationCount, &summary.VoidedAt,
		&summary.CreatedAt, &summary.UpdatedAt, &detail.Notes, &detail.VoidReason); errors.Is(err, pgx.ErrNoRows) {
		return PaymentDetail{}, ErrPaymentNotFound
	} else if err != nil {
		return PaymentDetail{}, fmt.Errorf("get payment: %w", err)
	}
	summary.DisplayNumber = paymentDisplayNumber(summary.PaymentNumber)
	detail.PaymentSummary = summary
	rows, err := r.pool.Query(ctx, `
		SELECT allocation.id, allocation.invoice_id, invoice.invoice_number,
		       invoice.invoice_prefix, allocation.amount::float8
		FROM payment_allocations allocation
		JOIN invoices invoice ON invoice.tenant_id = allocation.tenant_id AND invoice.id = allocation.invoice_id
		WHERE allocation.tenant_id = $1 AND allocation.payment_id = $2
		ORDER BY allocation.created_at, allocation.id
	`, tenantID, paymentID)
	if err != nil {
		return PaymentDetail{}, fmt.Errorf("list payment allocations: %w", err)
	}
	defer rows.Close()
	detail.Allocations = make([]PaymentAllocation, 0)
	for rows.Next() {
		var item PaymentAllocation
		if err := rows.Scan(&item.ID, &item.InvoiceID, &item.InvoiceNumber,
			&item.InvoicePrefix, &item.Amount); err != nil {
			return PaymentDetail{}, fmt.Errorf("scan payment allocation: %w", err)
		}
		item.PaymentID = paymentID
		paymentNumber := summary.PaymentNumber
		item.PaymentNumber = &paymentNumber
		item.PaymentDisplayNumber = summary.DisplayNumber
		item.DisplayNumber = invoiceDisplayNumber(item.InvoicePrefix, item.InvoiceNumber)
		detail.Allocations = append(detail.Allocations, item)
	}
	return detail, rows.Err()
}

func (r *Repository) CreatePayment(ctx context.Context, tenantID string, input normalizedPayment) (PaymentDetail, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return PaymentDetail{}, fmt.Errorf("begin create payment: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	allocations := append([]PaymentAllocationInput(nil), input.Allocations...)
	sort.Slice(allocations, func(i, j int) bool { return allocations[i].InvoiceID < allocations[j].InvoiceID })
	type lockedInvoice struct {
		ID      string
		Amount  float64
		Balance float64
	}
	locked := make([]lockedInvoice, 0, len(allocations))
	for _, allocation := range allocations {
		var customerID, currency, status string
		var balance float64
		err := tx.QueryRow(ctx, `
			SELECT customer_id, currency, status, balance_due::float8
			FROM invoices
			WHERE tenant_id = $1 AND id = $2
			FOR UPDATE
		`, tenantID, allocation.InvoiceID).Scan(&customerID, &currency, &status, &balance)
		if errors.Is(err, pgx.ErrNoRows) {
			return PaymentDetail{}, ErrInvoiceNotFound
		}
		if err != nil {
			return PaymentDetail{}, fmt.Errorf("lock invoice for payment: %w", err)
		}
		if customerID != input.CustomerID {
			return PaymentDetail{}, ErrCustomerMismatch
		}
		if currency != input.Currency {
			return PaymentDetail{}, ErrCurrencyMismatch
		}
		if status != "ISSUED" && status != "PARTIALLY_PAID" {
			return PaymentDetail{}, ErrInvoiceState
		}
		if moneyCents(allocation.Amount) > moneyCents(balance) {
			return PaymentDetail{}, ErrAllocationConflict
		}
		locked = append(locked, lockedInvoice{ID: allocation.InvoiceID, Amount: roundMoney(allocation.Amount), Balance: balance})
	}

	var paymentID string
	err = tx.QueryRow(ctx, `
		INSERT INTO payments (
			tenant_id, customer_id, status, amount, currency, method,
			reference, notes, received_at, created_by
		) VALUES ($1, $2, 'CONFIRMED', $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`, tenantID, input.CustomerID, input.Amount, input.Currency, input.Method,
		input.Reference, input.Notes, input.ReceivedAt, nullableUUID(webutil.UserID(ctx))).Scan(&paymentID)
	if err != nil {
		return PaymentDetail{}, fmt.Errorf("insert payment: %w", err)
	}
	for _, invoice := range locked {
		_, err := tx.Exec(ctx, `
			INSERT INTO payment_allocations (tenant_id, payment_id, invoice_id, amount)
			VALUES ($1, $2, $3, $4)
		`, tenantID, paymentID, invoice.ID, invoice.Amount)
		if err != nil {
			return PaymentDetail{}, fmt.Errorf("insert payment allocation: %w", err)
		}
		_, err = tx.Exec(ctx, `
			UPDATE invoices SET
				paid_amount = paid_amount + $3,
				status = CASE
				  WHEN paid_amount + $3 >= total_amount THEN 'PAID'
				  ELSE 'PARTIALLY_PAID'
				END
			WHERE tenant_id = $1 AND id = $2
		`, tenantID, invoice.ID, invoice.Amount)
		if err != nil {
			return PaymentDetail{}, fmt.Errorf("apply payment to invoice: %w", err)
		}
		if err := insertInvoiceEvent(ctx, tx, tenantID, invoice.ID, "PAYMENT_APPLIED", webutil.ActorID(ctx), map[string]any{
			"payment_id": paymentID,
			"amount":     invoice.Amount,
		}); err != nil {
			return PaymentDetail{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return PaymentDetail{}, fmt.Errorf("commit payment: %w", err)
	}
	return r.GetPayment(ctx, tenantID, paymentID)
}

func (r *Repository) VoidPayment(ctx context.Context, tenantID, paymentID, reason string) (PaymentDetail, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return PaymentDetail{}, fmt.Errorf("begin void payment: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status string
	if err := tx.QueryRow(ctx, `
		SELECT status FROM payments WHERE tenant_id = $1 AND id = $2 FOR UPDATE
	`, tenantID, paymentID).Scan(&status); errors.Is(err, pgx.ErrNoRows) {
		return PaymentDetail{}, ErrPaymentNotFound
	} else if err != nil {
		return PaymentDetail{}, fmt.Errorf("lock payment: %w", err)
	}
	if status != "CONFIRMED" {
		return PaymentDetail{}, ErrPaymentState
	}
	rows, err := tx.Query(ctx, `
		SELECT invoice_id, amount::float8
		FROM payment_allocations
		WHERE tenant_id = $1 AND payment_id = $2
		ORDER BY invoice_id
	`, tenantID, paymentID)
	if err != nil {
		return PaymentDetail{}, fmt.Errorf("load payment allocations for void: %w", err)
	}
	type allocation struct {
		InvoiceID string
		Amount    float64
	}
	allocations := make([]allocation, 0)
	for rows.Next() {
		var item allocation
		if err := rows.Scan(&item.InvoiceID, &item.Amount); err != nil {
			rows.Close()
			return PaymentDetail{}, fmt.Errorf("scan payment allocation for void: %w", err)
		}
		allocations = append(allocations, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return PaymentDetail{}, err
	}
	rows.Close()
	for _, allocation := range allocations {
		var currentStatus string
		var paid, total float64
		if err := tx.QueryRow(ctx, `
			SELECT status, paid_amount::float8, total_amount::float8
			FROM invoices
			WHERE tenant_id = $1 AND id = $2
			FOR UPDATE
		`, tenantID, allocation.InvoiceID).Scan(&currentStatus, &paid, &total); err != nil {
			return PaymentDetail{}, fmt.Errorf("lock allocated invoice for payment void: %w", err)
		}
		newPaidCents := moneyCents(paid) - moneyCents(allocation.Amount)
		if newPaidCents < 0 {
			return PaymentDetail{}, ErrPaymentState
		}
		newStatus := "ISSUED"
		if newPaidCents > 0 && newPaidCents < moneyCents(total) {
			newStatus = "PARTIALLY_PAID"
		} else if newPaidCents >= moneyCents(total) {
			newStatus = "PAID"
		}
		_, err := tx.Exec(ctx, `
			UPDATE invoices SET paid_amount = $3, status = $4
			WHERE tenant_id = $1 AND id = $2
		`, tenantID, allocation.InvoiceID, moneyValue(newPaidCents), newStatus)
		if err != nil {
			return PaymentDetail{}, fmt.Errorf("reverse invoice payment: %w", err)
		}
		if err := insertInvoiceEvent(ctx, tx, tenantID, allocation.InvoiceID, "PAYMENT_VOIDED", webutil.ActorID(ctx), map[string]any{
			"payment_id": paymentID,
			"amount":     allocation.Amount,
			"reason":     reason,
		}); err != nil {
			return PaymentDetail{}, err
		}
	}
	_, err = tx.Exec(ctx, `
		UPDATE payments SET
			status = 'VOIDED', voided_at = NOW(), voided_by = $3, void_reason = $4
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, paymentID, nullableUUID(webutil.UserID(ctx)), reason)
	if err != nil {
		return PaymentDetail{}, fmt.Errorf("void payment: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return PaymentDetail{}, fmt.Errorf("commit payment void: %w", err)
	}
	return r.GetPayment(ctx, tenantID, paymentID)
}

func (r *Repository) ListDeposits(ctx context.Context, tenantID, status string, limit int) ([]SecurityDeposit, error) {
	if limit <= 0 || limit > 250 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, `
		SELECT deposit.id, deposit.deposit_number, deposit.reservation_id,
		       reservation.reservation_number, deposit.customer_id,
		       COALESCE(NULLIF(TRIM(customer.company_name), ''), TRIM(customer.first_name || ' ' || customer.last_name)),
		       deposit.status, deposit.amount::float8, deposit.returned_amount::float8,
		       deposit.retained_amount::float8, deposit.balance_amount::float8,
		       deposit.currency, deposit.method, deposit.reference, deposit.notes,
		       deposit.received_at, deposit.settled_at, deposit.settlement_reason,
		       deposit.created_at, deposit.updated_at
		FROM security_deposits deposit
		JOIN reservations reservation ON reservation.tenant_id = deposit.tenant_id AND reservation.id = deposit.reservation_id
		JOIN customers customer ON customer.tenant_id = deposit.tenant_id AND customer.id = deposit.customer_id
		WHERE deposit.tenant_id = $1 AND ($2 = '' OR deposit.status = $2)
		ORDER BY deposit.created_at DESC
		LIMIT $3
	`, tenantID, strings.TrimSpace(status), limit)
	if err != nil {
		return nil, fmt.Errorf("list security deposits: %w", err)
	}
	defer rows.Close()
	items := make([]SecurityDeposit, 0)
	for rows.Next() {
		item, err := scanDeposit(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanDeposit(row rowScanner) (SecurityDeposit, error) {
	var item SecurityDeposit
	if err := row.Scan(
		&item.ID, &item.DepositNumber, &item.ReservationID, &item.ReservationNumber,
		&item.CustomerID, &item.CustomerName, &item.Status, &item.Amount,
		&item.ReturnedAmount, &item.RetainedAmount, &item.BalanceAmount,
		&item.Currency, &item.Method, &item.Reference, &item.Notes,
		&item.ReceivedAt, &item.SettledAt, &item.SettlementReason,
		&item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return SecurityDeposit{}, fmt.Errorf("scan security deposit: %w", err)
	}
	item.DisplayNumber = depositDisplayNumber(item.DepositNumber)
	return item, nil
}

func (r *Repository) GetDeposit(ctx context.Context, tenantID, depositID string) (SecurityDeposit, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT deposit.id, deposit.deposit_number, deposit.reservation_id,
		       reservation.reservation_number, deposit.customer_id,
		       COALESCE(NULLIF(TRIM(customer.company_name), ''), TRIM(customer.first_name || ' ' || customer.last_name)),
		       deposit.status, deposit.amount::float8, deposit.returned_amount::float8,
		       deposit.retained_amount::float8, deposit.balance_amount::float8,
		       deposit.currency, deposit.method, deposit.reference, deposit.notes,
		       deposit.received_at, deposit.settled_at, deposit.settlement_reason,
		       deposit.created_at, deposit.updated_at
		FROM security_deposits deposit
		JOIN reservations reservation ON reservation.tenant_id = deposit.tenant_id AND reservation.id = deposit.reservation_id
		JOIN customers customer ON customer.tenant_id = deposit.tenant_id AND customer.id = deposit.customer_id
		WHERE deposit.tenant_id = $1 AND deposit.id = $2
	`, tenantID, depositID)
	item, err := scanDeposit(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return SecurityDeposit{}, ErrDepositNotFound
	}
	return item, err
}

func (r *Repository) CreateDeposit(ctx context.Context, tenantID string, input CreateDepositInput, receivedAt *time.Time) (SecurityDeposit, error) {
	var customerID, currency string
	err := r.pool.QueryRow(ctx, `
		SELECT customer_id, tenant.currency
		FROM reservations reservation
		JOIN tenants tenant ON tenant.id = reservation.tenant_id
		WHERE reservation.tenant_id = $1 AND reservation.id = $2
		  AND reservation.status <> 'CANCELLED'
	`, tenantID, input.ReservationID).Scan(&customerID, &currency)
	if errors.Is(err, pgx.ErrNoRows) {
		return SecurityDeposit{}, ErrSourceNotFound
	}
	if err != nil {
		return SecurityDeposit{}, fmt.Errorf("load deposit reservation: %w", err)
	}
	if input.Currency != "" && input.Currency != currency {
		return SecurityDeposit{}, ErrCurrencyMismatch
	}
	status := "PENDING"
	if receivedAt != nil {
		status = "RECEIVED"
	}
	var depositID string
	err = r.pool.QueryRow(ctx, `
		INSERT INTO security_deposits (
			tenant_id, reservation_id, customer_id, status, amount, currency,
			method, reference, notes, received_at, created_by, updated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $11)
		RETURNING id
	`, tenantID, input.ReservationID, customerID, status, roundMoney(input.Amount),
		currency, input.Method, input.Reference, input.Notes, receivedAt,
		nullableUUID(webutil.UserID(ctx))).Scan(&depositID)
	if err != nil {
		return SecurityDeposit{}, fmt.Errorf("insert security deposit: %w", err)
	}
	return r.GetDeposit(ctx, tenantID, depositID)
}

func (r *Repository) ReceiveDeposit(ctx context.Context, tenantID, depositID string, input ReceiveDepositInput, receivedAt time.Time) (SecurityDeposit, error) {
	command, err := r.pool.Exec(ctx, `
		UPDATE security_deposits SET
			status = 'RECEIVED', received_at = $3, method = $4,
			reference = $5, notes = CASE WHEN $6 = '' THEN notes ELSE $6 END,
			updated_by = $7
		WHERE tenant_id = $1 AND id = $2 AND status = 'PENDING'
	`, tenantID, depositID, receivedAt, input.Method, input.Reference, input.Notes,
		nullableUUID(webutil.UserID(ctx)))
	if err != nil {
		return SecurityDeposit{}, fmt.Errorf("receive security deposit: %w", err)
	}
	if command.RowsAffected() == 0 {
		var exists bool
		if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM security_deposits WHERE tenant_id = $1 AND id = $2)`, tenantID, depositID).Scan(&exists); err != nil {
			return SecurityDeposit{}, fmt.Errorf("check security deposit: %w", err)
		}
		if !exists {
			return SecurityDeposit{}, ErrDepositNotFound
		}
		return SecurityDeposit{}, ErrDepositState
	}
	return r.GetDeposit(ctx, tenantID, depositID)
}

func (r *Repository) SettleDeposit(ctx context.Context, tenantID, depositID string, input SettleDepositInput, settledAt time.Time) (SecurityDeposit, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return SecurityDeposit{}, fmt.Errorf("begin settle security deposit: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status string
	var amount float64
	if err := tx.QueryRow(ctx, `
		SELECT status, amount::float8
		FROM security_deposits
		WHERE tenant_id = $1 AND id = $2
		FOR UPDATE
	`, tenantID, depositID).Scan(&status, &amount); errors.Is(err, pgx.ErrNoRows) {
		return SecurityDeposit{}, ErrDepositNotFound
	} else if err != nil {
		return SecurityDeposit{}, fmt.Errorf("lock security deposit: %w", err)
	}
	if status == "PENDING" || status == "RETURNED" || status == "RETAINED" || status == "SETTLED" {
		return SecurityDeposit{}, ErrDepositState
	}
	returnedCents := moneyCents(input.ReturnedAmount)
	retainedCents := moneyCents(input.RetainedAmount)
	amountCents := moneyCents(amount)
	if returnedCents < 0 || retainedCents < 0 || returnedCents+retainedCents > amountCents {
		return SecurityDeposit{}, ErrDepositState
	}
	newStatus := "RECEIVED"
	totalSettled := returnedCents + retainedCents
	switch {
	case totalSettled == 0:
		newStatus = "RECEIVED"
	case totalSettled < amountCents:
		newStatus = "PARTIALLY_SETTLED"
	case returnedCents == amountCents:
		newStatus = "RETURNED"
	case retainedCents == amountCents:
		newStatus = "RETAINED"
	default:
		newStatus = "SETTLED"
	}
	var settledValue *time.Time
	if totalSettled > 0 {
		settledValue = &settledAt
	}
	_, err = tx.Exec(ctx, `
		UPDATE security_deposits SET
			status = $3, returned_amount = $4, retained_amount = $5,
			settled_at = $6, settlement_reason = $7, updated_by = $8
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, depositID, newStatus, moneyValue(returnedCents), moneyValue(retainedCents),
		settledValue, input.Reason, nullableUUID(webutil.UserID(ctx)))
	if err != nil {
		return SecurityDeposit{}, fmt.Errorf("settle security deposit: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return SecurityDeposit{}, fmt.Errorf("commit security deposit settlement: %w", err)
	}
	return r.GetDeposit(ctx, tenantID, depositID)
}

func (r *Repository) Dashboard(ctx context.Context, tenantID string) (BillingDashboard, error) {
	settings, err := r.GetSettings(ctx, tenantID)
	if err != nil {
		return BillingDashboard{}, err
	}
	result := BillingDashboard{GeneratedAt: time.Now().UTC(), Settings: settings}
	if err := r.pool.QueryRow(ctx, `
		SELECT tenant.currency,
		       COALESCE(SUM(invoice.total_amount) FILTER (WHERE invoice.status IN ('ISSUED', 'PARTIALLY_PAID', 'PAID')), 0)::float8,
		       COALESCE(SUM(invoice.paid_amount) FILTER (WHERE invoice.status IN ('ISSUED', 'PARTIALLY_PAID', 'PAID')), 0)::float8,
		       COALESCE(SUM(invoice.balance_due) FILTER (WHERE invoice.status IN ('ISSUED', 'PARTIALLY_PAID')), 0)::float8,
		       COALESCE(SUM(invoice.balance_due) FILTER (WHERE invoice.status IN ('ISSUED', 'PARTIALLY_PAID') AND invoice.due_date < CURRENT_DATE), 0)::float8,
		       COALESCE(SUM(invoice.tax_amount) FILTER (WHERE invoice.status IN ('ISSUED', 'PARTIALLY_PAID', 'PAID')), 0)::float8,
		       COALESCE((SELECT SUM(deposit.balance_amount) FROM security_deposits deposit
		                 WHERE deposit.tenant_id = $1 AND deposit.status IN ('RECEIVED', 'PARTIALLY_SETTLED')), 0)::float8,
		       COUNT(*) FILTER (WHERE invoice.status = 'DRAFT')::int,
		       COUNT(*) FILTER (WHERE invoice.status IN ('ISSUED', 'PARTIALLY_PAID'))::int,
		       COUNT(*) FILTER (WHERE invoice.status IN ('ISSUED', 'PARTIALLY_PAID') AND invoice.due_date < CURRENT_DATE)::int,
		       COUNT(*) FILTER (WHERE invoice.status = 'PAID')::int
		FROM tenants tenant
		LEFT JOIN invoices invoice ON invoice.tenant_id = tenant.id
		WHERE tenant.id = $1
		GROUP BY tenant.id
	`, tenantID).Scan(
		&result.Currency,
		&result.Metrics.IssuedTotal,
		&result.Metrics.CollectedTotal,
		&result.Metrics.OutstandingTotal,
		&result.Metrics.OverdueTotal,
		&result.Metrics.TaxOutputTotal,
		&result.Metrics.DepositsHeldTotal,
		&result.Metrics.DraftCount,
		&result.Metrics.OpenInvoiceCount,
		&result.Metrics.OverdueCount,
		&result.Metrics.PaidCount,
	); err != nil {
		return BillingDashboard{}, fmt.Errorf("load billing dashboard metrics: %w", err)
	}
	result.RecentInvoices, err = r.ListInvoices(ctx, tenantID, "", "", 6)
	if err != nil {
		return BillingDashboard{}, err
	}
	result.RecentPayments, err = r.ListPayments(ctx, tenantID, "", "", 6)
	if err != nil {
		return BillingDashboard{}, err
	}
	result.MonthlyBilling, err = r.monthlyAmounts(ctx, tenantID, `
		SELECT TO_CHAR(months.month, 'YYYY-MM'),
		       COALESCE(SUM(invoice.total_amount), 0)::float8
		FROM generate_series(
			date_trunc('month', CURRENT_DATE) - INTERVAL '5 months',
			date_trunc('month', CURRENT_DATE),
			INTERVAL '1 month'
		) months(month)
		LEFT JOIN invoices invoice
		  ON invoice.tenant_id = $1
		 AND invoice.status IN ('ISSUED', 'PARTIALLY_PAID', 'PAID')
		 AND date_trunc('month', invoice.issue_date::timestamp) = months.month
		GROUP BY months.month
		ORDER BY months.month
	`)
	if err != nil {
		return BillingDashboard{}, err
	}
	result.MonthlyPayments, err = r.monthlyAmounts(ctx, tenantID, `
		SELECT TO_CHAR(months.month, 'YYYY-MM'),
		       COALESCE(SUM(payment.amount), 0)::float8
		FROM generate_series(
			date_trunc('month', CURRENT_DATE) - INTERVAL '5 months',
			date_trunc('month', CURRENT_DATE),
			INTERVAL '1 month'
		) months(month)
		LEFT JOIN payments payment
		  ON payment.tenant_id = $1
		 AND payment.status = 'CONFIRMED'
		 AND date_trunc('month', payment.received_at) = months.month
		GROUP BY months.month
		ORDER BY months.month
	`)
	if err != nil {
		return BillingDashboard{}, err
	}
	return result, nil
}

func (r *Repository) monthlyAmounts(ctx context.Context, tenantID, query string) ([]MonthlyAmount, error) {
	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("load monthly billing amounts: %w", err)
	}
	defer rows.Close()
	items := make([]MonthlyAmount, 0, 6)
	for rows.Next() {
		var item MonthlyAmount
		if err := rows.Scan(&item.Month, &item.Amount); err != nil {
			return nil, fmt.Errorf("scan monthly billing amount: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func nullableUUID(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
