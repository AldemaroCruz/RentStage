package customer

import (
	"context"
	"net/mail"
	"regexp"
	"strings"

	"github.com/rentstage/rentstage/apps/api/internal/core/audit"
)

var e164Pattern = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)

type Service struct {
	repository *Repository
	audit      *audit.Repository
}

func NewService(repository *Repository, auditRepository *audit.Repository) *Service {
	return &Service{repository: repository, audit: auditRepository}
}

func (s *Service) Create(ctx context.Context, tenantID string, input CreateInput) (Customer, map[string]string, error) {
	normalized, fields := normalize(input)
	if len(fields) > 0 {
		return Customer{}, fields, nil
	}
	item, err := s.repository.Create(ctx, tenantID, normalized)
	if err != nil {
		return Customer{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "CUSTOMER_CREATED", "customer", &item.ID, map[string]any{
		"name":   item.DisplayName,
		"source": item.Source,
	})
	return item, nil, nil
}

func (s *Service) Update(ctx context.Context, tenantID, customerID string, input UpdateInput) (Customer, map[string]string, error) {
	normalized, fields := normalize(input)
	if len(fields) > 0 {
		return Customer{}, fields, nil
	}
	item, err := s.repository.Update(ctx, tenantID, customerID, normalized)
	if err != nil {
		return Customer{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "CUSTOMER_UPDATED", "customer", &item.ID, map[string]any{
		"name":   item.DisplayName,
		"source": item.Source,
	})
	return item, nil, nil
}

func normalize(input CreateInput) (normalizedInput, map[string]string) {
	result := normalizedInput{
		FirstName:             strings.TrimSpace(input.FirstName),
		LastName:              strings.TrimSpace(input.LastName),
		Phone:                 cleanOptional(input.Phone),
		Email:                 cleanOptional(input.Email),
		CompanyName:           cleanOptional(input.CompanyName),
		TaxID:                 strings.TrimSpace(input.TaxID),
		TaxRegistrationNumber: strings.TrimSpace(input.TaxRegistrationNumber),
		BillingAddress:        strings.TrimSpace(input.BillingAddress),
		DocumentTypeCode:      strings.TrimSpace(input.DocumentTypeCode),
		TradeName:             strings.TrimSpace(input.TradeName),
		EconomicActivity:      strings.TrimSpace(input.EconomicActivity),
		EconomicActivityCode:  strings.TrimSpace(input.EconomicActivityCode),
		DepartmentCode:        strings.TrimSpace(input.DepartmentCode),
		MunicipalityCode:      strings.TrimSpace(input.MunicipalityCode),
		DistrictCode:          strings.TrimSpace(input.DistrictCode),
		PreferredLanguage:     strings.ToLower(strings.TrimSpace(input.PreferredLanguage)),
		Source:                strings.ToUpper(strings.TrimSpace(input.Source)),
		Notes:                 strings.TrimSpace(input.Notes),
	}
	if result.PreferredLanguage == "" {
		result.PreferredLanguage = "es"
	}
	if result.DocumentTypeCode == "" {
		result.DocumentTypeCode = "36"
	}
	if result.Source == "" {
		result.Source = "MANUAL"
	}
	if result.Phone != nil {
		value := normalizePhone(*result.Phone)
		result.Phone = &value
	}
	if result.Email != nil {
		value := strings.ToLower(*result.Email)
		result.Email = &value
	}

	fields := map[string]string{}
	if result.FirstName == "" {
		fields["first_name"] = "First name is required."
	}
	if len(result.FirstName) > 120 {
		fields["first_name"] = "First name must be 120 characters or fewer."
	}
	if len(result.LastName) > 120 {
		fields["last_name"] = "Last name must be 120 characters or fewer."
	}
	if result.CompanyName != nil && len(*result.CompanyName) > 180 {
		fields["company_name"] = "Company name must be 180 characters or fewer."
	}
	if len(result.TaxID) > 40 {
		fields["tax_id"] = "Tax ID must be 40 characters or fewer."
	}
	if len(result.TaxRegistrationNumber) > 40 {
		fields["tax_registration_number"] = "Tax registration number must be 40 characters or fewer."
	}
	if len(result.BillingAddress) > 4000 {
		fields["billing_address"] = "Billing address must be 4,000 characters or fewer."
	}
	if len(result.DocumentTypeCode) > 4 {
		fields["document_type_code"] = "Document type code must be 4 characters or fewer."
	}
	if len(result.TradeName) > 240 {
		fields["trade_name"] = "Trade name must be 240 characters or fewer."
	}
	if len(result.EconomicActivity) > 240 {
		fields["economic_activity"] = "Economic activity must be 240 characters or fewer."
	}
	if len(result.EconomicActivityCode) > 12 {
		fields["economic_activity_code"] = "Economic activity code must be 12 characters or fewer."
	}
	if len(result.DepartmentCode) > 4 {
		fields["department_code"] = "Department code must be 4 characters or fewer."
	}
	if len(result.MunicipalityCode) > 4 {
		fields["municipality_code"] = "Municipality code must be 4 characters or fewer."
	}
	if len(result.DistrictCode) > 8 {
		fields["district_code"] = "District code must be 8 characters or fewer."
	}
	if result.Phone != nil && !e164Pattern.MatchString(*result.Phone) {
		fields["phone"] = "Use international E.164 format, for example +50371234567."
	}
	if result.Email != nil {
		parsed, err := mail.ParseAddress(*result.Email)
		if err != nil || !strings.EqualFold(parsed.Address, *result.Email) {
			fields["email"] = "Enter a valid email address."
		}
	}
	if result.PreferredLanguage != "es" && result.PreferredLanguage != "en" {
		fields["preferred_language"] = "Supported languages are es and en."
	}
	if !contains([]string{"WEB", "WHATSAPP", "MANUAL", "IMPORT"}, result.Source) {
		fields["source"] = "Unsupported customer source."
	}
	if len(result.Notes) > 4000 {
		fields["notes"] = "Notes must be 4,000 characters or fewer."
	}
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

func normalizePhone(value string) string {
	var builder strings.Builder
	for index, character := range value {
		if character >= '0' && character <= '9' {
			builder.WriteRune(character)
		}
		if character == '+' && index == 0 {
			builder.WriteRune(character)
		}
	}
	return builder.String()
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
