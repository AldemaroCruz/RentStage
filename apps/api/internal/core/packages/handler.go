package packages

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/rentstage/rentstage/apps/api/internal/core/availability"
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

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("active")))
	if active != "" && active != "all" && active != "true" && active != "false" {
		webutil.WriteValidationError(w, r, map[string]string{"active": "Use all, true, or false."})
		return
	}
	items, err := h.repository.List(
		r.Context(),
		webutil.TenantID(r.Context()),
		strings.TrimSpace(r.URL.Query().Get("q")),
		active,
	)
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "package_list_failed", "Could not load packages.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	packageID, ok := pathID(w, r)
	if !ok {
		return
	}
	item, err := h.service.Get(r.Context(), webutil.TenantID(r.Context()), packageID)
	if h.writeFailure(w, r, nil, err, "package_load_failed", "Could not load the package.") {
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var input CreateInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	item, fields, err := h.service.Create(r.Context(), webutil.TenantID(r.Context()), input)
	if h.writeFailure(w, r, fields, err, "package_mutation_failed", "Could not save the package.") {
		return
	}
	webutil.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	packageID, ok := pathID(w, r)
	if !ok {
		return
	}
	var input CreateInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	item, fields, err := h.service.Update(r.Context(), webutil.TenantID(r.Context()), packageID, input)
	if h.writeFailure(w, r, fields, err, "package_mutation_failed", "Could not save the package.") {
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	packageID, ok := pathID(w, r)
	if !ok {
		return
	}
	item, err := h.service.Archive(r.Context(), webutil.TenantID(r.Context()), packageID)
	if h.writeFailure(w, r, nil, err, "package_archive_failed", "Could not archive the package.") {
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) QuoteTemplate(w http.ResponseWriter, r *http.Request) {
	packageID, ok := pathID(w, r)
	if !ok {
		return
	}
	quantity := 1
	if raw := strings.TrimSpace(r.URL.Query().Get("quantity")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			webutil.WriteValidationError(w, r, map[string]string{"quantity": "Quantity must be a whole number."})
			return
		}
		quantity = parsed
	}
	item, fields, err := h.service.QuoteTemplate(r.Context(), webutil.TenantID(r.Context()), packageID, quantity)
	if h.writeFailure(w, r, fields, err, "package_template_failed", "Could not prepare package quote items.") {
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) Availability(w http.ResponseWriter, r *http.Request) {
	packageID, ok := pathID(w, r)
	if !ok {
		return
	}
	var input AvailabilityInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	item, fields, err := h.service.Availability(r.Context(), webutil.TenantID(r.Context()), packageID, input)
	if h.writeFailure(w, r, fields, err, "package_availability_failed", "Could not calculate package availability.") {
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
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
	var availabilityNotFound *availability.ResourceNotFoundError
	switch {
	case errors.Is(err, ErrNotFound):
		webutil.WriteError(w, r, http.StatusNotFound, "package_not_found", "Package not found.")
	case errors.Is(err, ErrConflict):
		webutil.WriteError(w, r, http.StatusConflict, "package_conflict", "The package slug or one of its resources is duplicated.")
	case errors.Is(err, ErrResourceNotFound):
		webutil.WriteError(w, r, http.StatusUnprocessableEntity, "package_resource_not_found", "One or more resources do not exist or are archived.")
	case errors.Is(err, ErrUnavailable):
		webutil.WriteError(w, r, http.StatusConflict, "package_unavailable", "The package is archived, empty, or contains archived resources.")
	case errors.As(err, &availabilityNotFound):
		webutil.WriteError(w, r, http.StatusNotFound, "resource_not_found", "One of the package resources was not found or is archived.")
	case err != nil:
		webutil.WriteError(w, r, http.StatusInternalServerError, fallbackCode, fallbackMessage)
	default:
		return false
	}
	return true
}

func pathID(w http.ResponseWriter, r *http.Request) (string, bool) {
	value := r.PathValue("packageID")
	if !idutil.IsUUID(value) {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_package_id", "Package ID is invalid.")
		return "", false
	}
	return value, true
}
