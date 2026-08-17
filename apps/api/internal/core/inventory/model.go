package inventory

import (
	"errors"
	"time"

	"github.com/rentstage/rentstage/apps/api/internal/webutil"
)

var (
	ErrNotFound = errors.New("inventory item not found")
	ErrConflict = errors.New("inventory conflict")
)

type Asset struct {
	ID             string     `json:"id"`
	ResourceID     string     `json:"resource_id"`
	ResourceName   string     `json:"resource_name"`
	AssetCode      string     `json:"asset_code"`
	SerialNumber   *string    `json:"serial_number,omitempty"`
	PhysicalStatus string     `json:"physical_status"`
	PurchaseDate   *time.Time `json:"purchase_date,omitempty"`
	PurchasePrice  *float64   `json:"purchase_price,omitempty"`
	Notes          string     `json:"notes"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type CreateAssetInput struct {
	AssetCode      string   `json:"asset_code"`
	SerialNumber   *string  `json:"serial_number"`
	PhysicalStatus string   `json:"physical_status"`
	PurchaseDate   *string  `json:"purchase_date"`
	PurchasePrice  *float64 `json:"purchase_price"`
	Notes          string   `json:"notes"`
}

type UpdateAssetInput struct {
	AssetCode      *string                   `json:"asset_code"`
	SerialNumber   webutil.Optional[string]  `json:"serial_number"`
	PhysicalStatus *string                   `json:"physical_status"`
	PurchaseDate   webutil.Optional[string]  `json:"purchase_date"`
	PurchasePrice  webutil.Optional[float64] `json:"purchase_price"`
	Notes          *string                   `json:"notes"`
}
