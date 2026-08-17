package quoteportal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rentstage/rentstage/apps/api/internal/core/reservation"
)

type Repository struct {
	pool         *pgxpool.Pool
	reservations *reservation.Repository
}

func NewRepository(pool *pgxpool.Pool, reservations *reservation.Repository) *Repository {
	return &Repository{pool: pool, reservations: reservations}
}

func (r *Repository) Settings(ctx context.Context, tenantID string) (Settings, error) {
	if _, err := r.pool.Exec(ctx, `
		INSERT INTO quote_portal_settings (tenant_id)
		VALUES ($1)
		ON CONFLICT (tenant_id) DO NOTHING
	`, tenantID); err != nil {
		return Settings{}, fmt.Errorf("ensure quote portal settings: %w", err)
	}
	return loadSettings(ctx, r.pool, tenantID)
}

func (r *Repository) UpdateSettings(ctx context.Context, tenantID string, input normalizedSettings) (Settings, error) {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO quote_portal_settings (
			tenant_id, enabled, headline, introduction, accent_color,
			default_validity_days, allow_rejection, require_response_name,
			acceptance_terms_text, acceptance_terms_version
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (tenant_id) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			headline = EXCLUDED.headline,
			introduction = EXCLUDED.introduction,
			accent_color = EXCLUDED.accent_color,
			default_validity_days = EXCLUDED.default_validity_days,
			allow_rejection = EXCLUDED.allow_rejection,
			require_response_name = EXCLUDED.require_response_name,
			acceptance_terms_text = EXCLUDED.acceptance_terms_text,
			acceptance_terms_version = EXCLUDED.acceptance_terms_version
	`,
		tenantID,
		input.Enabled,
		input.Headline,
		input.Introduction,
		input.AccentColor,
		input.DefaultValidityDays,
		input.AllowRejection,
		input.RequireResponseName,
		input.AcceptanceTermsText,
		input.AcceptanceTermsVersion,
	)
	if err != nil {
		return Settings{}, fmt.Errorf("update quote portal settings: %w", err)
	}
	return loadSettings(ctx, r.pool, tenantID)
}

func (r *Repository) Issue(
	ctx context.Context,
	tenantID string,
	quoteID string,
	actorID string,
	tokenHash string,
	reissue bool,
) (IssueResult, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return IssueResult{}, fmt.Errorf("begin quote portal issue: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO quote_portal_settings (tenant_id)
		VALUES ($1)
		ON CONFLICT (tenant_id) DO NOTHING
	`, tenantID); err != nil {
		return IssueResult{}, fmt.Errorf("ensure quote portal settings: %w", err)
	}
	settings, err := loadSettingsForUpdate(ctx, tx, tenantID)
	if err != nil {
		return IssueResult{}, err
	}
	if !settings.Enabled {
		return IssueResult{}, ErrPortalDisabled
	}

	var quoteNumber int64
	var quoteStatus string
	var quoteExpiresAt *time.Time
	err = tx.QueryRow(ctx, `
		SELECT quote_number, status, expires_at
		FROM quotes
		WHERE tenant_id = $1 AND id = $2
		FOR UPDATE
	`, tenantID, quoteID).Scan(&quoteNumber, &quoteStatus, &quoteExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return IssueResult{}, ErrQuoteNotFound
	}
	if err != nil {
		return IssueResult{}, fmt.Errorf("lock quote for portal: %w", err)
	}
	if reissue {
		if quoteStatus != "SENT" {
			return IssueResult{}, ErrInvalidQuoteStatus
		}
	} else if quoteStatus != "DRAFT" {
		return IssueResult{}, ErrInvalidQuoteStatus
	}

	now := time.Now().UTC()
	expiresAt := now.Add(time.Duration(settings.DefaultValidityDays) * 24 * time.Hour)
	if quoteExpiresAt != nil && quoteExpiresAt.After(now.Add(5*time.Minute)) {
		expiresAt = quoteExpiresAt.UTC()
	}

	var existing bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM quote_portals WHERE tenant_id = $1 AND quote_id = $2
		)
	`, tenantID, quoteID).Scan(&existing); err != nil {
		return IssueResult{}, fmt.Errorf("check existing quote portal: %w", err)
	}

	if reissue {
		if _, err := tx.Exec(ctx, `
			UPDATE quotes SET expires_at = $3
			WHERE tenant_id = $1 AND id = $2
		`, tenantID, quoteID, expiresAt); err != nil {
			return IssueResult{}, fmt.Errorf("refresh quote expiration: %w", err)
		}
	} else {
		command, err := tx.Exec(ctx, `
			UPDATE quotes
			SET status = 'SENT', expires_at = $3
			WHERE tenant_id = $1 AND id = $2 AND status = 'DRAFT'
		`, tenantID, quoteID, expiresAt)
		if err != nil {
			return IssueResult{}, fmt.Errorf("mark quote sent: %w", err)
		}
		if command.RowsAffected() == 0 {
			return IssueResult{}, ErrInvalidQuoteStatus
		}
	}

	var result IssueResult
	result.QuoteID = quoteID
	result.QuoteNumber = quoteNumber
	result.ExpiresAt = expiresAt
	result.EventType = "CREATED"
	if existing {
		result.EventType = "REISSUED"
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO quote_portals (
			tenant_id, quote_id, token_hash, status, revision, expires_at,
			headline, introduction, accent_color, allow_rejection,
			require_response_name, terms_text, terms_version, created_by
		) VALUES (
			$1, $2, $3, 'ACTIVE', 1, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
		ON CONFLICT (tenant_id, quote_id) DO UPDATE SET
			token_hash = EXCLUDED.token_hash,
			status = 'ACTIVE',
			revision = quote_portals.revision + 1,
			expires_at = EXCLUDED.expires_at,
			headline = EXCLUDED.headline,
			introduction = EXCLUDED.introduction,
			accent_color = EXCLUDED.accent_color,
			allow_rejection = EXCLUDED.allow_rejection,
			require_response_name = EXCLUDED.require_response_name,
			terms_text = EXCLUDED.terms_text,
			terms_version = EXCLUDED.terms_version,
			created_by = EXCLUDED.created_by,
			last_viewed_at = NULL,
			view_count = 0,
			decision_at = NULL,
			decision_source = NULL,
			response_name = NULL,
			response_email = NULL,
			rejection_reason = NULL,
			origin_hash = NULL,
			user_agent = NULL
		RETURNING id, revision
	`,
		tenantID,
		quoteID,
		tokenHash,
		expiresAt,
		settings.Headline,
		settings.Introduction,
		settings.AccentColor,
		settings.AllowRejection,
		settings.RequireResponseName,
		settings.AcceptanceTermsText,
		settings.AcceptanceTermsVersion,
		actorID,
	).Scan(&result.PortalID, &result.Revision)
	if err != nil {
		return IssueResult{}, fmt.Errorf("issue quote portal: %w", err)
	}
	if err := insertEvent(ctx, tx, tenantID, result.PortalID, quoteID, result.EventType, "USER", actorID, "", "", map[string]any{
		"quote_number": quoteNumber,
		"revision":     result.Revision,
		"expires_at":   expiresAt,
	}); err != nil {
		return IssueResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return IssueResult{}, fmt.Errorf("commit quote portal issue: %w", err)
	}
	return result, nil
}

func (r *Repository) Revoke(ctx context.Context, tenantID, quoteID, actorID string) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin quote portal revoke: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var portalID string
	var status string
	err = tx.QueryRow(ctx, `
		SELECT id, status
		FROM quote_portals
		WHERE tenant_id = $1 AND quote_id = $2
		FOR UPDATE
	`, tenantID, quoteID).Scan(&portalID, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrPortalNotFound
	}
	if err != nil {
		return "", fmt.Errorf("lock quote portal: %w", err)
	}
	if status == "REVOKED" {
		if err := tx.Commit(ctx); err != nil {
			return "", fmt.Errorf("commit idempotent portal revoke: %w", err)
		}
		return portalID, nil
	}
	if status != "ACTIVE" {
		return "", ErrInvalidPortalStatus
	}
	if _, err := tx.Exec(ctx, `
		UPDATE quote_portals
		SET status = 'REVOKED', decision_at = NOW(), decision_source = 'ADMIN'
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, portalID); err != nil {
		return "", fmt.Errorf("revoke quote portal: %w", err)
	}
	if err := insertEvent(ctx, tx, tenantID, portalID, quoteID, "REVOKED", "USER", actorID, "", "", nil); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit quote portal revoke: %w", err)
	}
	return portalID, nil
}

func (r *Repository) TouchAndGet(ctx context.Context, tokenHash, originHash, userAgent string) (PublicView, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return PublicView{}, fmt.Errorf("begin quote portal view: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	identity, err := findPortalIdentity(ctx, tx, tokenHash)
	if err != nil {
		return PublicView{}, err
	}

	var quoteStatus string
	err = tx.QueryRow(ctx, `
		SELECT status
		FROM quotes
		WHERE tenant_id = $1 AND id = $2
		FOR UPDATE
	`, identity.TenantID, identity.QuoteID).Scan(&quoteStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return PublicView{}, ErrPortalNotFound
	}
	if err != nil {
		return PublicView{}, fmt.Errorf("lock quote for public portal view: %w", err)
	}

	var portalID, status string
	var expiresAt time.Time
	var viewCount int
	err = tx.QueryRow(ctx, `
		SELECT id, status, expires_at, view_count
		FROM quote_portals
		WHERE token_hash = $1 AND tenant_id = $2 AND quote_id = $3
		FOR UPDATE
	`, tokenHash, identity.TenantID, identity.QuoteID).Scan(&portalID, &status, &expiresAt, &viewCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return PublicView{}, ErrPortalNotFound
	}
	if err != nil {
		return PublicView{}, fmt.Errorf("lock public quote portal: %w", err)
	}
	tenantID := identity.TenantID
	quoteID := identity.QuoteID
	if status == "REVOKED" {
		return PublicView{}, ErrPortalUnavailable
	}

	now := time.Now().UTC()
	if status == "ACTIVE" && !expiresAt.After(now) {
		if quoteStatus == "SENT" {
			if _, err := tx.Exec(ctx, `
				UPDATE quotes SET status = 'EXPIRED'
				WHERE tenant_id = $1 AND id = $2 AND status = 'SENT'
			`, tenantID, quoteID); err != nil {
				return PublicView{}, fmt.Errorf("expire quote: %w", err)
			}
		}
		if _, err := tx.Exec(ctx, `
			UPDATE quote_portals
			SET status = 'EXPIRED', decision_at = COALESCE(decision_at, NOW()),
			    decision_source = COALESCE(decision_source, 'SYSTEM')
			WHERE tenant_id = $1 AND id = $2
		`, tenantID, portalID); err != nil {
			return PublicView{}, fmt.Errorf("expire quote portal: %w", err)
		}
		if err := insertEvent(ctx, tx, tenantID, portalID, quoteID, "EXPIRED", "SYSTEM", "quote-portal-expiration", originHash, userAgent, nil); err != nil {
			return PublicView{}, err
		}
	} else {
		if _, err := tx.Exec(ctx, `
			UPDATE quote_portals
			SET view_count = view_count + 1, last_viewed_at = NOW()
			WHERE tenant_id = $1 AND id = $2
		`, tenantID, portalID); err != nil {
			return PublicView{}, fmt.Errorf("record quote portal view: %w", err)
		}
		if viewCount == 0 {
			if err := insertEvent(ctx, tx, tenantID, portalID, quoteID, "VIEWED", "CUSTOMER", "public-visitor", originHash, userAgent, nil); err != nil {
				return PublicView{}, err
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return PublicView{}, fmt.Errorf("commit quote portal view: %w", err)
	}
	return r.publicView(ctx, tokenHash)
}

func (r *Repository) Accept(
	ctx context.Context,
	tokenHash string,
	decision normalizedDecision,
	originHash string,
	userAgent string,
) (DecisionResult, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return DecisionResult{}, fmt.Errorf("begin quote portal acceptance: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	locked, err := lockDecisionPortal(ctx, tx, tokenHash)
	if err != nil {
		return DecisionResult{}, err
	}
	if locked.Status == "ACCEPTED" {
		result := DecisionResult{
			Status:            "ACCEPTED",
			QuoteNumber:       locked.QuoteNumber,
			ReservationNumber: locked.ReservationNumber,
			DecisionAt:        valueOrNow(locked.DecisionAt),
			Idempotent:        true,
			TenantID:          locked.TenantID,
			PortalID:          locked.PortalID,
			QuoteID:           locked.QuoteID,
		}
		if err := tx.Commit(ctx); err != nil {
			return DecisionResult{}, fmt.Errorf("commit idempotent acceptance: %w", err)
		}
		return result, nil
	}
	if locked.Status == "REVOKED" {
		return DecisionResult{}, ErrPortalUnavailable
	}
	if locked.Status == "EXPIRED" || !locked.ExpiresAt.After(time.Now().UTC()) {
		if err := expireLockedPortal(ctx, tx, locked, originHash, userAgent); err != nil {
			return DecisionResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return DecisionResult{}, fmt.Errorf("commit quote portal expiration: %w", err)
		}
		return DecisionResult{}, ErrPortalExpired
	}
	if locked.Status != "ACTIVE" {
		return DecisionResult{}, ErrInvalidPortalStatus
	}
	if locked.QuoteStatus != "SENT" {
		return DecisionResult{}, ErrInvalidQuoteStatus
	}
	if locked.RequireResponseName && decision.ResponseName == "" {
		return DecisionResult{}, ErrResponseNameRequired
	}

	actorID := "quote-portal:" + locked.PortalID
	conversion, err := r.reservations.CreateFromQuoteTx(
		ctx, tx, locked.TenantID, locked.QuoteID, actorID, "SENT",
	)
	var conflict *reservation.AvailabilityConflictError
	if errors.As(err, &conflict) {
		publicConflict := publicAvailability(conflict)
		if eventErr := insertEvent(ctx, tx, locked.TenantID, locked.PortalID, locked.QuoteID, "ACCEPTANCE_BLOCKED", "CUSTOMER", "public-customer", originHash, userAgent, map[string]any{
			"availability": publicConflict,
		}); eventErr != nil {
			return DecisionResult{}, eventErr
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			return DecisionResult{}, fmt.Errorf("commit blocked acceptance: %w", commitErr)
		}
		return DecisionResult{}, conflict
	}
	if errors.Is(err, reservation.ErrQuoteStatus) {
		return DecisionResult{}, ErrInvalidQuoteStatus
	}
	if err != nil {
		return DecisionResult{}, err
	}

	now := time.Now().UTC()
	command, err := tx.Exec(ctx, `
		UPDATE quotes SET status = 'ACCEPTED'
		WHERE tenant_id = $1 AND id = $2 AND status = 'SENT'
	`, locked.TenantID, locked.QuoteID)
	if err != nil {
		return DecisionResult{}, fmt.Errorf("accept quote: %w", err)
	}
	if command.RowsAffected() == 0 {
		return DecisionResult{}, ErrInvalidQuoteStatus
	}
	if _, err := tx.Exec(ctx, `
		UPDATE quote_portals
		SET status = 'ACCEPTED', decision_at = $3, decision_source = 'CUSTOMER',
		    response_name = NULLIF($4, ''), response_email = $5,
		    rejection_reason = NULL, origin_hash = NULLIF($6, ''),
		    user_agent = NULLIF($7, '')
		WHERE tenant_id = $1 AND id = $2
	`, locked.TenantID, locked.PortalID, now, decision.ResponseName, decision.ResponseEmail, originHash, userAgent); err != nil {
		return DecisionResult{}, fmt.Errorf("record quote acceptance evidence: %w", err)
	}
	if err := insertEvent(ctx, tx, locked.TenantID, locked.PortalID, locked.QuoteID, "ACCEPTED", "CUSTOMER", decision.ResponseName, originHash, userAgent, map[string]any{
		"quote_number":       locked.QuoteNumber,
		"reservation_number": conversion.ReservationNumber,
		"terms_version":      locked.TermsVersion,
	}); err != nil {
		return DecisionResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return DecisionResult{}, fmt.Errorf("commit quote portal acceptance: %w", err)
	}
	reservationNumber := conversion.ReservationNumber
	return DecisionResult{
		Status:            "ACCEPTED",
		QuoteNumber:       locked.QuoteNumber,
		ReservationNumber: &reservationNumber,
		DecisionAt:        now,
		TenantID:          locked.TenantID,
		PortalID:          locked.PortalID,
		QuoteID:           locked.QuoteID,
		ReservationID:     conversion.ReservationID,
	}, nil
}

func (r *Repository) Reject(
	ctx context.Context,
	tokenHash string,
	decision normalizedDecision,
	originHash string,
	userAgent string,
) (DecisionResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return DecisionResult{}, fmt.Errorf("begin quote portal rejection: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	locked, err := lockDecisionPortal(ctx, tx, tokenHash)
	if err != nil {
		return DecisionResult{}, err
	}
	if locked.Status == "REJECTED" {
		result := DecisionResult{
			Status:      "REJECTED",
			QuoteNumber: locked.QuoteNumber,
			DecisionAt:  valueOrNow(locked.DecisionAt),
			Idempotent:  true,
			TenantID:    locked.TenantID,
			PortalID:    locked.PortalID,
			QuoteID:     locked.QuoteID,
		}
		if err := tx.Commit(ctx); err != nil {
			return DecisionResult{}, fmt.Errorf("commit idempotent rejection: %w", err)
		}
		return result, nil
	}
	if locked.Status == "REVOKED" {
		return DecisionResult{}, ErrPortalUnavailable
	}
	if locked.Status == "EXPIRED" || !locked.ExpiresAt.After(time.Now().UTC()) {
		if err := expireLockedPortal(ctx, tx, locked, originHash, userAgent); err != nil {
			return DecisionResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return DecisionResult{}, fmt.Errorf("commit quote portal expiration: %w", err)
		}
		return DecisionResult{}, ErrPortalExpired
	}
	if locked.Status != "ACTIVE" {
		return DecisionResult{}, ErrInvalidPortalStatus
	}
	if !locked.AllowRejection {
		return DecisionResult{}, ErrRejectionDisabled
	}
	if locked.QuoteStatus != "SENT" {
		return DecisionResult{}, ErrInvalidQuoteStatus
	}
	if locked.RequireResponseName && decision.ResponseName == "" {
		return DecisionResult{}, ErrResponseNameRequired
	}

	now := time.Now().UTC()
	command, err := tx.Exec(ctx, `
		UPDATE quotes SET status = 'REJECTED'
		WHERE tenant_id = $1 AND id = $2 AND status = 'SENT'
	`, locked.TenantID, locked.QuoteID)
	if err != nil {
		return DecisionResult{}, fmt.Errorf("reject quote: %w", err)
	}
	if command.RowsAffected() == 0 {
		return DecisionResult{}, ErrInvalidQuoteStatus
	}
	if _, err := tx.Exec(ctx, `
		UPDATE quote_portals
		SET status = 'REJECTED', decision_at = $3, decision_source = 'CUSTOMER',
		    response_name = NULLIF($4, ''), response_email = $5,
		    rejection_reason = $6, origin_hash = NULLIF($7, ''),
		    user_agent = NULLIF($8, '')
		WHERE tenant_id = $1 AND id = $2
	`, locked.TenantID, locked.PortalID, now, decision.ResponseName, decision.ResponseEmail, decision.RejectionReason, originHash, userAgent); err != nil {
		return DecisionResult{}, fmt.Errorf("record quote rejection evidence: %w", err)
	}
	if err := insertEvent(ctx, tx, locked.TenantID, locked.PortalID, locked.QuoteID, "REJECTED", "CUSTOMER", decision.ResponseName, originHash, userAgent, map[string]any{
		"quote_number": locked.QuoteNumber,
	}); err != nil {
		return DecisionResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return DecisionResult{}, fmt.Errorf("commit quote portal rejection: %w", err)
	}
	return DecisionResult{
		Status:      "REJECTED",
		QuoteNumber: locked.QuoteNumber,
		DecisionAt:  now,
		TenantID:    locked.TenantID,
		PortalID:    locked.PortalID,
		QuoteID:     locked.QuoteID,
	}, nil
}

func (r *Repository) publicView(ctx context.Context, tokenHash string) (PublicView, error) {
	var view PublicView
	var portalStatus, quoteStatus string
	var portalEnabled bool
	err := r.pool.QueryRow(ctx, `
		SELECT
			p.id, p.tenant_id, p.quote_id, p.status, p.headline,
			p.introduction, p.accent_color, p.allow_rejection,
			p.require_response_name, p.terms_text, p.terms_version,
			p.expires_at, p.decision_at, p.decision_source,
			p.response_name, p.rejection_reason,
			t.name, t.slug, t.logo_url, t.email, t.phone, t.address,
			t.currency, t.timezone,
			settings.enabled,
			q.quote_number, q.status,
			TRIM(c.first_name || ' ' || c.last_name),
			q.start_at, q.end_at, q.event_type, q.event_location,
			q.subtotal::float8, q.discount_amount::float8,
			q.extra_charges::float8, q.total::float8, q.created_at,
			reservation.reservation_number
		FROM quote_portals p
		JOIN quotes q ON q.tenant_id = p.tenant_id AND q.id = p.quote_id
		JOIN tenants t ON t.id = p.tenant_id
		JOIN quote_portal_settings settings ON settings.tenant_id = p.tenant_id
		JOIN customers c ON c.tenant_id = q.tenant_id AND c.id = q.customer_id
		LEFT JOIN reservations reservation
		  ON reservation.tenant_id = q.tenant_id AND reservation.quote_id = q.id
		WHERE p.token_hash = $1
	`, tokenHash).Scan(
		&view.PortalID,
		&view.TenantID,
		&view.QuoteID,
		&portalStatus,
		&view.Portal.Headline,
		&view.Portal.Introduction,
		&view.Portal.AccentColor,
		&view.Portal.AllowRejection,
		&view.Portal.RequireResponseName,
		&view.Portal.TermsText,
		&view.Portal.TermsVersion,
		&view.Portal.ExpiresAt,
		&view.Portal.DecisionAt,
		&view.Portal.DecisionSource,
		&view.Portal.ResponseName,
		&view.Portal.RejectionReason,
		&view.Tenant.Name,
		&view.Tenant.Slug,
		&view.Tenant.LogoURL,
		&view.Tenant.Email,
		&view.Tenant.Phone,
		&view.Tenant.Address,
		&view.Tenant.Currency,
		&view.Tenant.Timezone,
		&portalEnabled,
		&view.Quote.QuoteNumber,
		&quoteStatus,
		&view.Quote.CustomerName,
		&view.Quote.StartAt,
		&view.Quote.EndAt,
		&view.Quote.EventType,
		&view.Quote.EventLocation,
		&view.Quote.Subtotal,
		&view.Quote.DiscountAmount,
		&view.Quote.ExtraCharges,
		&view.Quote.Total,
		&view.Quote.CreatedAt,
		&view.Quote.ReservationNumber,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return PublicView{}, ErrPortalNotFound
	}
	if err != nil {
		return PublicView{}, fmt.Errorf("load public quote portal: %w", err)
	}
	if !portalEnabled {
		return PublicView{}, ErrPortalUnavailable
	}
	view.Portal.Status = portalStatus
	view.Quote.Status = quoteStatus
	view.Portal.CanAccept = portalStatus == "ACTIVE" && quoteStatus == "SENT" && view.Portal.ExpiresAt.After(time.Now().UTC())
	view.Portal.CanReject = view.Portal.CanAccept && view.Portal.AllowRejection

	rows, err := r.pool.Query(ctx, `
		SELECT qi.description, resource.name, qi.quantity,
		       qi.unit_price::float8, qi.discount_amount::float8,
		       qi.line_total::float8
		FROM quote_items qi
		JOIN resources resource
		  ON resource.tenant_id = qi.tenant_id AND resource.id = qi.resource_id
		WHERE qi.tenant_id = $1 AND qi.quote_id = $2
		ORDER BY qi.created_at, qi.id
	`, view.TenantID, view.QuoteID)
	if err != nil {
		return PublicView{}, fmt.Errorf("load public quote items: %w", err)
	}
	defer rows.Close()
	view.Quote.Items = make([]PublicQuoteItem, 0)
	for rows.Next() {
		var item PublicQuoteItem
		if err := rows.Scan(
			&item.Description,
			&item.ResourceName,
			&item.Quantity,
			&item.UnitPrice,
			&item.DiscountAmount,
			&item.LineTotal,
		); err != nil {
			return PublicView{}, fmt.Errorf("scan public quote item: %w", err)
		}
		view.Quote.Items = append(view.Quote.Items, item)
	}
	if err := rows.Err(); err != nil {
		return PublicView{}, fmt.Errorf("iterate public quote items: %w", err)
	}
	return view, nil
}

type lockedPortal struct {
	PortalID            string
	TenantID            string
	QuoteID             string
	Status              string
	ExpiresAt           time.Time
	AllowRejection      bool
	RequireResponseName bool
	TermsVersion        string
	QuoteNumber         int64
	QuoteStatus         string
	DecisionAt          *time.Time
	ReservationNumber   *int64
}

type portalIdentity struct {
	TenantID      string
	QuoteID       string
	PortalEnabled bool
}

type rowQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func findPortalIdentity(ctx context.Context, q rowQuerier, tokenHash string) (portalIdentity, error) {
	var item portalIdentity
	err := q.QueryRow(ctx, `
		SELECT portal.tenant_id, portal.quote_id, settings.enabled
		FROM quote_portals portal
		JOIN quote_portal_settings settings ON settings.tenant_id = portal.tenant_id
		WHERE portal.token_hash = $1
	`, tokenHash).Scan(&item.TenantID, &item.QuoteID, &item.PortalEnabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return portalIdentity{}, ErrPortalNotFound
	}
	if err != nil {
		return portalIdentity{}, fmt.Errorf("find quote portal: %w", err)
	}
	if !item.PortalEnabled {
		return portalIdentity{}, ErrPortalUnavailable
	}
	return item, nil
}

func lockDecisionPortal(ctx context.Context, tx pgx.Tx, tokenHash string) (lockedPortal, error) {
	identity, err := findPortalIdentity(ctx, tx, tokenHash)
	if err != nil {
		return lockedPortal{}, err
	}

	var item lockedPortal
	item.TenantID = identity.TenantID
	item.QuoteID = identity.QuoteID
	err = tx.QueryRow(ctx, `
		SELECT q.quote_number, q.status, reservation.reservation_number
		FROM quotes q
		LEFT JOIN reservations reservation
		  ON reservation.tenant_id = q.tenant_id AND reservation.quote_id = q.id
		WHERE q.tenant_id = $1 AND q.id = $2
		FOR UPDATE OF q
	`, item.TenantID, item.QuoteID).Scan(
		&item.QuoteNumber,
		&item.QuoteStatus,
		&item.ReservationNumber,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return lockedPortal{}, ErrPortalNotFound
	}
	if err != nil {
		return lockedPortal{}, fmt.Errorf("lock quote for portal decision: %w", err)
	}

	err = tx.QueryRow(ctx, `
		SELECT id, status, expires_at, allow_rejection,
		       require_response_name, terms_version, decision_at
		FROM quote_portals
		WHERE token_hash = $1 AND tenant_id = $2 AND quote_id = $3
		FOR UPDATE
	`, tokenHash, item.TenantID, item.QuoteID).Scan(
		&item.PortalID,
		&item.Status,
		&item.ExpiresAt,
		&item.AllowRejection,
		&item.RequireResponseName,
		&item.TermsVersion,
		&item.DecisionAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return lockedPortal{}, ErrPortalNotFound
	}
	if err != nil {
		return lockedPortal{}, fmt.Errorf("lock quote portal decision: %w", err)
	}
	return item, nil
}

func expireLockedPortal(ctx context.Context, tx pgx.Tx, item lockedPortal, originHash, userAgent string) error {
	if item.Status == "EXPIRED" {
		return nil
	}
	if item.QuoteStatus == "SENT" {
		if _, err := tx.Exec(ctx, `
			UPDATE quotes SET status = 'EXPIRED'
			WHERE tenant_id = $1 AND id = $2 AND status = 'SENT'
		`, item.TenantID, item.QuoteID); err != nil {
			return fmt.Errorf("expire quote: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE quote_portals
		SET status = 'EXPIRED', decision_at = COALESCE(decision_at, NOW()),
		    decision_source = COALESCE(decision_source, 'SYSTEM')
		WHERE tenant_id = $1 AND id = $2
	`, item.TenantID, item.PortalID); err != nil {
		return fmt.Errorf("expire quote portal: %w", err)
	}
	return insertEvent(ctx, tx, item.TenantID, item.PortalID, item.QuoteID, "EXPIRED", "SYSTEM", "quote-portal-expiration", originHash, userAgent, nil)
}

func loadSettings(ctx context.Context, q rowQuerier, tenantID string) (Settings, error) {
	var item Settings
	err := q.QueryRow(ctx, `
		SELECT tenant_id, enabled, headline, introduction, accent_color,
		       default_validity_days, allow_rejection, require_response_name,
		       acceptance_terms_text, acceptance_terms_version,
		       created_at, updated_at
		FROM quote_portal_settings
		WHERE tenant_id = $1
	`, tenantID).Scan(
		&item.TenantID,
		&item.Enabled,
		&item.Headline,
		&item.Introduction,
		&item.AccentColor,
		&item.DefaultValidityDays,
		&item.AllowRejection,
		&item.RequireResponseName,
		&item.AcceptanceTermsText,
		&item.AcceptanceTermsVersion,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return Settings{}, fmt.Errorf("load quote portal settings: %w", err)
	}
	return item, nil
}

func loadSettingsForUpdate(ctx context.Context, tx pgx.Tx, tenantID string) (Settings, error) {
	var item Settings
	err := tx.QueryRow(ctx, `
		SELECT tenant_id, enabled, headline, introduction, accent_color,
		       default_validity_days, allow_rejection, require_response_name,
		       acceptance_terms_text, acceptance_terms_version,
		       created_at, updated_at
		FROM quote_portal_settings
		WHERE tenant_id = $1
		FOR UPDATE
	`, tenantID).Scan(
		&item.TenantID,
		&item.Enabled,
		&item.Headline,
		&item.Introduction,
		&item.AccentColor,
		&item.DefaultValidityDays,
		&item.AllowRejection,
		&item.RequireResponseName,
		&item.AcceptanceTermsText,
		&item.AcceptanceTermsVersion,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return Settings{}, fmt.Errorf("lock quote portal settings: %w", err)
	}
	return item, nil
}

func insertEvent(
	ctx context.Context,
	executor interface {
		Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	},
	tenantID, portalID, quoteID, eventType, actorType, actorID, originHash, userAgent string,
	metadata map[string]any,
) error {
	if metadata == nil {
		metadata = map[string]any{}
	}
	payload, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal quote portal event metadata: %w", err)
	}
	_, err = executor.Exec(ctx, `
		INSERT INTO quote_portal_events (
			tenant_id, portal_id, quote_id, event_type, actor_type,
			actor_id, origin_hash, user_agent, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), NULLIF($8, ''), $9::jsonb)
	`, tenantID, portalID, quoteID, eventType, actorType, actorID, originHash, userAgent, string(payload))
	if err != nil {
		return fmt.Errorf("insert quote portal event: %w", err)
	}
	return nil
}

func valueOrNow(value *time.Time) time.Time {
	if value != nil {
		return value.UTC()
	}
	return time.Now().UTC()
}

func publicAvailability(conflict *reservation.AvailabilityConflictError) PublicAvailabilityConflict {
	result := PublicAvailabilityConflict{Available: false, Items: make([]PublicAvailabilityItem, 0, len(conflict.Result.Items))}
	for _, item := range conflict.Result.Items {
		result.Items = append(result.Items, PublicAvailabilityItem{
			ResourceName:      item.ResourceName,
			RequestedQuantity: item.RequestedQuantity,
			CanFulfill:        item.CanFulfill,
		})
	}
	return result
}
