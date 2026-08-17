package operations

import "time"

type CalendarFilter struct {
	From       time.Time
	To         time.Time
	Statuses   []string
	CustomerID *string
	ResourceID *string
}

type CalendarEvent struct {
	ID                 string     `json:"id"`
	ReservationNumber  int64      `json:"reservation_number"`
	CustomerID         string     `json:"customer_id"`
	CustomerName       string     `json:"customer_name"`
	CustomerPhone      *string    `json:"customer_phone,omitempty"`
	Source             string     `json:"source"`
	Status             string     `json:"status"`
	BlockStartAt       time.Time  `json:"block_start_at"`
	BlockEndAt         time.Time  `json:"block_end_at"`
	EventStartAt       time.Time  `json:"event_start_at"`
	EventEndAt         time.Time  `json:"event_end_at"`
	EventType          *string    `json:"event_type,omitempty"`
	EventLocation      *string    `json:"event_location,omitempty"`
	Total              float64    `json:"total"`
	ItemCount          int        `json:"item_count"`
	RequiredAssetCount int        `json:"required_asset_count"`
	AssignedAssetCount int        `json:"assigned_asset_count"`
	ResourceSummary    string     `json:"resource_summary"`
	CheckedOutAt       *time.Time `json:"checked_out_at,omitempty"`
	ReturnedAt         *time.Time `json:"returned_at,omitempty"`
}

type CalendarResult struct {
	From     time.Time       `json:"from"`
	To       time.Time       `json:"to"`
	Timezone string          `json:"timezone"`
	Items    []CalendarEvent `json:"items"`
}

type AgendaResult struct {
	Date           string          `json:"date"`
	Timezone       string          `json:"timezone"`
	DayStart       time.Time       `json:"day_start"`
	DayEnd         time.Time       `json:"day_end"`
	Departures     []CalendarEvent `json:"departures"`
	Events         []CalendarEvent `json:"events"`
	Returns        []CalendarEvent `json:"returns"`
	PendingClose   []CalendarEvent `json:"pending_close"`
	OverdueReturns []Alert         `json:"overdue_returns"`
}

type Alert struct {
	ID                 string     `json:"id"`
	Type               string     `json:"type"`
	Severity           string     `json:"severity"`
	ReservationID      string     `json:"reservation_id"`
	ReservationNumber  int64      `json:"reservation_number"`
	CustomerName       string     `json:"customer_name"`
	EventType          *string    `json:"event_type,omitempty"`
	Status             string     `json:"status"`
	Title              string     `json:"title"`
	Message            string     `json:"message"`
	DueAt              *time.Time `json:"due_at,omitempty"`
	MissingAssetCount  int        `json:"missing_asset_count"`
	RequiredAssetCount int        `json:"required_asset_count"`
	AssignedAssetCount int        `json:"assigned_asset_count"`
	MinutesOverdue     int64      `json:"minutes_overdue"`
}

type AlertCounts struct {
	Critical int `json:"critical"`
	Warning  int `json:"warning"`
	Info     int `json:"info"`
	Total    int `json:"total"`
}

type AlertResult struct {
	GeneratedAt time.Time   `json:"generated_at"`
	Counts      AlertCounts `json:"counts"`
	Items       []Alert     `json:"items"`
}
