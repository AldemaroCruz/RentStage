package inventory

import (
	"errors"
	"net/http"

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

func (h *Handler) ListAssets(w http.ResponseWriter, r *http.Request) {
	resourceID, ok := pathUUID(w, r, "resourceID", "invalid_resource_id")
	if !ok {
		return
	}
	items, err := h.repository.ListAssets(r.Context(), webutil.TenantID(r.Context()), resourceID)
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "asset_list_failed", "Could not load physical assets.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) CreateAsset(w http.ResponseWriter, r *http.Request) {
	resourceID, ok := pathUUID(w, r, "resourceID", "invalid_resource_id")
	if !ok {
		return
	}
	var input CreateAssetInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	item, fields, err := h.service.CreateAsset(r.Context(), webutil.TenantID(r.Context()), resourceID, input)
	if len(fields) > 0 {
		webutil.WriteValidationError(w, r, fields)
		return
	}
	h.writeMutation(w, r, item, err, http.StatusCreated)
}

func (h *Handler) UpdateAsset(w http.ResponseWriter, r *http.Request) {
	assetID, ok := pathUUID(w, r, "assetID", "invalid_asset_id")
	if !ok {
		return
	}
	var input UpdateAssetInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	item, fields, err := h.service.UpdateAsset(r.Context(), webutil.TenantID(r.Context()), assetID, input)
	if len(fields) > 0 {
		webutil.WriteValidationError(w, r, fields)
		return
	}
	h.writeMutation(w, r, item, err, http.StatusOK)
}

func (h *Handler) RetireAsset(w http.ResponseWriter, r *http.Request) {
	assetID, ok := pathUUID(w, r, "assetID", "invalid_asset_id")
	if !ok {
		return
	}
	item, err := h.service.RetireAsset(r.Context(), webutil.TenantID(r.Context()), assetID)
	h.writeMutation(w, r, item, err, http.StatusOK)
}

func (h *Handler) writeMutation(w http.ResponseWriter, r *http.Request, item Asset, err error, status int) {
	if errors.Is(err, ErrNotFound) {
		webutil.WriteError(w, r, http.StatusNotFound, "asset_not_found", "Asset or resource not found.")
		return
	}
	if errors.Is(err, ErrConflict) {
		webutil.WriteError(w, r, http.StatusConflict, "asset_conflict", "The asset code or serial number is already in use.")
		return
	}
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "asset_mutation_failed", "Could not save the physical asset.")
		return
	}
	webutil.WriteJSON(w, status, item)
}

func pathUUID(w http.ResponseWriter, r *http.Request, pathKey, errorCode string) (string, bool) {
	value := r.PathValue(pathKey)
	if !idutil.IsUUID(value) {
		webutil.WriteError(w, r, http.StatusBadRequest, errorCode, "The supplied identifier is invalid.")
		return "", false
	}
	return value, true
}
