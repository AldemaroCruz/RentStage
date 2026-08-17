package customer

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

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	source := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("source")))
	if source != "" && !contains([]string{"WEB", "WHATSAPP", "MANUAL", "IMPORT"}, source) {
		webutil.WriteValidationError(w, r, map[string]string{"source": "Unsupported customer source."})
		return
	}
	items, err := h.repository.List(
		r.Context(),
		webutil.TenantID(r.Context()),
		strings.TrimSpace(r.URL.Query().Get("q")),
		source,
	)
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "customer_list_failed", "Could not load customers.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	customerID, ok := pathID(w, r)
	if !ok {
		return
	}
	item, err := h.repository.Get(r.Context(), webutil.TenantID(r.Context()), customerID)
	if errors.Is(err, ErrNotFound) {
		webutil.WriteError(w, r, http.StatusNotFound, "customer_not_found", "Customer not found.")
		return
	}
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "customer_load_failed", "Could not load the customer.")
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
	customerID, ok := pathID(w, r)
	if !ok {
		return
	}
	var input UpdateInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	item, fields, err := h.service.Update(r.Context(), webutil.TenantID(r.Context()), customerID, input)
	h.writeMutation(w, r, item, fields, err, http.StatusOK)
}

func (h *Handler) writeMutation(
	w http.ResponseWriter,
	r *http.Request,
	item Customer,
	fields map[string]string,
	err error,
	status int,
) {
	if len(fields) > 0 {
		webutil.WriteValidationError(w, r, fields)
		return
	}
	if errors.Is(err, ErrNotFound) {
		webutil.WriteError(w, r, http.StatusNotFound, "customer_not_found", "Customer not found.")
		return
	}
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "customer_mutation_failed", "Could not save the customer.")
		return
	}
	webutil.WriteJSON(w, status, item)
}

func pathID(w http.ResponseWriter, r *http.Request) (string, bool) {
	value := r.PathValue("customerID")
	if !idutil.IsUUID(value) {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_customer_id", "Customer ID is invalid.")
		return "", false
	}
	return value, true
}
