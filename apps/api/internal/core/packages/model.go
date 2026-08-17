package packages

import (
	"errors"
	"time"

	"github.com/rentstage/rentstage/apps/api/internal/core/availability"
)

var (
	ErrNotFound         = errors.New("package not found")
	ErrConflict         = errors.New("package conflict")
	ErrResourceNotFound = errors.New("package resource not found")
	ErrUnavailable      = errors.New("package unavailable")
)

const (
	PricingModeSumItems = "SUM_ITEMS"
	PricingModeFixed    = "FIXED"
)

type Summary struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	Slug                 string    `json:"slug"`
	Description          string    `json:"description"`
	GuestCapacity        *int      `json:"guest_capacity,omitempty"`
	PricingMode          string    `json:"pricing_mode"`
	FixedPrice           *float64  `json:"fixed_price,omitempty"`
	ImageURL             *string   `json:"image_url,omitempty"`
	PublicVisible        bool      `json:"public_visible"`
	PublicFeatured       bool      `json:"public_featured"`
	PublicSortOrder      int       `json:"public_sort_order"`
	CalculatedPrice      float64   `json:"calculated_price"`
	EffectivePrice       float64   `json:"effective_price"`
	DiscountValue        float64   `json:"discount_value"`
	SurchargeValue       float64   `json:"surcharge_value"`
	ItemCount            int       `json:"item_count"`
	TotalQuantity        int       `json:"total_quantity"`
	UnavailableItemCount int       `json:"unavailable_item_count"`
	Ready                bool      `json:"ready"`
	Active               bool      `json:"active"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type Item struct {
	ID                  string    `json:"id"`
	ResourceID          string    `json:"resource_id"`
	ResourceName        string    `json:"resource_name"`
	ResourceType        string    `json:"resource_type"`
	PricingUnit         string    `json:"pricing_unit"`
	ResourceActive      bool      `json:"resource_active"`
	Description         string    `json:"description"`
	Quantity            int       `json:"quantity"`
	BasePrice           float64   `json:"base_price"`
	UnitPriceOverride   *float64  `json:"unit_price_override,omitempty"`
	UnitPrice           float64   `json:"unit_price"`
	LineTotal           float64   `json:"line_total"`
	AssetCount          int       `json:"asset_count"`
	AvailableAssetCount int       `json:"available_asset_count"`
	AttentionAssetCount int       `json:"attention_asset_count"`
	SortOrder           int       `json:"sort_order"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type Detail struct {
	Summary
	Items         []Item        `json:"items"`
	QuoteTemplate QuoteTemplate `json:"quote_template"`
}

type ItemInput struct {
	ResourceID        string   `json:"resource_id"`
	Description       string   `json:"description"`
	Quantity          int      `json:"quantity"`
	UnitPriceOverride *float64 `json:"unit_price_override"`
	SortOrder         int      `json:"sort_order"`
}

type CreateInput struct {
	Name          string      `json:"name"`
	Slug          string      `json:"slug"`
	Description   string      `json:"description"`
	GuestCapacity *int        `json:"guest_capacity"`
	PricingMode   string      `json:"pricing_mode"`
	FixedPrice    *float64    `json:"fixed_price"`
	ImageURL      *string     `json:"image_url"`
	Active        *bool       `json:"active"`
	Items         []ItemInput `json:"items"`
}

type QuoteTemplateItem struct {
	ResourceID     string  `json:"resource_id"`
	ResourceName   string  `json:"resource_name"`
	Description    string  `json:"description"`
	Quantity       int     `json:"quantity"`
	UnitPrice      float64 `json:"unit_price"`
	DiscountAmount float64 `json:"discount_amount"`
	LineTotal      float64 `json:"line_total"`
}

type QuoteTemplate struct {
	PackageID       string              `json:"package_id"`
	PackageName     string              `json:"package_name"`
	PackageQuantity int                 `json:"package_quantity"`
	PricingMode     string              `json:"pricing_mode"`
	CalculatedPrice float64             `json:"calculated_price"`
	EffectivePrice  float64             `json:"effective_price"`
	DiscountAmount  float64             `json:"discount_amount"`
	ExtraCharges    float64             `json:"extra_charges"`
	Items           []QuoteTemplateItem `json:"items"`
}

type AvailabilityInput struct {
	StartAt  string `json:"start_at"`
	EndAt    string `json:"end_at"`
	Quantity int    `json:"quantity"`
}

type AvailabilityResult struct {
	PackageID       string                    `json:"package_id"`
	PackageName     string                    `json:"package_name"`
	PackageQuantity int                       `json:"package_quantity"`
	StartAt         time.Time                 `json:"start_at"`
	EndAt           time.Time                 `json:"end_at"`
	Available       bool                      `json:"available"`
	Items           []availability.ItemResult `json:"items"`
}

type normalizedItem struct {
	ResourceID        string
	Description       string
	Quantity          int
	UnitPriceOverride *float64
	SortOrder         int
}

type normalizedInput struct {
	Name          string
	Slug          string
	Description   string
	GuestCapacity *int
	PricingMode   string
	FixedPrice    *float64
	ImageURL      *string
	Active        bool
	Items         []normalizedItem
}
