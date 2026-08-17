package billing

import (
	"errors"
	"time"
)

var (
	ErrInvoiceNotFound     = errors.New("invoice not found")
	ErrPaymentNotFound     = errors.New("payment not found")
	ErrDepositNotFound     = errors.New("security deposit not found")
	ErrCustomerNotFound    = errors.New("billing customer not found")
	ErrSourceNotFound      = errors.New("billing source not found")
	ErrSourceConflict      = errors.New("billing source already invoiced")
	ErrInvoiceImmutable    = errors.New("invoice is immutable")
	ErrInvoiceState        = errors.New("invalid invoice state")
	ErrPaymentState        = errors.New("invalid payment state")
	ErrDepositState        = errors.New("invalid security deposit state")
	ErrTaxRuleNotFound     = errors.New("tax rule not found")
	ErrAllocationConflict  = errors.New("payment allocation conflict")
	ErrCurrencyMismatch    = errors.New("currency mismatch")
	ErrCustomerMismatch    = errors.New("customer mismatch")
	ErrFiscalProfileNeeded = errors.New("fiscal profile is incomplete")
	ErrBillingDisabled     = errors.New("billing is disabled")
)

type Settings struct {
	TenantID                   string    `json:"tenant_id"`
	Enabled                    bool      `json:"enabled"`
	LegalName                  string    `json:"legal_name"`
	TradeName                  string    `json:"trade_name"`
	TaxID                      string    `json:"tax_id"`
	TaxRegistrationNumber      string    `json:"tax_registration_number"`
	EconomicActivity           string    `json:"economic_activity"`
	EconomicActivityCode       string    `json:"economic_activity_code"`
	FiscalAddress              string    `json:"fiscal_address"`
	Department                 string    `json:"department"`
	Municipality               string    `json:"municipality"`
	District                   string    `json:"district"`
	DepartmentCode             string    `json:"department_code"`
	MunicipalityCode           string    `json:"municipality_code"`
	DistrictCode               string    `json:"district_code"`
	Email                      string    `json:"email"`
	Phone                      string    `json:"phone"`
	PricesIncludeTax           bool      `json:"prices_include_tax"`
	DefaultTaxRate             float64   `json:"default_tax_rate"`
	DefaultPaymentTermsDays    int       `json:"default_payment_terms_days"`
	InvoicePrefix              string    `json:"invoice_prefix"`
	NextInvoiceNumber          int64     `json:"next_invoice_number"`
	FiscalProfileComplete      bool      `json:"fiscal_profile_complete"`
	FiscalProfileMissingFields []string  `json:"fiscal_profile_missing_fields"`
	CreatedAt                  time.Time `json:"created_at"`
	UpdatedAt                  time.Time `json:"updated_at"`
}

type SettingsInput struct {
	Enabled                 bool    `json:"enabled"`
	LegalName               string  `json:"legal_name"`
	TradeName               string  `json:"trade_name"`
	TaxID                   string  `json:"tax_id"`
	TaxRegistrationNumber   string  `json:"tax_registration_number"`
	EconomicActivity        string  `json:"economic_activity"`
	EconomicActivityCode    string  `json:"economic_activity_code"`
	FiscalAddress           string  `json:"fiscal_address"`
	Department              string  `json:"department"`
	Municipality            string  `json:"municipality"`
	District                string  `json:"district"`
	DepartmentCode          string  `json:"department_code"`
	MunicipalityCode        string  `json:"municipality_code"`
	DistrictCode            string  `json:"district_code"`
	Email                   string  `json:"email"`
	Phone                   string  `json:"phone"`
	PricesIncludeTax        bool    `json:"prices_include_tax"`
	DefaultTaxRate          float64 `json:"default_tax_rate"`
	DefaultPaymentTermsDays int     `json:"default_payment_terms_days"`
	InvoicePrefix           string  `json:"invoice_prefix"`
}

type TaxRule struct {
	ID         string    `json:"id"`
	Code       string    `json:"code"`
	Name       string    `json:"name"`
	Category   string    `json:"category"`
	Rate       float64   `json:"rate"`
	Active     bool      `json:"active"`
	IsDefault  bool      `json:"is_default"`
	ValidFrom  string    `json:"valid_from"`
	ValidUntil *string   `json:"valid_until,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type InvoiceItem struct {
	ID             string  `json:"id"`
	ResourceID     *string `json:"resource_id,omitempty"`
	TaxRuleID      *string `json:"tax_rule_id,omitempty"`
	Description    string  `json:"description"`
	Quantity       float64 `json:"quantity"`
	UnitPrice      float64 `json:"unit_price"`
	DiscountAmount float64 `json:"discount_amount"`
	GrossAmount    float64 `json:"gross_amount"`
	NetAmount      float64 `json:"net_amount"`
	TaxCode        string  `json:"tax_code"`
	TaxCategory    string  `json:"tax_category"`
	TaxRate        float64 `json:"tax_rate"`
	TaxAmount      float64 `json:"tax_amount"`
	LineTotal      float64 `json:"line_total"`
	SortOrder      int     `json:"sort_order"`
}

type InvoiceSummary struct {
	ID                string     `json:"id"`
	InvoiceNumber     *int64     `json:"invoice_number,omitempty"`
	InvoicePrefix     string     `json:"invoice_prefix"`
	DisplayNumber     string     `json:"display_number"`
	CustomerID        string     `json:"customer_id"`
	CustomerName      string     `json:"customer_name"`
	QuoteID           *string    `json:"quote_id,omitempty"`
	QuoteNumber       *int64     `json:"quote_number,omitempty"`
	ReservationID     *string    `json:"reservation_id,omitempty"`
	ReservationNumber *int64     `json:"reservation_number,omitempty"`
	SourceType        string     `json:"source_type"`
	Status            string     `json:"status"`
	DisplayStatus     string     `json:"display_status"`
	IssueDate         string     `json:"issue_date"`
	DueDate           string     `json:"due_date"`
	Currency          string     `json:"currency"`
	PricesIncludeTax  bool       `json:"prices_include_tax"`
	TaxableAmount     float64    `json:"taxable_amount"`
	ExemptAmount      float64    `json:"exempt_amount"`
	NonTaxableAmount  float64    `json:"non_taxable_amount"`
	TaxAmount         float64    `json:"tax_amount"`
	TotalAmount       float64    `json:"total_amount"`
	PaidAmount        float64    `json:"paid_amount"`
	BalanceDue        float64    `json:"balance_due"`
	FiscalStatus      string     `json:"fiscal_status"`
	ItemCount         int        `json:"item_count"`
	IssuedAt          *time.Time `json:"issued_at,omitempty"`
	VoidedAt          *time.Time `json:"voided_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type InvoiceEvent struct {
	ID        string         `json:"id"`
	EventType string         `json:"event_type"`
	ActorID   string         `json:"actor_id"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt time.Time      `json:"created_at"`
}

type PaymentAllocation struct {
	ID                   string  `json:"id"`
	PaymentID            string  `json:"payment_id,omitempty"`
	PaymentNumber        *int64  `json:"payment_number,omitempty"`
	PaymentDisplayNumber string  `json:"payment_display_number,omitempty"`
	InvoiceID            string  `json:"invoice_id"`
	InvoiceNumber        *int64  `json:"invoice_number,omitempty"`
	InvoicePrefix        string  `json:"invoice_prefix"`
	DisplayNumber        string  `json:"display_number"`
	Amount               float64 `json:"amount"`
}

type InvoiceDetail struct {
	InvoiceSummary
	CustomerTaxID              string              `json:"customer_tax_id"`
	CustomerEmail              string              `json:"customer_email"`
	CustomerPhone              string              `json:"customer_phone"`
	CustomerAddress            string              `json:"customer_address"`
	SellerLegalName            string              `json:"seller_legal_name"`
	SellerTradeName            string              `json:"seller_trade_name"`
	SellerTaxID                string              `json:"seller_tax_id"`
	SellerRegistrationNumber   string              `json:"seller_registration_number"`
	SellerEconomicActivity     string              `json:"seller_economic_activity"`
	SellerEconomicActivityCode string              `json:"seller_economic_activity_code"`
	SellerAddress              string              `json:"seller_address"`
	SellerEmail                string              `json:"seller_email"`
	SellerPhone                string              `json:"seller_phone"`
	Notes                      string              `json:"notes"`
	Terms                      string              `json:"terms"`
	VoidReason                 string              `json:"void_reason"`
	Items                      []InvoiceItem       `json:"items"`
	Events                     []InvoiceEvent      `json:"events"`
	Allocations                []PaymentAllocation `json:"allocations"`
}

type InvoiceItemInput struct {
	ResourceID     *string `json:"resource_id"`
	TaxRuleID      string  `json:"tax_rule_id"`
	Description    string  `json:"description"`
	Quantity       float64 `json:"quantity"`
	UnitPrice      float64 `json:"unit_price"`
	DiscountAmount float64 `json:"discount_amount"`
}

type CreateInvoiceInput struct {
	SourceType string             `json:"source_type"`
	SourceID   *string            `json:"source_id"`
	CustomerID string             `json:"customer_id"`
	IssueDate  string             `json:"issue_date"`
	DueDate    string             `json:"due_date"`
	Currency   string             `json:"currency"`
	Notes      string             `json:"notes"`
	Terms      string             `json:"terms"`
	Items      []InvoiceItemInput `json:"items"`
}

type UpdateInvoiceInput struct {
	CustomerID       string             `json:"customer_id"`
	IssueDate        string             `json:"issue_date"`
	DueDate          string             `json:"due_date"`
	Currency         string             `json:"currency"`
	PricesIncludeTax bool               `json:"prices_include_tax"`
	Notes            string             `json:"notes"`
	Terms            string             `json:"terms"`
	Items            []InvoiceItemInput `json:"items"`
}

type VoidInput struct {
	Reason string `json:"reason"`
}

type PaymentSummary struct {
	ID              string     `json:"id"`
	PaymentNumber   int64      `json:"payment_number"`
	DisplayNumber   string     `json:"display_number"`
	CustomerID      string     `json:"customer_id"`
	CustomerName    string     `json:"customer_name"`
	Status          string     `json:"status"`
	Amount          float64    `json:"amount"`
	Currency        string     `json:"currency"`
	Method          string     `json:"method"`
	Reference       string     `json:"reference"`
	ReceivedAt      time.Time  `json:"received_at"`
	AllocationCount int        `json:"allocation_count"`
	VoidedAt        *time.Time `json:"voided_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type PaymentDetail struct {
	PaymentSummary
	Notes       string              `json:"notes"`
	VoidReason  string              `json:"void_reason"`
	Allocations []PaymentAllocation `json:"allocations"`
}

type PaymentAllocationInput struct {
	InvoiceID string  `json:"invoice_id"`
	Amount    float64 `json:"amount"`
}

type CreatePaymentInput struct {
	CustomerID  string                   `json:"customer_id"`
	Amount      float64                  `json:"amount"`
	Currency    string                   `json:"currency"`
	Method      string                   `json:"method"`
	Reference   string                   `json:"reference"`
	Notes       string                   `json:"notes"`
	ReceivedAt  string                   `json:"received_at"`
	Allocations []PaymentAllocationInput `json:"allocations"`
}

type SecurityDeposit struct {
	ID                string     `json:"id"`
	DepositNumber     int64      `json:"deposit_number"`
	DisplayNumber     string     `json:"display_number"`
	ReservationID     string     `json:"reservation_id"`
	ReservationNumber int64      `json:"reservation_number"`
	CustomerID        string     `json:"customer_id"`
	CustomerName      string     `json:"customer_name"`
	Status            string     `json:"status"`
	Amount            float64    `json:"amount"`
	ReturnedAmount    float64    `json:"returned_amount"`
	RetainedAmount    float64    `json:"retained_amount"`
	BalanceAmount     float64    `json:"balance_amount"`
	Currency          string     `json:"currency"`
	Method            string     `json:"method"`
	Reference         string     `json:"reference"`
	Notes             string     `json:"notes"`
	ReceivedAt        *time.Time `json:"received_at,omitempty"`
	SettledAt         *time.Time `json:"settled_at,omitempty"`
	SettlementReason  string     `json:"settlement_reason"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type CreateDepositInput struct {
	ReservationID string  `json:"reservation_id"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	Method        string  `json:"method"`
	Reference     string  `json:"reference"`
	Notes         string  `json:"notes"`
	MarkReceived  bool    `json:"mark_received"`
	ReceivedAt    *string `json:"received_at"`
}

type ReceiveDepositInput struct {
	ReceivedAt string `json:"received_at"`
	Method     string `json:"method"`
	Reference  string `json:"reference"`
	Notes      string `json:"notes"`
}

type SettleDepositInput struct {
	ReturnedAmount float64 `json:"returned_amount"`
	RetainedAmount float64 `json:"retained_amount"`
	SettledAt      string  `json:"settled_at"`
	Reason         string  `json:"reason"`
}

type DashboardMetrics struct {
	IssuedTotal       float64 `json:"issued_total"`
	CollectedTotal    float64 `json:"collected_total"`
	OutstandingTotal  float64 `json:"outstanding_total"`
	OverdueTotal      float64 `json:"overdue_total"`
	TaxOutputTotal    float64 `json:"tax_output_total"`
	DepositsHeldTotal float64 `json:"deposits_held_total"`
	DraftCount        int     `json:"draft_count"`
	OpenInvoiceCount  int     `json:"open_invoice_count"`
	OverdueCount      int     `json:"overdue_count"`
	PaidCount         int     `json:"paid_count"`
}

type MonthlyAmount struct {
	Month  string  `json:"month"`
	Amount float64 `json:"amount"`
}

type BillingDashboard struct {
	GeneratedAt     time.Time        `json:"generated_at"`
	Currency        string           `json:"currency"`
	Settings        Settings         `json:"settings"`
	Metrics         DashboardMetrics `json:"metrics"`
	RecentInvoices  []InvoiceSummary `json:"recent_invoices"`
	RecentPayments  []PaymentSummary `json:"recent_payments"`
	MonthlyBilling  []MonthlyAmount  `json:"monthly_billing"`
	MonthlyPayments []MonthlyAmount  `json:"monthly_payments"`
}

type sourceDraft struct {
	SourceType        string
	QuoteID           *string
	QuoteNumber       *int64
	ReservationID     *string
	ReservationNumber *int64
	CustomerID        string
	Currency          string
	Notes             string
	HeaderDiscount    float64
	ExtraCharges      float64
	Items             []sourceItem
}

type sourceItem struct {
	ResourceID     *string
	Description    string
	Quantity       float64
	UnitPrice      float64
	DiscountAmount float64
}

type normalizedInvoiceItem struct {
	ResourceID     *string
	TaxRuleID      *string
	Description    string
	Quantity       float64
	UnitPrice      float64
	DiscountAmount float64
	GrossAmount    float64
	NetAmount      float64
	TaxCode        string
	TaxCategory    string
	TaxRate        float64
	TaxAmount      float64
	LineTotal      float64
	SortOrder      int
}

type normalizedInvoice struct {
	SourceType       string
	QuoteID          *string
	ReservationID    *string
	CustomerID       string
	IssueDate        time.Time
	DueDate          time.Time
	Currency         string
	PricesIncludeTax bool
	TaxableAmount    float64
	ExemptAmount     float64
	NonTaxableAmount float64
	TaxAmount        float64
	TotalAmount      float64
	Notes            string
	Terms            string
	Items            []normalizedInvoiceItem
}

type normalizedPayment struct {
	CustomerID  string
	Amount      float64
	Currency    string
	Method      string
	Reference   string
	Notes       string
	ReceivedAt  time.Time
	Allocations []PaymentAllocationInput
}
