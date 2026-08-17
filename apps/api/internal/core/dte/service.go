package dte

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rentstage/rentstage/apps/api/internal/core/audit"
	"github.com/rentstage/rentstage/apps/api/internal/idutil"
	"github.com/rentstage/rentstage/apps/api/internal/webutil"
)

type Service struct {
	repository *Repository
	audit      *audit.Repository
}

func NewService(repository *Repository, auditRepository *audit.Repository) *Service {
	return &Service{repository: repository, audit: auditRepository}
}

func (s *Service) Settings(ctx context.Context, tenantID string) (Settings, error) {
	item, err := s.repository.GetSettings(ctx, tenantID)
	if err != nil {
		return Settings{}, err
	}
	return enrichSettings(item), nil
}

func (s *Service) UpdateSettings(ctx context.Context, tenantID string, input SettingsInput) (Settings, map[string]string, error) {
	normalized, fields := normalizeSettings(input)
	if len(fields) > 0 {
		return Settings{}, fields, nil
	}
	item, err := s.repository.UpdateSettings(ctx, tenantID, normalized)
	if err != nil {
		return Settings{}, nil, err
	}
	item = enrichSettings(item)
	_ = s.audit.Record(ctx, tenantID, "DTE_SETTINGS_UPDATED", "dte_settings", nil, map[string]any{
		"provider_mode":         item.ProviderMode,
		"environment":           item.Environment,
		"default_document_type": item.DefaultDocumentType,
		"enabled":               item.Enabled,
	})
	return item, nil, nil
}

func (s *Service) Prepare(ctx context.Context, tenantID, invoiceID string, input PrepareInput) (DocumentDetail, map[string]string, error) {
	settings, err := s.Settings(ctx, tenantID)
	if err != nil {
		return DocumentDetail{}, nil, err
	}
	if !settings.Enabled {
		return DocumentDetail{}, nil, ErrSettingsDisabled
	}
	invoice, err := s.repository.LoadInvoiceSnapshot(ctx, tenantID, invoiceID)
	if err != nil {
		return DocumentDetail{}, nil, err
	}
	documentType := normalizeDocumentType(input.DocumentType, settings, invoice)
	fields := validateSnapshot(settings, invoice, documentType)
	if !settings.ConfigurationReady {
		for _, field := range settings.MissingConfiguration {
			if _, exists := fields[field]; !exists {
				fields[field] = "Completa este valor en la configuración DTE."
			}
		}
	}
	if len(fields) > 0 {
		return DocumentDetail{}, fields, nil
	}
	generationCode := idutil.NewUUID()
	actorID := webutil.UserID(ctx)
	item, err := s.repository.CreateDocument(
		ctx, tenantID, invoiceID, documentType, generationCode, actorID, settings,
		func(control string, lockedSettings Settings) (map[string]any, error) {
			return buildPayload(lockedSettings, invoice, documentType, generationCode, control), nil
		},
	)
	if err != nil {
		return DocumentDetail{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "DTE_PREPARED", "dte_document", &item.ID, map[string]any{
		"invoice_id":     item.InvoiceID,
		"invoice_number": item.InvoiceDisplayNumber,
		"document_type":  item.DocumentType,
		"control_number": item.ControlNumber,
		"environment":    item.Environment,
	})
	return item, nil, nil
}

func (s *Service) Submit(ctx context.Context, tenantID, documentID string) (DocumentDetail, error) {
	settings, err := s.Settings(ctx, tenantID)
	if err != nil {
		return DocumentDetail{}, err
	}
	if !settings.Enabled {
		return DocumentDetail{}, ErrSettingsDisabled
	}
	prepared, err := s.repository.GetDocument(ctx, tenantID, documentID)
	if err != nil {
		return DocumentDetail{}, err
	}
	executionSettings := settingsForDocument(settings, prepared.DocumentSummary)
	if !executionSettings.ConfigurationReady {
		return DocumentDetail{}, ErrSettingsIncomplete
	}
	provider, err := providerFor(executionSettings)
	if err != nil {
		return DocumentDetail{}, err
	}
	actorID := webutil.UserID(ctx)
	item, err := s.repository.MarkSubmitting(ctx, tenantID, documentID, actorID, executionSettings.MaxAttempts)
	if err != nil {
		return DocumentDetail{}, err
	}

	result, providerErr := provider.Submit(ctx, ProviderSubmission{Settings: executionSettings, Document: item})
	if providerErr != nil {
		result = ProviderResult{
			Accepted:       false,
			Retryable:      true,
			ProviderStatus: "TRANSPORT_ERROR",
			ErrorCode:      "PROVIDER_UNAVAILABLE",
			ErrorMessage:   providerErr.Error(),
			Request:        map[string]any{"provider": executionSettings.ProviderMode},
			Response:       map[string]any{},
		}
	}
	var retryAt *time.Time
	if result.Retryable && item.AttemptCount < executionSettings.MaxAttempts {
		delay := retryDelay(executionSettings.RetryBaseSeconds, item.AttemptCount)
		value := time.Now().Add(delay)
		retryAt = &value
	} else if result.Retryable {
		result.Retryable = false
		result.ErrorCode = firstNonEmpty(result.ErrorCode, "MAX_ATTEMPTS_REACHED")
		result.ErrorMessage = firstNonEmpty(result.ErrorMessage, "Se alcanzó el máximo de intentos configurado.")
	}
	updated, err := s.repository.ApplySubmissionResult(ctx, tenantID, documentID, actorID, result, retryAt)
	if err != nil {
		return DocumentDetail{}, err
	}
	action := "DTE_REJECTED"
	if updated.Status == "ACCEPTED" {
		action = "DTE_ACCEPTED"
	} else if updated.Status == "RETRY_REQUIRED" {
		action = "DTE_RETRY_REQUIRED"
	}
	_ = s.audit.Record(ctx, tenantID, action, "dte_document", &updated.ID, map[string]any{
		"invoice_id":      updated.InvoiceID,
		"control_number":  updated.ControlNumber,
		"receipt_seal":    updated.ReceiptSeal,
		"provider_status": updated.ProviderStatus,
		"attempt_count":   updated.AttemptCount,
	})
	return updated, nil
}

func (s *Service) Cancel(ctx context.Context, tenantID, documentID string) (DocumentDetail, error) {
	actorID := webutil.UserID(ctx)
	item, err := s.repository.CancelDocument(ctx, tenantID, documentID, actorID)
	if err != nil {
		return DocumentDetail{}, err
	}
	_ = s.audit.Record(ctx, tenantID, "DTE_CANCELLED", "dte_document", &item.ID, map[string]any{
		"invoice_id":     item.InvoiceID,
		"control_number": item.ControlNumber,
	})
	return item, nil
}

func (s *Service) Invalidate(ctx context.Context, tenantID, documentID string, input InvalidateInput) (DocumentDetail, map[string]string, error) {
	reason := strings.TrimSpace(input.Reason)
	fields := map[string]string{}
	if len(reason) < 5 {
		fields["reason"] = "Indica un motivo de al menos cinco caracteres."
	} else if len(reason) > 500 {
		fields["reason"] = "El motivo no puede superar 500 caracteres."
	}
	if len(fields) > 0 {
		return DocumentDetail{}, fields, nil
	}
	settings, err := s.Settings(ctx, tenantID)
	if err != nil {
		return DocumentDetail{}, nil, err
	}
	prepared, err := s.repository.GetDocument(ctx, tenantID, documentID)
	if err != nil {
		return DocumentDetail{}, nil, err
	}
	executionSettings := settingsForDocument(settings, prepared.DocumentSummary)
	if !executionSettings.ConfigurationReady {
		return DocumentDetail{}, nil, ErrSettingsIncomplete
	}
	provider, err := providerFor(executionSettings)
	if err != nil {
		return DocumentDetail{}, nil, err
	}
	actorID := webutil.UserID(ctx)
	item, err := s.repository.BeginInvalidation(ctx, tenantID, documentID, actorID, reason)
	if err != nil {
		return DocumentDetail{}, nil, err
	}
	result, providerErr := provider.Invalidate(ctx, ProviderSubmission{Settings: executionSettings, Document: item}, reason)
	if providerErr != nil {
		result = InvalidationResult{
			Accepted:       false,
			Retryable:      errors.Is(providerErr, ErrProviderUnavailable),
			ProviderStatus: "INVALIDATION_ERROR",
			ErrorCode:      "INVALIDATION_FAILED",
			ErrorMessage:   providerErr.Error(),
		}
	}
	updated, err := s.repository.ApplyInvalidationResult(ctx, tenantID, documentID, actorID, result)
	if err != nil {
		return DocumentDetail{}, nil, err
	}
	action := "DTE_INVALIDATION_FAILED"
	if updated.Status == "INVALIDATED" {
		action = "DTE_INVALIDATED"
	}
	_ = s.audit.Record(ctx, tenantID, action, "dte_document", &updated.ID, map[string]any{
		"invoice_id":      updated.InvoiceID,
		"control_number":  updated.ControlNumber,
		"reason":          reason,
		"provider_status": updated.ProviderStatus,
	})
	return updated, nil, nil
}

func settingsForDocument(current Settings, document DocumentSummary) Settings {
	current.ProviderMode = document.ProviderMode
	current.Environment = document.Environment
	current.SchemaVersion = document.SchemaVersion
	return enrichSettings(current)
}

func normalizeSettings(input SettingsInput) (SettingsInput, map[string]string) {
	input.ProviderMode = strings.ToUpper(strings.TrimSpace(input.ProviderMode))
	input.Environment = strings.ToUpper(strings.TrimSpace(input.Environment))
	input.DefaultDocumentType = strings.TrimSpace(input.DefaultDocumentType)
	input.EstablishmentType = strings.ToUpper(strings.TrimSpace(input.EstablishmentType))
	input.EstablishmentCode = strings.ToUpper(strings.TrimSpace(input.EstablishmentCode))
	input.PointOfSaleCode = strings.ToUpper(strings.TrimSpace(input.PointOfSaleCode))
	input.AuthURL = strings.TrimSpace(input.AuthURL)
	input.SignerURL = strings.TrimSpace(input.SignerURL)
	input.ReceptionURL = strings.TrimSpace(input.ReceptionURL)
	input.InvalidationURL = strings.TrimSpace(input.InvalidationURL)
	input.QueryURL = strings.TrimSpace(input.QueryURL)
	input.UserSecretRef = strings.TrimSpace(input.UserSecretRef)
	input.PasswordSecretRef = strings.TrimSpace(input.PasswordSecretRef)
	input.SigningPasswordSecretRef = strings.TrimSpace(input.SigningPasswordSecretRef)
	if input.ProviderMode == "" {
		input.ProviderMode = "MOCK"
	}
	if input.Environment == "" {
		input.Environment = "TEST"
	}
	if input.DefaultDocumentType == "" {
		input.DefaultDocumentType = "01"
	}
	if input.SchemaVersion == 0 {
		input.SchemaVersion = 1
	}
	if input.EstablishmentType == "" {
		input.EstablishmentType = "01"
	}
	if input.EstablishmentCode == "" {
		input.EstablishmentCode = "M001"
	}
	if input.PointOfSaleCode == "" {
		input.PointOfSaleCode = "P001"
	}
	if input.MaxAttempts == 0 {
		input.MaxAttempts = 5
	}
	if input.RetryBaseSeconds == 0 {
		input.RetryBaseSeconds = 60
	}

	fields := map[string]string{}
	if input.ProviderMode != "MOCK" && input.ProviderMode != "MH_HTTP" {
		fields["provider_mode"] = "Selecciona MOCK o MH_HTTP."
	}
	if input.Environment != "TEST" && input.Environment != "PRODUCTION" {
		fields["environment"] = "Selecciona TEST o PRODUCTION."
	}
	if input.ProviderMode == "MOCK" && input.Environment == "PRODUCTION" {
		fields["environment"] = "El proveedor MOCK solo puede utilizarse en ambiente TEST."
	}
	if input.AutoSubmitOnIssue {
		fields["auto_submit_on_issue"] = "El envío automático está reservado; v0.12 requiere confirmación manual."
		input.AutoSubmitOnIssue = false
	}
	if input.DefaultDocumentType != "01" && input.DefaultDocumentType != "03" {
		fields["default_document_type"] = "Selecciona Factura (01) o CCF (03)."
	}
	if input.SchemaVersion < 1 || input.SchemaVersion > 99 {
		fields["schema_version"] = "La versión debe estar entre 1 y 99."
	}
	if len(input.EstablishmentCode) != 4 {
		fields["establishment_code"] = "El código de establecimiento debe tener cuatro caracteres."
	}
	if len(input.PointOfSaleCode) != 4 {
		fields["point_of_sale_code"] = "El punto de venta debe tener cuatro caracteres."
	}
	if input.MaxAttempts < 1 || input.MaxAttempts > 20 {
		fields["max_attempts"] = "Los intentos deben estar entre 1 y 20."
	}
	if input.RetryBaseSeconds < 5 || input.RetryBaseSeconds > 86400 {
		fields["retry_base_seconds"] = "El intervalo debe estar entre 5 y 86,400 segundos."
	}
	if input.ProviderMode == "MH_HTTP" {
		for key, value := range map[string]string{
			"auth_url":      input.AuthURL,
			"signer_url":    input.SignerURL,
			"reception_url": input.ReceptionURL,
		} {
			if !validEndpoint(value, input.Environment == "PRODUCTION") {
				fields[key] = "Configura una URL válida" + productionHTTPS(input.Environment) + "."
			}
		}
		for key, value := range map[string]string{
			"user_secret_ref":             input.UserSecretRef,
			"password_secret_ref":         input.PasswordSecretRef,
			"signing_password_secret_ref": input.SigningPasswordSecretRef,
		} {
			if !strings.HasPrefix(value, "env://") || len(strings.TrimPrefix(value, "env://")) < 2 {
				fields[key] = "Usa una referencia env://NOMBRE_VARIABLE; nunca guardes el secreto aquí."
			}
		}
	}
	return input, fields
}

func enrichSettings(item Settings) Settings {
	missing := make([]string, 0)
	if strings.TrimSpace(item.EstablishmentCode) == "" {
		missing = append(missing, "establishment_code")
	}
	if strings.TrimSpace(item.PointOfSaleCode) == "" {
		missing = append(missing, "point_of_sale_code")
	}
	if item.ProviderMode == "MOCK" && item.Environment == "PRODUCTION" {
		missing = append(missing, "environment")
	}
	if item.ProviderMode == "MH_HTTP" {
		for key, value := range map[string]string{
			"auth_url":                    item.AuthURL,
			"signer_url":                  item.SignerURL,
			"reception_url":               item.ReceptionURL,
			"user_secret_ref":             item.UserSecretRef,
			"password_secret_ref":         item.PasswordSecretRef,
			"signing_password_secret_ref": item.SigningPasswordSecretRef,
		} {
			if strings.TrimSpace(value) == "" {
				missing = append(missing, key)
			}
		}
	}
	item.MissingConfiguration = missing
	item.ConfigurationReady = len(missing) == 0
	item.ProductionSafetyReady = item.ProviderMode == "MH_HTTP" && item.Environment == "PRODUCTION" &&
		validEndpoint(item.AuthURL, true) && validEndpoint(item.SignerURL, true) && validEndpoint(item.ReceptionURL, true) &&
		strings.HasPrefix(item.UserSecretRef, "env://") && strings.HasPrefix(item.PasswordSecretRef, "env://") &&
		strings.HasPrefix(item.SigningPasswordSecretRef, "env://")
	if item.ProviderMode == "MOCK" {
		item.ConfigurationReady = true
	}
	return item
}

func validEndpoint(value string, requireHTTPS bool) bool {
	return validateProviderEndpoint(value, requireHTTPS) == nil
}

func productionHTTPS(environment string) string {
	if environment == "PRODUCTION" {
		return " con HTTPS"
	}
	return ""
}

func retryDelay(baseSeconds, attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	multiplier := 1 << min(attempt-1, 6)
	return time.Duration(baseSeconds*multiplier) * time.Second
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func formatSettingsError(settings Settings) error {
	if settings.ConfigurationReady {
		return nil
	}
	return fmt.Errorf("missing DTE configuration: %s: %w", strings.Join(settings.MissingConfiguration, ", "), ErrSettingsIncomplete)
}
