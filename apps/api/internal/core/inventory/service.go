package inventory

import (
	"context"
	"strings"
	"time"

	"github.com/rentstage/rentstage/apps/api/internal/core/audit"
)

type normalizedAssetInput struct {
	AssetCode      string
	SerialNumber   *string
	PhysicalStatus string
	PurchaseDate   *time.Time
	PurchasePrice  *float64
	Notes          string
}

type Service struct {
	repository *Repository
	audit      *audit.Repository
}

func NewService(repository *Repository, auditRepository *audit.Repository) *Service {
	return &Service{repository: repository, audit: auditRepository}
}

func (s *Service) CreateAsset(
	ctx context.Context,
	tenantID string,
	resourceID string,
	input CreateAssetInput,
) (Asset, map[string]string, error) {
	normalized, fields := normalizeAssetInput(input)
	if len(fields) > 0 {
		return Asset{}, fields, nil
	}
	item, err := s.repository.CreateAsset(ctx, tenantID, resourceID, normalized)
	if err != nil {
		return Asset{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "ASSET_CREATED", "asset", &item.ID, map[string]any{
		"asset_code":  item.AssetCode,
		"resource_id": item.ResourceID,
		"status":      item.PhysicalStatus,
	})
	return item, nil, nil
}

func (s *Service) UpdateAsset(
	ctx context.Context,
	tenantID string,
	assetID string,
	patch UpdateAssetInput,
) (Asset, map[string]string, error) {
	current, err := s.repository.GetAsset(ctx, tenantID, assetID)
	if err != nil {
		return Asset{}, nil, err
	}

	input := CreateAssetInput{
		AssetCode:      current.AssetCode,
		SerialNumber:   current.SerialNumber,
		PhysicalStatus: current.PhysicalStatus,
		PurchasePrice:  current.PurchasePrice,
		Notes:          current.Notes,
	}
	if current.PurchaseDate != nil {
		date := current.PurchaseDate.Format("2006-01-02")
		input.PurchaseDate = &date
	}
	if patch.AssetCode != nil {
		input.AssetCode = *patch.AssetCode
	}
	if patch.SerialNumber.Set {
		input.SerialNumber = patch.SerialNumber.Value
	}
	if patch.PhysicalStatus != nil {
		input.PhysicalStatus = *patch.PhysicalStatus
	}
	if patch.PurchaseDate.Set {
		input.PurchaseDate = patch.PurchaseDate.Value
	}
	if patch.PurchasePrice.Set {
		input.PurchasePrice = patch.PurchasePrice.Value
	}
	if patch.Notes != nil {
		input.Notes = *patch.Notes
	}

	normalized, fields := normalizeAssetInput(input)
	if len(fields) > 0 {
		return Asset{}, fields, nil
	}
	item, err := s.repository.UpdateAsset(ctx, tenantID, assetID, normalized)
	if err != nil {
		return Asset{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "ASSET_UPDATED", "asset", &item.ID, map[string]any{
		"asset_code": item.AssetCode,
		"status":     item.PhysicalStatus,
	})
	return item, nil, nil
}

func (s *Service) RetireAsset(ctx context.Context, tenantID, assetID string) (Asset, error) {
	status := "RETIRED"
	item, _, err := s.UpdateAsset(ctx, tenantID, assetID, UpdateAssetInput{PhysicalStatus: &status})
	if err == nil {
		_ = s.audit.Record(ctx, tenantID, "ASSET_RETIRED", "asset", &item.ID, map[string]any{"asset_code": item.AssetCode})
	}
	return item, err
}

func normalizeAssetInput(input CreateAssetInput) (normalizedAssetInput, map[string]string) {
	fields := map[string]string{}
	result := normalizedAssetInput{
		AssetCode:      strings.ToUpper(strings.TrimSpace(input.AssetCode)),
		PhysicalStatus: strings.ToUpper(strings.TrimSpace(input.PhysicalStatus)),
		PurchasePrice:  input.PurchasePrice,
		Notes:          strings.TrimSpace(input.Notes),
	}
	if result.PhysicalStatus == "" {
		result.PhysicalStatus = "AVAILABLE"
	}
	if input.SerialNumber != nil {
		value := strings.ToUpper(strings.TrimSpace(*input.SerialNumber))
		if value != "" {
			result.SerialNumber = &value
		}
	}
	if input.PurchaseDate != nil && strings.TrimSpace(*input.PurchaseDate) != "" {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(*input.PurchaseDate))
		if err != nil {
			fields["purchase_date"] = "Use YYYY-MM-DD format."
		} else {
			result.PurchaseDate = &parsed
		}
	}
	if result.AssetCode == "" {
		fields["asset_code"] = "Asset code is required."
	}
	if len(result.AssetCode) > 120 {
		fields["asset_code"] = "Asset code must be 120 characters or fewer."
	}
	if !contains(statuses(), result.PhysicalStatus) {
		fields["physical_status"] = "Unsupported asset status."
	}
	if result.PurchasePrice != nil && *result.PurchasePrice < 0 {
		fields["purchase_price"] = "Purchase price cannot be negative."
	}
	return result, fields
}

func statuses() []string {
	return []string{"AVAILABLE", "MAINTENANCE", "DAMAGED", "LOST", "RETIRED"}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
