package publiccatalog

import (
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/rentstage/rentstage/apps/api/internal/core/availability"
	"github.com/rentstage/rentstage/apps/api/internal/core/packages"
	"github.com/rentstage/rentstage/apps/api/internal/idutil"
	"github.com/rentstage/rentstage/apps/api/internal/webutil"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) PublicCatalog(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.Catalog(r.Context(), r.PathValue("tenantSlug"))
	if h.writeFailure(w, r, nil, err, "public_catalog_load_failed", "Could not load the public catalog.") {
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) PublicPackage(w http.ResponseWriter, r *http.Request) {
	tenant, settings, item, err := h.service.Package(r.Context(), r.PathValue("tenantSlug"), r.PathValue("packageSlug"))
	if h.writeFailure(w, r, nil, err, "public_package_load_failed", "Could not load the package.") {
		return
	}
	webutil.WriteJSON(w, http.StatusOK, map[string]any{"tenant": tenant, "settings": settings, "package": item})
}

func (h *Handler) PublicResource(w http.ResponseWriter, r *http.Request) {
	tenant, settings, item, err := h.service.Resource(r.Context(), r.PathValue("tenantSlug"), r.PathValue("resourceSlug"))
	if h.writeFailure(w, r, nil, err, "public_resource_load_failed", "Could not load the resource.") {
		return
	}
	webutil.WriteJSON(w, http.StatusOK, map[string]any{"tenant": tenant, "settings": settings, "resource": item})
}

func (h *Handler) PublicAvailability(w http.ResponseWriter, r *http.Request) {
	var input AvailabilityInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	item, fields, err := h.service.Availability(r.Context(), r.PathValue("tenantSlug"), input)
	if h.writeFailure(w, r, fields, err, "public_availability_failed", "Could not calculate availability.") {
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) SubmitQuoteRequest(w http.ResponseWriter, r *http.Request) {
	var input QuoteRequestInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	item, fields, err := h.service.SubmitQuoteRequest(
		r.Context(), r.PathValue("tenantSlug"), input, clientIP(r), r.UserAgent(),
	)
	if h.writeFailure(w, r, fields, err, "quote_request_create_failed", "Could not submit the quote request.") {
		return
	}
	webutil.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) AdminCatalog(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.AdminCatalog(r.Context(), webutil.TenantID(r.Context()))
	if h.writeFailure(w, r, nil, err, "public_catalog_admin_load_failed", "Could not load public catalog settings.") {
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
	if h.writeFailure(w, r, fields, err, "public_catalog_update_failed", "Could not update public catalog settings.") {
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) UpdatePackagePublication(w http.ResponseWriter, r *http.Request) {
	packageID := r.PathValue("packageID")
	if !idutil.IsUUID(packageID) {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_package_id", "Package ID is invalid.")
		return
	}
	var input PackagePublicationInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	item, fields, err := h.service.UpdatePackagePublication(r.Context(), webutil.TenantID(r.Context()), packageID, input)
	if h.writeFailure(w, r, fields, err, "package_publication_failed", "Could not update package publication.") {
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) UpdateResourcePublication(w http.ResponseWriter, r *http.Request) {
	resourceID := r.PathValue("resourceID")
	if !idutil.IsUUID(resourceID) {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_resource_id", "Resource ID is invalid.")
		return
	}
	var input ResourcePublicationInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	item, fields, err := h.service.UpdateResourcePublication(r.Context(), webutil.TenantID(r.Context()), resourceID, input)
	if h.writeFailure(w, r, fields, err, "resource_publication_failed", "Could not update resource publication.") {
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) ListQuoteRequests(w http.ResponseWriter, r *http.Request) {
	item, fields, err := h.service.ListQuoteRequests(
		r.Context(), webutil.TenantID(r.Context()), r.URL.Query().Get("q"), r.URL.Query().Get("status"),
	)
	if h.writeFailure(w, r, fields, err, "quote_request_list_failed", "Could not load quote requests.") {
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) GetQuoteRequest(w http.ResponseWriter, r *http.Request) {
	requestID, ok := requestPathID(w, r)
	if !ok {
		return
	}
	item, err := h.service.GetQuoteRequest(r.Context(), webutil.TenantID(r.Context()), requestID)
	if h.writeFailure(w, r, nil, err, "quote_request_load_failed", "Could not load the quote request.") {
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) UpdateQuoteRequestStatus(w http.ResponseWriter, r *http.Request) {
	requestID, ok := requestPathID(w, r)
	if !ok {
		return
	}
	var input QuoteRequestStatusInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	item, fields, err := h.service.UpdateQuoteRequestStatus(
		r.Context(), webutil.TenantID(r.Context()), requestID, webutil.ActorID(r.Context()), input,
	)
	if h.writeFailure(w, r, fields, err, "quote_request_status_failed", "Could not update the quote request.") {
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) ConvertQuoteRequest(w http.ResponseWriter, r *http.Request) {
	requestID, ok := requestPathID(w, r)
	if !ok {
		return
	}
	item, err := h.service.ConvertQuoteRequest(
		r.Context(), webutil.TenantID(r.Context()), requestID, webutil.ActorID(r.Context()),
	)
	if h.writeFailure(w, r, nil, err, "quote_request_conversion_failed", "Could not convert the quote request.") {
		return
	}
	webutil.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) writeFailure(
	w http.ResponseWriter,
	r *http.Request,
	fields map[string]string,
	err error,
	fallbackCode string,
	fallbackMessage string,
) bool {
	if len(fields) > 0 {
		webutil.WriteValidationError(w, r, fields)
		return true
	}
	var resourceNotFound *availability.ResourceNotFoundError
	switch {
	case errors.Is(err, ErrCatalogNotFound):
		webutil.WriteError(w, r, http.StatusNotFound, "public_catalog_not_found", "Public catalog not found.")
	case errors.Is(err, ErrPackageNotPublic), errors.Is(err, packages.ErrNotFound):
		webutil.WriteError(w, r, http.StatusNotFound, "public_package_not_found", "Package not found in this public catalog.")
	case errors.Is(err, ErrResourceNotPublic):
		webutil.WriteError(w, r, http.StatusNotFound, "public_resource_not_found", "Resource not found in this public catalog.")
	case errors.Is(err, ErrQuoteRequestsDisabled):
		webutil.WriteError(w, r, http.StatusForbidden, "quote_requests_disabled", "This catalog is not accepting quote requests.")
	case errors.Is(err, ErrQuoteRequestNotFound):
		webutil.WriteError(w, r, http.StatusNotFound, "quote_request_not_found", "Quote request not found.")
	case errors.Is(err, ErrQuoteRequestRateLimited):
		w.Header().Set("Retry-After", "3600")
		webutil.WriteError(w, r, http.StatusTooManyRequests, "quote_request_rate_limited", "Too many requests were submitted. Try again later.")
	case errors.Is(err, ErrPublicationConflict):
		webutil.WriteError(w, r, http.StatusConflict, "public_slug_conflict", "That public URL is already used by another resource.")
	case errors.Is(err, ErrQuoteRequestConflict), errors.Is(err, ErrConversionConflict):
		webutil.WriteError(w, r, http.StatusConflict, "quote_request_conflict", "The quote request cannot be changed in its current state.")
	case errors.As(err, &resourceNotFound):
		webutil.WriteError(w, r, http.StatusConflict, "public_package_unavailable", "A package component is no longer available.")
	case err != nil:
		webutil.WriteError(w, r, http.StatusInternalServerError, fallbackCode, fallbackMessage)
	default:
		return false
	}
	return true
}

func requestPathID(w http.ResponseWriter, r *http.Request) (string, bool) {
	value := r.PathValue("requestID")
	if !idutil.IsUUID(value) {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_quote_request_id", "Quote request ID is invalid.")
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
