package publiccatalog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/rentstage/rentstage/apps/api/internal/core/audit"
	"github.com/rentstage/rentstage/apps/api/internal/core/availability"
	"github.com/rentstage/rentstage/apps/api/internal/core/packages"
)

var (
	accentPattern       = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
	publicationSlugExpr = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
)

type Service struct {
	repository      *Repository
	packageService  *packages.Service
	availability    *availability.Service
	audit           *audit.Repository
	webBaseURL      string
	fingerprintSalt string
	now             func() time.Time
}

func NewService(
	repository *Repository,
	packageService *packages.Service,
	availabilityService *availability.Service,
	auditRepository *audit.Repository,
	webBaseURL string,
	fingerprintSalt string,
) *Service {
	return &Service{
		repository:      repository,
		packageService:  packageService,
		availability:    availabilityService,
		audit:           auditRepository,
		webBaseURL:      strings.TrimRight(webBaseURL, "/"),
		fingerprintSalt: fingerprintSalt,
		now:             time.Now,
	}
}

func (s *Service) AdminCatalog(ctx context.Context, tenantID string) (AdminCatalog, error) {
	settings, err := s.repository.GetSettings(ctx, tenantID)
	if err != nil {
		return AdminCatalog{}, err
	}
	tenant, err := s.repository.GetTenant(ctx, tenantID)
	if err != nil {
		return AdminCatalog{}, err
	}
	packageItems, err := s.repository.ListAdminPackages(ctx, tenantID)
	if err != nil {
		return AdminCatalog{}, err
	}
	resources, err := s.repository.ListAdminResources(ctx, tenantID)
	if err != nil {
		return AdminCatalog{}, err
	}
	return AdminCatalog{
		Settings:  settings,
		Tenant:    tenant,
		PublicURL: s.webBaseURL + "/p/" + tenant.Slug,
		Packages:  packageItems,
		Resources: resources,
	}, nil
}

func (s *Service) UpdateSettings(ctx context.Context, tenantID string, input SettingsInput) (Settings, map[string]string, error) {
	normalized, fields := normalizeSettings(input)
	if len(fields) > 0 {
		return Settings{}, fields, nil
	}
	item, err := s.repository.UpsertSettings(ctx, tenantID, normalized)
	if err != nil {
		return Settings{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "PUBLIC_CATALOG_UPDATED", "public_catalog", nil, map[string]any{
		"enabled":                item.Enabled,
		"show_prices":            item.ShowPrices,
		"show_resources":         item.ShowResources,
		"quote_requests_enabled": item.QuoteRequestsEnabled,
		"web_chat_enabled":       item.WebChatEnabled,
		"terms_version":          item.TermsVersion,
	})
	return item, nil, nil
}

func (s *Service) UpdatePackagePublication(ctx context.Context, tenantID, packageID string, input PackagePublicationInput) (AdminCatalog, map[string]string, error) {
	fields := map[string]string{}
	if input.SortOrder < 0 || input.SortOrder > 1_000_000 {
		fields["sort_order"] = "Sort order must be between 0 and 1,000,000."
	}
	if input.Featured && !input.Visible {
		fields["featured"] = "A featured package must also be visible."
	}
	item, err := s.packageService.Get(ctx, tenantID, packageID)
	if err != nil {
		return AdminCatalog{}, nil, err
	}
	if input.Visible && (!item.Active || !item.Ready) {
		fields["visible"] = "Only active packages with available components can be published."
	}
	if len(fields) > 0 {
		return AdminCatalog{}, fields, nil
	}
	if err := s.repository.UpdatePackagePublication(ctx, tenantID, packageID, input); err != nil {
		return AdminCatalog{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "PACKAGE_PUBLICATION_UPDATED", "package", &packageID, map[string]any{
		"visible": input.Visible, "featured": input.Featured, "sort_order": input.SortOrder,
	})
	catalog, err := s.AdminCatalog(ctx, tenantID)
	return catalog, nil, err
}

func (s *Service) UpdateResourcePublication(ctx context.Context, tenantID, resourceID string, input ResourcePublicationInput) (AdminCatalog, map[string]string, error) {
	input.PublicSlug = strings.ToLower(strings.TrimSpace(input.PublicSlug))
	input.PublicDescription = strings.TrimSpace(input.PublicDescription)
	input.PublicImageURL = cleanOptional(input.PublicImageURL)
	if input.Visible && input.PublicSlug == "" {
		items, err := s.repository.ListAdminResources(ctx, tenantID)
		if err != nil {
			return AdminCatalog{}, nil, err
		}
		for _, resource := range items {
			if resource.ID == resourceID {
				input.PublicSlug = slugify(resource.Name)
				break
			}
		}
	}
	fields := map[string]string{}
	if input.Visible && input.PublicSlug == "" {
		fields["public_slug"] = "A public slug is required for visible resources."
	} else if input.PublicSlug != "" && !publicationSlugExpr.MatchString(input.PublicSlug) {
		fields["public_slug"] = "Use lowercase letters, numbers, and single hyphens."
	} else if len(input.PublicSlug) > 140 {
		fields["public_slug"] = "Public slug must be 140 characters or fewer."
	}
	if len(input.PublicDescription) > 4000 {
		fields["public_description"] = "Public description must be 4,000 characters or fewer."
	}
	if input.PublicImageURL != nil && !validHTTPURL(*input.PublicImageURL) {
		fields["public_image_url"] = "Image URL must use http or https."
	}
	if input.SortOrder < 0 || input.SortOrder > 1_000_000 {
		fields["sort_order"] = "Sort order must be between 0 and 1,000,000."
	}
	if input.Featured && !input.Visible {
		fields["featured"] = "A featured resource must also be visible."
	}
	resources, err := s.repository.ListAdminResources(ctx, tenantID)
	if err != nil {
		return AdminCatalog{}, nil, err
	}
	found := false
	for _, resource := range resources {
		if resource.ID == resourceID {
			found = true
			if input.Visible && !resource.Active {
				fields["visible"] = "Archived resources cannot be published."
			}
			break
		}
	}
	if !found {
		return AdminCatalog{}, nil, ErrResourceNotPublic
	}
	if len(fields) > 0 {
		return AdminCatalog{}, fields, nil
	}
	if !input.Visible {
		input.Featured = false
	}
	if err := s.repository.UpdateResourcePublication(ctx, tenantID, resourceID, input); err != nil {
		return AdminCatalog{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "RESOURCE_PUBLICATION_UPDATED", "resource", &resourceID, map[string]any{
		"visible": input.Visible, "featured": input.Featured, "sort_order": input.SortOrder,
		"public_slug": input.PublicSlug,
	})
	catalog, err := s.AdminCatalog(ctx, tenantID)
	return catalog, nil, err
}

func (s *Service) Catalog(ctx context.Context, tenantSlug string) (PublicCatalog, error) {
	tenant, settings, err := s.repository.ResolvePublicCatalog(ctx, strings.ToLower(strings.TrimSpace(tenantSlug)))
	if err != nil {
		return PublicCatalog{}, err
	}
	packageItems, err := s.repository.ListPublicPackages(ctx, settings.TenantID, settings.ShowPrices)
	if err != nil {
		return PublicCatalog{}, err
	}
	resources := make([]PublicResource, 0)
	if settings.ShowResources {
		resources, err = s.repository.ListPublicResources(ctx, settings.TenantID, settings.ShowPrices)
		if err != nil {
			return PublicCatalog{}, err
		}
	}
	return PublicCatalog{
		Tenant:    tenant,
		Settings:  publicSettings(settings),
		Packages:  packageItems,
		Resources: resources,
	}, nil
}

func (s *Service) Package(ctx context.Context, tenantSlug, packageSlug string) (PublicTenant, PublicSettings, PublicPackageDetail, error) {
	tenant, settings, err := s.repository.ResolvePublicCatalog(ctx, strings.ToLower(strings.TrimSpace(tenantSlug)))
	if err != nil {
		return PublicTenant{}, PublicSettings{}, PublicPackageDetail{}, err
	}
	item, err := s.repository.GetPublicPackage(ctx, settings.TenantID, strings.ToLower(strings.TrimSpace(packageSlug)), settings.ShowPrices)
	return tenant, publicSettings(settings), item, err
}

func (s *Service) Resource(ctx context.Context, tenantSlug, resourceSlug string) (PublicTenant, PublicSettings, PublicResource, error) {
	tenant, settings, err := s.repository.ResolvePublicCatalog(ctx, strings.ToLower(strings.TrimSpace(tenantSlug)))
	if err != nil {
		return PublicTenant{}, PublicSettings{}, PublicResource{}, err
	}
	if !settings.ShowResources {
		return PublicTenant{}, PublicSettings{}, PublicResource{}, ErrResourceNotPublic
	}
	item, err := s.repository.GetPublicResource(ctx, settings.TenantID, strings.ToLower(strings.TrimSpace(resourceSlug)), settings.ShowPrices)
	return tenant, publicSettings(settings), item, err
}

func (s *Service) Availability(ctx context.Context, tenantSlug string, input AvailabilityInput) (PublicAvailabilityResult, map[string]string, error) {
	_, settings, err := s.repository.ResolvePublicCatalog(ctx, strings.ToLower(strings.TrimSpace(tenantSlug)))
	if err != nil {
		return PublicAvailabilityResult{}, nil, err
	}
	if !settings.QuoteRequestsEnabled {
		return PublicAvailabilityResult{}, nil, ErrQuoteRequestsDisabled
	}
	normalized, fields := normalizeAvailability(input)
	if len(fields) > 0 {
		return PublicAvailabilityResult{}, fields, nil
	}
	prepared, prepareFields, err := s.prepare(ctx, settings.TenantID, normalized.StartAt, normalized.EndAt, normalized.Selections)
	if len(prepareFields) > 0 || err != nil {
		return PublicAvailabilityResult{}, prepareFields, err
	}
	return sanitizeAvailability(prepared.Availability), nil, nil
}

func (s *Service) SubmitQuoteRequest(
	ctx context.Context,
	tenantSlug string,
	input QuoteRequestInput,
	clientIP string,
	userAgent string,
) (QuoteRequestReceipt, map[string]string, error) {
	tenant, settings, err := s.repository.ResolvePublicCatalog(ctx, strings.ToLower(strings.TrimSpace(tenantSlug)))
	if err != nil {
		return QuoteRequestReceipt{}, nil, err
	}
	if !settings.QuoteRequestsEnabled {
		return QuoteRequestReceipt{}, nil, ErrQuoteRequestsDisabled
	}
	// A filled honeypot receives a harmless generic acknowledgement but is never persisted.
	if strings.TrimSpace(input.Website) != "" {
		now := s.now().UTC()
		return QuoteRequestReceipt{
			ReferenceCode: "RQ-RECEIVED", Status: QuoteRequestStatusNew,
			Availability: PublicAvailabilityResult{}, CreatedAt: now,
		}, nil, nil
	}
	normalized, fields := normalizeQuoteRequest(input, s.now())
	if len(fields) > 0 {
		return QuoteRequestReceipt{}, fields, nil
	}
	hash := s.submitterHash(settings.TenantID, clientIP)
	count, err := s.repository.CountRecentRequests(ctx, settings.TenantID, hash, s.now().Add(-time.Hour))
	if err != nil {
		return QuoteRequestReceipt{}, nil, err
	}
	if count >= 5 {
		return QuoteRequestReceipt{}, nil, ErrQuoteRequestRateLimited
	}
	prepared, prepareFields, err := s.prepare(ctx, settings.TenantID, normalized.StartAt, normalized.EndAt, normalized.Selections)
	if len(prepareFields) > 0 || err != nil {
		return QuoteRequestReceipt{}, prepareFields, err
	}
	prepared.Input = normalized
	prepared.TermsText = settings.TermsText
	prepared.TermsVersion = settings.TermsVersion
	prepared.SubmitterHash = hash
	prepared.UserAgent = truncate(strings.TrimSpace(userAgent), 500)
	receipt, err := s.repository.CreateQuoteRequest(ctx, settings.TenantID, tenant.Currency, prepared)
	if err != nil {
		return QuoteRequestReceipt{}, nil, err
	}
	receipt.Availability = sanitizeAvailability(prepared.Availability)
	estimatedTotal := prepared.EstimatedTotal
	receipt.EstimatedTotal = &estimatedTotal
	if !settings.ShowPrices {
		receipt.EstimatedTotal = nil
	}
	_ = s.audit.RecordAs(ctx, settings.TenantID, "API", "public-catalog", "QUOTE_REQUEST_CREATED", "quote_request", &receipt.RequestID, map[string]any{
		"reference_code":  receipt.ReferenceCode,
		"available":       receipt.AvailabilityAvailable,
		"estimated_total": prepared.EstimatedTotal,
		"package_count":   len(prepared.Packages),
	})
	return receipt, nil, nil
}

func (s *Service) ListQuoteRequests(ctx context.Context, tenantID, search, status string) (QuoteRequestList, map[string]string, error) {
	status = strings.ToUpper(strings.TrimSpace(status))
	fields := map[string]string{}
	if status != "" && !validQuoteRequestStatus(status) {
		fields["status"] = "Unsupported quote request status."
	}
	if len(fields) > 0 {
		return QuoteRequestList{}, fields, nil
	}
	item, err := s.repository.ListQuoteRequests(ctx, tenantID, strings.TrimSpace(search), status)
	return item, nil, err
}

func (s *Service) GetQuoteRequest(ctx context.Context, tenantID, requestID string) (QuoteRequestDetail, error) {
	return s.repository.GetQuoteRequest(ctx, tenantID, requestID)
}

func (s *Service) UpdateQuoteRequestStatus(ctx context.Context, tenantID, requestID, actorID string, input QuoteRequestStatusInput) (QuoteRequestDetail, map[string]string, error) {
	status := strings.ToUpper(strings.TrimSpace(input.Status))
	fields := map[string]string{}
	if !validQuoteRequestStatus(status) || status == QuoteRequestStatusConverted {
		fields["status"] = "Use NEW, IN_REVIEW, CLOSED, or SPAM."
	}
	if len(fields) > 0 {
		return QuoteRequestDetail{}, fields, nil
	}
	item, err := s.repository.UpdateQuoteRequestStatus(ctx, tenantID, requestID, actorID, status)
	if err != nil {
		return QuoteRequestDetail{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "QUOTE_REQUEST_STATUS_UPDATED", "quote_request", &requestID, map[string]any{
		"reference_code": item.ReferenceCode, "status": item.Status,
	})
	return item, nil, nil
}

func (s *Service) ConvertQuoteRequest(ctx context.Context, tenantID, requestID, actorID string) (ConversionResult, error) {
	result, err := s.repository.ConvertQuoteRequest(ctx, tenantID, requestID, actorID)
	if err != nil {
		return ConversionResult{}, err
	}
	_ = s.audit.Record(ctx, tenantID, "QUOTE_REQUEST_CONVERTED", "quote_request", &requestID, map[string]any{
		"reference_code": result.ReferenceCode,
		"customer_id":    result.CustomerID,
		"quote_id":       result.QuoteID,
		"quote_number":   result.QuoteNumber,
	})
	return result, nil
}

type normalizedAvailability struct {
	StartAt    time.Time
	EndAt      time.Time
	Selections []QuoteRequestSelection
}

func normalizeAvailability(input AvailabilityInput) (normalizedAvailability, map[string]string) {
	start, startErr := time.Parse(time.RFC3339, strings.TrimSpace(input.StartAt))
	end, endErr := time.Parse(time.RFC3339, strings.TrimSpace(input.EndAt))
	fields := validatePeriod(start, end, startErr, endErr)
	selections, selectionFields := normalizeSelections(input.Selections)
	for key, value := range selectionFields {
		fields[key] = value
	}
	return normalizedAvailability{StartAt: start, EndAt: end, Selections: selections}, fields
}

func normalizeQuoteRequest(input QuoteRequestInput, now time.Time) (normalizedQuoteRequest, map[string]string) {
	result := normalizedQuoteRequest{
		FirstName:         strings.TrimSpace(input.FirstName),
		LastName:          strings.TrimSpace(input.LastName),
		Phone:             cleanOptional(input.Phone),
		Email:             cleanOptional(input.Email),
		CompanyName:       cleanOptional(input.CompanyName),
		PreferredLanguage: strings.ToLower(strings.TrimSpace(input.PreferredLanguage)),
		EventType:         cleanOptional(input.EventType),
		EventLocation:     cleanOptional(input.EventLocation),
		Notes:             strings.TrimSpace(input.Notes),
		ConsentAccepted:   input.ConsentAccepted,
	}
	if result.Email != nil {
		lower := strings.ToLower(*result.Email)
		result.Email = &lower
	}
	if result.PreferredLanguage == "" {
		result.PreferredLanguage = "es"
	}
	start, startErr := time.Parse(time.RFC3339, strings.TrimSpace(input.StartAt))
	end, endErr := time.Parse(time.RFC3339, strings.TrimSpace(input.EndAt))
	result.StartAt = start
	result.EndAt = end
	fields := validatePeriod(start, end, startErr, endErr)
	if !start.IsZero() && start.Before(now.Add(-5*time.Minute)) {
		fields["start_at"] = "The event period cannot begin in the past."
	}
	if !start.IsZero() && start.After(now.AddDate(3, 0, 0)) {
		fields["start_at"] = "The event period must be within the next three years."
	}
	if result.FirstName == "" {
		fields["first_name"] = "First name is required."
	} else if len(result.FirstName) > 120 {
		fields["first_name"] = "First name must be 120 characters or fewer."
	}
	if len(result.LastName) > 120 {
		fields["last_name"] = "Last name must be 120 characters or fewer."
	}
	if result.Email == nil && result.Phone == nil {
		fields["contact"] = "Provide an email address or phone number."
	}
	if result.Email != nil {
		if len(*result.Email) > 320 {
			fields["email"] = "Email must be 320 characters or fewer."
		} else if !validEmail(*result.Email) {
			fields["email"] = "Email address is invalid."
		}
	}
	if result.Phone != nil && len(*result.Phone) > 40 {
		fields["phone"] = "Phone number must be 40 characters or fewer."
	}
	if result.CompanyName != nil && len(*result.CompanyName) > 180 {
		fields["company_name"] = "Company name must be 180 characters or fewer."
	}
	if result.EventType != nil && len(*result.EventType) > 120 {
		fields["event_type"] = "Event type must be 120 characters or fewer."
	}
	if result.EventLocation != nil && len(*result.EventLocation) > 500 {
		fields["event_location"] = "Event location must be 500 characters or fewer."
	}
	if len(result.Notes) > 6000 {
		fields["notes"] = "Notes must be 6,000 characters or fewer."
	}
	if result.PreferredLanguage != "es" && result.PreferredLanguage != "en" {
		fields["preferred_language"] = "Use es or en."
	}
	if !result.ConsentAccepted {
		fields["consent_accepted"] = "Accept the contact and privacy notice to continue."
	}
	var selectionFields map[string]string
	result.Selections, selectionFields = normalizeSelections(input.Selections)
	for key, value := range selectionFields {
		fields[key] = value
	}
	return result, fields
}

func normalizeSelections(input []QuoteRequestSelection) ([]QuoteRequestSelection, map[string]string) {
	result := make([]QuoteRequestSelection, 0, len(input))
	fields := map[string]string{}
	if len(input) == 0 {
		fields["selections"] = "Select at least one package."
	}
	if len(input) > 10 {
		fields["selections"] = "Select at most 10 packages."
	}
	seen := map[string]struct{}{}
	for index, item := range input {
		slug := strings.ToLower(strings.TrimSpace(item.PackageSlug))
		prefix := fmt.Sprintf("selections[%d]", index)
		if slug == "" || !publicationSlugExpr.MatchString(slug) {
			fields[prefix+".package_slug"] = "Package slug is invalid."
		}
		if _, exists := seen[slug]; slug != "" && exists {
			fields[prefix+".package_slug"] = "Each package can appear only once."
		}
		seen[slug] = struct{}{}
		if item.Quantity <= 0 || item.Quantity > 100 {
			fields[prefix+".quantity"] = "Quantity must be between 1 and 100."
		}
		result = append(result, QuoteRequestSelection{PackageSlug: slug, Quantity: item.Quantity})
	}
	return result, fields
}

func validatePeriod(start, end time.Time, startErr, endErr error) map[string]string {
	fields := map[string]string{}
	if startErr != nil {
		fields["start_at"] = "Use an RFC3339 timestamp."
	}
	if endErr != nil {
		fields["end_at"] = "Use an RFC3339 timestamp."
	}
	if startErr == nil && endErr == nil {
		if !end.After(start) {
			fields["end_at"] = "End must be after start."
		} else if end.Sub(start) > 31*24*time.Hour {
			fields["end_at"] = "The requested period cannot exceed 31 days."
		}
	}
	return fields
}

func (s *Service) prepare(ctx context.Context, tenantID string, startAt, endAt time.Time, selections []QuoteRequestSelection) (preparedRequest, map[string]string, error) {
	result := preparedRequest{Packages: make([]preparedPackage, 0, len(selections))}
	fields := map[string]string{}
	type accumulator struct {
		ResourceID   string
		ResourceName string
		Description  string
		Quantity     int
		GrossCents   int64
		NetCents     int64
	}
	lines := map[string]*accumulator{}
	order := make([]string, 0)
	availabilityQuantities := map[string]int{}
	for index, selection := range selections {
		template, templateFields, err := s.packageService.PublicQuoteTemplateBySlug(ctx, tenantID, selection.PackageSlug, selection.Quantity)
		if err != nil {
			if err == packages.ErrNotFound || err == packages.ErrUnavailable {
				fields[fmt.Sprintf("selections[%d].package_slug", index)] = "Package is not available in the public catalog."
				continue
			}
			return preparedRequest{}, nil, err
		}
		for key, value := range templateFields {
			fields[fmt.Sprintf("selections[%d].%s", index, key)] = value
		}
		if len(templateFields) > 0 {
			continue
		}
		unitPrice := roundMoney(template.EffectivePrice / float64(selection.Quantity))
		result.Packages = append(result.Packages, preparedPackage{
			PackageID: template.PackageID, PackageName: template.PackageName,
			PackageSlug: selection.PackageSlug, Quantity: selection.Quantity,
			UnitPrice: unitPrice, LineTotal: template.EffectivePrice, Template: template,
		})
		result.EstimatedExtraCharges = roundMoney(result.EstimatedExtraCharges + template.ExtraCharges)
		for _, item := range template.Items {
			entry, exists := lines[item.ResourceID]
			if !exists {
				entry = &accumulator{ResourceID: item.ResourceID, ResourceName: item.ResourceName, Description: item.Description}
				lines[item.ResourceID] = entry
				order = append(order, item.ResourceID)
			}
			entry.Quantity += item.Quantity
			entry.GrossCents += moneyCents(float64(item.Quantity) * item.UnitPrice)
			entry.NetCents += moneyCents(item.LineTotal)
			availabilityQuantities[item.ResourceID] += item.Quantity
		}
	}
	if len(fields) > 0 {
		return preparedRequest{}, fields, nil
	}
	availabilityItems := make([]availability.ItemInput, 0, len(order))
	for _, resourceID := range order {
		entry := lines[resourceID]
		grossUnitCents := entry.GrossCents / int64(entry.Quantity)
		if entry.GrossCents%int64(entry.Quantity) != 0 {
			grossUnitCents++
		}
		grossCents := grossUnitCents * int64(entry.Quantity)
		discountCents := grossCents - entry.NetCents
		if discountCents < 0 {
			discountCents = 0
		}
		result.Items = append(result.Items, preparedLine{
			ResourceID: entry.ResourceID, ResourceName: entry.ResourceName,
			Description: entry.Description, Quantity: entry.Quantity,
			UnitPrice: centsMoney(grossUnitCents), DiscountAmount: centsMoney(discountCents),
			LineTotal: centsMoney(grossCents - discountCents),
		})
		result.EstimatedSubtotal = roundMoney(result.EstimatedSubtotal + centsMoney(grossCents))
		availabilityItems = append(availabilityItems, availability.ItemInput{ResourceID: resourceID, Quantity: availabilityQuantities[resourceID]})
	}
	result.EstimatedDiscount = totalDiscount(result.Items)
	result.EstimatedTotal = roundMoney(result.EstimatedSubtotal - result.EstimatedDiscount + result.EstimatedExtraCharges)
	availabilityResult, availabilityFields, err := s.availability.Check(ctx, tenantID, availability.CheckInput{
		StartAt: startAt.Format(time.RFC3339), EndAt: endAt.Format(time.RFC3339), Items: availabilityItems,
	})
	if err != nil {
		return preparedRequest{}, nil, err
	}
	if len(availabilityFields) > 0 {
		return preparedRequest{}, availabilityFields, nil
	}
	result.Availability = availabilityResult
	return result, nil, nil
}

func normalizeSettings(input SettingsInput) (normalizedSettings, map[string]string) {
	result := normalizedSettings{
		Enabled: input.Enabled, Headline: strings.TrimSpace(input.Headline),
		Description: strings.TrimSpace(input.Description), CoverImageURL: cleanOptional(input.CoverImageURL),
		AccentColor: strings.ToUpper(strings.TrimSpace(input.AccentColor)), ShowPrices: input.ShowPrices,
		ShowResources: input.ShowResources, QuoteRequestsEnabled: input.QuoteRequestsEnabled,
		ContactEmail: cleanOptional(input.ContactEmail), ContactPhone: cleanOptional(input.ContactPhone),
		ContactAddress: cleanOptional(input.ContactAddress), TermsText: strings.TrimSpace(input.TermsText),
		TermsVersion:   strings.TrimSpace(input.TermsVersion),
		WebChatEnabled: input.WebChatEnabled,
	}
	if result.AccentColor == "" {
		result.AccentColor = "#6657F7"
	}
	if result.TermsVersion == "" {
		result.TermsVersion = "1.0"
	}
	fields := map[string]string{}
	if len(result.Headline) > 180 {
		fields["headline"] = "Headline must be 180 characters or fewer."
	}
	if len(result.Description) > 4000 {
		fields["description"] = "Description must be 4,000 characters or fewer."
	}
	if result.CoverImageURL != nil && !validHTTPURL(*result.CoverImageURL) {
		fields["cover_image_url"] = "Cover image URL must use http or https."
	}
	if !accentPattern.MatchString(result.AccentColor) {
		fields["accent_color"] = "Use a six-digit hexadecimal color such as #6657F7."
	}
	if result.ContactEmail != nil {
		if len(*result.ContactEmail) > 320 {
			fields["contact_email"] = "Contact email must be 320 characters or fewer."
		} else if !validEmail(*result.ContactEmail) {
			fields["contact_email"] = "Contact email is invalid."
		}
	}
	if result.ContactPhone != nil && len(*result.ContactPhone) > 40 {
		fields["contact_phone"] = "Contact phone must be 40 characters or fewer."
	}
	if result.ContactAddress != nil && len(*result.ContactAddress) > 500 {
		fields["contact_address"] = "Contact address must be 500 characters or fewer."
	}
	if result.TermsText == "" {
		fields["terms_text"] = "Contact and privacy terms are required."
	} else if len(result.TermsText) > 4000 {
		fields["terms_text"] = "Terms must be 4,000 characters or fewer."
	}
	if len(result.TermsVersion) > 40 {
		fields["terms_version"] = "Terms version must be 40 characters or fewer."
	}
	if !result.Enabled {
		result.WebChatEnabled = false
	}
	return result, fields
}

func publicSettings(item Settings) PublicSettings {
	return PublicSettings{
		Headline: item.Headline, Description: item.Description, CoverImageURL: item.CoverImageURL,
		AccentColor: item.AccentColor, ShowPrices: item.ShowPrices, ShowResources: item.ShowResources,
		QuoteRequestsEnabled: item.QuoteRequestsEnabled, WebChatEnabled: item.WebChatEnabled, ContactEmail: item.ContactEmail,
		ContactPhone: item.ContactPhone, ContactAddress: item.ContactAddress,
		TermsText: item.TermsText, TermsVersion: item.TermsVersion,
	}
}

func sanitizeAvailability(item availability.Result) PublicAvailabilityResult {
	result := PublicAvailabilityResult{StartAt: item.StartAt, EndAt: item.EndAt, Available: item.Available, Items: make([]PublicAvailabilityItem, 0, len(item.Items))}
	for _, source := range item.Items {
		result.Items = append(result.Items, PublicAvailabilityItem{
			ResourceName: source.ResourceName, RequestedQuantity: source.RequestedQuantity, CanFulfill: source.CanFulfill,
		})
	}
	return result
}

func validQuoteRequestStatus(value string) bool {
	switch value {
	case QuoteRequestStatusNew, QuoteRequestStatusInReview, QuoteRequestStatusConverted, QuoteRequestStatusClosed, QuoteRequestStatusSpam:
		return true
	default:
		return false
	}
}

func (s *Service) submitterHash(tenantID, clientIP string) string {
	source := s.fingerprintSalt + "|" + tenantID + "|" + strings.TrimSpace(clientIP)
	digest := sha256.Sum256([]byte(source))
	return hex.EncodeToString(digest[:])
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

func validEmail(value string) bool {
	parsed, err := mail.ParseAddress(value)
	return err == nil && parsed.Name == "" && strings.EqualFold(parsed.Address, value)
}

func validHTTPURL(value string) bool {
	if len(value) > 2000 {
		return false
	}
	parsed, err := url.ParseRequestURI(value)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https")
}

func truncate(value string, maximum int) string {
	runes := []rune(value)
	if len(runes) <= maximum {
		return value
	}
	return string(runes[:maximum])
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

func totalDiscount(items []preparedLine) float64 {
	value := 0.0
	for _, item := range items {
		value += item.DiscountAmount
	}
	return roundMoney(value)
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
