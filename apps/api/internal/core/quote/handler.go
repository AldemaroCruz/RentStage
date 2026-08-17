package quote

import (
	"context"
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

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	status := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
	if status != "" && !validStatus(status) {
		webutil.WriteValidationError(w, r, map[string]string{"status": "Unsupported quote status."})
		return
	}
	customerID := strings.TrimSpace(r.URL.Query().Get("customer_id"))
	if customerID != "" && !idutil.IsUUID(customerID) {
		webutil.WriteValidationError(w, r, map[string]string{"customer_id": "Customer ID is invalid."})
		return
	}
	items, err := h.repository.List(
		r.Context(),
		webutil.TenantID(r.Context()),
		strings.TrimSpace(r.URL.Query().Get("q")),
		status,
		customerID,
	)
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "quote_list_failed", "Could not load quotes.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	quoteID, ok := pathID(w, r)
	if !ok {
		return
	}
	item, err := h.repository.Get(r.Context(), webutil.TenantID(r.Context()), quoteID)
	if errors.Is(err, ErrNotFound) {
		webutil.WriteError(w, r, http.StatusNotFound, "quote_not_found", "Quote not found.")
		return
	}
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "quote_load_failed", "Could not load the quote.")
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
	h.writeMutation(w, r, item, fields, err, http.StatusCreated)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	quoteID, ok := pathID(w, r)
	if !ok {
		return
	}
	var input CreateInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	item, fields, err := h.service.Update(r.Context(), webutil.TenantID(r.Context()), quoteID, input)
	h.writeMutation(w, r, item, fields, err, http.StatusOK)
}

func (h *Handler) Send(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.service.Send)
}

func (h *Handler) Accept(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.service.Accept)
}

func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.service.Reject)
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.service.Cancel)
}

func (h *Handler) transition(
	w http.ResponseWriter,
	r *http.Request,
	action func(context.Context, string, string) (Detail, error),
) {
	quoteID, ok := pathID(w, r)
	if !ok {
		return
	}
	item, err := action(r.Context(), webutil.TenantID(r.Context()), quoteID)
	h.writeMutation(w, r, item, nil, err, http.StatusOK)
}

func (h *Handler) writeMutation(
	w http.ResponseWriter,
	r *http.Request,
	item Detail,
	fields map[string]string,
	err error,
	status int,
) {
	if len(fields) > 0 {
		webutil.WriteValidationError(w, r, fields)
		return
	}
	switch {
	case errors.Is(err, ErrNotFound):
		webutil.WriteError(w, r, http.StatusNotFound, "quote_not_found", "Quote not found.")
	case errors.Is(err, ErrCustomerNotFound):
		webutil.WriteError(w, r, http.StatusUnprocessableEntity, "quote_customer_not_found", "The selected customer does not exist.")
	case errors.Is(err, ErrResourceNotFound):
		webutil.WriteError(w, r, http.StatusUnprocessableEntity, "quote_resource_not_found", "One or more selected resources do not exist or are archived.")
	case errors.Is(err, ErrImmutable):
		webutil.WriteError(w, r, http.StatusConflict, "quote_immutable", "Only draft quotes can be edited.")
	case errors.Is(err, ErrInvalidTransition):
		webutil.WriteError(w, r, http.StatusConflict, "invalid_quote_transition", "The quote cannot move to that status from its current state.")
	case err != nil:
		webutil.WriteError(w, r, http.StatusInternalServerError, "quote_mutation_failed", "Could not save the quote.")
	default:
		webutil.WriteJSON(w, status, item)
	}
}

func pathID(w http.ResponseWriter, r *http.Request) (string, bool) {
	value := r.PathValue("quoteID")
	if !idutil.IsUUID(value) {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_quote_id", "Quote ID is invalid.")
		return "", false
	}
	return value, true
}

func validStatus(value string) bool {
	for _, status := range []string{"DRAFT", "SENT", "ACCEPTED", "REJECTED", "EXPIRED", "CANCELLED"} {
		if value == status {
			return true
		}
	}
	return false
}
