package dte

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

func (r *Repository) ensureSettings(ctx context.Context, tenantID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO dte_settings (tenant_id)
		VALUES ($1)
		ON CONFLICT (tenant_id) DO NOTHING
	`, tenantID)
	if err != nil {
		return fmt.Errorf("ensure dte settings: %w", err)
	}
	return nil
}

func (r *Repository) GetSettings(ctx context.Context, tenantID string) (Settings, error) {
	if err := r.ensureSettings(ctx, tenantID); err != nil {
		return Settings{}, err
	}
	return scanSettings(r.pool.QueryRow(ctx, settingsSelect+` WHERE tenant_id = $1`, tenantID))
}

func (r *Repository) UpdateSettings(ctx context.Context, tenantID string, input SettingsInput) (Settings, error) {
	if err := r.ensureSettings(ctx, tenantID); err != nil {
		return Settings{}, err
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE dte_settings SET
			enabled = $2,
			provider_mode = $3,
			environment = $4,
			default_document_type = $5,
			schema_version = $6,
			establishment_type = $7,
			establishment_code = $8,
			point_of_sale_code = $9,
			auth_url = $10,
			signer_url = $11,
			reception_url = $12,
			invalidation_url = $13,
			query_url = $14,
			user_secret_ref = $15,
			password_secret_ref = $16,
			signing_password_secret_ref = $17,
			auto_submit_on_issue = $18,
			max_attempts = $19,
			retry_base_seconds = $20
		WHERE tenant_id = $1
	`, tenantID, input.Enabled, input.ProviderMode, input.Environment,
		input.DefaultDocumentType, input.SchemaVersion, input.EstablishmentType,
		input.EstablishmentCode, input.PointOfSaleCode, input.AuthURL, input.SignerURL,
		input.ReceptionURL, input.InvalidationURL, input.QueryURL, input.UserSecretRef,
		input.PasswordSecretRef, input.SigningPasswordSecretRef, input.AutoSubmitOnIssue,
		input.MaxAttempts, input.RetryBaseSeconds)
	if err != nil {
		return Settings{}, fmt.Errorf("update dte settings: %w", err)
	}
	return r.GetSettings(ctx, tenantID)
}

const settingsSelect = `
	SELECT tenant_id, enabled, provider_mode, environment, default_document_type,
	       schema_version, establishment_type, establishment_code, point_of_sale_code,
	       auth_url, signer_url, reception_url, invalidation_url, query_url,
	       user_secret_ref, password_secret_ref, signing_password_secret_ref,
	       auto_submit_on_issue, max_attempts, retry_base_seconds, next_control_number,
	       created_at, updated_at
	FROM dte_settings`

func scanSettings(row interface{ Scan(...any) error }) (Settings, error) {
	var item Settings
	if err := row.Scan(
		&item.TenantID, &item.Enabled, &item.ProviderMode, &item.Environment,
		&item.DefaultDocumentType, &item.SchemaVersion, &item.EstablishmentType,
		&item.EstablishmentCode, &item.PointOfSaleCode, &item.AuthURL, &item.SignerURL,
		&item.ReceptionURL, &item.InvalidationURL, &item.QueryURL, &item.UserSecretRef,
		&item.PasswordSecretRef, &item.SigningPasswordSecretRef, &item.AutoSubmitOnIssue,
		&item.MaxAttempts, &item.RetryBaseSeconds, &item.NextControlNumber,
		&item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return Settings{}, fmt.Errorf("scan dte settings: %w", err)
	}
	return item, nil
}

func (r *Repository) LoadInvoiceSnapshot(ctx context.Context, tenantID, invoiceID string) (InvoiceSnapshot, error) {
	var item InvoiceSnapshot
	err := r.pool.QueryRow(ctx, `
		SELECT invoice.id, COALESCE(invoice.invoice_number, 0), invoice.invoice_prefix,
		       invoice.status, invoice.fiscal_status, invoice.issue_date::text,
		       invoice.due_date::text, invoice.currency, invoice.prices_include_tax,
		       invoice.customer_id, invoice.customer_name, invoice.customer_tax_id,
		       invoice.customer_registration_number, invoice.customer_document_type_code,
		       invoice.customer_trade_name, invoice.customer_economic_activity,
		       invoice.customer_economic_activity_code, invoice.customer_email,
		       invoice.customer_phone, invoice.customer_address,
		       invoice.customer_department_code, invoice.customer_municipality_code,
		       invoice.customer_district_code,
		       invoice.seller_legal_name, invoice.seller_trade_name, invoice.seller_tax_id,
		       invoice.seller_registration_number, invoice.seller_economic_activity,
		       invoice.seller_economic_activity_code, invoice.seller_email,
		       invoice.seller_phone, invoice.seller_address,
		       invoice.seller_department_code, invoice.seller_municipality_code,
		       invoice.seller_district_code,
		       invoice.taxable_amount::float8, invoice.exempt_amount::float8,
		       invoice.non_taxable_amount::float8, invoice.tax_amount::float8,
		       invoice.total_amount::float8, invoice.notes, invoice.terms
		FROM invoices invoice
		WHERE invoice.tenant_id = $1 AND invoice.id = $2
	`, tenantID, invoiceID).Scan(
		&item.ID, &item.InvoiceNumber, &item.InvoicePrefix, &item.Status,
		&item.FiscalStatus, &item.IssueDate, &item.DueDate, &item.Currency,
		&item.PricesIncludeTax, &item.CustomerID, &item.CustomerName,
		&item.CustomerTaxID, &item.CustomerRegistrationNumber,
		&item.CustomerDocumentType, &item.CustomerTradeName,
		&item.CustomerEconomicActivity, &item.CustomerEconomicCode,
		&item.CustomerEmail, &item.CustomerPhone, &item.CustomerAddress,
		&item.CustomerDepartmentCode, &item.CustomerMunicipalityCode,
		&item.CustomerDistrictCode, &item.SellerLegalName, &item.SellerTradeName,
		&item.SellerTaxID, &item.SellerRegistrationNumber,
		&item.SellerEconomicActivity, &item.SellerEconomicCode,
		&item.SellerEmail, &item.SellerPhone, &item.SellerAddress,
		&item.SellerDepartmentCode, &item.SellerMunicipalityCode,
		&item.SellerDistrictCode, &item.TaxableAmount, &item.ExemptAmount,
		&item.NonTaxableAmount, &item.TaxAmount, &item.TotalAmount,
		&item.Notes, &item.Terms,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return InvoiceSnapshot{}, ErrInvoiceNotFound
	}
	if err != nil {
		return InvoiceSnapshot{}, fmt.Errorf("load DTE invoice snapshot: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, description, quantity::float8, unit_price::float8,
		       discount_amount::float8, net_amount::float8, tax_category,
		       tax_rate::float8, tax_amount::float8, line_total::float8,
		       dte_item_type, dte_unit_code, dte_product_code
		FROM invoice_items
		WHERE tenant_id = $1 AND invoice_id = $2
		ORDER BY sort_order, id
	`, tenantID, invoiceID)
	if err != nil {
		return InvoiceSnapshot{}, fmt.Errorf("load DTE invoice items: %w", err)
	}
	defer rows.Close()
	item.Items = make([]InvoiceItemSnapshot, 0)
	for rows.Next() {
		var line InvoiceItemSnapshot
		if err := rows.Scan(&line.ID, &line.Description, &line.Quantity, &line.UnitPrice,
			&line.DiscountAmount, &line.NetAmount, &line.TaxCategory, &line.TaxRate,
			&line.TaxAmount, &line.LineTotal, &line.DTEItemType, &line.DTEUnitCode,
			&line.DTEProductCode); err != nil {
			return InvoiceSnapshot{}, fmt.Errorf("scan DTE invoice item: %w", err)
		}
		item.Items = append(item.Items, line)
	}
	if err := rows.Err(); err != nil {
		return InvoiceSnapshot{}, err
	}
	return item, nil
}

type payloadBuilder func(controlNumber string, settings Settings) (map[string]any, error)

func (r *Repository) CreateDocument(
	ctx context.Context,
	tenantID, invoiceID, documentType, generationCode, actorID string,
	settings Settings,
	build payloadBuilder,
) (DocumentDetail, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return DocumentDetail{}, fmt.Errorf("begin prepare DTE: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var invoiceStatus, fiscalStatus string
	if err := tx.QueryRow(ctx, `
		SELECT status, fiscal_status
		FROM invoices
		WHERE tenant_id = $1 AND id = $2
		FOR UPDATE
	`, tenantID, invoiceID).Scan(&invoiceStatus, &fiscalStatus); errors.Is(err, pgx.ErrNoRows) {
		return DocumentDetail{}, ErrInvoiceNotFound
	} else if err != nil {
		return DocumentDetail{}, fmt.Errorf("lock invoice for DTE preparation: %w", err)
	}
	if invoiceStatus == "DRAFT" || invoiceStatus == "VOID" {
		return DocumentDetail{}, ErrInvoiceState
	}
	if fiscalStatus != "READY_FOR_DTE" && fiscalStatus != "REJECTED" {
		return DocumentDetail{}, ErrInvoiceFiscalState
	}

	lockedSettings, err := scanSettings(tx.QueryRow(ctx, settingsSelect+`
		WHERE tenant_id = $1
		FOR UPDATE
	`, tenantID))
	if err != nil {
		return DocumentDetail{}, fmt.Errorf("lock DTE settings and sequence: %w", err)
	}
	lockedSettings = enrichSettings(lockedSettings)
	if !lockedSettings.Enabled {
		return DocumentDetail{}, ErrSettingsDisabled
	}
	if !lockedSettings.ConfigurationReady {
		return DocumentDetail{}, ErrSettingsIncomplete
	}
	if !samePreparationSettings(settings, lockedSettings) {
		return DocumentDetail{}, ErrSettingsChanged
	}
	sequence := lockedSettings.NextControlNumber
	control := controlNumber(documentType, lockedSettings.EstablishmentCode, lockedSettings.PointOfSaleCode, sequence)
	payload, err := build(control, lockedSettings)
	if err != nil {
		return DocumentDetail{}, err
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return DocumentDetail{}, fmt.Errorf("encode DTE payload: %w", err)
	}
	idempotency := hashString(tenantID + ":" + invoiceID + ":" + generationCode)

	var documentID string
	err = tx.QueryRow(ctx, `
		INSERT INTO dte_documents (
			tenant_id, invoice_id, document_type, schema_version,
			provider_mode, environment, status, generation_code,
			control_number, idempotency_key, payload, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, 'READY_TO_SIGN', $7,
			$8, $9, $10::jsonb, $11, $11
		)
		RETURNING id
	`, tenantID, invoiceID, documentType, lockedSettings.SchemaVersion,
		lockedSettings.ProviderMode, lockedSettings.Environment, generationCode, control,
		idempotency, string(encoded), nullableUUID(actorID)).Scan(&documentID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return DocumentDetail{}, ErrDocumentConflict
		}
		return DocumentDetail{}, fmt.Errorf("insert DTE document: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE dte_settings
		SET next_control_number = next_control_number + 1
		WHERE tenant_id = $1
	`, tenantID); err != nil {
		return DocumentDetail{}, fmt.Errorf("advance DTE control number: %w", err)
	}
	if err := insertEvent(ctx, tx, tenantID, documentID, "DTE_PREPARED", actorID, map[string]any{
		"document_type":   documentType,
		"generation_code": generationCode,
		"control_number":  control,
		"provider":        lockedSettings.ProviderMode,
		"environment":     lockedSettings.Environment,
	}); err != nil {
		return DocumentDetail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return DocumentDetail{}, fmt.Errorf("commit prepare DTE: %w", err)
	}
	return r.GetDocument(ctx, tenantID, documentID)
}

func samePreparationSettings(expected, locked Settings) bool {
	return expected.ProviderMode == locked.ProviderMode &&
		expected.Environment == locked.Environment &&
		expected.DefaultDocumentType == locked.DefaultDocumentType &&
		expected.SchemaVersion == locked.SchemaVersion &&
		expected.EstablishmentType == locked.EstablishmentType &&
		expected.EstablishmentCode == locked.EstablishmentCode &&
		expected.PointOfSaleCode == locked.PointOfSaleCode
}

func (r *Repository) ListDocuments(ctx context.Context, tenantID, search, status string, limit int) ([]DocumentSummary, error) {
	if limit <= 0 || limit > 250 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, documentSummarySelect+`
		WHERE document.tenant_id = $1
		  AND ($2 = '' OR document.status = $2)
		  AND ($3 = '' OR document.control_number ILIKE '%' || $3 || '%'
		       OR document.generation_code::text ILIKE '%' || $3 || '%'
		       OR invoice.customer_name ILIKE '%' || $3 || '%'
		       OR (invoice.invoice_prefix || '-' || LPAD(COALESCE(invoice.invoice_number, 0)::text, 6, '0')) ILIKE '%' || $3 || '%')
		ORDER BY document.created_at DESC
		LIMIT $4
	`, tenantID, status, search, limit)
	if err != nil {
		return nil, fmt.Errorf("list DTE documents: %w", err)
	}
	defer rows.Close()
	items := make([]DocumentSummary, 0)
	for rows.Next() {
		item, err := scanDocumentSummary(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

const documentSummarySelect = `
	SELECT document.id, document.invoice_id, invoice.invoice_number, invoice.invoice_prefix,
	       invoice.customer_name, document.document_type, document.schema_version,
	       document.provider_mode, document.environment, document.status,
	       document.generation_code::text, document.control_number,
	       document.receipt_seal, document.provider_status,
	       document.error_code, document.error_message, document.attempt_count,
	       document.next_attempt_at, document.submitted_at, document.accepted_at,
	       document.rejected_at, document.invalidated_at,
	       document.created_at, document.updated_at
	FROM dte_documents document
	JOIN invoices invoice ON invoice.tenant_id = document.tenant_id AND invoice.id = document.invoice_id`

func scanDocumentSummary(row interface{ Scan(...any) error }) (DocumentSummary, error) {
	var item DocumentSummary
	if err := row.Scan(
		&item.ID, &item.InvoiceID, &item.InvoiceNumber, &item.InvoicePrefix,
		&item.CustomerName, &item.DocumentType, &item.SchemaVersion,
		&item.ProviderMode, &item.Environment, &item.Status,
		&item.GenerationCode, &item.ControlNumber, &item.ReceiptSeal,
		&item.ProviderStatus, &item.ErrorCode, &item.ErrorMessage,
		&item.AttemptCount, &item.NextAttemptAt, &item.SubmittedAt,
		&item.AcceptedAt, &item.RejectedAt, &item.InvalidatedAt,
		&item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return DocumentSummary{}, fmt.Errorf("scan DTE document: %w", err)
	}
	item.InvoiceDisplayNumber = invoiceDisplayNumber(item.InvoicePrefix, item.InvoiceNumber)
	item.DocumentTypeLabel = documentTypeLabel(item.DocumentType)
	return item, nil
}

func (r *Repository) GetDocument(ctx context.Context, tenantID, documentID string) (DocumentDetail, error) {
	row := r.pool.QueryRow(ctx, documentSummarySelect+`
		WHERE document.tenant_id = $1 AND document.id = $2
	`, tenantID, documentID)
	var item DocumentDetail
	var summary DocumentSummary
	var payload, request, response, invalidationRequest, invalidationResponse []byte
	err := row.Scan(
		&summary.ID, &summary.InvoiceID, &summary.InvoiceNumber, &summary.InvoicePrefix,
		&summary.CustomerName, &summary.DocumentType, &summary.SchemaVersion,
		&summary.ProviderMode, &summary.Environment, &summary.Status,
		&summary.GenerationCode, &summary.ControlNumber, &summary.ReceiptSeal,
		&summary.ProviderStatus, &summary.ErrorCode, &summary.ErrorMessage,
		&summary.AttemptCount, &summary.NextAttemptAt, &summary.SubmittedAt,
		&summary.AcceptedAt, &summary.RejectedAt, &summary.InvalidatedAt,
		&summary.CreatedAt, &summary.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return DocumentDetail{}, ErrDocumentNotFound
	}
	if err != nil {
		return DocumentDetail{}, fmt.Errorf("get DTE document: %w", err)
	}
	summary.InvoiceDisplayNumber = invoiceDisplayNumber(summary.InvoicePrefix, summary.InvoiceNumber)
	summary.DocumentTypeLabel = documentTypeLabel(summary.DocumentType)
	item.DocumentSummary = summary

	err = r.pool.QueryRow(ctx, `
		SELECT idempotency_key, payload, signed_document,
		       provider_request, provider_response, invalidation_request, invalidation_response,
		       invalidation_reason
		FROM dte_documents
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, documentID).Scan(&item.IDempotencyKey, &payload, &item.SignedDocument,
		&request, &response, &invalidationRequest, &invalidationResponse, &item.InvalidationReason)
	if err != nil {
		return DocumentDetail{}, fmt.Errorf("load DTE document payload: %w", err)
	}
	item.Payload = decodeJSONMap(payload)
	item.ProviderRequest = decodeJSONMap(request)
	item.ProviderResponse = decodeJSONMap(response)
	item.InvalidationRequest = decodeJSONMap(invalidationRequest)
	item.InvalidationResponse = decodeJSONMap(invalidationResponse)

	rows, err := r.pool.Query(ctx, `
		SELECT id, event_type, actor_id, metadata, created_at
		FROM dte_events
		WHERE tenant_id = $1 AND dte_document_id = $2
		ORDER BY created_at DESC, id DESC
	`, tenantID, documentID)
	if err != nil {
		return DocumentDetail{}, fmt.Errorf("list DTE events: %w", err)
	}
	defer rows.Close()
	item.Events = make([]Event, 0)
	for rows.Next() {
		var event Event
		var metadata []byte
		if err := rows.Scan(&event.ID, &event.EventType, &event.ActorID, &metadata, &event.CreatedAt); err != nil {
			return DocumentDetail{}, fmt.Errorf("scan DTE event: %w", err)
		}
		event.Metadata = decodeJSONMap(metadata)
		item.Events = append(item.Events, event)
	}
	return item, rows.Err()
}

func (r *Repository) GetDocumentByInvoice(ctx context.Context, tenantID, invoiceID string) (DocumentDetail, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		SELECT id FROM dte_documents
		WHERE tenant_id = $1 AND invoice_id = $2
		ORDER BY created_at DESC
		LIMIT 1
	`, tenantID, invoiceID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return DocumentDetail{}, ErrDocumentNotFound
	}
	if err != nil {
		return DocumentDetail{}, err
	}
	return r.GetDocument(ctx, tenantID, id)
}

func (r *Repository) MarkSubmitting(ctx context.Context, tenantID, documentID, actorID string, maxAttempts int) (DocumentDetail, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return DocumentDetail{}, fmt.Errorf("begin DTE submission: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status string
	var attempts int
	var invoiceID string
	if err := tx.QueryRow(ctx, `
		SELECT status, attempt_count, invoice_id
		FROM dte_documents
		WHERE tenant_id = $1 AND id = $2
		FOR UPDATE
	`, tenantID, documentID).Scan(&status, &attempts, &invoiceID); errors.Is(err, pgx.ErrNoRows) {
		return DocumentDetail{}, ErrDocumentNotFound
	} else if err != nil {
		return DocumentDetail{}, fmt.Errorf("lock DTE document: %w", err)
	}
	if status != "READY_TO_SIGN" && status != "RETRY_REQUIRED" {
		return DocumentDetail{}, ErrDocumentState
	}
	if attempts >= maxAttempts {
		return DocumentDetail{}, ErrDocumentState
	}
	_, err = tx.Exec(ctx, `
		UPDATE dte_documents SET
			status = 'SUBMITTING',
			attempt_count = attempt_count + 1,
			submitted_at = NOW(),
			next_attempt_at = NULL,
			error_code = '',
			error_message = '',
			updated_by = $3
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, documentID, nullableUUID(actorID))
	if err != nil {
		return DocumentDetail{}, fmt.Errorf("mark DTE submitting: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE invoices SET fiscal_status = 'SUBMITTED' WHERE tenant_id = $1 AND id = $2`, tenantID, invoiceID); err != nil {
		return DocumentDetail{}, fmt.Errorf("mark invoice DTE submitted: %w", err)
	}
	if err := insertEvent(ctx, tx, tenantID, documentID, "DTE_SUBMISSION_STARTED", actorID, map[string]any{"attempt": attempts + 1}); err != nil {
		return DocumentDetail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return DocumentDetail{}, fmt.Errorf("commit DTE submission start: %w", err)
	}
	return r.GetDocument(ctx, tenantID, documentID)
}

func (r *Repository) ApplySubmissionResult(ctx context.Context, tenantID, documentID, actorID string, result ProviderResult, retryAt *time.Time) (DocumentDetail, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return DocumentDetail{}, fmt.Errorf("begin DTE result: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status, invoiceID string
	if err := tx.QueryRow(ctx, `
		SELECT status, invoice_id FROM dte_documents
		WHERE tenant_id = $1 AND id = $2 FOR UPDATE
	`, tenantID, documentID).Scan(&status, &invoiceID); errors.Is(err, pgx.ErrNoRows) {
		return DocumentDetail{}, ErrDocumentNotFound
	} else if err != nil {
		return DocumentDetail{}, err
	}
	if status != "SUBMITTING" {
		return DocumentDetail{}, ErrDocumentState
	}

	nextStatus := "REJECTED"
	invoiceFiscalStatus := "REJECTED"
	eventType := "DTE_REJECTED"
	if result.Accepted {
		nextStatus = "ACCEPTED"
		invoiceFiscalStatus = "ACCEPTED"
		eventType = "DTE_ACCEPTED"
	} else if result.Retryable {
		nextStatus = "RETRY_REQUIRED"
		invoiceFiscalStatus = "READY_FOR_DTE"
		eventType = "DTE_RETRY_REQUIRED"
	}
	requestJSON, err := json.Marshal(result.Request)
	if err != nil {
		return DocumentDetail{}, fmt.Errorf("encode DTE provider request: %w", err)
	}
	responseJSON, err := json.Marshal(result.Response)
	if err != nil {
		return DocumentDetail{}, fmt.Errorf("encode DTE provider response: %w", err)
	}

	// Keep the transition flags as dedicated boolean parameters. Reusing the
	// VARCHAR status parameter inside CASE expressions can make PostgreSQL infer
	// both character varying and text for the same prepared-statement parameter.
	acceptedNow := nextStatus == "ACCEPTED"
	rejectedNow := nextStatus == "REJECTED"
	_, err = tx.Exec(ctx, `
		UPDATE dte_documents SET
			status = $3,
			signed_document = $4,
			provider_request = $5::jsonb,
			provider_response = $6::jsonb,
			receipt_seal = $7,
			provider_status = $8,
			error_code = $9,
			error_message = $10,
			next_attempt_at = $11,
			accepted_at = CASE WHEN $12::boolean THEN NOW() ELSE accepted_at END,
			rejected_at = CASE WHEN $13::boolean THEN NOW() ELSE rejected_at END,
			updated_by = $14
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, documentID, nextStatus, result.SignedDocument, string(requestJSON),
		string(responseJSON), result.ReceiptSeal, result.ProviderStatus, result.ErrorCode,
		result.ErrorMessage, retryAt, acceptedNow, rejectedNow, nullableUUID(actorID))
	if err != nil {
		return DocumentDetail{}, fmt.Errorf("apply DTE provider result: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE invoices SET fiscal_status = $3 WHERE tenant_id = $1 AND id = $2`, tenantID, invoiceID, invoiceFiscalStatus); err != nil {
		return DocumentDetail{}, fmt.Errorf("update invoice fiscal status: %w", err)
	}
	if err := insertEvent(ctx, tx, tenantID, documentID, eventType, actorID, map[string]any{
		"provider_status": result.ProviderStatus,
		"receipt_seal":    result.ReceiptSeal,
		"error_code":      result.ErrorCode,
		"error_message":   result.ErrorMessage,
		"next_attempt_at": retryAt,
	}); err != nil {
		return DocumentDetail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return DocumentDetail{}, fmt.Errorf("commit DTE provider result: %w", err)
	}
	return r.GetDocument(ctx, tenantID, documentID)
}

func (r *Repository) CancelDocument(ctx context.Context, tenantID, documentID, actorID string) (DocumentDetail, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return DocumentDetail{}, fmt.Errorf("begin cancel DTE: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var status, invoiceID string
	if err := tx.QueryRow(ctx, `
		SELECT status, invoice_id
		FROM dte_documents
		WHERE tenant_id = $1 AND id = $2
		FOR UPDATE
	`, tenantID, documentID).Scan(&status, &invoiceID); errors.Is(err, pgx.ErrNoRows) {
		return DocumentDetail{}, ErrDocumentNotFound
	} else if err != nil {
		return DocumentDetail{}, fmt.Errorf("lock DTE for cancellation: %w", err)
	}
	if status != "READY_TO_SIGN" && status != "RETRY_REQUIRED" {
		return DocumentDetail{}, ErrDocumentState
	}
	if _, err := tx.Exec(ctx, `
		UPDATE dte_documents
		SET status = 'CANCELLED', next_attempt_at = NULL,
		    error_code = '', error_message = '', updated_by = $3
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, documentID, nullableUUID(actorID)); err != nil {
		return DocumentDetail{}, fmt.Errorf("cancel DTE: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE invoices
		SET fiscal_status = 'READY_FOR_DTE'
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, invoiceID); err != nil {
		return DocumentDetail{}, fmt.Errorf("restore invoice fiscal readiness: %w", err)
	}
	if err := insertEvent(ctx, tx, tenantID, documentID, "DTE_CANCELLED", actorID, map[string]any{
		"previous_status": status,
	}); err != nil {
		return DocumentDetail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return DocumentDetail{}, fmt.Errorf("commit cancel DTE: %w", err)
	}
	return r.GetDocument(ctx, tenantID, documentID)
}

func (r *Repository) BeginInvalidation(ctx context.Context, tenantID, documentID, actorID, reason string) (DocumentDetail, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return DocumentDetail{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status string
	if err := tx.QueryRow(ctx, `
		SELECT status FROM dte_documents
		WHERE tenant_id = $1 AND id = $2 FOR UPDATE
	`, tenantID, documentID).Scan(&status); errors.Is(err, pgx.ErrNoRows) {
		return DocumentDetail{}, ErrDocumentNotFound
	} else if err != nil {
		return DocumentDetail{}, err
	}
	if status != "ACCEPTED" {
		return DocumentDetail{}, ErrDocumentState
	}
	_, err = tx.Exec(ctx, `
		UPDATE dte_documents SET status = 'INVALIDATION_PENDING',
		       invalidation_reason = $3, updated_by = $4
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, documentID, reason, nullableUUID(actorID))
	if err != nil {
		return DocumentDetail{}, err
	}
	if err := insertEvent(ctx, tx, tenantID, documentID, "DTE_INVALIDATION_STARTED", actorID, map[string]any{"reason": reason}); err != nil {
		return DocumentDetail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return DocumentDetail{}, err
	}
	return r.GetDocument(ctx, tenantID, documentID)
}

func (r *Repository) ApplyInvalidationResult(ctx context.Context, tenantID, documentID, actorID string, result InvalidationResult) (DocumentDetail, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return DocumentDetail{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status, invoiceID string
	if err := tx.QueryRow(ctx, `
		SELECT status, invoice_id FROM dte_documents
		WHERE tenant_id = $1 AND id = $2 FOR UPDATE
	`, tenantID, documentID).Scan(&status, &invoiceID); errors.Is(err, pgx.ErrNoRows) {
		return DocumentDetail{}, ErrDocumentNotFound
	} else if err != nil {
		return DocumentDetail{}, err
	}
	if status != "INVALIDATION_PENDING" {
		return DocumentDetail{}, ErrDocumentState
	}
	requestJSON, err := json.Marshal(result.Request)
	if err != nil {
		return DocumentDetail{}, fmt.Errorf("encode DTE invalidation request: %w", err)
	}
	responseJSON, err := json.Marshal(result.Response)
	if err != nil {
		return DocumentDetail{}, fmt.Errorf("encode DTE invalidation response: %w", err)
	}
	nextStatus := "ACCEPTED"
	eventType := "DTE_INVALIDATION_FAILED"
	if result.Accepted {
		nextStatus = "INVALIDATED"
		eventType = "DTE_INVALIDATED"
	}
	_, err = tx.Exec(ctx, `
		UPDATE dte_documents SET
			status = $3,
			invalidation_request = $4::jsonb,
			invalidation_response = $5::jsonb,
			provider_status = $6,
			error_code = $7,
			error_message = $8,
			invalidated_at = CASE WHEN $9::boolean THEN NOW() ELSE invalidated_at END,
			updated_by = $10
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, documentID, nextStatus, string(requestJSON), string(responseJSON),
		result.ProviderStatus, result.ErrorCode, result.ErrorMessage, result.Accepted, nullableUUID(actorID))
	if err != nil {
		return DocumentDetail{}, err
	}
	if result.Accepted {
		if _, err := tx.Exec(ctx, `UPDATE invoices SET fiscal_status = 'VOIDED' WHERE tenant_id = $1 AND id = $2`, tenantID, invoiceID); err != nil {
			return DocumentDetail{}, err
		}
	}
	if err := insertEvent(ctx, tx, tenantID, documentID, eventType, actorID, map[string]any{
		"provider_status": result.ProviderStatus,
		"error_code":      result.ErrorCode,
		"error_message":   result.ErrorMessage,
	}); err != nil {
		return DocumentDetail{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return DocumentDetail{}, err
	}
	return r.GetDocument(ctx, tenantID, documentID)
}

func insertEvent(ctx context.Context, tx pgx.Tx, tenantID, documentID, eventType, actorID string, metadata map[string]any) error {
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO dte_events (tenant_id, dte_document_id, event_type, actor_id, metadata)
		VALUES ($1, $2, $3, $4, $5::jsonb)
	`, tenantID, documentID, eventType, nullableUUID(actorID), string(encoded))
	if err != nil {
		return fmt.Errorf("insert DTE event: %w", err)
	}
	return nil
}

func decodeJSONMap(value []byte) map[string]any {
	result := map[string]any{}
	if len(value) == 0 {
		return result
	}
	if err := json.Unmarshal(value, &result); err != nil {
		return map[string]any{"raw": string(value)}
	}
	return result
}

func nullableUUID(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func hashString(value string) string {
	// Kept here to avoid storing any idempotency source material.
	sum := sha256.Sum256([]byte(value))
	return fmt.Sprintf("%x", sum[:])
}
