package availability

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/rentstage/rentstage/apps/api/internal/idutil"
	"github.com/rentstage/rentstage/apps/api/internal/webutil"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Check(w http.ResponseWriter, r *http.Request) {
	var input CheckInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	result, fields, err := h.service.Check(r.Context(), webutil.TenantID(r.Context()), input)
	if h.writeFailure(w, r, fields, err) {
		return
	}
	webutil.WriteJSON(w, http.StatusOK, result)
}

// Get preserves the original flat single-resource response contract while
// delegating calculations to the same bulk availability engine.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	resourceID := strings.TrimSpace(r.PathValue("resourceID"))
	if !idutil.IsUUID(resourceID) {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_resource_id", "Resource ID is invalid.")
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
	input := CheckInput{
		StartAt: r.URL.Query().Get("start"),
		EndAt:   r.URL.Query().Get("end"),
		Items:   []ItemInput{{ResourceID: resourceID, Quantity: quantity}},
	}
	result, fields, err := h.service.Check(r.Context(), webutil.TenantID(r.Context()), input)
	if h.writeFailure(w, r, fields, err) {
		return
	}
	item := result.Items[0]
	webutil.WriteJSON(w, http.StatusOK, SingleResult{
		ResourceID:        item.ResourceID,
		ResourceName:      item.ResourceName,
		Start:             result.StartAt,
		End:               result.EndAt,
		RequestedQuantity: item.RequestedQuantity,
		EligibleAssets:    item.EligibleAssets,
		ReservedQuantity:  item.ReservedQuantity,
		AvailableQuantity: item.AvailableQuantity,
		CanFulfill:        item.CanFulfill,
	})
}

func (h *Handler) writeFailure(
	w http.ResponseWriter,
	r *http.Request,
	fields map[string]string,
	err error,
) bool {
	if len(fields) > 0 {
		webutil.WriteValidationError(w, r, fields)
		return true
	}
	var notFound *ResourceNotFoundError
	if errors.As(err, &notFound) {
		webutil.WriteError(w, r, http.StatusNotFound, "resource_not_found", "One of the requested resources was not found or is archived.")
		return true
	}
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "availability_failed", "Could not calculate availability.")
		return true
	}
	return false
}
