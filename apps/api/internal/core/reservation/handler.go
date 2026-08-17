package reservation

import (
	"context"
	"errors"
	"net/http"
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
	status := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
	if status != "" && !validStatus(status) {
		webutil.WriteValidationError(w, r, map[string]string{"status": "Unsupported reservation status."})
		return
	}
	items, err := h.repository.List(
		r.Context(),
		webutil.TenantID(r.Context()),
		strings.TrimSpace(r.URL.Query().Get("q")),
		status,
	)
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "reservation_list_failed", "Could not load reservations.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var input CreateInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	item, fields, err := h.service.CreateManual(r.Context(), webutil.TenantID(r.Context()), input)
	if len(fields) > 0 {
		webutil.WriteValidationError(w, r, fields)
		return
	}
	var conflict *AvailabilityConflictError
	var resourceNotFound *availability.ResourceNotFoundError
	switch {
	case errors.Is(err, ErrCustomerNotFound):
		webutil.WriteError(w, r, http.StatusUnprocessableEntity, "reservation_customer_not_found", "The selected customer does not exist.")
	case errors.As(err, &resourceNotFound):
		webutil.WriteError(w, r, http.StatusUnprocessableEntity, "reservation_resource_not_found", "One of the selected resources does not exist or is archived.")
	case errors.As(err, &conflict):
		webutil.WriteJSON(w, http.StatusConflict, map[string]any{
			"error":        "availability_conflict",
			"message":      "The manual reservation cannot be created because one or more resources are unavailable.",
			"request_id":   webutil.RequestID(r.Context()),
			"availability": conflict.Result,
		})
	case err != nil:
		webutil.WriteError(w, r, http.StatusInternalServerError, "reservation_create_failed", "Could not create the manual reservation.")
	default:
		webutil.WriteJSON(w, http.StatusCreated, item)
	}
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	reservationID, ok := reservationPathID(w, r)
	if !ok {
		return
	}
	item, err := h.repository.Get(r.Context(), webutil.TenantID(r.Context()), reservationID)
	if errors.Is(err, ErrNotFound) {
		webutil.WriteError(w, r, http.StatusNotFound, "reservation_not_found", "Reservation not found.")
		return
	}
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "reservation_load_failed", "Could not load the reservation.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) Reschedule(w http.ResponseWriter, r *http.Request) {
	reservationID, ok := reservationPathID(w, r)
	if !ok {
		return
	}
	var input RescheduleInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	item, fields, err := h.service.Reschedule(r.Context(), webutil.TenantID(r.Context()), reservationID, input)
	if len(fields) > 0 {
		webutil.WriteValidationError(w, r, fields)
		return
	}
	var conflict *AvailabilityConflictError
	var assetConflict *AssetScheduleConflictError
	switch {
	case errors.Is(err, ErrNotFound):
		webutil.WriteError(w, r, http.StatusNotFound, "reservation_not_found", "Reservation not found.")
	case errors.Is(err, ErrRescheduleState):
		webutil.WriteError(w, r, http.StatusConflict, "reservation_reschedule_state", "The reservation cannot be rescheduled from its current state.")
	case errors.As(err, &conflict):
		webutil.WriteJSON(w, http.StatusConflict, map[string]any{
			"error":        "availability_conflict",
			"message":      "The reservation cannot be moved because one or more resources are unavailable in the requested period.",
			"request_id":   webutil.RequestID(r.Context()),
			"availability": conflict.Result,
		})
	case errors.As(err, &assetConflict):
		webutil.WriteJSON(w, http.StatusConflict, map[string]any{
			"error":                       "asset_schedule_conflict",
			"message":                     "An assigned physical unit is already committed to another reservation in the requested period.",
			"request_id":                  webutil.RequestID(r.Context()),
			"asset_id":                    assetConflict.AssetID,
			"asset_code":                  assetConflict.AssetCode,
			"conflict_reservation_id":     assetConflict.ReservationID,
			"conflict_reservation_number": assetConflict.ReservationNumber,
		})
	case err != nil:
		webutil.WriteError(w, r, http.StatusInternalServerError, "reservation_reschedule_failed", "Could not reschedule the reservation.")
	default:
		webutil.WriteJSON(w, http.StatusOK, item)
	}
}

func (h *Handler) Warehouse(w http.ResponseWriter, r *http.Request) {
	reservationID, ok := reservationPathID(w, r)
	if !ok {
		return
	}
	item, err := h.repository.GetWarehouseInventory(r.Context(), webutil.TenantID(r.Context()), reservationID)
	if errors.Is(err, ErrNotFound) {
		webutil.WriteError(w, r, http.StatusNotFound, "reservation_not_found", "Reservation not found.")
		return
	}
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "warehouse_inventory_load_failed", "Could not load warehouse preparation data.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) ConvertQuote(w http.ResponseWriter, r *http.Request) {
	quoteID := strings.TrimSpace(r.PathValue("quoteID"))
	if !idutil.IsUUID(quoteID) {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_quote_id", "Quote ID is invalid.")
		return
	}
	item, err := h.service.ConvertQuote(r.Context(), webutil.TenantID(r.Context()), quoteID)
	if err == nil {
		webutil.WriteJSON(w, http.StatusCreated, item)
		return
	}
	var conflict *AvailabilityConflictError
	if errors.As(err, &conflict) {
		webutil.WriteJSON(w, http.StatusConflict, map[string]any{
			"error":        "availability_conflict",
			"message":      "The accepted quote cannot be reserved because one or more resources are no longer available.",
			"request_id":   webutil.RequestID(r.Context()),
			"availability": conflict.Result,
		})
		return
	}
	var converted *QuoteAlreadyConvertedError
	if errors.As(err, &converted) {
		webutil.WriteJSON(w, http.StatusConflict, map[string]any{
			"error":          "quote_already_converted",
			"message":        "This quote already has a reservation.",
			"request_id":     webutil.RequestID(r.Context()),
			"reservation_id": converted.ReservationID,
		})
		return
	}
	var resourceNotFound *availability.ResourceNotFoundError
	switch {
	case errors.Is(err, ErrQuoteNotFound):
		webutil.WriteError(w, r, http.StatusNotFound, "quote_not_found", "Quote not found.")
	case errors.Is(err, ErrQuoteNotAccepted):
		webutil.WriteError(w, r, http.StatusConflict, "quote_not_accepted", "Only accepted quotes can be converted into reservations.")
	case errors.As(err, &resourceNotFound):
		webutil.WriteError(w, r, http.StatusUnprocessableEntity, "reservation_resource_not_found", "One of the quote resources no longer exists or is archived.")
	default:
		webutil.WriteError(w, r, http.StatusInternalServerError, "quote_conversion_failed", "Could not convert the quote into a reservation.")
	}
}

func (h *Handler) Confirm(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.service.Confirm)
}

func (h *Handler) Prepare(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, h.service.Prepare)
}

func (h *Handler) MarkReady(w http.ResponseWriter, r *http.Request) {
	reservationID, ok := reservationPathID(w, r)
	if !ok {
		return
	}
	item, err := h.service.MarkReady(r.Context(), webutil.TenantID(r.Context()), reservationID)
	var incomplete *InventoryIncompleteError
	switch {
	case errors.Is(err, ErrNotFound):
		webutil.WriteError(w, r, http.StatusNotFound, "reservation_not_found", "Reservation not found.")
	case errors.As(err, &incomplete):
		webutil.WriteJSON(w, http.StatusConflict, map[string]any{
			"error":         "reservation_inventory_incomplete",
			"message":       "Assign every required physical unit before marking the reservation ready.",
			"request_id":    webutil.RequestID(r.Context()),
			"missing_items": incomplete.Items,
		})
	case errors.Is(err, ErrInvalidTransition):
		webutil.WriteError(w, r, http.StatusConflict, "invalid_reservation_transition", "The reservation cannot move to ready from its current state.")
	case err != nil:
		webutil.WriteError(w, r, http.StatusInternalServerError, "reservation_ready_failed", "Could not mark the reservation ready.")
	default:
		webutil.WriteJSON(w, http.StatusOK, item)
	}
}

func (h *Handler) AssignAsset(w http.ResponseWriter, r *http.Request) {
	reservationID, ok := reservationPathID(w, r)
	if !ok {
		return
	}
	var input AssignAssetInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	item, fields, err := h.service.AssignAsset(r.Context(), webutil.TenantID(r.Context()), reservationID, input)
	if len(fields) > 0 {
		webutil.WriteValidationError(w, r, fields)
		return
	}
	var conflict *AssetConflictError
	switch {
	case errors.Is(err, ErrNotFound):
		webutil.WriteError(w, r, http.StatusNotFound, "reservation_not_found", "Reservation not found.")
	case errors.Is(err, ErrAssetNotFound):
		webutil.WriteError(w, r, http.StatusNotFound, "asset_not_found", "Physical asset not found.")
	case errors.Is(err, ErrWarehouseState):
		webutil.WriteError(w, r, http.StatusConflict, "warehouse_state_conflict", "Assets can only be assigned while the reservation is being prepared.")
	case errors.Is(err, ErrAssetUnavailable):
		webutil.WriteError(w, r, http.StatusConflict, "asset_not_available", "This physical asset is not in AVAILABLE condition.")
	case errors.Is(err, ErrAssetResourceMismatch):
		webutil.WriteError(w, r, http.StatusUnprocessableEntity, "asset_resource_mismatch", "This asset does not belong to a resource requested by the reservation.")
	case errors.Is(err, ErrAssetAlreadyAssigned):
		webutil.WriteError(w, r, http.StatusConflict, "asset_already_assigned", "This asset is already assigned to the reservation.")
	case errors.Is(err, ErrAssignmentCapacity):
		webutil.WriteError(w, r, http.StatusConflict, "reservation_item_fully_assigned", "The requested quantity for this resource is already fully assigned.")
	case errors.As(err, &conflict):
		webutil.WriteJSON(w, http.StatusConflict, map[string]any{
			"error":                       "asset_assignment_conflict",
			"message":                     "This asset is already assigned to an overlapping reservation.",
			"request_id":                  webutil.RequestID(r.Context()),
			"asset_id":                    conflict.AssetID,
			"asset_code":                  conflict.AssetCode,
			"conflict_reservation_id":     conflict.ReservationID,
			"conflict_reservation_number": conflict.ReservationNumber,
		})
	case err != nil:
		webutil.WriteError(w, r, http.StatusInternalServerError, "asset_assignment_failed", "Could not assign the physical asset.")
	default:
		webutil.WriteJSON(w, http.StatusCreated, item)
	}
}

func (h *Handler) UnassignAsset(w http.ResponseWriter, r *http.Request) {
	reservationID, ok := reservationPathID(w, r)
	if !ok {
		return
	}
	assetID := strings.TrimSpace(r.PathValue("assetID"))
	if !idutil.IsUUID(assetID) {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_asset_id", "Asset ID is invalid.")
		return
	}
	item, err := h.service.UnassignAsset(r.Context(), webutil.TenantID(r.Context()), reservationID, assetID)
	switch {
	case errors.Is(err, ErrNotFound):
		webutil.WriteError(w, r, http.StatusNotFound, "reservation_not_found", "Reservation not found.")
	case errors.Is(err, ErrAssignmentNotFound):
		webutil.WriteError(w, r, http.StatusNotFound, "asset_assignment_not_found", "Active asset assignment not found.")
	case errors.Is(err, ErrWarehouseState):
		webutil.WriteError(w, r, http.StatusConflict, "warehouse_state_conflict", "Assets can only be unassigned while the reservation is being prepared.")
	case err != nil:
		webutil.WriteError(w, r, http.StatusInternalServerError, "asset_unassignment_failed", "Could not unassign the physical asset.")
	default:
		webutil.WriteJSON(w, http.StatusOK, item)
	}
}

func (h *Handler) CheckOut(w http.ResponseWriter, r *http.Request) {
	reservationID, ok := reservationPathID(w, r)
	if !ok {
		return
	}
	var input CheckoutInput
	if r.ContentLength != 0 {
		if err := webutil.DecodeJSON(r, &input); err != nil {
			webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
			return
		}
	}
	item, fields, err := h.service.CheckOut(r.Context(), webutil.TenantID(r.Context()), reservationID, input)
	if len(fields) > 0 {
		webutil.WriteValidationError(w, r, fields)
		return
	}
	var incomplete *InventoryIncompleteError
	switch {
	case errors.Is(err, ErrNotFound):
		webutil.WriteError(w, r, http.StatusNotFound, "reservation_not_found", "Reservation not found.")
	case errors.As(err, &incomplete):
		webutil.WriteJSON(w, http.StatusConflict, map[string]any{
			"error":         "reservation_inventory_incomplete",
			"message":       "The reservation cannot be checked out until all required physical units are assigned.",
			"request_id":    webutil.RequestID(r.Context()),
			"missing_items": incomplete.Items,
		})
	case errors.Is(err, ErrInvalidTransition):
		webutil.WriteError(w, r, http.StatusConflict, "invalid_reservation_transition", "Only a ready reservation can be checked out.")
	case err != nil:
		webutil.WriteError(w, r, http.StatusInternalServerError, "reservation_checkout_failed", "Could not register the checkout.")
	default:
		webutil.WriteJSON(w, http.StatusOK, item)
	}
}

func (h *Handler) Return(w http.ResponseWriter, r *http.Request) {
	reservationID, ok := reservationPathID(w, r)
	if !ok {
		return
	}
	var input ReturnInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	item, fields, err := h.service.Return(r.Context(), webutil.TenantID(r.Context()), reservationID, input)
	if len(fields) > 0 {
		webutil.WriteValidationError(w, r, fields)
		return
	}
	var mismatch *ReturnMismatchError
	switch {
	case errors.Is(err, ErrNotFound):
		webutil.WriteError(w, r, http.StatusNotFound, "reservation_not_found", "Reservation not found.")
	case errors.As(err, &mismatch):
		webutil.WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"error":                "return_asset_mismatch",
			"message":              "The return inspection must include every checked-out asset exactly once.",
			"request_id":           webutil.RequestID(r.Context()),
			"expected_asset_ids":   mismatch.ExpectedAssetIDs,
			"missing_asset_ids":    mismatch.MissingAssetIDs,
			"unexpected_asset_ids": mismatch.UnexpectedAssetIDs,
		})
	case errors.Is(err, ErrWarehouseState):
		webutil.WriteError(w, r, http.StatusConflict, "warehouse_state_conflict", "The reservation contains an inconsistent checkout assignment.")
	case errors.Is(err, ErrInvalidTransition):
		webutil.WriteError(w, r, http.StatusConflict, "invalid_reservation_transition", "Only a checked-out reservation can be returned.")
	case err != nil:
		webutil.WriteError(w, r, http.StatusInternalServerError, "reservation_return_failed", "Could not register the return inspection.")
	default:
		webutil.WriteJSON(w, http.StatusOK, item)
	}
}

func (h *Handler) Complete(w http.ResponseWriter, r *http.Request) {
	reservationID, ok := reservationPathID(w, r)
	if !ok {
		return
	}
	item, err := h.service.Complete(r.Context(), webutil.TenantID(r.Context()), reservationID)
	switch {
	case errors.Is(err, ErrNotFound):
		webutil.WriteError(w, r, http.StatusNotFound, "reservation_not_found", "Reservation not found.")
	case errors.Is(err, ErrAssetsNotReturned):
		webutil.WriteError(w, r, http.StatusConflict, "assets_not_returned", "Every checked-out asset must be returned before completing the reservation.")
	case errors.Is(err, ErrInvalidTransition):
		webutil.WriteError(w, r, http.StatusConflict, "invalid_reservation_transition", "Only a returned reservation can be completed.")
	case err != nil:
		webutil.WriteError(w, r, http.StatusInternalServerError, "reservation_complete_failed", "Could not complete the reservation.")
	default:
		webutil.WriteJSON(w, http.StatusOK, item)
	}
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	reservationID, ok := reservationPathID(w, r)
	if !ok {
		return
	}
	item, err := h.service.Cancel(r.Context(), webutil.TenantID(r.Context()), reservationID)
	switch {
	case errors.Is(err, ErrNotFound):
		webutil.WriteError(w, r, http.StatusNotFound, "reservation_not_found", "Reservation not found.")
	case errors.Is(err, ErrInvalidTransition):
		webutil.WriteError(w, r, http.StatusConflict, "invalid_reservation_transition", "The reservation cannot be cancelled from its current state.")
	case err != nil:
		webutil.WriteError(w, r, http.StatusInternalServerError, "reservation_cancel_failed", "Could not cancel the reservation.")
	default:
		webutil.WriteJSON(w, http.StatusOK, item)
	}
}

func (h *Handler) transition(
	w http.ResponseWriter,
	r *http.Request,
	action func(context.Context, string, string) (Detail, error),
) {
	reservationID, ok := reservationPathID(w, r)
	if !ok {
		return
	}
	item, err := action(r.Context(), webutil.TenantID(r.Context()), reservationID)
	switch {
	case errors.Is(err, ErrNotFound):
		webutil.WriteError(w, r, http.StatusNotFound, "reservation_not_found", "Reservation not found.")
	case errors.Is(err, ErrInvalidTransition):
		webutil.WriteError(w, r, http.StatusConflict, "invalid_reservation_transition", "The reservation cannot move to that status from its current state.")
	case err != nil:
		webutil.WriteError(w, r, http.StatusInternalServerError, "reservation_transition_failed", "Could not change the reservation status.")
	default:
		webutil.WriteJSON(w, http.StatusOK, item)
	}
}

func reservationPathID(w http.ResponseWriter, r *http.Request) (string, bool) {
	value := strings.TrimSpace(r.PathValue("reservationID"))
	if !idutil.IsUUID(value) {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_reservation_id", "Reservation ID is invalid.")
		return "", false
	}
	return value, true
}

func validStatus(value string) bool {
	for _, status := range []string{
		"PENDING", "CONFIRMED", "PREPARING", "READY",
		"CHECKED_OUT", "RETURNED", "COMPLETED", "CANCELLED",
	} {
		if value == status {
			return true
		}
	}
	return false
}
