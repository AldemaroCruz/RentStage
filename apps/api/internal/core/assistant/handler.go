package assistant

import (
	"errors"
	"net/http"

	"github.com/rentstage/rentstage/apps/api/internal/core/quoteportal"
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
	items, err := h.repository.List(r.Context(), webutil.TenantID(r.Context()))
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "assistant_list_failed", "Could not load assistant conversations.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	conversationID, ok := conversationPathID(w, r)
	if !ok {
		return
	}
	item, err := h.repository.Get(r.Context(), webutil.TenantID(r.Context()), conversationID)
	if h.writeFailure(w, r, nil, err) {
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) Simulate(w http.ResponseWriter, r *http.Request) {
	var input SimulateInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	item, fields, err := h.service.Simulate(r.Context(), webutil.TenantID(r.Context()), input)
	if h.writeFailure(w, r, fields, err) {
		return
	}
	webutil.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	conversationID, ok := conversationPathID(w, r)
	if !ok {
		return
	}
	var input ApproveInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	item, fields, err := h.service.Approve(
		r.Context(), webutil.TenantID(r.Context()), conversationID, input,
	)
	if h.writeFailure(w, r, fields, err) {
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) LinkCustomer(w http.ResponseWriter, r *http.Request) {
	conversationID, ok := conversationPathID(w, r)
	if !ok {
		return
	}
	var input LinkCustomerInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	item, fields, err := h.service.LinkCustomer(
		r.Context(), webutil.TenantID(r.Context()), conversationID, input,
	)
	if h.writeFailure(w, r, fields, err) {
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) SendDemo(w http.ResponseWriter, r *http.Request) {
	conversationID, ok := conversationPathID(w, r)
	if !ok {
		return
	}
	var input SendDemoInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	item, fields, err := h.service.SendDemo(
		r.Context(), webutil.TenantID(r.Context()), conversationID, input,
	)
	if h.writeFailure(w, r, fields, err) {
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) ReceiveDemo(w http.ResponseWriter, r *http.Request) {
	conversationID, ok := conversationPathID(w, r)
	if !ok {
		return
	}
	var input ReceiveDemoInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	item, fields, err := h.service.ReceiveDemo(
		r.Context(), webutil.TenantID(r.Context()), conversationID, input,
	)
	if h.writeFailure(w, r, fields, err) {
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) ShareQuoteDemo(w http.ResponseWriter, r *http.Request) {
	conversationID, ok := conversationPathID(w, r)
	if !ok {
		return
	}
	var input ShareQuoteDemoInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	item, fields, err := h.service.ShareQuoteDemo(
		r.Context(), webutil.TenantID(r.Context()), conversationID, input,
	)
	if h.writeFailure(w, r, fields, err) {
		return
	}
	// A newly issued portal URL contains a raw bearer token. It is returned
	// exactly once and must never be cached by browsers or intermediaries.
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Referrer-Policy", "no-referrer")
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) writeFailure(w http.ResponseWriter, r *http.Request, fields map[string]string, err error) bool {
	if len(fields) > 0 {
		webutil.WriteValidationError(w, r, fields)
		return true
	}
	switch {
	case errors.Is(err, ErrNotFound):
		webutil.WriteError(w, r, http.StatusNotFound, "assistant_conversation_not_found", "Assistant conversation not found.")
	case errors.Is(err, ErrNoReadyPackage):
		webutil.WriteError(w, r, http.StatusConflict, "assistant_package_missing", "Create at least one ready commercial package before simulating a conversation.")
	case errors.Is(err, ErrUnavailable):
		webutil.WriteError(w, r, http.StatusConflict, "assistant_package_unavailable", "Availability changed. Review the package before creating a quote draft.")
	case errors.Is(err, ErrAlreadyApproved):
		webutil.WriteError(w, r, http.StatusConflict, "assistant_already_approved", "This proposal already created a quote draft.")
	case errors.Is(err, ErrCustomerMissing):
		webutil.WriteError(w, r, http.StatusUnprocessableEntity, "assistant_customer_missing", "Select an existing customer for the quote draft.")
	case errors.Is(err, ErrDemoOnly):
		webutil.WriteError(w, r, http.StatusConflict, "assistant_demo_only", "This operation is available only in the simulated demo channel.")
	case errors.Is(err, ErrMessageNotReady):
		webutil.WriteError(w, r, http.StatusConflict, "assistant_message_not_ready", "The selected message was already sent or is no longer awaiting approval.")
	case errors.Is(err, ErrQuoteMissing):
		webutil.WriteError(w, r, http.StatusConflict, "assistant_quote_missing", "Create the quote draft before sharing a customer portal.")
	case errors.Is(err, ErrPortalDeliveryMissing):
		webutil.WriteError(w, r, http.StatusInternalServerError, "assistant_portal_delivery_missing", "The secure customer link was not returned.")
	case errors.Is(err, quoteportal.ErrPortalDisabled):
		webutil.WriteError(w, r, http.StatusConflict, "quote_portal_disabled", "Enable the Quote Portal for this workspace before sharing the quote.")
	case errors.Is(err, quoteportal.ErrQuoteNotFound):
		webutil.WriteError(w, r, http.StatusNotFound, "quote_not_found", "The quote linked to this conversation was not found.")
	case errors.Is(err, quoteportal.ErrInvalidQuoteStatus), errors.Is(err, quoteportal.ErrInvalidPortalStatus):
		webutil.WriteError(w, r, http.StatusConflict, "invalid_quote_portal_status", "The quote or its portal can no longer be shared from this state.")
	case err != nil:
		webutil.WriteError(w, r, http.StatusInternalServerError, "assistant_operation_failed", "Could not complete the assistant operation.")
	default:
		return false
	}
	return true
}

func conversationPathID(w http.ResponseWriter, r *http.Request) (string, bool) {
	value := r.PathValue("conversationID")
	if !idutil.IsUUID(value) {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_conversation_id", "Conversation ID is invalid.")
		return "", false
	}
	return value, true
}
