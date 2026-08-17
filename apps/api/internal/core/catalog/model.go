package catalog

import (
	"errors"
	"time"

	"github.com/rentstage/rentstage/apps/api/internal/webutil"
)

var (
	ErrNotFound = errors.New("catalog item not found")
	ErrConflict = errors.New("catalog conflict")
)

type Category struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	ResourceCount int       `json:"resource_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Resource struct {
	ID                    string         `json:"id"`
	CategoryID            *string        `json:"category_id,omitempty"`
	CategoryName          *string        `json:"category_name,omitempty"`
	ResourceType          string         `json:"resource_type"`
	Name                  string         `json:"name"`
	Description           string         `json:"description"`
	SKU                   *string        `json:"sku,omitempty"`
	BasePrice             float64        `json:"base_price"`
	PricingUnit           string         `json:"pricing_unit"`
	DepositAmount         float64        `json:"deposit_amount"`
	TrackIndividualAssets bool           `json:"track_individual_assets"`
	Active                bool           `json:"active"`
	Metadata              map[string]any `json:"metadata"`
	AssetCount            int            `json:"asset_count"`
	AvailableAssetCount   int            `json:"available_asset_count"`
	AttentionAssetCount   int            `json:"attention_asset_count"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
}

type CreateCategoryInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type CreateResourceInput struct {
	CategoryID            *string        `json:"category_id"`
	ResourceType          string         `json:"resource_type"`
	Name                  string         `json:"name"`
	Description           string         `json:"description"`
	SKU                   *string        `json:"sku"`
	BasePrice             float64        `json:"base_price"`
	PricingUnit           string         `json:"pricing_unit"`
	DepositAmount         float64        `json:"deposit_amount"`
	TrackIndividualAssets *bool          `json:"track_individual_assets"`
	Active                *bool          `json:"active"`
	Metadata              map[string]any `json:"metadata"`
}

type UpdateResourceInput struct {
	CategoryID            webutil.Optional[string] `json:"category_id"`
	ResourceType          *string                  `json:"resource_type"`
	Name                  *string                  `json:"name"`
	Description           *string                  `json:"description"`
	SKU                   webutil.Optional[string] `json:"sku"`
	BasePrice             *float64                 `json:"base_price"`
	PricingUnit           *string                  `json:"pricing_unit"`
	DepositAmount         *float64                 `json:"deposit_amount"`
	TrackIndividualAssets *bool                    `json:"track_individual_assets"`
	Active                *bool                    `json:"active"`
	Metadata              *map[string]any          `json:"metadata"`
}
