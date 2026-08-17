package reservation

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/rentstage/rentstage/apps/api/internal/core/audit"
	"github.com/rentstage/rentstage/apps/api/internal/idutil"
	"github.com/rentstage/rentstage/apps/api/internal/webutil"
)

type Service struct {
	repository *Repository
	audit      *audit.Repository
}

func NewService(repository *Repository, auditRepository *audit.Repository) *Service {
	return &Service{repository: repository, audit: auditRepository}
}

func (s *Service) CreateManual(ctx context.Context, tenantID string, input CreateInput) (Detail, map[string]string, error) {
	normalized, fields := normalizeCreateInput(input)
	if len(fields) > 0 {
		return Detail{}, fields, nil
	}
	item, err := s.repository.CreateManual(ctx, tenantID, normalized, webutil.ActorID(ctx))
	if err != nil {
		return Detail{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "RESERVATION_CREATED_MANUALLY", "reservation", &item.ID, map[string]any{
		"reservation_number": item.ReservationNumber,
		"customer":           item.CustomerName,
		"source":             item.Source,
		"total":              item.Total,
	})
	return item, nil, nil
}

func (s *Service) Reschedule(ctx context.Context, tenantID, reservationID string, input RescheduleInput) (Detail, map[string]string, error) {
	normalized, fields := normalizeRescheduleInput(input)
	if len(fields) > 0 {
		return Detail{}, fields, nil
	}
	item, err := s.repository.Reschedule(ctx, tenantID, reservationID, normalized, webutil.ActorID(ctx))
	if err != nil {
		return Detail{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "RESERVATION_RESCHEDULED", "reservation", &item.ID, map[string]any{
		"reservation_number": item.ReservationNumber,
		"block_start_at":     item.BlockStartAt,
		"block_end_at":       item.BlockEndAt,
		"event_start_at":     item.EventStartAt,
		"event_end_at":       item.EventEndAt,
		"reason":             normalized.Reason,
	})
	return item, nil, nil
}

func (s *Service) ConvertQuote(ctx context.Context, tenantID, quoteID string) (Detail, error) {
	item, err := s.repository.CreateFromQuote(ctx, tenantID, quoteID, webutil.ActorID(ctx))
	if err != nil {
		return Detail{}, err
	}
	_ = s.audit.Record(ctx, tenantID, "RESERVATION_CREATED_FROM_QUOTE", "reservation", &item.ID, map[string]any{
		"reservation_number": item.ReservationNumber,
		"quote_id":           quoteID,
		"customer":           item.CustomerName,
		"total":              item.Total,
	})
	return item, nil
}

func (s *Service) Confirm(ctx context.Context, tenantID, reservationID string) (Detail, error) {
	return s.transition(ctx, tenantID, reservationID, []string{"PENDING"}, "CONFIRMED", "RESERVATION_CONFIRMED")
}

func (s *Service) Prepare(ctx context.Context, tenantID, reservationID string) (Detail, error) {
	return s.transition(ctx, tenantID, reservationID, []string{"CONFIRMED"}, "PREPARING", "RESERVATION_PREPARING")
}

func (s *Service) MarkReady(ctx context.Context, tenantID, reservationID string) (Detail, error) {
	item, err := s.repository.MarkReadyWithInventory(ctx, tenantID, reservationID, webutil.ActorID(ctx))
	if err != nil {
		return Detail{}, err
	}
	_ = s.audit.Record(ctx, tenantID, "RESERVATION_READY", "reservation", &item.ID, map[string]any{
		"reservation_number": item.ReservationNumber,
		"assigned_assets":    item.AssignedAssetCount,
	})
	return item, nil
}

func (s *Service) AssignAsset(ctx context.Context, tenantID, reservationID string, input AssignAssetInput) (Detail, map[string]string, error) {
	input.AssetID = strings.TrimSpace(input.AssetID)
	if !idutil.IsUUID(input.AssetID) {
		return Detail{}, map[string]string{"asset_id": "Asset ID is invalid."}, nil
	}
	item, err := s.repository.AssignAsset(ctx, tenantID, reservationID, input.AssetID, webutil.ActorID(ctx))
	if err != nil {
		return Detail{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "RESERVATION_ASSET_ASSIGNED", "reservation", &item.ID, map[string]any{
		"reservation_number": item.ReservationNumber,
		"asset_id":           input.AssetID,
	})
	return item, nil, nil
}

func (s *Service) UnassignAsset(ctx context.Context, tenantID, reservationID, assetID string) (Detail, error) {
	item, err := s.repository.UnassignAsset(ctx, tenantID, reservationID, assetID, webutil.ActorID(ctx))
	if err != nil {
		return Detail{}, err
	}
	_ = s.audit.Record(ctx, tenantID, "RESERVATION_ASSET_UNASSIGNED", "reservation", &item.ID, map[string]any{
		"reservation_number": item.ReservationNumber,
		"asset_id":           assetID,
	})
	return item, nil
}

func (s *Service) CheckOut(ctx context.Context, tenantID, reservationID string, input CheckoutInput) (Detail, map[string]string, error) {
	input.Notes = strings.TrimSpace(input.Notes)
	if len(input.Notes) > 2000 {
		return Detail{}, map[string]string{"notes": "Checkout notes must be 2000 characters or fewer."}, nil
	}
	item, err := s.repository.CheckOutWithAssets(ctx, tenantID, reservationID, webutil.ActorID(ctx), input.Notes)
	if err != nil {
		return Detail{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "RESERVATION_CHECKED_OUT", "reservation", &item.ID, map[string]any{
		"reservation_number": item.ReservationNumber,
		"assets":             item.AssignedAssetCount,
		"notes":              input.Notes,
	})
	return item, nil, nil
}

func (s *Service) Return(ctx context.Context, tenantID, reservationID string, input ReturnInput) (Detail, map[string]string, error) {
	normalized, fields := normalizeReturnInput(input)
	if len(fields) > 0 {
		return Detail{}, fields, nil
	}
	item, err := s.repository.ReturnWithInspection(ctx, tenantID, reservationID, webutil.ActorID(ctx), normalized)
	if err != nil {
		return Detail{}, nil, err
	}
	conditionCounts := map[string]int{}
	for _, asset := range normalized.Assets {
		conditionCounts[asset.Condition]++
	}
	_ = s.audit.Record(ctx, tenantID, "RESERVATION_RETURNED", "reservation", &item.ID, map[string]any{
		"reservation_number": item.ReservationNumber,
		"assets":             len(normalized.Assets),
		"conditions":         conditionCounts,
		"notes":              normalized.Notes,
	})
	return item, nil, nil
}

func (s *Service) Complete(ctx context.Context, tenantID, reservationID string) (Detail, error) {
	item, err := s.repository.CompleteAfterReturn(ctx, tenantID, reservationID, webutil.ActorID(ctx))
	if err != nil {
		return Detail{}, err
	}
	_ = s.audit.Record(ctx, tenantID, "RESERVATION_COMPLETED", "reservation", &item.ID, map[string]any{
		"reservation_number": item.ReservationNumber,
		"status":             item.Status,
	})
	return item, nil
}

func (s *Service) Cancel(ctx context.Context, tenantID, reservationID string) (Detail, error) {
	item, err := s.repository.CancelAndRelease(ctx, tenantID, reservationID, webutil.ActorID(ctx))
	if err != nil {
		return Detail{}, err
	}
	_ = s.audit.Record(ctx, tenantID, "RESERVATION_CANCELLED", "reservation", &item.ID, map[string]any{
		"reservation_number": item.ReservationNumber,
		"status":             item.Status,
	})
	return item, nil
}

func (s *Service) transition(
	ctx context.Context,
	tenantID string,
	reservationID string,
	allowed []string,
	target string,
	action string,
) (Detail, error) {
	item, err := s.repository.Transition(ctx, tenantID, reservationID, allowed, target, webutil.ActorID(ctx))
	if err != nil {
		return Detail{}, err
	}
	_ = s.audit.Record(ctx, tenantID, action, "reservation", &item.ID, map[string]any{
		"reservation_number": item.ReservationNumber,
		"status":             item.Status,
	})
	return item, nil
}

func normalizeReturnInput(input ReturnInput) (ReturnInput, map[string]string) {
	fields := map[string]string{}
	result := ReturnInput{
		Notes:  strings.TrimSpace(input.Notes),
		Assets: make([]ReturnAssetInput, 0, len(input.Assets)),
	}
	if len(result.Notes) > 2000 {
		fields["notes"] = "Return notes must be 2000 characters or fewer."
	}
	seen := map[string]struct{}{}
	for index, item := range input.Assets {
		assetID := strings.TrimSpace(item.AssetID)
		condition := strings.ToUpper(strings.TrimSpace(item.Condition))
		notes := strings.TrimSpace(item.Notes)
		prefix := fmt.Sprintf("assets.%d", index)
		if !idutil.IsUUID(assetID) {
			fields[prefix+".asset_id"] = "Asset ID is invalid."
		}
		if _, exists := seen[assetID]; assetID != "" && exists {
			fields[prefix+".asset_id"] = "Each asset can only be inspected once."
		}
		seen[assetID] = struct{}{}
		if !contains([]string{"GOOD", "MAINTENANCE_REQUIRED", "DAMAGED", "LOST"}, condition) {
			fields[prefix+".condition"] = "Unsupported return condition."
		}
		if len(notes) > 1000 {
			fields[prefix+".notes"] = "Asset return notes must be 1000 characters or fewer."
		}
		result.Assets = append(result.Assets, ReturnAssetInput{
			AssetID:   assetID,
			Condition: condition,
			Notes:     notes,
		})
	}
	return result, fields
}
func normalizeCreateInput(input CreateInput) (normalizedCreateInput, map[string]string) {
	result := normalizedCreateInput{
		CustomerID:     strings.TrimSpace(input.CustomerID),
		EventType:      cleanOptionalString(input.EventType),
		EventLocation:  cleanOptionalString(input.EventLocation),
		DiscountAmount: roundReservationMoney(input.DiscountAmount),
		ExtraCharges:   roundReservationMoney(input.ExtraCharges),
		Notes:          strings.TrimSpace(input.Notes),
		Items:          make([]normalizedCreateItem, 0, len(input.Items)),
	}
	fields := map[string]string{}

	if !idutil.IsUUID(result.CustomerID) {
		fields["customer_id"] = "Customer ID is invalid."
	}
	result.BlockStartAt = parseReservationTime(input.BlockStartAt, "block_start_at", fields)
	result.BlockEndAt = parseReservationTime(input.BlockEndAt, "block_end_at", fields)
	result.EventStartAt = parseReservationTime(input.EventStartAt, "event_start_at", fields)
	result.EventEndAt = parseReservationTime(input.EventEndAt, "event_end_at", fields)
	validateSchedule(result.BlockStartAt, result.BlockEndAt, result.EventStartAt, result.EventEndAt, fields)

	if result.EventType != nil && len(*result.EventType) > 120 {
		fields["event_type"] = "Event type must be 120 characters or fewer."
	}
	if result.EventLocation != nil && len(*result.EventLocation) > 500 {
		fields["event_location"] = "Event location must be 500 characters or fewer."
	}
	if !validReservationMoney(input.DiscountAmount) || result.DiscountAmount < 0 {
		fields["discount_amount"] = "Discount must be a finite non-negative amount."
	}
	if !validReservationMoney(input.ExtraCharges) || result.ExtraCharges < 0 {
		fields["extra_charges"] = "Extra charges must be a finite non-negative amount."
	}
	if len(result.Notes) > 6000 {
		fields["notes"] = "Notes must be 6,000 characters or fewer."
	}
	if len(input.Items) == 0 {
		fields["items"] = "Add at least one resource."
	}
	if len(input.Items) > 100 {
		fields["items"] = "A reservation can contain at most 100 items."
	}

	seen := map[string]struct{}{}
	for index, item := range input.Items {
		prefix := fmt.Sprintf("items[%d]", index)
		normalized := normalizedCreateItem{
			ResourceID:     strings.TrimSpace(item.ResourceID),
			Description:    strings.TrimSpace(item.Description),
			Quantity:       item.Quantity,
			UnitPrice:      roundReservationMoney(item.UnitPrice),
			DiscountAmount: roundReservationMoney(item.DiscountAmount),
		}
		if !idutil.IsUUID(normalized.ResourceID) {
			fields[prefix+".resource_id"] = "Resource ID is invalid."
		}
		if _, exists := seen[normalized.ResourceID]; normalized.ResourceID != "" && exists {
			fields[prefix+".resource_id"] = "Each resource can appear only once."
		}
		seen[normalized.ResourceID] = struct{}{}
		if len(normalized.Description) > 500 {
			fields[prefix+".description"] = "Description must be 500 characters or fewer."
		}
		if normalized.Quantity <= 0 || normalized.Quantity > 10000 {
			fields[prefix+".quantity"] = "Quantity must be between 1 and 10,000."
		}
		if !validReservationMoney(item.UnitPrice) || normalized.UnitPrice < 0 {
			fields[prefix+".unit_price"] = "Unit price must be a finite non-negative amount."
		}
		if !validReservationMoney(item.DiscountAmount) || normalized.DiscountAmount < 0 {
			fields[prefix+".discount_amount"] = "Item discount must be a finite non-negative amount."
		}
		gross := roundReservationMoney(float64(normalized.Quantity) * normalized.UnitPrice)
		if normalized.DiscountAmount > gross {
			fields[prefix+".discount_amount"] = "Item discount cannot exceed the item amount."
		}
		normalized.LineTotal = roundReservationMoney(math.Max(0, gross-normalized.DiscountAmount))
		result.Subtotal = roundReservationMoney(result.Subtotal + normalized.LineTotal)
		result.Items = append(result.Items, normalized)
	}
	if result.DiscountAmount > result.Subtotal {
		fields["discount_amount"] = "Reservation discount cannot exceed the subtotal."
	}
	result.Total = roundReservationMoney(math.Max(0, result.Subtotal-result.DiscountAmount+result.ExtraCharges))
	return result, fields
}

func normalizeRescheduleInput(input RescheduleInput) (normalizedRescheduleInput, map[string]string) {
	result := normalizedRescheduleInput{Reason: strings.TrimSpace(input.Reason)}
	fields := map[string]string{}
	result.BlockStartAt = parseReservationTime(input.BlockStartAt, "block_start_at", fields)
	result.BlockEndAt = parseReservationTime(input.BlockEndAt, "block_end_at", fields)
	result.EventStartAt = parseReservationTime(input.EventStartAt, "event_start_at", fields)
	result.EventEndAt = parseReservationTime(input.EventEndAt, "event_end_at", fields)
	validateSchedule(result.BlockStartAt, result.BlockEndAt, result.EventStartAt, result.EventEndAt, fields)
	if len(result.Reason) > 1000 {
		fields["reason"] = "Reason must be 1,000 characters or fewer."
	}
	return result, fields
}

func parseReservationTime(value, field string, fields map[string]string) time.Time {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		fields[field] = "Use an RFC3339 timestamp."
		return time.Time{}
	}
	return parsed
}

func validateSchedule(blockStart, blockEnd, eventStart, eventEnd time.Time, fields map[string]string) {
	if !blockStart.IsZero() && !blockEnd.IsZero() && !blockEnd.After(blockStart) {
		fields["block_end_at"] = "Block end must be after block start."
	}
	if !eventStart.IsZero() && !eventEnd.IsZero() && !eventEnd.After(eventStart) {
		fields["event_end_at"] = "Event end must be after event start."
	}
	if !blockStart.IsZero() && !eventStart.IsZero() && eventStart.Before(blockStart) {
		fields["event_start_at"] = "Event start must be inside the blocked period."
	}
	if !blockEnd.IsZero() && !eventEnd.IsZero() && eventEnd.After(blockEnd) {
		fields["event_end_at"] = "Event end must be inside the blocked period."
	}
}

func cleanOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func validReservationMoney(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func roundReservationMoney(value float64) float64 {
	return math.Round(value*100) / 100
}
