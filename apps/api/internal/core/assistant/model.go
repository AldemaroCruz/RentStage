package assistant

import (
	"errors"
	"time"
)

var (
	ErrNotFound              = errors.New("assistant conversation not found")
	ErrNoReadyPackage        = errors.New("no ready commercial package")
	ErrUnavailable           = errors.New("assistant proposal is unavailable")
	ErrAlreadyApproved       = errors.New("assistant proposal is already approved")
	ErrCustomerMissing       = errors.New("assistant customer not found")
	ErrDemoOnly              = errors.New("assistant operation is only available in demo")
	ErrMessageNotReady       = errors.New("assistant message is not ready to send")
	ErrQuoteMissing          = errors.New("assistant quote is missing")
	ErrPortalDeliveryMissing = errors.New("assistant quote portal delivery is missing")
	ErrProviderDisabled      = errors.New("WhatsApp provider is disabled")
	ErrProviderDelivery      = errors.New("WhatsApp provider delivery failed")
	ErrServiceWindowClosed   = errors.New("WhatsApp customer service window is closed")
	ErrConsentRevoked        = errors.New("WhatsApp customer consent is revoked")
)

type ConversationSummary struct {
	ID                     string     `json:"id"`
	Channel                string     `json:"channel"`
	CustomerID             *string    `json:"customer_id,omitempty"`
	CustomerName           *string    `json:"customer_name,omitempty"`
	ContactName            string     `json:"contact_name"`
	ContactPhone           string     `json:"contact_phone"`
	ContactEmail           *string    `json:"contact_email,omitempty"`
	Status                 string     `json:"status"`
	ConsentStatus          string     `json:"consent_status"`
	ServiceWindowExpiresAt *time.Time `json:"service_window_expires_at,omitempty"`
	Summary                string     `json:"summary"`
	LastMessage            string     `json:"last_message"`
	LastMessageAt          time.Time  `json:"last_message_at"`
	QuoteID                *string    `json:"quote_id,omitempty"`
	QuoteNumber            *int64     `json:"quote_number,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

type Message struct {
	ID         string         `json:"id"`
	Direction  string         `json:"direction"`
	SenderType string         `json:"sender_type"`
	Provider   string         `json:"provider"`
	Body       string         `json:"body"`
	Status     string         `json:"status"`
	Metadata   map[string]any `json:"metadata"`
	ApprovedBy *string        `json:"approved_by,omitempty"`
	ApprovedAt *time.Time     `json:"approved_at,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

type Proposal struct {
	ID                string         `json:"id"`
	Status            string         `json:"status"`
	Provider          string         `json:"provider"`
	EventType         string         `json:"event_type"`
	StartAt           time.Time      `json:"start_at"`
	EndAt             time.Time      `json:"end_at"`
	EventLocation     string         `json:"event_location"`
	GuestCount        int            `json:"guest_count"`
	PackageID         string         `json:"package_id"`
	PackageQuantity   int            `json:"package_quantity"`
	PackageName       string         `json:"package_name"`
	PackagePrice      float64        `json:"package_price"`
	Available         bool           `json:"available"`
	Recommendation    string         `json:"recommendation"`
	ResponseDraft     string         `json:"response_draft"`
	Evidence          map[string]any `json:"evidence"`
	QuoteID           *string        `json:"quote_id,omitempty"`
	QuoteNumber       *int64         `json:"quote_number,omitempty"`
	QuoteStatus       *string        `json:"quote_status,omitempty"`
	PortalStatus      *string        `json:"portal_status,omitempty"`
	PortalViewCount   int            `json:"portal_view_count"`
	PortalViewedAt    *time.Time     `json:"portal_viewed_at,omitempty"`
	PortalDecisionAt  *time.Time     `json:"portal_decision_at,omitempty"`
	ReservationID     *string        `json:"reservation_id,omitempty"`
	ReservationNumber *int64         `json:"reservation_number,omitempty"`
	ApprovedBy        *string        `json:"approved_by,omitempty"`
	ApprovedAt        *time.Time     `json:"approved_at,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

type ConversationDetail struct {
	ConversationSummary
	Messages       []Message       `json:"messages"`
	Proposal       *Proposal       `json:"proposal,omitempty"`
	PortalDelivery *PortalDelivery `json:"portal_delivery,omitempty"`
}

// PortalDelivery deliberately exists only in the response that issues a new
// bearer link. The repository never persists PublicURL or its raw token.
type PortalDelivery struct {
	QuoteID     string    `json:"quote_id"`
	QuoteNumber int64     `json:"quote_number"`
	PublicURL   string    `json:"public_url"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type SimulateInput struct {
	ContactName   string `json:"contact_name"`
	ContactPhone  string `json:"contact_phone"`
	Message       string `json:"message"`
	EventType     string `json:"event_type"`
	StartAt       string `json:"start_at"`
	EndAt         string `json:"end_at"`
	EventLocation string `json:"event_location"`
	GuestCount    int    `json:"guest_count"`
}

type ApproveInput struct {
	CustomerID   string `json:"customer_id"`
	ResponseBody string `json:"response_body"`
}

type LinkCustomerInput struct {
	CustomerID string `json:"customer_id"`
}

type SendDemoInput struct {
	MessageID string `json:"message_id"`
	Body      string `json:"body"`
}

type SendInput = SendDemoInput

type ReceiveDemoInput struct {
	Body string `json:"body"`
}

type ShareQuoteDemoInput struct {
	Body string `json:"body"`
}

type normalizedSimulation struct {
	ContactName   string
	ContactPhone  string
	Message       string
	EventType     string
	StartAt       time.Time
	EndAt         time.Time
	EventLocation string
	GuestCount    int
}

type proposalRecord struct {
	EventType       string
	StartAt         time.Time
	EndAt           time.Time
	EventLocation   string
	GuestCount      int
	PackageID       string
	PackageQuantity int
	PackageName     string
	PackagePrice    float64
	Available       bool
	Recommendation  string
	ResponseDraft   string
	Evidence        map[string]any
}
