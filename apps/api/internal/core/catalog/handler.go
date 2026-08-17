package catalog

import (
	"errors"
	"net/http"
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

func (h *Handler) ListCategories(w http.ResponseWriter, r *http.Request) {
	items, err := h.repository.ListCategories(r.Context(), webutil.TenantID(r.Context()))
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "category_list_failed", "Could not load categories.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var input CreateCategoryInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	item, fields, err := h.service.CreateCategory(r.Context(), webutil.TenantID(r.Context()), input)
	if len(fields) > 0 {
		webutil.WriteValidationError(w, r, fields)
		return
	}
	if errors.Is(err, ErrConflict) {
		webutil.WriteError(w, r, http.StatusConflict, "category_exists", "A category with that name already exists.")
		return
	}
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "category_create_failed", "Could not create the category.")
		return
	}
	webutil.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	categoryID := r.PathValue("categoryID")
	if !idutil.IsUUID(categoryID) {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_category_id", "Category ID is invalid.")
		return
	}
	err := h.service.DeleteCategory(r.Context(), webutil.TenantID(r.Context()), categoryID)
	if errors.Is(err, ErrNotFound) {
		webutil.WriteError(w, r, http.StatusNotFound, "category_not_found", "Category not found.")
		return
	}
	if errors.Is(err, ErrConflict) {
		webutil.WriteError(w, r, http.StatusConflict, "category_in_use", "Move the category's resources before deleting it.")
		return
	}
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "category_delete_failed", "Could not delete the category.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListResources(w http.ResponseWriter, r *http.Request) {
	activeOnly := strings.EqualFold(r.URL.Query().Get("active"), "true")
	categoryID := strings.TrimSpace(r.URL.Query().Get("category_id"))
	if categoryID != "" && !idutil.IsUUID(categoryID) {
		webutil.WriteValidationError(w, r, map[string]string{"category_id": "Category ID is invalid."})
		return
	}
	items, err := h.repository.ListResources(
		r.Context(),
		webutil.TenantID(r.Context()),
		strings.TrimSpace(r.URL.Query().Get("q")),
		categoryID,
		activeOnly,
	)
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "resource_list_failed", "Could not load inventory resources.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) GetResource(w http.ResponseWriter, r *http.Request) {
	resourceID, ok := validPathUUID(w, r, "resourceID", "resource")
	if !ok {
		return
	}
	item, err := h.repository.GetResource(r.Context(), webutil.TenantID(r.Context()), resourceID)
	if errors.Is(err, ErrNotFound) {
		webutil.WriteError(w, r, http.StatusNotFound, "resource_not_found", "Resource not found.")
		return
	}
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "resource_load_failed", "Could not load the resource.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) CreateResource(w http.ResponseWriter, r *http.Request) {
	var input CreateResourceInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	item, fields, err := h.service.CreateResource(r.Context(), webutil.TenantID(r.Context()), input)
	if len(fields) > 0 {
		webutil.WriteValidationError(w, r, fields)
		return
	}
	h.writeResourceMutation(w, r, item, err, http.StatusCreated)
}

func (h *Handler) UpdateResource(w http.ResponseWriter, r *http.Request) {
	resourceID, ok := validPathUUID(w, r, "resourceID", "resource")
	if !ok {
		return
	}
	var input UpdateResourceInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	item, fields, err := h.service.UpdateResource(r.Context(), webutil.TenantID(r.Context()), resourceID, input)
	if len(fields) > 0 {
		webutil.WriteValidationError(w, r, fields)
		return
	}
	h.writeResourceMutation(w, r, item, err, http.StatusOK)
}

func (h *Handler) ArchiveResource(w http.ResponseWriter, r *http.Request) {
	resourceID, ok := validPathUUID(w, r, "resourceID", "resource")
	if !ok {
		return
	}
	active := false
	item, _, err := h.service.UpdateResource(
		r.Context(),
		webutil.TenantID(r.Context()),
		resourceID,
		UpdateResourceInput{Active: &active},
	)
	h.writeResourceMutation(w, r, item, err, http.StatusOK)
}

func (h *Handler) writeResourceMutation(w http.ResponseWriter, r *http.Request, item Resource, err error, status int) {
	if errors.Is(err, ErrNotFound) {
		webutil.WriteError(w, r, http.StatusNotFound, "resource_not_found", "Resource or category not found.")
		return
	}
	if errors.Is(err, ErrConflict) {
		webutil.WriteError(w, r, http.StatusConflict, "resource_conflict", "The SKU is already in use.")
		return
	}
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "resource_mutation_failed", "Could not save the resource.")
		return
	}
	webutil.WriteJSON(w, status, item)
}

func validPathUUID(w http.ResponseWriter, r *http.Request, pathKey, entity string) (string, bool) {
	value := r.PathValue(pathKey)
	if !idutil.IsUUID(value) {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_"+entity+"_id", "The supplied identifier is invalid.")
		return "", false
	}
	return value, true
}
