package availability

import (
	"context"
	"strings"
	"time"

	"github.com/rentstage/rentstage/apps/api/internal/idutil"
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Check(ctx context.Context, tenantID string, input CheckInput) (Result, map[string]string, error) {
	normalized, fields := Normalize(input)
	if len(fields) > 0 {
		return Result{}, fields, nil
	}
	result, err := s.repository.Check(ctx, tenantID, normalized)
	return result, nil, err
}

func Normalize(input CheckInput) (NormalizedInput, map[string]string) {
	result := NormalizedInput{Items: make([]NormalizedItem, 0, len(input.Items))}
	fields := map[string]string{}

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
	if !result.StartAt.IsZero() && !result.EndAt.IsZero() && !result.EndAt.After(result.StartAt) {
		fields["end_at"] = "End must be after start."
	}
	if len(input.Items) == 0 {
		fields["items"] = "Add at least one resource."
	}
	if len(input.Items) > 100 {
		fields["items"] = "Availability can be checked for at most 100 resources at a time."
	}

	seen := map[string]struct{}{}
	for index, item := range input.Items {
		prefix := "items[" + intString(index) + "]"
		resourceID := strings.TrimSpace(item.ResourceID)
		if !idutil.IsUUID(resourceID) {
			fields[prefix+".resource_id"] = "Resource ID is invalid."
		}
		if _, exists := seen[resourceID]; resourceID != "" && exists {
			fields[prefix+".resource_id"] = "Each resource can appear only once."
		}
		seen[resourceID] = struct{}{}
		if item.Quantity <= 0 || item.Quantity > 10000 {
			fields[prefix+".quantity"] = "Quantity must be between 1 and 10,000."
		}
		result.Items = append(result.Items, NormalizedItem{ResourceID: resourceID, Quantity: item.Quantity})
	}
	return result, fields
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
