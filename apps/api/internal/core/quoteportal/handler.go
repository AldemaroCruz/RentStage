package quoteportal

import (
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/rentstage/rentstage/apps/api/internal/core/quote"
	"github.com/rentstage/rentstage/apps/api/internal/idutil"
	"github.com/rentstage/rentstage/apps/api/internal/webutil"
)

const quotePortalTokenHeader = "X-RentStage-Quote-Token"

func setPublicHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow, noarchive")
	w.Header().Add("Vary", quotePortalTokenHeader)
}

func publicToken(w http.ResponseWriter, r *http.Request) (string, bool) {
	token := strings.TrimSpace(r.Header.Get(quotePortalTokenHeader))
	if token == "" {
		webutil.WriteError(w, r, http.StatusNotFound, "quote_portal_not_found", "El enlace de cotización no existe o ya fue reemplazado.")
		return "", false
	}
	return token, true
}

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Settings(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.Settings(r.Context(), webutil.TenantID(r.Context()))
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "quote_portal_settings_load_failed", "No fue posible cargar la configuración del portal.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var input SettingsInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	item, fields, err := h.service.UpdateSettings(r.Context(), webutil.TenantID(r.Context()), input)
	if len(fields) > 0 {
		webutil.WriteValidationError(w, r, fields)
		return
	}
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "quote_portal_settings_update_failed", "No fue posible guardar la configuración del portal.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) Send(w http.ResponseWriter, r *http.Request) {
	quoteID, ok := quotePathID(w, r)
	if !ok {
		return
	}
	item, err := h.service.Send(r.Context(), webutil.TenantID(r.Context()), quoteID, webutil.ActorID(r.Context()))
	h.writeAdminMutation(w, r, item, err)
}

func (h *Handler) Reissue(w http.ResponseWriter, r *http.Request) {
	quoteID, ok := quotePathID(w, r)
	if !ok {
		return
	}
	item, err := h.service.Reissue(r.Context(), webutil.TenantID(r.Context()), quoteID, webutil.ActorID(r.Context()))
	h.writeAdminMutation(w, r, item, err)
}

func (h *Handler) Revoke(w http.ResponseWriter, r *http.Request) {
	quoteID, ok := quotePathID(w, r)
	if !ok {
		return
	}
	item, err := h.service.Revoke(r.Context(), webutil.TenantID(r.Context()), quoteID, webutil.ActorID(r.Context()))
	h.writeAdminMutation(w, r, item, err)
}

func (h *Handler) PublicView(w http.ResponseWriter, r *http.Request) {
	setPublicHeaders(w)
	token, ok := publicToken(w, r)
	if !ok {
		return
	}
	item, err := h.service.PublicView(r.Context(), token, clientIP(r), r.UserAgent())
	switch {
	case errors.Is(err, ErrPortalNotFound):
		webutil.WriteError(w, r, http.StatusNotFound, "quote_portal_not_found", "El enlace de cotización no existe o ya fue reemplazado.")
	case errors.Is(err, ErrPortalUnavailable):
		webutil.WriteError(w, r, http.StatusGone, "quote_portal_unavailable", "Este enlace de cotización ya no está disponible.")
	case err != nil:
		webutil.WriteError(w, r, http.StatusInternalServerError, "quote_portal_load_failed", "No fue posible cargar la cotización.")
	default:
		webutil.WriteJSON(w, http.StatusOK, item)
	}
}

func (h *Handler) Accept(w http.ResponseWriter, r *http.Request) {
	setPublicHeaders(w)
	token, ok := publicToken(w, r)
	if !ok {
		return
	}
	var input AcceptInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	item, fields, conflict, err := h.service.Accept(
		r.Context(), token, input, clientIP(r), r.UserAgent(),
	)
	if len(fields) > 0 {
		webutil.WriteValidationError(w, r, fields)
		return
	}
	if conflict != nil {
		webutil.WriteJSON(w, http.StatusConflict, map[string]any{
			"error":        "availability_conflict",
			"message":      "La cotización sigue vigente, pero uno o más recursos ya no están disponibles para esas fechas. Contacta a la empresa para ajustar la propuesta.",
			"request_id":   webutil.RequestID(r.Context()),
			"availability": conflict,
		})
		return
	}
	if h.writePublicDecisionFailure(w, r, err) {
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	setPublicHeaders(w)
	token, ok := publicToken(w, r)
	if !ok {
		return
	}
	var input RejectInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	item, fields, err := h.service.Reject(
		r.Context(), token, input, clientIP(r), r.UserAgent(),
	)
	if len(fields) > 0 {
		webutil.WriteValidationError(w, r, fields)
		return
	}
	if h.writePublicDecisionFailure(w, r, err) {
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) writeAdminMutation(w http.ResponseWriter, r *http.Request, item quote.Detail, err error) {
	// Send/reissue responses may contain the raw one-time bearer link. Never let
	// browsers or intermediaries cache a response handled by this mutation path.
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Referrer-Policy", "no-referrer")
	switch {
	case errors.Is(err, ErrQuoteNotFound), errors.Is(err, quote.ErrNotFound):
		webutil.WriteError(w, r, http.StatusNotFound, "quote_not_found", "Quote not found.")
	case errors.Is(err, ErrPortalNotFound):
		webutil.WriteError(w, r, http.StatusNotFound, "quote_portal_not_found", "La cotización todavía no tiene un portal.")
	case errors.Is(err, ErrPortalDisabled):
		webutil.WriteError(w, r, http.StatusConflict, "quote_portal_disabled", "El portal de cotizaciones está deshabilitado para este workspace.")
	case errors.Is(err, ErrInvalidQuoteStatus):
		webutil.WriteError(w, r, http.StatusConflict, "invalid_quote_transition", "La cotización no permite esa acción desde su estado actual.")
	case errors.Is(err, ErrInvalidPortalStatus):
		webutil.WriteError(w, r, http.StatusConflict, "invalid_quote_portal_status", "El portal ya fue respondido, vencido o revocado.")
	case err != nil:
		webutil.WriteError(w, r, http.StatusInternalServerError, "quote_portal_mutation_failed", "No fue posible actualizar el portal de la cotización.")
	default:
		webutil.WriteJSON(w, http.StatusOK, item)
	}
}

func (h *Handler) writePublicDecisionFailure(w http.ResponseWriter, r *http.Request, err error) bool {
	switch {
	case errors.Is(err, ErrPortalNotFound):
		webutil.WriteError(w, r, http.StatusNotFound, "quote_portal_not_found", "El enlace de cotización no existe o ya fue reemplazado.")
	case errors.Is(err, ErrPortalUnavailable):
		webutil.WriteError(w, r, http.StatusGone, "quote_portal_unavailable", "Este enlace de cotización fue revocado.")
	case errors.Is(err, ErrPortalExpired):
		webutil.WriteError(w, r, http.StatusGone, "quote_portal_expired", "La vigencia de esta cotización terminó. Contacta a la empresa para solicitar una actualización.")
	case errors.Is(err, ErrRejectionDisabled):
		webutil.WriteError(w, r, http.StatusForbidden, "quote_rejection_disabled", "Esta cotización no permite rechazo desde el portal.")
	case errors.Is(err, ErrInvalidQuoteStatus), errors.Is(err, ErrInvalidPortalStatus):
		webutil.WriteError(w, r, http.StatusConflict, "quote_portal_decision_conflict", "La cotización ya no puede responderse desde este enlace.")
	case err != nil:
		webutil.WriteError(w, r, http.StatusInternalServerError, "quote_portal_decision_failed", "No fue posible registrar la respuesta.")
	default:
		return false
	}
	return true
}

func quotePathID(w http.ResponseWriter, r *http.Request) (string, bool) {
	value := strings.TrimSpace(r.PathValue("quoteID"))
	if !idutil.IsUUID(value) {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_quote_id", "Quote ID is invalid.")
		return "", false
	}
	return value, true
}

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}
