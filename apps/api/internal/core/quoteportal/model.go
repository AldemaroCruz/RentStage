package quoteportal

import (
	"errors"
	"time"
)

var (
	ErrPortalNotFound       = errors.New("quote portal not found")
	ErrPortalUnavailable    = errors.New("quote portal is unavailable")
	ErrPortalDisabled       = errors.New("quote portal is disabled")
	ErrQuoteNotFound        = errors.New("quote not found")
	ErrInvalidQuoteStatus   = errors.New("quote status does not permit portal action")
	ErrInvalidPortalStatus  = errors.New("quote portal status does not permit action")
	ErrPortalExpired        = errors.New("quote portal expired")
	ErrRejectionDisabled    = errors.New("quote portal rejection is disabled")
	ErrResponseNameRequired = errors.New("response name is required")
)

type Settings struct {
	TenantID               string    `json:"tenant_id"`
	Enabled                bool      `json:"enabled"`
	Headline               string    `json:"headline"`
	Introduction           string    `json:"introduction"`
	AccentColor            string    `json:"accent_color"`
	DefaultValidityDays    int       `json:"default_validity_days"`
	AllowRejection         bool      `json:"allow_rejection"`
	RequireResponseName    bool      `json:"require_response_name"`
	AcceptanceTermsText    string    `json:"acceptance_terms_text"`
	AcceptanceTermsVersion string    `json:"acceptance_terms_version"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type SettingsInput struct {
	Enabled                bool   `json:"enabled"`
	Headline               string `json:"headline"`
	Introduction           string `json:"introduction"`
	AccentColor            string `json:"accent_color"`
	DefaultValidityDays    int    `json:"default_validity_days"`
	AllowRejection         bool   `json:"allow_rejection"`
	RequireResponseName    bool   `json:"require_response_name"`
	AcceptanceTermsText    string `json:"acceptance_terms_text"`
	AcceptanceTermsVersion string `json:"acceptance_terms_version"`
}

type normalizedSettings struct {
	Enabled                bool
	Headline               string
	Introduction           string
	AccentColor            string
	DefaultValidityDays    int
	AllowRejection         bool
	RequireResponseName    bool
	AcceptanceTermsText    string
	AcceptanceTermsVersion string
}

type IssueResult struct {
	PortalID    string
	QuoteID     string
	QuoteNumber int64
	Token       string
	PublicURL   string
	ExpiresAt   time.Time
	Revision    int
	EventType   string
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

type PublicPortal struct {
	Status              string     `json:"status"`
	Headline            string     `json:"headline"`
	Introduction        string     `json:"introduction"`
	AccentColor         string     `json:"accent_color"`
	AllowRejection      bool       `json:"allow_rejection"`
	RequireResponseName bool       `json:"require_response_name"`
	TermsText           string     `json:"terms_text"`
	TermsVersion        string     `json:"terms_version"`
	ExpiresAt           time.Time  `json:"expires_at"`
	DecisionAt          *time.Time `json:"decision_at,omitempty"`
	DecisionSource      *string    `json:"decision_source,omitempty"`
	ResponseName        *string    `json:"response_name,omitempty"`
	RejectionReason     *string    `json:"rejection_reason,omitempty"`
	CanAccept           bool       `json:"can_accept"`
	CanReject           bool       `json:"can_reject"`
}

type PublicQuoteItem struct {
	Description    string  `json:"description"`
	ResourceName   string  `json:"resource_name"`
	Quantity       int     `json:"quantity"`
	UnitPrice      float64 `json:"unit_price"`
	DiscountAmount float64 `json:"discount_amount"`
	LineTotal      float64 `json:"line_total"`
}

type PublicQuote struct {
	QuoteNumber       int64             `json:"quote_number"`
	Status            string            `json:"status"`
	CustomerName      string            `json:"customer_name"`
	StartAt           time.Time         `json:"start_at"`
	EndAt             time.Time         `json:"end_at"`
	EventType         *string           `json:"event_type,omitempty"`
	EventLocation     *string           `json:"event_location,omitempty"`
	Subtotal          float64           `json:"subtotal"`
	DiscountAmount    float64           `json:"discount_amount"`
	ExtraCharges      float64           `json:"extra_charges"`
	Total             float64           `json:"total"`
	CreatedAt         time.Time         `json:"created_at"`
	ReservationNumber *int64            `json:"reservation_number,omitempty"`
	Items             []PublicQuoteItem `json:"items"`
}

type PublicView struct {
	Tenant PublicTenant `json:"tenant"`
	Portal PublicPortal `json:"portal"`
	Quote  PublicQuote  `json:"quote"`

	PortalID string `json:"-"`
	TenantID string `json:"-"`
	QuoteID  string `json:"-"`
}

type AcceptInput struct {
	ResponseName  string `json:"response_name"`
	ResponseEmail string `json:"response_email"`
	TermsAccepted bool   `json:"terms_accepted"`
}

type RejectInput struct {
	ResponseName    string `json:"response_name"`
	ResponseEmail   string `json:"response_email"`
	RejectionReason string `json:"rejection_reason"`
}

type normalizedDecision struct {
	ResponseName    string
	ResponseEmail   *string
	RejectionReason *string
}

type DecisionResult struct {
	Status            string    `json:"status"`
	QuoteNumber       int64     `json:"quote_number"`
	ReservationNumber *int64    `json:"reservation_number,omitempty"`
	DecisionAt        time.Time `json:"decision_at"`
	Idempotent        bool      `json:"idempotent"`

	TenantID      string `json:"-"`
	PortalID      string `json:"-"`
	QuoteID       string `json:"-"`
	ReservationID string `json:"-"`
}

type PublicAvailabilityItem struct {
	ResourceName      string `json:"resource_name"`
	RequestedQuantity int    `json:"requested_quantity"`
	CanFulfill        bool   `json:"can_fulfill"`
}

type PublicAvailabilityConflict struct {
	Available bool                     `json:"available"`
	Items     []PublicAvailabilityItem `json:"items"`
}
