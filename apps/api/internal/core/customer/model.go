package customer

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("customer not found")

type Customer struct {
	ID                    string     `json:"id"`
	FirstName             string     `json:"first_name"`
	LastName              string     `json:"last_name"`
	DisplayName           string     `json:"display_name"`
	Phone                 *string    `json:"phone,omitempty"`
	Email                 *string    `json:"email,omitempty"`
	CompanyName           *string    `json:"company_name,omitempty"`
	TaxID                 string     `json:"tax_id"`
	TaxRegistrationNumber string     `json:"tax_registration_number"`
	BillingAddress        string     `json:"billing_address"`
	DocumentTypeCode      string     `json:"document_type_code"`
	TradeName             string     `json:"trade_name"`
	EconomicActivity      string     `json:"economic_activity"`
	EconomicActivityCode  string     `json:"economic_activity_code"`
	DepartmentCode        string     `json:"department_code"`
	MunicipalityCode      string     `json:"municipality_code"`
	DistrictCode          string     `json:"district_code"`
	PreferredLanguage     string     `json:"preferred_language"`
	Source                string     `json:"source"`
	Notes                 string     `json:"notes"`
	QuoteCount            int        `json:"quote_count"`
	AcceptedQuoteCount    int        `json:"accepted_quote_count"`
	AcceptedQuoteRevenue  float64    `json:"accepted_quote_revenue"`
	LastQuoteAt           *time.Time `json:"last_quote_at,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

type CreateInput struct {
	FirstName             string  `json:"first_name"`
	LastName              string  `json:"last_name"`
	Phone                 *string `json:"phone"`
	Email                 *string `json:"email"`
	CompanyName           *string `json:"company_name"`
	TaxID                 string  `json:"tax_id"`
	TaxRegistrationNumber string  `json:"tax_registration_number"`
	BillingAddress        string  `json:"billing_address"`
	DocumentTypeCode      string  `json:"document_type_code"`
	TradeName             string  `json:"trade_name"`
	EconomicActivity      string  `json:"economic_activity"`
	EconomicActivityCode  string  `json:"economic_activity_code"`
	DepartmentCode        string  `json:"department_code"`
	MunicipalityCode      string  `json:"municipality_code"`
	DistrictCode          string  `json:"district_code"`
	PreferredLanguage     string  `json:"preferred_language"`
	Source                string  `json:"source"`
	Notes                 string  `json:"notes"`
}

type UpdateInput = CreateInput

type normalizedInput struct {
	FirstName             string
	LastName              string
	Phone                 *string
	Email                 *string
	CompanyName           *string
	TaxID                 string
	TaxRegistrationNumber string
	BillingAddress        string
	DocumentTypeCode      string
	TradeName             string
	EconomicActivity      string
	EconomicActivityCode  string
	DepartmentCode        string
	MunicipalityCode      string
	DistrictCode          string
	PreferredLanguage     string
	Source                string
	Notes                 string
}
