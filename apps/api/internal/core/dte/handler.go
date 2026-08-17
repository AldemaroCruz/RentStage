package dte

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/rentstage/rentstage/apps/api/internal/idutil"
	"github.com/rentstage/rentstage/apps/api/internal/webutil"
)

type Handler struct {
	repository *Repository
	service    *Service
}

func NewHandler(repository *Repository, service *Service) *Handler {
	return &Handler{repository: repository, service: service}
}

func (h *Handler) Settings(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.Settings(r.Context(), webutil.TenantID(r.Context()))
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "dte_settings_load_failed", "No fue posible cargar la configuración DTE.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var input SettingsInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "El cuerpo no contiene JSON válido.")
		return
	}
	item, fields, err := h.service.UpdateSettings(r.Context(), webutil.TenantID(r.Context()), input)
	if len(fields) > 0 {
		webutil.WriteValidationError(w, r, fields)
		return
	}
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "dte_settings_update_failed", "No fue posible guardar la configuración DTE.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	status := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
	if status != "" && !validStatus(status) {
		webutil.WriteValidationError(w, r, map[string]string{"status": "Estado DTE no soportado."})
		return
	}
	items, err := h.repository.ListDocuments(
		r.Context(), webutil.TenantID(r.Context()),
		strings.TrimSpace(r.URL.Query().Get("q")), status, queryLimit(r, 100),
	)
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "dte_list_failed", "No fue posible cargar los DTE.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	documentID, ok := pathUUID(w, r, "documentID", "invalid_dte_id", "El ID del DTE es inválido.")
	if !ok {
		return
	}
	item, err := h.repository.GetDocument(r.Context(), webutil.TenantID(r.Context()), documentID)
	if errors.Is(err, ErrDocumentNotFound) {
		webutil.WriteError(w, r, http.StatusNotFound, "dte_not_found", "DTE no encontrado.")
		return
	}
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "dte_load_failed", "No fue posible cargar el DTE.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) GetByInvoice(w http.ResponseWriter, r *http.Request) {
	invoiceID, ok := pathUUID(w, r, "invoiceID", "invalid_invoice_id", "El ID de factura es inválido.")
	if !ok {
		return
	}
	item, err := h.repository.GetDocumentByInvoice(r.Context(), webutil.TenantID(r.Context()), invoiceID)
	if errors.Is(err, ErrDocumentNotFound) {
		webutil.WriteError(w, r, http.StatusNotFound, "dte_not_found", "La factura todavía no tiene un DTE.")
		return
	}
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "dte_load_failed", "No fue posible cargar el DTE.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) Prepare(w http.ResponseWriter, r *http.Request) {
	invoiceID, ok := pathUUID(w, r, "invoiceID", "invalid_invoice_id", "El ID de factura es inválido.")
	if !ok {
		return
	}
	var input PrepareInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "El cuerpo no contiene JSON válido.")
		return
	}
	item, fields, err := h.service.Prepare(r.Context(), webutil.TenantID(r.Context()), invoiceID, input)
	if len(fields) > 0 {
		webutil.WriteValidationError(w, r, fields)
		return
	}
	h.writeMutation(w, r, item, err, http.StatusCreated)
}

func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	documentID, ok := pathUUID(w, r, "documentID", "invalid_dte_id", "El ID del DTE es inválido.")
	if !ok {
		return
	}
	item, err := h.service.Submit(r.Context(), webutil.TenantID(r.Context()), documentID)
	h.writeMutation(w, r, item, err, http.StatusOK)
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	documentID, ok := pathUUID(w, r, "documentID", "invalid_dte_id", "El ID del DTE es inválido.")
	if !ok {
		return
	}
	item, err := h.service.Cancel(r.Context(), webutil.TenantID(r.Context()), documentID)
	h.writeMutation(w, r, item, err, http.StatusOK)
}

func (h *Handler) Invalidate(w http.ResponseWriter, r *http.Request) {
	documentID, ok := pathUUID(w, r, "documentID", "invalid_dte_id", "El ID del DTE es inválido.")
	if !ok {
		return
	}
	var input InvalidateInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "El cuerpo no contiene JSON válido.")
		return
	}
	item, fields, err := h.service.Invalidate(r.Context(), webutil.TenantID(r.Context()), documentID, input)
	if len(fields) > 0 {
		webutil.WriteValidationError(w, r, fields)
		return
	}
	h.writeMutation(w, r, item, err, http.StatusOK)
}

func (h *Handler) writeMutation(w http.ResponseWriter, r *http.Request, item DocumentDetail, err error, status int) {
	switch {
	case errors.Is(err, ErrDocumentNotFound):
		webutil.WriteError(w, r, http.StatusNotFound, "dte_not_found", "DTE no encontrado.")
	case errors.Is(err, ErrInvoiceNotFound):
		webutil.WriteError(w, r, http.StatusNotFound, "invoice_not_found", "Factura no encontrada.")
	case errors.Is(err, ErrDocumentConflict):
		webutil.WriteError(w, r, http.StatusConflict, "dte_document_conflict", "La factura ya tiene un DTE activo.")
	case errors.Is(err, ErrSettingsChanged):
		webutil.WriteError(w, r, http.StatusConflict, "dte_settings_changed", "La configuración DTE cambió durante la preparación. Revisa los valores e inténtalo nuevamente.")
	case errors.Is(err, ErrDocumentState):
		webutil.WriteError(w, r, http.StatusConflict, "invalid_dte_state", "El DTE no admite esta operación en su estado actual.")
	case errors.Is(err, ErrSettingsDisabled):
		webutil.WriteError(w, r, http.StatusConflict, "dte_disabled", "La integración DTE está deshabilitada.")
	case errors.Is(err, ErrSettingsIncomplete), errors.Is(err, ErrProviderConfiguration):
		webutil.WriteError(w, r, http.StatusUnprocessableEntity, "dte_configuration_incomplete", "Completa y valida la configuración DTE antes de continuar.")
	case errors.Is(err, ErrInvoiceState), errors.Is(err, ErrInvoiceFiscalState):
		webutil.WriteError(w, r, http.StatusConflict, "invoice_not_dte_eligible", "La factura no está lista para DTE.")
	case errors.Is(err, ErrInvalidationUnsupported):
		webutil.WriteError(w, r, http.StatusUnprocessableEntity, "dte_invalidation_unavailable", "El proveedor no tiene configurado el servicio de invalidación.")
	case err != nil:
		slog.Error("DTE mutation failed",
			"request_id", webutil.RequestID(r.Context()),
			"tenant_id", webutil.TenantID(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
			"error", err,
		)
		webutil.WriteError(w, r, http.StatusInternalServerError, "dte_mutation_failed", "No fue posible completar la operación DTE.")
	default:
		webutil.WriteJSON(w, status, item)
	}
}

func pathUUID(w http.ResponseWriter, r *http.Request, name, code, message string) (string, bool) {
	value := r.PathValue(name)
	if !idutil.IsUUID(value) {
		webutil.WriteError(w, r, http.StatusBadRequest, code, message)
		return "", false
	}
	return value, true
}

func queryLimit(r *http.Request, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("limit")))
	if err != nil || value <= 0 {
		return fallback
	}
	if value > 250 {
		return 250
	}
	return value
}

func validStatus(value string) bool {
	switch value {
	case "READY_TO_SIGN", "SUBMITTING", "ACCEPTED", "REJECTED", "RETRY_REQUIRED", "INVALIDATION_PENDING", "INVALIDATED", "CANCELLED":
		return true
	default:
		return false
	}
}
