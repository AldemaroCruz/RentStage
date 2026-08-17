package quoteportal

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/mail"
	"regexp"
	"strings"

	"github.com/rentstage/rentstage/apps/api/internal/core/audit"
	"github.com/rentstage/rentstage/apps/api/internal/core/quote"
	"github.com/rentstage/rentstage/apps/api/internal/core/reservation"
)

var accentPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

type Service struct {
	repository      *Repository
	quotes          *quote.Repository
	audit           *audit.Repository
	webBaseURL      string
	fingerprintSalt string
}

func NewService(
	repository *Repository,
	quoteRepository *quote.Repository,
	auditRepository *audit.Repository,
	webBaseURL string,
	fingerprintSalt string,
) *Service {
	return &Service{
		repository:      repository,
		quotes:          quoteRepository,
		audit:           auditRepository,
		webBaseURL:      strings.TrimRight(webBaseURL, "/"),
		fingerprintSalt: fingerprintSalt,
	}
}

func (s *Service) Settings(ctx context.Context, tenantID string) (Settings, error) {
	return s.repository.Settings(ctx, tenantID)
}

func (s *Service) UpdateSettings(
	ctx context.Context,
	tenantID string,
	input SettingsInput,
) (Settings, map[string]string, error) {
	normalized, fields := normalizeSettings(input)
	if len(fields) > 0 {
		return Settings{}, fields, nil
	}
	item, err := s.repository.UpdateSettings(ctx, tenantID, normalized)
	if err != nil {
		return Settings{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "QUOTE_PORTAL_SETTINGS_UPDATED", "tenant", &tenantID, map[string]any{
		"enabled":                  item.Enabled,
		"default_validity_days":    item.DefaultValidityDays,
		"allow_rejection":          item.AllowRejection,
		"require_response_name":    item.RequireResponseName,
		"acceptance_terms_version": item.AcceptanceTermsVersion,
	})
	return item, nil, nil
}

func (s *Service) Send(ctx context.Context, tenantID, quoteID, actorID string) (quote.Detail, error) {
	return s.issue(ctx, tenantID, quoteID, actorID, false)
}

func (s *Service) Reissue(ctx context.Context, tenantID, quoteID, actorID string) (quote.Detail, error) {
	return s.issue(ctx, tenantID, quoteID, actorID, true)
}

func (s *Service) issue(ctx context.Context, tenantID, quoteID, actorID string, reissue bool) (quote.Detail, error) {
	token, tokenHash, err := newToken()
	if err != nil {
		return quote.Detail{}, err
	}
	issued, err := s.repository.Issue(ctx, tenantID, quoteID, actorID, tokenHash, reissue)
	if err != nil {
		return quote.Detail{}, err
	}
	issued.Token = token
	issued.PublicURL = s.webBaseURL + "/q#" + token
	item, err := s.quotes.Get(ctx, tenantID, quoteID)
	if err != nil {
		return quote.Detail{}, err
	}
	if item.Portal != nil {
		item.Portal.PublicURL = issued.PublicURL
	}
	action := "QUOTE_PORTAL_CREATED"
	if issued.EventType == "REISSUED" {
		action = "QUOTE_PORTAL_REISSUED"
	}
	_ = s.audit.Record(ctx, tenantID, action, "quote", &quoteID, map[string]any{
		"quote_number": item.QuoteNumber,
		"expires_at":   issued.ExpiresAt,
		"revision":     issued.Revision,
	})
	return item, nil
}

func (s *Service) Revoke(ctx context.Context, tenantID, quoteID, actorID string) (quote.Detail, error) {
	portalID, err := s.repository.Revoke(ctx, tenantID, quoteID, actorID)
	if err != nil {
		return quote.Detail{}, err
	}
	item, err := s.quotes.Get(ctx, tenantID, quoteID)
	if err != nil {
		return quote.Detail{}, err
	}
	_ = s.audit.Record(ctx, tenantID, "QUOTE_PORTAL_REVOKED", "quote", &quoteID, map[string]any{
		"quote_number": item.QuoteNumber,
		"portal_id":    portalID,
	})
	return item, nil
}

func (s *Service) PublicView(ctx context.Context, token, clientIP, userAgent string) (PublicView, error) {
	tokenHash, err := hashToken(token)
	if err != nil {
		return PublicView{}, ErrPortalNotFound
	}
	return s.repository.TouchAndGet(ctx, tokenHash, s.originHash(tokenHash, clientIP), cleanUserAgent(userAgent))
}

func (s *Service) Accept(
	ctx context.Context,
	token string,
	input AcceptInput,
	clientIP string,
	userAgent string,
) (DecisionResult, map[string]string, *PublicAvailabilityConflict, error) {
	tokenHash, err := hashToken(token)
	if err != nil {
		return DecisionResult{}, nil, nil, ErrPortalNotFound
	}
	decision, fields := normalizeAccept(input)
	if len(fields) > 0 {
		return DecisionResult{}, fields, nil, nil
	}
	result, err := s.repository.Accept(
		ctx,
		tokenHash,
		decision,
		s.originHash(tokenHash, clientIP),
		cleanUserAgent(userAgent),
	)
	if errors.Is(err, ErrResponseNameRequired) {
		return DecisionResult{}, map[string]string{"response_name": "Escribe el nombre de la persona que acepta."}, nil, nil
	}
	var conflict *reservation.AvailabilityConflictError
	if errors.As(err, &conflict) {
		public := publicAvailability(conflict)
		return DecisionResult{}, nil, &public, nil
	}
	if err != nil {
		return DecisionResult{}, nil, nil, err
	}
	if !result.Idempotent {
		_ = s.audit.RecordAs(ctx, result.TenantID, "API", "quote-portal-customer", "QUOTE_ACCEPTED_ONLINE", "quote", &result.QuoteID, map[string]any{
			"quote_number":       result.QuoteNumber,
			"reservation_number": result.ReservationNumber,
		})
		if result.ReservationID != "" {
			_ = s.audit.RecordAs(ctx, result.TenantID, "API", "quote-portal-customer", "RESERVATION_CREATED_FROM_QUOTE", "reservation", &result.ReservationID, map[string]any{
				"quote_id":           result.QuoteID,
				"quote_number":       result.QuoteNumber,
				"reservation_number": result.ReservationNumber,
				"source":             "QUOTE_PORTAL",
			})
		}
	}
	return result, nil, nil, nil
}

func (s *Service) Reject(
	ctx context.Context,
	token string,
	input RejectInput,
	clientIP string,
	userAgent string,
) (DecisionResult, map[string]string, error) {
	tokenHash, err := hashToken(token)
	if err != nil {
		return DecisionResult{}, nil, ErrPortalNotFound
	}
	decision, fields := normalizeReject(input)
	if len(fields) > 0 {
		return DecisionResult{}, fields, nil
	}
	result, err := s.repository.Reject(
		ctx,
		tokenHash,
		decision,
		s.originHash(tokenHash, clientIP),
		cleanUserAgent(userAgent),
	)
	if errors.Is(err, ErrResponseNameRequired) {
		return DecisionResult{}, map[string]string{"response_name": "Escribe el nombre de la persona que responde."}, nil
	}
	if err != nil {
		return DecisionResult{}, nil, err
	}
	if !result.Idempotent {
		_ = s.audit.RecordAs(ctx, result.TenantID, "API", "quote-portal-customer", "QUOTE_REJECTED_ONLINE", "quote", &result.QuoteID, map[string]any{
			"quote_number": result.QuoteNumber,
		})
	}
	return result, nil, nil
}

func normalizeSettings(input SettingsInput) (normalizedSettings, map[string]string) {
	result := normalizedSettings{
		Enabled:                input.Enabled,
		Headline:               strings.TrimSpace(input.Headline),
		Introduction:           strings.TrimSpace(input.Introduction),
		AccentColor:            strings.ToLower(strings.TrimSpace(input.AccentColor)),
		DefaultValidityDays:    input.DefaultValidityDays,
		AllowRejection:         input.AllowRejection,
		RequireResponseName:    input.RequireResponseName,
		AcceptanceTermsText:    strings.TrimSpace(input.AcceptanceTermsText),
		AcceptanceTermsVersion: strings.TrimSpace(input.AcceptanceTermsVersion),
	}
	fields := map[string]string{}
	if result.Headline == "" || len(result.Headline) > 180 {
		fields["headline"] = "El título debe contener entre 1 y 180 caracteres."
	}
	if len(result.Introduction) > 2000 {
		fields["introduction"] = "La introducción no puede superar 2,000 caracteres."
	}
	if !accentPattern.MatchString(result.AccentColor) {
		fields["accent_color"] = "Usa un color hexadecimal como #6558e8."
	}
	if result.DefaultValidityDays < 1 || result.DefaultValidityDays > 60 {
		fields["default_validity_days"] = "La vigencia debe estar entre 1 y 60 días."
	}
	if result.AcceptanceTermsText == "" || len(result.AcceptanceTermsText) > 12000 {
		fields["acceptance_terms_text"] = "Los términos deben contener entre 1 y 12,000 caracteres."
	}
	if result.AcceptanceTermsVersion == "" || len(result.AcceptanceTermsVersion) > 40 {
		fields["acceptance_terms_version"] = "La versión debe contener entre 1 y 40 caracteres."
	}
	return result, fields
}

func normalizeAccept(input AcceptInput) (normalizedDecision, map[string]string) {
	decision, fields := normalizeDecision(input.ResponseName, input.ResponseEmail, "")
	if !input.TermsAccepted {
		fields["terms_accepted"] = "Debes aceptar los términos para confirmar la cotización."
	}
	return decision, fields
}

func normalizeReject(input RejectInput) (normalizedDecision, map[string]string) {
	return normalizeDecision(input.ResponseName, input.ResponseEmail, input.RejectionReason)
}

func normalizeDecision(name, email, reason string) (normalizedDecision, map[string]string) {
	result := normalizedDecision{ResponseName: strings.TrimSpace(name)}
	fields := map[string]string{}
	if len(result.ResponseName) > 180 {
		fields["response_name"] = "El nombre no puede superar 180 caracteres."
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if email != "" {
		parsed, err := mail.ParseAddress(email)
		if err != nil || !strings.EqualFold(parsed.Address, email) || len(email) > 320 {
			fields["response_email"] = "Escribe un correo válido."
		} else {
			result.ResponseEmail = &email
		}
	}
	reason = strings.TrimSpace(reason)
	if len(reason) > 2000 {
		fields["rejection_reason"] = "El motivo no puede superar 2,000 caracteres."
	} else if reason != "" {
		result.RejectionReason = &reason
	}
	return result, fields
}

func newToken() (string, string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(buffer)
	digest := sha256.Sum256([]byte(token))
	return token, hex.EncodeToString(digest[:]), nil
}

func hashToken(token string) (string, error) {
	token = strings.TrimSpace(token)
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(decoded) != 32 {
		return "", ErrPortalNotFound
	}
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:]), nil
}

func (s *Service) originHash(tokenHash, clientIP string) string {
	digest := sha256.Sum256([]byte(s.fingerprintSalt + "|quote-portal|" + tokenHash + "|" + strings.TrimSpace(clientIP)))
	return hex.EncodeToString(digest[:])
}

func cleanUserAgent(value string) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) > 500 {
		value = string(runes[:500])
	}
	return value
}
