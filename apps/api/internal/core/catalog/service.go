package catalog

import (
	"context"
	"strings"

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

func (s *Service) CreateCategory(ctx context.Context, tenantID string, input CreateCategoryInput) (Category, map[string]string, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	fields := map[string]string{}
	if input.Name == "" {
		fields["name"] = "Name is required."
	}
	if len(input.Name) > 120 {
		fields["name"] = "Name must be 120 characters or fewer."
	}
	if len(fields) > 0 {
		return Category{}, fields, nil
	}

	item, err := s.repository.CreateCategory(ctx, tenantID, input)
	if err != nil {
		return Category{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "CATEGORY_CREATED", "category", &item.ID, map[string]any{"name": item.Name})
	return item, nil, nil
}

func (s *Service) DeleteCategory(ctx context.Context, tenantID, categoryID string) error {
	if err := s.repository.DeleteCategory(ctx, tenantID, categoryID); err != nil {
		return err
	}
	_ = s.audit.Record(ctx, tenantID, "CATEGORY_DELETED", "category", &categoryID, nil)
	return nil
}

func (s *Service) CreateResource(ctx context.Context, tenantID string, input CreateResourceInput) (Resource, map[string]string, error) {
	normalizeResourceInput(&input)
	fields := validateResourceInput(input)
	if len(fields) > 0 {
		return Resource{}, fields, nil
	}
	item, err := s.repository.CreateResource(ctx, tenantID, input)
	if err != nil {
		return Resource{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "RESOURCE_CREATED", "resource", &item.ID, map[string]any{"name": item.Name, "sku": item.SKU})
	return item, nil, nil
}

func (s *Service) UpdateResource(
	ctx context.Context,
	tenantID string,
	resourceID string,
	patch UpdateResourceInput,
) (Resource, map[string]string, error) {
	current, err := s.repository.GetResource(ctx, tenantID, resourceID)
	if err != nil {
		return Resource{}, nil, err
	}

	input := CreateResourceInput{
		CategoryID:            current.CategoryID,
		ResourceType:          current.ResourceType,
		Name:                  current.Name,
		Description:           current.Description,
		SKU:                   current.SKU,
		BasePrice:             current.BasePrice,
		PricingUnit:           current.PricingUnit,
		DepositAmount:         current.DepositAmount,
		TrackIndividualAssets: boolPtr(current.TrackIndividualAssets),
		Active:                boolPtr(current.Active),
		Metadata:              current.Metadata,
	}
	if patch.CategoryID.Set {
		input.CategoryID = patch.CategoryID.Value
	}
	if patch.ResourceType != nil {
		input.ResourceType = *patch.ResourceType
	}
	if patch.Name != nil {
		input.Name = *patch.Name
	}
	if patch.Description != nil {
		input.Description = *patch.Description
	}
	if patch.SKU.Set {
		input.SKU = patch.SKU.Value
	}
	if patch.BasePrice != nil {
		input.BasePrice = *patch.BasePrice
	}
	if patch.PricingUnit != nil {
		input.PricingUnit = *patch.PricingUnit
	}
	if patch.DepositAmount != nil {
		input.DepositAmount = *patch.DepositAmount
	}
	if patch.TrackIndividualAssets != nil {
		input.TrackIndividualAssets = patch.TrackIndividualAssets
	}
	if patch.Active != nil {
		input.Active = patch.Active
	}
	if patch.Metadata != nil {
		input.Metadata = *patch.Metadata
	}

	normalizeResourceInput(&input)
	fields := validateResourceInput(input)
	if len(fields) > 0 {
		return Resource{}, fields, nil
	}

	item, err := s.repository.UpdateResource(ctx, tenantID, resourceID, input)
	if err != nil {
		return Resource{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "RESOURCE_UPDATED", "resource", &item.ID, map[string]any{"name": item.Name, "active": item.Active})
	return item, nil, nil
}

func normalizeResourceInput(input *CreateResourceInput) {
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	input.ResourceType = strings.ToUpper(strings.TrimSpace(input.ResourceType))
	input.PricingUnit = strings.ToUpper(strings.TrimSpace(input.PricingUnit))
	if input.ResourceType == "" {
		input.ResourceType = "EQUIPMENT"
	}
	if input.PricingUnit == "" {
		input.PricingUnit = "DAY"
	}
	if input.CategoryID != nil {
		value := strings.TrimSpace(*input.CategoryID)
		if value == "" {
			input.CategoryID = nil
		} else {
			input.CategoryID = &value
		}
	}
	if input.SKU != nil {
		value := strings.ToUpper(strings.TrimSpace(*input.SKU))
		if value == "" {
			input.SKU = nil
		} else {
			input.SKU = &value
		}
	}
	if input.TrackIndividualAssets == nil {
		input.TrackIndividualAssets = boolPtr(true)
	}
	if input.Active == nil {
		input.Active = boolPtr(true)
	}
	if input.Metadata == nil {
		input.Metadata = map[string]any{}
	}
}

func validateResourceInput(input CreateResourceInput) map[string]string {
	fields := map[string]string{}
	if input.Name == "" {
		fields["name"] = "Name is required."
	}
	if len(input.Name) > 180 {
		fields["name"] = "Name must be 180 characters or fewer."
	}
	if input.BasePrice < 0 {
		fields["base_price"] = "Base price cannot be negative."
	}
	if input.DepositAmount < 0 {
		fields["deposit_amount"] = "Deposit cannot be negative."
	}
	if input.CategoryID != nil && !idutil.IsUUID(*input.CategoryID) {
		fields["category_id"] = "Category ID is invalid."
	}
	if !contains([]string{"EQUIPMENT", "SPACE", "SERVICE"}, input.ResourceType) {
		fields["resource_type"] = "Unsupported resource type."
	}
	if !contains([]string{"HOUR", "DAY", "EVENT", "FIXED"}, input.PricingUnit) {
		fields["pricing_unit"] = "Unsupported pricing unit."
	}
	return fields
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func boolPtr(value bool) *bool { return &value }
