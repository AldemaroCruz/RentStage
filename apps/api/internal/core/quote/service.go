package quote

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/rentstage/rentstage/apps/api/internal/core/audit"
	"github.com/rentstage/rentstage/apps/api/internal/idutil"
)

type Service struct {
	repository *Repository
	audit      *audit.Repository
}

func NewService(repository *Repository, auditRepository *audit.Repository) *Service {
	return &Service{repository: repository, audit: auditRepository}
}

func (s *Service) Create(ctx context.Context, tenantID string, input CreateInput) (Detail, map[string]string, error) {
	normalized, fields := normalize(input)
	if len(fields) > 0 {
		return Detail{}, fields, nil
	}
	item, err := s.repository.Create(ctx, tenantID, normalized)
	if err != nil {
		return Detail{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "QUOTE_CREATED", "quote", &item.ID, map[string]any{
		"quote_number": item.QuoteNumber,
		"customer":     item.CustomerName,
		"total":        item.Total,
	})
	return item, nil, nil
}

func (s *Service) Update(ctx context.Context, tenantID, quoteID string, input CreateInput) (Detail, map[string]string, error) {
	normalized, fields := normalize(input)
	if len(fields) > 0 {
		return Detail{}, fields, nil
	}
	item, err := s.repository.Update(ctx, tenantID, quoteID, normalized)
	if err != nil {
		return Detail{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "QUOTE_UPDATED", "quote", &item.ID, map[string]any{
		"quote_number": item.QuoteNumber,
		"total":        item.Total,
	})
	return item, nil, nil
}

func (s *Service) Send(ctx context.Context, tenantID, quoteID string) (Detail, error) {
	return s.transition(ctx, tenantID, quoteID, []string{"DRAFT"}, "SENT", "QUOTE_SENT")
}

func (s *Service) Accept(ctx context.Context, tenantID, quoteID string) (Detail, error) {
	return s.transition(ctx, tenantID, quoteID, []string{"SENT"}, "ACCEPTED", "QUOTE_ACCEPTED")
}

func (s *Service) Reject(ctx context.Context, tenantID, quoteID string) (Detail, error) {
	return s.transition(ctx, tenantID, quoteID, []string{"SENT"}, "REJECTED", "QUOTE_REJECTED")
}

func (s *Service) Cancel(ctx context.Context, tenantID, quoteID string) (Detail, error) {
	return s.transition(ctx, tenantID, quoteID, []string{"DRAFT", "SENT"}, "CANCELLED", "QUOTE_CANCELLED")
}

func (s *Service) transition(
	ctx context.Context,
	tenantID string,
	quoteID string,
	allowed []string,
	target string,
	action string,
) (Detail, error) {
	item, err := s.repository.Transition(ctx, tenantID, quoteID, allowed, target)
	if err != nil {
		return Detail{}, err
	}
	_ = s.audit.Record(ctx, tenantID, action, "quote", &item.ID, map[string]any{
		"quote_number": item.QuoteNumber,
		"status":       item.Status,
	})
	return item, nil
}

func normalize(input CreateInput) (normalizedInput, map[string]string) {
	result := normalizedInput{
		CustomerID:     strings.TrimSpace(input.CustomerID),
		EventType:      cleanOptional(input.EventType),
		EventLocation:  cleanOptional(input.EventLocation),
		DiscountAmount: roundMoney(input.DiscountAmount),
		ExtraCharges:   roundMoney(input.ExtraCharges),
		Notes:          strings.TrimSpace(input.Notes),
		Items:          make([]normalizedItem, 0, len(input.Items)),
	}
	fields := map[string]string{}

	if !idutil.IsUUID(result.CustomerID) {
		fields["customer_id"] = "Customer ID is invalid."
	}
	start, err := time.Parse(time.RFC3339, strings.TrimSpace(input.StartAt))
	if err != nil {
		fields["start_at"] = "Use an RFC3339 timestamp."
	} else {
		result.StartAt = start
	}
	end, err := time.Parse(time.RFC3339, strings.TrimSpace(input.EndAt))
	if err != nil {
		fields["end_at"] = "Use an RFC3339 timestamp."
	} else {
		result.EndAt = end
	}
	if err == nil && !result.StartAt.IsZero() && !result.EndAt.After(result.StartAt) {
		fields["end_at"] = "End must be after start."
	}
	if input.ExpiresAt != nil && strings.TrimSpace(*input.ExpiresAt) != "" {
		value, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(*input.ExpiresAt))
		if parseErr != nil {
			fields["expires_at"] = "Use an RFC3339 timestamp."
		} else {
			result.ExpiresAt = &value
		}
	}
	if result.EventType != nil && len(*result.EventType) > 120 {
		fields["event_type"] = "Event type must be 120 characters or fewer."
	}
	if result.EventLocation != nil && len(*result.EventLocation) > 500 {
		fields["event_location"] = "Event location must be 500 characters or fewer."
	}
	if result.DiscountAmount < 0 {
		fields["discount_amount"] = "Discount cannot be negative."
	}
	if result.ExtraCharges < 0 {
		fields["extra_charges"] = "Extra charges cannot be negative."
	}
	if len(result.Notes) > 6000 {
		fields["notes"] = "Notes must be 6,000 characters or fewer."
	}
	if len(input.Items) == 0 {
		fields["items"] = "Add at least one resource."
	}
	if len(input.Items) > 100 {
		fields["items"] = "A quote can contain at most 100 items."
	}

	seen := map[string]struct{}{}
	for index, item := range input.Items {
		prefix := "items[" + intString(index) + "]"
		normalized := normalizedItem{
			ResourceID:     strings.TrimSpace(item.ResourceID),
			Description:    strings.TrimSpace(item.Description),
			Quantity:       item.Quantity,
			UnitPrice:      roundMoney(item.UnitPrice),
			DiscountAmount: roundMoney(item.DiscountAmount),
		}
		if !idutil.IsUUID(normalized.ResourceID) {
			fields[prefix+".resource_id"] = "Resource ID is invalid."
		}
		if _, exists := seen[normalized.ResourceID]; normalized.ResourceID != "" && exists {
			fields[prefix+".resource_id"] = "Each resource can appear only once."
		}
		seen[normalized.ResourceID] = struct{}{}
		if normalized.Quantity <= 0 || normalized.Quantity > 10000 {
			fields[prefix+".quantity"] = "Quantity must be between 1 and 10,000."
		}
		if normalized.UnitPrice < 0 {
			fields[prefix+".unit_price"] = "Unit price cannot be negative."
		}
		if normalized.DiscountAmount < 0 {
			fields[prefix+".discount_amount"] = "Item discount cannot be negative."
		}
		gross := roundMoney(float64(normalized.Quantity) * normalized.UnitPrice)
		if normalized.DiscountAmount > gross {
			fields[prefix+".discount_amount"] = "Item discount cannot exceed the item amount."
		}
		normalized.LineTotal = roundMoney(math.Max(0, gross-normalized.DiscountAmount))
		result.Subtotal = roundMoney(result.Subtotal + normalized.LineTotal)
		result.Items = append(result.Items, normalized)
	}
	if result.DiscountAmount > result.Subtotal {
		fields["discount_amount"] = "Quote discount cannot exceed the subtotal."
	}
	result.Total = roundMoney(math.Max(0, result.Subtotal-result.DiscountAmount+result.ExtraCharges))
	return result, fields
}

func cleanOptional(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func roundMoney(value float64) float64 {
	return math.Round(value*100) / 100
}

func intString(value int) string {
	if value == 0 {
		return "0"
	}
	var buffer [20]byte
	position := len(buffer)
	for value > 0 {
		position--
		buffer[position] = byte('0' + value%10)
		value /= 10
	}
	return string(buffer[position:])
}
