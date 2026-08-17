package packages

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"github.com/rentstage/rentstage/apps/api/internal/core/audit"
	"github.com/rentstage/rentstage/apps/api/internal/core/availability"
	"github.com/rentstage/rentstage/apps/api/internal/idutil"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type Service struct {
	repository   *Repository
	availability *availability.Service
	audit        *audit.Repository
}

func NewService(repository *Repository, availabilityService *availability.Service, auditRepository *audit.Repository) *Service {
	return &Service{repository: repository, availability: availabilityService, audit: auditRepository}
}

func (s *Service) Get(ctx context.Context, tenantID, packageID string) (Detail, error) {
	item, err := s.repository.Get(ctx, tenantID, packageID)
	if err != nil {
		return Detail{}, err
	}
	item.QuoteTemplate = buildQuoteTemplate(item, 1)
	return item, nil
}

func (s *Service) PublicGetBySlug(ctx context.Context, tenantID, slug string) (Detail, error) {
	item, err := s.repository.GetBySlug(ctx, tenantID, strings.ToLower(strings.TrimSpace(slug)))
	if err != nil {
		return Detail{}, err
	}
	if !item.Active || !item.PublicVisible || !item.Ready {
		return Detail{}, ErrNotFound
	}
	item.QuoteTemplate = buildQuoteTemplate(item, 1)
	return item, nil
}

func (s *Service) PublicQuoteTemplateBySlug(ctx context.Context, tenantID, slug string, quantity int) (QuoteTemplate, map[string]string, error) {
	fields := validateQuantity(quantity)
	if len(fields) > 0 {
		return QuoteTemplate{}, fields, nil
	}
	item, err := s.PublicGetBySlug(ctx, tenantID, slug)
	if err != nil {
		return QuoteTemplate{}, nil, err
	}
	if packageFields := validatePackageQuantity(item, quantity); len(packageFields) > 0 {
		return QuoteTemplate{}, packageFields, nil
	}
	return buildQuoteTemplate(item, quantity), nil, nil
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
	item.QuoteTemplate = buildQuoteTemplate(item, 1)
	_ = s.audit.Record(ctx, tenantID, "PACKAGE_CREATED", "package", &item.ID, map[string]any{
		"name":              item.Name,
		"pricing_mode":      item.PricingMode,
		"effective_price":   item.EffectivePrice,
		"item_count":        item.ItemCount,
		"guest_capacity":    item.GuestCapacity,
		"public_visible":    item.PublicVisible,
		"public_featured":   item.PublicFeatured,
		"public_sort_order": item.PublicSortOrder,
	})
	return item, nil, nil
}

func (s *Service) Update(ctx context.Context, tenantID, packageID string, input CreateInput) (Detail, map[string]string, error) {
	normalized, fields := normalize(input)
	if len(fields) > 0 {
		return Detail{}, fields, nil
	}
	item, err := s.repository.Update(ctx, tenantID, packageID, normalized)
	if err != nil {
		return Detail{}, nil, err
	}
	item.QuoteTemplate = buildQuoteTemplate(item, 1)
	_ = s.audit.Record(ctx, tenantID, "PACKAGE_UPDATED", "package", &item.ID, map[string]any{
		"name":              item.Name,
		"pricing_mode":      item.PricingMode,
		"effective_price":   item.EffectivePrice,
		"active":            item.Active,
		"item_count":        item.ItemCount,
		"public_visible":    item.PublicVisible,
		"public_featured":   item.PublicFeatured,
		"public_sort_order": item.PublicSortOrder,
	})
	return item, nil, nil
}

func (s *Service) Archive(ctx context.Context, tenantID, packageID string) (Detail, error) {
	item, err := s.repository.Archive(ctx, tenantID, packageID)
	if err != nil {
		return Detail{}, err
	}
	item.QuoteTemplate = buildQuoteTemplate(item, 1)
	_ = s.audit.Record(ctx, tenantID, "PACKAGE_ARCHIVED", "package", &item.ID, map[string]any{
		"name": item.Name,
	})
	return item, nil
}

func (s *Service) QuoteTemplate(ctx context.Context, tenantID, packageID string, quantity int) (QuoteTemplate, map[string]string, error) {
	fields := validateQuantity(quantity)
	if len(fields) > 0 {
		return QuoteTemplate{}, fields, nil
	}
	item, err := s.repository.Get(ctx, tenantID, packageID)
	if err != nil {
		return QuoteTemplate{}, nil, err
	}
	if packageFields := validatePackageQuantity(item, quantity); len(packageFields) > 0 {
		return QuoteTemplate{}, packageFields, nil
	}
	if !item.Ready {
		return QuoteTemplate{}, nil, ErrUnavailable
	}
	return buildQuoteTemplate(item, quantity), nil, nil
}

func (s *Service) Availability(ctx context.Context, tenantID, packageID string, input AvailabilityInput) (AvailabilityResult, map[string]string, error) {
	fields := validateQuantity(input.Quantity)
	if input.Quantity == 0 {
		input.Quantity = 1
		fields = nil
	}
	if len(fields) > 0 {
		return AvailabilityResult{}, fields, nil
	}
	item, err := s.repository.Get(ctx, tenantID, packageID)
	if err != nil {
		return AvailabilityResult{}, nil, err
	}
	if packageFields := validatePackageQuantity(item, input.Quantity); len(packageFields) > 0 {
		return AvailabilityResult{}, packageFields, nil
	}
	if !item.Ready {
		return AvailabilityResult{}, nil, ErrUnavailable
	}
	availabilityItems := make([]availability.ItemInput, 0, len(item.Items))
	for _, packageItem := range item.Items {
		availabilityItems = append(availabilityItems, availability.ItemInput{
			ResourceID: packageItem.ResourceID,
			Quantity:   packageItem.Quantity * input.Quantity,
		})
	}
	result, availabilityFields, err := s.availability.Check(ctx, tenantID, availability.CheckInput{
		StartAt: input.StartAt,
		EndAt:   input.EndAt,
		Items:   availabilityItems,
	})
	if len(availabilityFields) > 0 {
		return AvailabilityResult{}, availabilityFields, nil
	}
	if err != nil {
		return AvailabilityResult{}, nil, err
	}
	return AvailabilityResult{
		PackageID:       item.ID,
		PackageName:     item.Name,
		PackageQuantity: input.Quantity,
		StartAt:         result.StartAt,
		EndAt:           result.EndAt,
		Available:       result.Available,
		Items:           result.Items,
	}, nil, nil
}

func packageQuantityLimit(items []Item) int {
	limit := 100
	for _, item := range items {
		if item.Quantity <= 0 {
			continue
		}
		candidate := 10_000 / item.Quantity
		if candidate < limit {
			limit = candidate
		}
	}
	if limit < 1 {
		return 1
	}
	return limit
}

func validateQuantity(quantity int) map[string]string {
	fields := map[string]string{}
	if quantity <= 0 || quantity > 100 {
		fields["quantity"] = "Quantity must be between 1 and 100."
	}
	return fields
}

func validatePackageQuantity(item Detail, quantity int) map[string]string {
	maximum := packageQuantityLimit(item.Items)
	if quantity > maximum {
		return map[string]string{
			"quantity": fmt.Sprintf("This package supports at most %d units per request because of its component quantities.", maximum),
		}
	}
	return nil
}

func normalize(input CreateInput) (normalizedInput, map[string]string) {
	result := normalizedInput{
		Name:          strings.TrimSpace(input.Name),
		Slug:          strings.ToLower(strings.TrimSpace(input.Slug)),
		Description:   strings.TrimSpace(input.Description),
		GuestCapacity: input.GuestCapacity,
		PricingMode:   strings.ToUpper(strings.TrimSpace(input.PricingMode)),
		FixedPrice:    roundMoneyPointer(input.FixedPrice),
		ImageURL:      cleanOptional(input.ImageURL),
		Active:        true,
		Items:         make([]normalizedItem, 0, len(input.Items)),
	}
	if result.Slug == "" {
		result.Slug = slugify(result.Name)
	}
	if result.PricingMode == "" {
		result.PricingMode = PricingModeSumItems
	}
	if input.Active != nil {
		result.Active = *input.Active
	}

	fields := map[string]string{}
	if result.Name == "" {
		fields["name"] = "Name is required."
	} else if len(result.Name) > 180 {
		fields["name"] = "Name must be 180 characters or fewer."
	}
	if result.Slug == "" || !slugPattern.MatchString(result.Slug) {
		fields["slug"] = "Use lowercase letters, numbers, and single hyphens."
	} else if len(result.Slug) > 140 {
		fields["slug"] = "Slug must be 140 characters or fewer."
	}
	if len(result.Description) > 4000 {
		fields["description"] = "Description must be 4,000 characters or fewer."
	}
	if result.GuestCapacity != nil && (*result.GuestCapacity <= 0 || *result.GuestCapacity > 1_000_000) {
		fields["guest_capacity"] = "Guest capacity must be between 1 and 1,000,000."
	}
	if result.PricingMode != PricingModeSumItems && result.PricingMode != PricingModeFixed {
		fields["pricing_mode"] = "Pricing mode must be SUM_ITEMS or FIXED."
	}
	if result.PricingMode == PricingModeFixed {
		if result.FixedPrice == nil {
			fields["fixed_price"] = "Fixed price is required for fixed-price packages."
		} else if *result.FixedPrice < 0 {
			fields["fixed_price"] = "Fixed price cannot be negative."
		}
	} else {
		result.FixedPrice = nil
	}
	if result.ImageURL != nil {
		parsed, err := url.ParseRequestURI(*result.ImageURL)
		if len(*result.ImageURL) > 2000 {
			fields["image_url"] = "Image URL must be 2,000 characters or fewer."
		} else if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			fields["image_url"] = "Image URL must use http or https."
		}
	}
	if len(input.Items) == 0 {
		fields["items"] = "Add at least one resource."
	}
	if len(input.Items) > 50 {
		fields["items"] = "A package can contain at most 50 resources."
	}

	seen := map[string]struct{}{}
	for index, item := range input.Items {
		prefix := "items[" + intString(index) + "]"
		normalized := normalizedItem{
			ResourceID:        strings.TrimSpace(item.ResourceID),
			Description:       strings.TrimSpace(item.Description),
			Quantity:          item.Quantity,
			UnitPriceOverride: roundMoneyPointer(item.UnitPriceOverride),
			SortOrder:         index,
		}
		if item.SortOrder >= 0 {
			normalized.SortOrder = item.SortOrder
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
		if normalized.Quantity <= 0 || normalized.Quantity > 10_000 {
			fields[prefix+".quantity"] = "Quantity must be between 1 and 10,000."
		}
		if normalized.UnitPriceOverride != nil && *normalized.UnitPriceOverride < 0 {
			fields[prefix+".unit_price_override"] = "Price override cannot be negative."
		}
		if normalized.SortOrder < 0 {
			fields[prefix+".sort_order"] = "Sort order cannot be negative."
		}
		result.Items = append(result.Items, normalized)
	}
	return result, fields
}

func buildQuoteTemplate(item Detail, packageQuantity int) QuoteTemplate {
	calculatedTotal := roundMoney(item.CalculatedPrice * float64(packageQuantity))
	effectiveTotal := roundMoney(item.EffectivePrice * float64(packageQuantity))
	template := QuoteTemplate{
		PackageID:       item.ID,
		PackageName:     item.Name,
		PackageQuantity: packageQuantity,
		PricingMode:     item.PricingMode,
		CalculatedPrice: calculatedTotal,
		EffectivePrice:  effectiveTotal,
		Items:           make([]QuoteTemplateItem, 0, len(item.Items)),
	}

	for _, source := range item.Items {
		quantity := source.Quantity * packageQuantity
		unitPrice := roundMoney(source.UnitPrice)
		lineTotal := roundMoney(float64(quantity) * unitPrice)
		template.Items = append(template.Items, QuoteTemplateItem{
			ResourceID:   source.ResourceID,
			ResourceName: source.ResourceName,
			Description:  source.Description,
			Quantity:     quantity,
			UnitPrice:    unitPrice,
			LineTotal:    lineTotal,
		})
	}

	if effectiveTotal < calculatedTotal {
		template.DiscountAmount = roundMoney(calculatedTotal - effectiveTotal)
		allocateDiscount(template.Items, template.DiscountAmount)
	}
	if effectiveTotal > calculatedTotal {
		template.ExtraCharges = roundMoney(effectiveTotal - calculatedTotal)
	}
	return template
}

func allocateDiscount(items []QuoteTemplateItem, discount float64) {
	if len(items) == 0 || discount <= 0 {
		return
	}
	totalGrossCents := int64(0)
	grossCents := make([]int64, len(items))
	for index, item := range items {
		gross := moneyCents(float64(item.Quantity) * item.UnitPrice)
		grossCents[index] = gross
		totalGrossCents += gross
	}
	remaining := moneyCents(discount)
	if totalGrossCents == 0 {
		return
	}
	for index := range items {
		allocated := int64(0)
		if index == len(items)-1 {
			allocated = remaining
		} else {
			allocated = moneyCents(discount) * grossCents[index] / totalGrossCents
			if allocated > remaining {
				allocated = remaining
			}
		}
		if allocated > grossCents[index] {
			allocated = grossCents[index]
		}
		remaining -= allocated
		items[index].DiscountAmount = centsMoney(allocated)
		items[index].LineTotal = centsMoney(grossCents[index] - allocated)
	}
	if remaining > 0 {
		for index := len(items) - 1; index >= 0 && remaining > 0; index-- {
			capacity := grossCents[index] - moneyCents(items[index].DiscountAmount)
			allocated := remaining
			if allocated > capacity {
				allocated = capacity
			}
			items[index].DiscountAmount = centsMoney(moneyCents(items[index].DiscountAmount) + allocated)
			items[index].LineTotal = centsMoney(grossCents[index] - moneyCents(items[index].DiscountAmount))
			remaining -= allocated
		}
	}
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastHyphen := false
	for _, current := range value {
		current = foldRune(current)
		switch {
		case current >= 'a' && current <= 'z', current >= '0' && current <= '9':
			builder.WriteRune(current)
			lastHyphen = false
		case unicode.IsSpace(current) || current == '-' || current == '_' || current == '/':
			if builder.Len() > 0 && !lastHyphen {
				builder.WriteByte('-')
				lastHyphen = true
			}
		}
	}
	return strings.Trim(builder.String(), "-")
}

func foldRune(value rune) rune {
	switch value {
	case 'á', 'à', 'ä', 'â', 'ã':
		return 'a'
	case 'é', 'è', 'ë', 'ê':
		return 'e'
	case 'í', 'ì', 'ï', 'î':
		return 'i'
	case 'ó', 'ò', 'ö', 'ô', 'õ':
		return 'o'
	case 'ú', 'ù', 'ü', 'û':
		return 'u'
	case 'ñ':
		return 'n'
	default:
		return value
	}
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

func roundMoneyPointer(value *float64) *float64 {
	if value == nil {
		return nil
	}
	rounded := roundMoney(*value)
	return &rounded
}

func roundMoney(value float64) float64 {
	return math.Round(value*100) / 100
}

func moneyCents(value float64) int64 {
	return int64(math.Round(value * 100))
}

func centsMoney(value int64) float64 {
	return float64(value) / 100
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
