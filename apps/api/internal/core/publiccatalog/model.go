package publiccatalog

import (
	"errors"
	"time"

	"github.com/rentstage/rentstage/apps/api/internal/core/availability"
	"github.com/rentstage/rentstage/apps/api/internal/core/packages"
)

var (
	ErrCatalogNotFound         = errors.New("public catalog not found")
	ErrPackageNotPublic        = errors.New("package is not public")
	ErrResourceNotPublic       = errors.New("resource is not public")
	ErrQuoteRequestsDisabled   = errors.New("quote requests disabled")
	ErrQuoteRequestNotFound    = errors.New("quote request not found")
	ErrQuoteRequestConflict    = errors.New("quote request conflict")
	ErrQuoteRequestRateLimited = errors.New("quote request rate limited")
	ErrConversionConflict      = errors.New("quote request conversion conflict")
	ErrPublicationConflict     = errors.New("public catalog publication conflict")
)

const (
	QuoteRequestStatusNew       = "NEW"
	QuoteRequestStatusInReview  = "IN_REVIEW"
	QuoteRequestStatusConverted = "CONVERTED"
	QuoteRequestStatusClosed    = "CLOSED"
	QuoteRequestStatusSpam      = "SPAM"
)

type Settings struct {
	TenantID             string    `json:"tenant_id"`
	Enabled              bool      `json:"enabled"`
	Headline             string    `json:"headline"`
	Description          string    `json:"description"`
	CoverImageURL        *string   `json:"cover_image_url,omitempty"`
	AccentColor          string    `json:"accent_color"`
	ShowPrices           bool      `json:"show_prices"`
	ShowResources        bool      `json:"show_resources"`
	QuoteRequestsEnabled bool      `json:"quote_requests_enabled"`
	ContactEmail         *string   `json:"contact_email,omitempty"`
	ContactPhone         *string   `json:"contact_phone,omitempty"`
	ContactAddress       *string   `json:"contact_address,omitempty"`
	TermsText            string    `json:"terms_text"`
	TermsVersion         string    `json:"terms_version"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type SettingsInput struct {
	Enabled              bool    `json:"enabled"`
	Headline             string  `json:"headline"`
	Description          string  `json:"description"`
	CoverImageURL        *string `json:"cover_image_url"`
	AccentColor          string  `json:"accent_color"`
	ShowPrices           bool    `json:"show_prices"`
	ShowResources        bool    `json:"show_resources"`
	QuoteRequestsEnabled bool    `json:"quote_requests_enabled"`
	ContactEmail         *string `json:"contact_email"`
	ContactPhone         *string `json:"contact_phone"`
	ContactAddress       *string `json:"contact_address"`
	TermsText            string  `json:"terms_text"`
	TermsVersion         string  `json:"terms_version"`
}

type PackagePublicationInput struct {
	Visible   bool `json:"visible"`
	Featured  bool `json:"featured"`
	SortOrder int  `json:"sort_order"`
}

type ResourcePublicationInput struct {
	PublicSlug        string  `json:"public_slug"`
	PublicDescription string  `json:"public_description"`
	PublicImageURL    *string `json:"public_image_url"`
	Visible           bool    `json:"visible"`
	Featured          bool    `json:"featured"`
	SortOrder         int     `json:"sort_order"`
}

type AdminPackage struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Slug            string  `json:"slug"`
	Description     string  `json:"description"`
	GuestCapacity   *int    `json:"guest_capacity,omitempty"`
	ImageURL        *string `json:"image_url,omitempty"`
	EffectivePrice  float64 `json:"effective_price"`
	ItemCount       int     `json:"item_count"`
	TotalQuantity   int     `json:"total_quantity"`
	Ready           bool    `json:"ready"`
	Active          bool    `json:"active"`
	PublicVisible   bool    `json:"public_visible"`
	PublicFeatured  bool    `json:"public_featured"`
	PublicSortOrder int     `json:"public_sort_order"`
}

type AdminResource struct {
	ID                string  `json:"id"`
	CategoryName      *string `json:"category_name,omitempty"`
	ResourceType      string  `json:"resource_type"`
	Name              string  `json:"name"`
	Description       string  `json:"description"`
	BasePrice         float64 `json:"base_price"`
	PricingUnit       string  `json:"pricing_unit"`
	Active            bool    `json:"active"`
	PublicSlug        *string `json:"public_slug,omitempty"`
	PublicDescription string  `json:"public_description"`
	PublicImageURL    *string `json:"public_image_url,omitempty"`
	PublicVisible     bool    `json:"public_visible"`
	PublicFeatured    bool    `json:"public_featured"`
	PublicSortOrder   int     `json:"public_sort_order"`
}

type AdminCatalog struct {
	Settings  Settings        `json:"settings"`
	Tenant    PublicTenant    `json:"tenant"`
	PublicURL string          `json:"public_url"`
	Packages  []AdminPackage  `json:"packages"`
	Resources []AdminResource `json:"resources"`
}

type PublicTenant struct {
	Name     string  `json:"name"`
	Slug     string  `json:"slug"`
	LogoURL  *string `json:"logo_url,omitempty"`
	Email    *string `json:"email,omitempty"`
	Phone    *string `json:"phone,omitempty"`
	Address  *string `json:"address,omitempty"`
	Currency string  `json:"currency"`
	Timezone string  `json:"timezone"`
}

type PublicSettings struct {
	Headline             string  `json:"headline"`
	Description          string  `json:"description"`
	CoverImageURL        *string `json:"cover_image_url,omitempty"`
	AccentColor          string  `json:"accent_color"`
	ShowPrices           bool    `json:"show_prices"`
	ShowResources        bool    `json:"show_resources"`
	QuoteRequestsEnabled bool    `json:"quote_requests_enabled"`
	ContactEmail         *string `json:"contact_email,omitempty"`
	ContactPhone         *string `json:"contact_phone,omitempty"`
	ContactAddress       *string `json:"contact_address,omitempty"`
	TermsText            string  `json:"terms_text"`
	TermsVersion         string  `json:"terms_version"`
}

type PublicPackageSummary struct {
	Name           string   `json:"name"`
	Slug           string   `json:"slug"`
	Description    string   `json:"description"`
	GuestCapacity  *int     `json:"guest_capacity,omitempty"`
	ImageURL       *string  `json:"image_url,omitempty"`
	EffectivePrice *float64 `json:"effective_price,omitempty"`
	ItemCount      int      `json:"item_count"`
	TotalQuantity  int      `json:"total_quantity"`
	Featured       bool     `json:"featured"`
}

type PublicPackageItem struct {
	ResourceName string `json:"resource_name"`
	Description  string `json:"description"`
	Quantity     int    `json:"quantity"`
}

type PublicPackageDetail struct {
	PublicPackageSummary
	Items []PublicPackageItem `json:"items"`
}

type PublicResource struct {
	Slug         string   `json:"slug"`
	CategoryName *string  `json:"category_name,omitempty"`
	ResourceType string   `json:"resource_type"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	ImageURL     *string  `json:"image_url,omitempty"`
	BasePrice    *float64 `json:"base_price,omitempty"`
	PricingUnit  string   `json:"pricing_unit"`
	Featured     bool     `json:"featured"`
}

type PublicCatalog struct {
	Tenant    PublicTenant           `json:"tenant"`
	Settings  PublicSettings         `json:"settings"`
	Packages  []PublicPackageSummary `json:"packages"`
	Resources []PublicResource       `json:"resources"`
}

type QuoteRequestSelection struct {
	PackageSlug string `json:"package_slug"`
	Quantity    int    `json:"quantity"`
}

type AvailabilityInput struct {
	StartAt    string                  `json:"start_at"`
	EndAt      string                  `json:"end_at"`
	Selections []QuoteRequestSelection `json:"selections"`
}

type PublicAvailabilityItem struct {
	ResourceName      string `json:"resource_name"`
	RequestedQuantity int    `json:"requested_quantity"`
	CanFulfill        bool   `json:"can_fulfill"`
}

type PublicAvailabilityResult struct {
	StartAt   time.Time                `json:"start_at"`
	EndAt     time.Time                `json:"end_at"`
	Available bool                     `json:"available"`
	Items     []PublicAvailabilityItem `json:"items"`
}

type QuoteRequestInput struct {
	FirstName         string                  `json:"first_name"`
	LastName          string                  `json:"last_name"`
	Phone             *string                 `json:"phone"`
	Email             *string                 `json:"email"`
	CompanyName       *string                 `json:"company_name"`
	PreferredLanguage string                  `json:"preferred_language"`
	EventType         *string                 `json:"event_type"`
	EventLocation     *string                 `json:"event_location"`
	StartAt           string                  `json:"start_at"`
	EndAt             string                  `json:"end_at"`
	Notes             string                  `json:"notes"`
	ConsentAccepted   bool                    `json:"consent_accepted"`
	Website           string                  `json:"website"`
	Selections        []QuoteRequestSelection `json:"selections"`
}

type QuoteRequestReceipt struct {
	RequestID             string                   `json:"-"`
	ReferenceCode         string                   `json:"reference_code"`
	Status                string                   `json:"status"`
	EstimatedTotal        *float64                 `json:"estimated_total,omitempty"`
	AvailabilityAvailable bool                     `json:"availability_available"`
	Availability          PublicAvailabilityResult `json:"availability"`
	CreatedAt             time.Time                `json:"created_at"`
}

type RequestPackage struct {
	ID          string                 `json:"id"`
	PackageID   *string                `json:"package_id,omitempty"`
	PackageName string                 `json:"package_name"`
	PackageSlug string                 `json:"package_slug"`
	Quantity    int                    `json:"quantity"`
	UnitPrice   float64                `json:"unit_price"`
	LineTotal   float64                `json:"line_total"`
	Template    packages.QuoteTemplate `json:"template"`
	CreatedAt   time.Time              `json:"created_at"`
}

type RequestItem struct {
	ID             string    `json:"id"`
	ResourceID     string    `json:"resource_id"`
	ResourceName   string    `json:"resource_name"`
	Description    string    `json:"description"`
	Quantity       int       `json:"quantity"`
	UnitPrice      float64   `json:"unit_price"`
	DiscountAmount float64   `json:"discount_amount"`
	LineTotal      float64   `json:"line_total"`
	CreatedAt      time.Time `json:"created_at"`
}

type QuoteRequestSummary struct {
	ID                    string     `json:"id"`
	ReferenceCode         string     `json:"reference_code"`
	Status                string     `json:"status"`
	CustomerName          string     `json:"customer_name"`
	Phone                 *string    `json:"phone,omitempty"`
	Email                 *string    `json:"email,omitempty"`
	EventType             *string    `json:"event_type,omitempty"`
	EventLocation         *string    `json:"event_location,omitempty"`
	StartAt               time.Time  `json:"start_at"`
	EndAt                 time.Time  `json:"end_at"`
	EstimatedTotal        float64    `json:"estimated_total"`
	Currency              string     `json:"currency"`
	AvailabilityAvailable bool       `json:"availability_available"`
	PackageCount          int        `json:"package_count"`
	ConvertedQuoteID      *string    `json:"converted_quote_id,omitempty"`
	HandledAt             *time.Time `json:"handled_at,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

type QuoteRequestDetail struct {
	QuoteRequestSummary
	FirstName             string              `json:"first_name"`
	LastName              string              `json:"last_name"`
	CompanyName           *string             `json:"company_name,omitempty"`
	PreferredLanguage     string              `json:"preferred_language"`
	Notes                 string              `json:"notes"`
	EstimatedSubtotal     float64             `json:"estimated_subtotal"`
	EstimatedDiscount     float64             `json:"estimated_discount_amount"`
	EstimatedExtraCharges float64             `json:"estimated_extra_charges"`
	Availability          availability.Result `json:"availability"`
	TermsText             string              `json:"terms_text"`
	TermsVersion          string              `json:"terms_version"`
	ConsentAccepted       bool                `json:"consent_accepted"`
	ConvertedCustomerID   *string             `json:"converted_customer_id,omitempty"`
	Packages              []RequestPackage    `json:"packages"`
	Items                 []RequestItem       `json:"items"`
}

type QuoteRequestList struct {
	Items  []QuoteRequestSummary `json:"items"`
	Counts map[string]int        `json:"counts"`
}

type QuoteRequestStatusInput struct {
	Status string `json:"status"`
}

type ConversionResult struct {
	RequestID     string `json:"request_id"`
	ReferenceCode string `json:"reference_code"`
	CustomerID    string `json:"customer_id"`
	QuoteID       string `json:"quote_id"`
	QuoteNumber   int64  `json:"quote_number"`
}

type normalizedSettings struct {
	Enabled              bool
	Headline             string
	Description          string
	CoverImageURL        *string
	AccentColor          string
	ShowPrices           bool
	ShowResources        bool
	QuoteRequestsEnabled bool
	ContactEmail         *string
	ContactPhone         *string
	ContactAddress       *string
	TermsText            string
	TermsVersion         string
}

type normalizedQuoteRequest struct {
	FirstName         string
	LastName          string
	Phone             *string
	Email             *string
	CompanyName       *string
	PreferredLanguage string
	EventType         *string
	EventLocation     *string
	StartAt           time.Time
	EndAt             time.Time
	Notes             string
	ConsentAccepted   bool
	Selections        []QuoteRequestSelection
}

type preparedPackage struct {
	PackageID   string
	PackageName string
	PackageSlug string
	Quantity    int
	UnitPrice   float64
	LineTotal   float64
	Template    packages.QuoteTemplate
}

type preparedLine struct {
	ResourceID     string
	ResourceName   string
	Description    string
	Quantity       int
	UnitPrice      float64
	DiscountAmount float64
	LineTotal      float64
}

type preparedRequest struct {
	Input                 normalizedQuoteRequest
	EstimatedSubtotal     float64
	EstimatedDiscount     float64
	EstimatedExtraCharges float64
	EstimatedTotal        float64
	Availability          availability.Result
	TermsText             string
	TermsVersion          string
	SubmitterHash         string
	UserAgent             string
	Packages              []preparedPackage
	Items                 []preparedLine
}
