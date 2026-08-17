package reservation

import (
	"errors"
	"time"

	"github.com/rentstage/rentstage/apps/api/internal/core/availability"
)

var (
	ErrNotFound              = errors.New("reservation not found")
	ErrQuoteNotFound         = errors.New("quote not found")
	ErrQuoteNotAccepted      = errors.New("quote is not accepted")
	ErrQuoteStatus           = errors.New("quote status does not permit conversion")
	ErrInvalidTransition     = errors.New("invalid reservation status transition")
	ErrWarehouseState        = errors.New("reservation is not in a warehouse-operable state")
	ErrAssetNotFound         = errors.New("asset not found")
	ErrAssetUnavailable      = errors.New("asset is not physically available")
	ErrAssetResourceMismatch = errors.New("asset does not belong to a requested resource")
	ErrAssignmentNotFound    = errors.New("active asset assignment not found")
	ErrAssetAlreadyAssigned  = errors.New("asset is already assigned to this reservation")
	ErrAssignmentCapacity    = errors.New("reservation item already has all required assets")
	ErrReturnPayload         = errors.New("return payload does not match checked-out assets")
	ErrAssetsNotReturned     = errors.New("one or more checked-out assets have not been returned")
	ErrCustomerNotFound      = errors.New("reservation customer not found")
	ErrRescheduleState       = errors.New("reservation cannot be rescheduled from its current state")
)

// QuoteConversionResult is returned by the transactional quote conversion
// helper so customer portals can create a reservation and accept the quote in
// one PostgreSQL transaction without loading the public-facing detail first.
type QuoteConversionResult struct {
	ReservationID     string
	ReservationNumber int64
}

type QuoteAlreadyConvertedError struct {
	ReservationID string
}

func (e *QuoteAlreadyConvertedError) Error() string {
	return "quote already converted to reservation"
}

type AvailabilityConflictError struct {
	Result availability.Result
}

func (e *AvailabilityConflictError) Error() string {
	return "reservation availability conflict"
}

type AssetConflictError struct {
	AssetID           string `json:"asset_id"`
	AssetCode         string `json:"asset_code"`
	ReservationID     string `json:"reservation_id"`
	ReservationNumber int64  `json:"reservation_number"`
}

func (e *AssetConflictError) Error() string {
	return "asset is assigned to an overlapping reservation"
}

// AssetScheduleConflictError reports an exact physical unit that would overlap
// another blocking reservation after a schedule change.
type AssetScheduleConflictError struct {
	AssetID           string `json:"asset_id"`
	AssetCode         string `json:"asset_code"`
	ReservationID     string `json:"reservation_id"`
	ReservationNumber int64  `json:"reservation_number"`
}

func (e *AssetScheduleConflictError) Error() string {
	return "assigned asset conflicts with the requested schedule"
}

type InventoryGap struct {
	ReservationItemID string `json:"reservation_item_id"`
	ResourceID        string `json:"resource_id"`
	ResourceName      string `json:"resource_name"`
	RequiredQuantity  int    `json:"required_quantity"`
	AssignedQuantity  int    `json:"assigned_quantity"`
	MissingQuantity   int    `json:"missing_quantity"`
}

type InventoryIncompleteError struct {
	Items []InventoryGap `json:"items"`
}

func (e *InventoryIncompleteError) Error() string {
	return "reservation inventory is incomplete"
}

type ReturnMismatchError struct {
	ExpectedAssetIDs   []string `json:"expected_asset_ids"`
	MissingAssetIDs    []string `json:"missing_asset_ids"`
	UnexpectedAssetIDs []string `json:"unexpected_asset_ids"`
}

func (e *ReturnMismatchError) Error() string {
	return "return assets do not match checked-out assignments"
}

type AssignedAsset struct {
	AssignmentID    string     `json:"assignment_id"`
	AssetID         string     `json:"asset_id"`
	AssetCode       string     `json:"asset_code"`
	SerialNumber    *string    `json:"serial_number,omitempty"`
	PhysicalStatus  string     `json:"physical_status"`
	State           string     `json:"state"`
	AssignedAt      time.Time  `json:"assigned_at"`
	AssignedBy      string     `json:"assigned_by"`
	CheckedOutAt    *time.Time `json:"checked_out_at,omitempty"`
	CheckedOutBy    *string    `json:"checked_out_by,omitempty"`
	ReturnedAt      *time.Time `json:"returned_at,omitempty"`
	ReturnedBy      *string    `json:"returned_by,omitempty"`
	ReturnCondition *string    `json:"return_condition,omitempty"`
	ReturnNotes     string     `json:"return_notes"`
	ReleasedAt      *time.Time `json:"released_at,omitempty"`
	ReleasedBy      *string    `json:"released_by,omitempty"`
	ReleaseReason   string     `json:"release_reason"`
}

type Item struct {
	ID                    string          `json:"id"`
	ResourceID            string          `json:"resource_id"`
	ResourceName          string          `json:"resource_name"`
	Description           string          `json:"description"`
	Quantity              int             `json:"quantity"`
	UnitPrice             float64         `json:"unit_price"`
	DiscountAmount        float64         `json:"discount_amount"`
	LineTotal             float64         `json:"line_total"`
	TrackIndividualAssets bool            `json:"track_individual_assets"`
	AssignedQuantity      int             `json:"assigned_quantity"`
	MissingQuantity       int             `json:"missing_quantity"`
	Assignments           []AssignedAsset `json:"assignments"`
}

type StatusHistory struct {
	ID         string    `json:"id"`
	FromStatus *string   `json:"from_status,omitempty"`
	ToStatus   string    `json:"to_status"`
	ActorID    string    `json:"actor_id"`
	Note       string    `json:"note"`
	CreatedAt  time.Time `json:"created_at"`
}

type ActivityEvent struct {
	ID           string         `json:"id"`
	EventType    string         `json:"event_type"`
	AssetID      *string        `json:"asset_id,omitempty"`
	AssetCode    *string        `json:"asset_code,omitempty"`
	ResourceName *string        `json:"resource_name,omitempty"`
	ActorID      string         `json:"actor_id"`
	Note         string         `json:"note"`
	Metadata     map[string]any `json:"metadata"`
	CreatedAt    time.Time      `json:"created_at"`
}

type Summary struct {
	ID                string     `json:"id"`
	ReservationNumber int64      `json:"reservation_number"`
	CustomerID        string     `json:"customer_id"`
	CustomerName      string     `json:"customer_name"`
	CustomerPhone     *string    `json:"customer_phone,omitempty"`
	QuoteID           *string    `json:"quote_id,omitempty"`
	QuoteNumber       *int64     `json:"quote_number,omitempty"`
	Source            string     `json:"source"`
	Status            string     `json:"status"`
	BlockStartAt      time.Time  `json:"block_start_at"`
	BlockEndAt        time.Time  `json:"block_end_at"`
	EventStartAt      time.Time  `json:"event_start_at"`
	EventEndAt        time.Time  `json:"event_end_at"`
	EventType         *string    `json:"event_type,omitempty"`
	EventLocation     *string    `json:"event_location,omitempty"`
	Subtotal          float64    `json:"subtotal"`
	DiscountAmount    float64    `json:"discount_amount"`
	ExtraCharges      float64    `json:"extra_charges"`
	Total             float64    `json:"total"`
	ItemCount         int        `json:"item_count"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	CheckedOutAt      *time.Time `json:"checked_out_at,omitempty"`
	CheckedOutBy      *string    `json:"checked_out_by,omitempty"`
	ReturnedAt        *time.Time `json:"returned_at,omitempty"`
	ReturnedBy        *string    `json:"returned_by,omitempty"`
}

type ScheduleHistory struct {
	ID                   string    `json:"id"`
	PreviousBlockStartAt time.Time `json:"previous_block_start_at"`
	PreviousBlockEndAt   time.Time `json:"previous_block_end_at"`
	PreviousEventStartAt time.Time `json:"previous_event_start_at"`
	PreviousEventEndAt   time.Time `json:"previous_event_end_at"`
	NewBlockStartAt      time.Time `json:"new_block_start_at"`
	NewBlockEndAt        time.Time `json:"new_block_end_at"`
	NewEventStartAt      time.Time `json:"new_event_start_at"`
	NewEventEndAt        time.Time `json:"new_event_end_at"`
	Reason               string    `json:"reason"`
	ActorID              string    `json:"actor_id"`
	CreatedAt            time.Time `json:"created_at"`
}

type Detail struct {
	Summary
	Notes              string            `json:"notes"`
	CheckoutNotes      string            `json:"checkout_notes"`
	ReturnNotes        string            `json:"return_notes"`
	WarehouseComplete  bool              `json:"warehouse_complete"`
	RequiredAssetCount int               `json:"required_asset_count"`
	AssignedAssetCount int               `json:"assigned_asset_count"`
	Items              []Item            `json:"items"`
	StatusHistory      []StatusHistory   `json:"status_history"`
	ActivityHistory    []ActivityEvent   `json:"activity_history"`
	ScheduleHistory    []ScheduleHistory `json:"schedule_history"`
}

type AssignableAsset struct {
	ID             string  `json:"id"`
	ResourceID     string  `json:"resource_id"`
	AssetCode      string  `json:"asset_code"`
	SerialNumber   *string `json:"serial_number,omitempty"`
	PhysicalStatus string  `json:"physical_status"`
	Notes          string  `json:"notes"`
}

type WarehouseItem struct {
	ReservationItemID     string            `json:"reservation_item_id"`
	ResourceID            string            `json:"resource_id"`
	ResourceName          string            `json:"resource_name"`
	TrackIndividualAssets bool              `json:"track_individual_assets"`
	RequiredQuantity      int               `json:"required_quantity"`
	AssignedQuantity      int               `json:"assigned_quantity"`
	MissingQuantity       int               `json:"missing_quantity"`
	Assignments           []AssignedAsset   `json:"assignments"`
	AvailableAssets       []AssignableAsset `json:"available_assets"`
}

type WarehouseInventory struct {
	ReservationID        string          `json:"reservation_id"`
	Status               string          `json:"status"`
	CanManageAssignments bool            `json:"can_manage_assignments"`
	Complete             bool            `json:"complete"`
	RequiredAssetCount   int             `json:"required_asset_count"`
	AssignedAssetCount   int             `json:"assigned_asset_count"`
	Items                []WarehouseItem `json:"items"`
}

type AssignAssetInput struct {
	AssetID string `json:"asset_id"`
}

type CheckoutInput struct {
	Notes string `json:"notes"`
}

type ReturnAssetInput struct {
	AssetID   string `json:"asset_id"`
	Condition string `json:"condition"`
	Notes     string `json:"notes"`
}

type ReturnInput struct {
	Notes  string             `json:"notes"`
	Assets []ReturnAssetInput `json:"assets"`
}

type CreateItemInput struct {
	ResourceID     string  `json:"resource_id"`
	Description    string  `json:"description"`
	Quantity       int     `json:"quantity"`
	UnitPrice      float64 `json:"unit_price"`
	DiscountAmount float64 `json:"discount_amount"`
}

type CreateInput struct {
	CustomerID     string            `json:"customer_id"`
	BlockStartAt   string            `json:"block_start_at"`
	BlockEndAt     string            `json:"block_end_at"`
	EventStartAt   string            `json:"event_start_at"`
	EventEndAt     string            `json:"event_end_at"`
	EventType      *string           `json:"event_type"`
	EventLocation  *string           `json:"event_location"`
	DiscountAmount float64           `json:"discount_amount"`
	ExtraCharges   float64           `json:"extra_charges"`
	Notes          string            `json:"notes"`
	Items          []CreateItemInput `json:"items"`
}

type normalizedCreateItem struct {
	ResourceID     string
	Description    string
	Quantity       int
	UnitPrice      float64
	DiscountAmount float64
	LineTotal      float64
}

type normalizedCreateInput struct {
	CustomerID     string
	BlockStartAt   time.Time
	BlockEndAt     time.Time
	EventStartAt   time.Time
	EventEndAt     time.Time
	EventType      *string
	EventLocation  *string
	Subtotal       float64
	DiscountAmount float64
	ExtraCharges   float64
	Total          float64
	Notes          string
	Items          []normalizedCreateItem
}

type RescheduleInput struct {
	BlockStartAt string `json:"block_start_at"`
	BlockEndAt   string `json:"block_end_at"`
	EventStartAt string `json:"event_start_at"`
	EventEndAt   string `json:"event_end_at"`
	Reason       string `json:"reason"`
}

type normalizedRescheduleInput struct {
	BlockStartAt time.Time
	BlockEndAt   time.Time
	EventStartAt time.Time
	EventEndAt   time.Time
	Reason       string
}
