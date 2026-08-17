package quote

import (
	"errors"
	"time"
)

var (
	ErrNotFound          = errors.New("quote not found")
	ErrCustomerNotFound  = errors.New("quote customer not found")
	ErrResourceNotFound  = errors.New("quote resource not found")
	ErrImmutable         = errors.New("quote is immutable")
	ErrInvalidTransition = errors.New("invalid quote status transition")
)

type Item struct {
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

type Summary struct {
	ID                string     `json:"id"`
	QuoteNumber       int64      `json:"quote_number"`
	CustomerID        string     `json:"customer_id"`
	CustomerName      string     `json:"customer_name"`
	CustomerPhone     *string    `json:"customer_phone,omitempty"`
	ReservationID     *string    `json:"reservation_id,omitempty"`
	ReservationNumber *int64     `json:"reservation_number,omitempty"`
	Status            string     `json:"status"`
	StartAt           time.Time  `json:"start_at"`
	EndAt             time.Time  `json:"end_at"`
	EventType         *string    `json:"event_type,omitempty"`
	EventLocation     *string    `json:"event_location,omitempty"`
	Subtotal          float64    `json:"subtotal"`
	DiscountAmount    float64    `json:"discount_amount"`
	ExtraCharges      float64    `json:"extra_charges"`
	Total             float64    `json:"total"`
	ItemCount         int        `json:"item_count"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type PortalSummary struct {
	ID              string     `json:"id"`
	Status          string     `json:"status"`
	Revision        int        `json:"revision"`
	ExpiresAt       time.Time  `json:"expires_at"`
	LastViewedAt    *time.Time `json:"last_viewed_at,omitempty"`
	ViewCount       int        `json:"view_count"`
	DecisionAt      *time.Time `json:"decision_at,omitempty"`
	DecisionSource  *string    `json:"decision_source,omitempty"`
	ResponseName    *string    `json:"response_name,omitempty"`
	ResponseEmail   *string    `json:"response_email,omitempty"`
	RejectionReason *string    `json:"rejection_reason,omitempty"`
	TermsVersion    string     `json:"terms_version"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	PublicURL       string     `json:"public_url,omitempty"`
}

type Detail struct {
	Summary
	Notes  string         `json:"notes"`
	Items  []Item         `json:"items"`
	Portal *PortalSummary `json:"portal,omitempty"`
}

type ItemInput struct {
	ResourceID     string  `json:"resource_id"`
	Description    string  `json:"description"`
	Quantity       int     `json:"quantity"`
	UnitPrice      float64 `json:"unit_price"`
	DiscountAmount float64 `json:"discount_amount"`
}

type CreateInput struct {
	CustomerID     string      `json:"customer_id"`
	StartAt        string      `json:"start_at"`
	EndAt          string      `json:"end_at"`
	EventType      *string     `json:"event_type"`
	EventLocation  *string     `json:"event_location"`
	DiscountAmount float64     `json:"discount_amount"`
	ExtraCharges   float64     `json:"extra_charges"`
	Notes          string      `json:"notes"`
	ExpiresAt      *string     `json:"expires_at"`
	Items          []ItemInput `json:"items"`
}

type normalizedItem struct {
	ResourceID     string
	Description    string
	Quantity       int
	UnitPrice      float64
	DiscountAmount float64
	LineTotal      float64
}

type normalizedInput struct {
	CustomerID     string
	StartAt        time.Time
	EndAt          time.Time
	EventType      *string
	EventLocation  *string
	Subtotal       float64
	DiscountAmount float64
	ExtraCharges   float64
	Total          float64
	Notes          string
	ExpiresAt      *time.Time
	Items          []normalizedItem
}
