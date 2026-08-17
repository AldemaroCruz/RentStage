package availability

import (
	"errors"
	"time"
)

var ErrInvalidPeriod = errors.New("invalid availability period")

type ResourceNotFoundError struct {
	ResourceID string
}

func (e *ResourceNotFoundError) Error() string {
	return "availability resource not found: " + e.ResourceID
}

type ItemInput struct {
	ResourceID string `json:"resource_id"`
	Quantity   int    `json:"quantity"`
}

type CheckInput struct {
	StartAt string      `json:"start_at"`
	EndAt   string      `json:"end_at"`
	Items   []ItemInput `json:"items"`
}

type NormalizedItem struct {
	ResourceID string
	Quantity   int
}

type NormalizedInput struct {
	StartAt              time.Time
	EndAt                time.Time
	Items                []NormalizedItem
	ExcludeReservationID *string
}

type ItemResult struct {
	ResourceID        string `json:"resource_id"`
	ResourceName      string `json:"resource_name"`
	RequestedQuantity int    `json:"requested_quantity"`
	EligibleAssets    int    `json:"eligible_assets"`
	ReservedQuantity  int    `json:"reserved_quantity"`
	AvailableQuantity int    `json:"available_quantity"`
	CanFulfill        bool   `json:"can_fulfill"`
}

// SingleResult preserves the v0.2 response contract of the original
// single-resource GET endpoint while the bulk POST endpoint uses Result.
type SingleResult struct {
	ResourceID        string    `json:"resource_id"`
	ResourceName      string    `json:"resource_name"`
	Start             time.Time `json:"start"`
	End               time.Time `json:"end"`
	RequestedQuantity int       `json:"requested_quantity"`
	EligibleAssets    int       `json:"eligible_assets"`
	ReservedQuantity  int       `json:"reserved_quantity"`
	AvailableQuantity int       `json:"available_quantity"`
	CanFulfill        bool      `json:"can_fulfill"`
}

type Result struct {
	StartAt   time.Time    `json:"start_at"`
	EndAt     time.Time    `json:"end_at"`
	Available bool         `json:"available"`
	Items     []ItemResult `json:"items"`
}
