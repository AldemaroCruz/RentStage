package dte

import (
	"errors"
	"time"
)

var (
	ErrSettingsDisabled        = errors.New("dte integration is disabled")
	ErrSettingsIncomplete      = errors.New("dte settings are incomplete")
	ErrSettingsChanged         = errors.New("dte settings changed during preparation")
	ErrDocumentNotFound        = errors.New("dte document not found")
	ErrDocumentConflict        = errors.New("an active dte already exists for the invoice")
	ErrDocumentState           = errors.New("invalid dte document state")
	ErrInvoiceNotFound         = errors.New("invoice not found")
	ErrInvoiceState            = errors.New("invoice is not eligible for dte")
	ErrInvoiceFiscalState      = errors.New("invoice fiscal state is not eligible")
	ErrProviderUnavailable     = errors.New("dte provider is unavailable")
	ErrProviderConfiguration   = errors.New("dte provider configuration is invalid")
	ErrProviderRejected        = errors.New("dte provider rejected document")
	ErrInvalidationUnsupported = errors.New("dte invalidation is not available")
)

type Settings struct {
	TenantID                 string    `json:"tenant_id"`
	Enabled                  bool      `json:"enabled"`
	ProviderMode             string    `json:"provider_mode"`
	Environment              string    `json:"environment"`
	DefaultDocumentType      string    `json:"default_document_type"`
	SchemaVersion            int       `json:"schema_version"`
	EstablishmentType        string    `json:"establishment_type"`
	EstablishmentCode        string    `json:"establishment_code"`
	PointOfSaleCode          string    `json:"point_of_sale_code"`
	AuthURL                  string    `json:"auth_url"`
	SignerURL                string    `json:"signer_url"`
	ReceptionURL             string    `json:"reception_url"`
	InvalidationURL          string    `json:"invalidation_url"`
	QueryURL                 string    `json:"query_url"`
	UserSecretRef            string    `json:"user_secret_ref"`
	PasswordSecretRef        string    `json:"password_secret_ref"`
	SigningPasswordSecretRef string    `json:"signing_password_secret_ref"`
	AutoSubmitOnIssue        bool      `json:"auto_submit_on_issue"`
	MaxAttempts              int       `json:"max_attempts"`
	RetryBaseSeconds         int       `json:"retry_base_seconds"`
	NextControlNumber        int64     `json:"next_control_number"`
	ConfigurationReady       bool      `json:"configuration_ready"`
	ProductionSafetyReady    bool      `json:"production_safety_ready"`
	MissingConfiguration     []string  `json:"missing_configuration"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}

type SettingsInput struct {
	Enabled                  bool   `json:"enabled"`
	ProviderMode             string `json:"provider_mode"`
	Environment              string `json:"environment"`
	DefaultDocumentType      string `json:"default_document_type"`
	SchemaVersion            int    `json:"schema_version"`
	EstablishmentType        string `json:"establishment_type"`
	EstablishmentCode        string `json:"establishment_code"`
	PointOfSaleCode          string `json:"point_of_sale_code"`
	AuthURL                  string `json:"auth_url"`
	SignerURL                string `json:"signer_url"`
	ReceptionURL             string `json:"reception_url"`
	InvalidationURL          string `json:"invalidation_url"`
	QueryURL                 string `json:"query_url"`
	UserSecretRef            string `json:"user_secret_ref"`
	PasswordSecretRef        string `json:"password_secret_ref"`
	SigningPasswordSecretRef string `json:"signing_password_secret_ref"`
	AutoSubmitOnIssue        bool   `json:"auto_submit_on_issue"`
	MaxAttempts              int    `json:"max_attempts"`
	RetryBaseSeconds         int    `json:"retry_base_seconds"`
}

type PrepareInput struct {
	DocumentType string `json:"document_type"`
}

type InvalidateInput struct {
	Reason string `json:"reason"`
}

type Event struct {
	ID        string         `json:"id"`
	EventType string         `json:"event_type"`
	ActorID   *string        `json:"actor_id,omitempty"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt time.Time      `json:"created_at"`
}

type DocumentSummary struct {
	ID                   string     `json:"id"`
	InvoiceID            string     `json:"invoice_id"`
	InvoiceNumber        *int64     `json:"invoice_number,omitempty"`
	InvoicePrefix        string     `json:"invoice_prefix"`
	InvoiceDisplayNumber string     `json:"invoice_display_number"`
	CustomerName         string     `json:"customer_name"`
	DocumentType         string     `json:"document_type"`
	DocumentTypeLabel    string     `json:"document_type_label"`
	SchemaVersion        int        `json:"schema_version"`
	ProviderMode         string     `json:"provider_mode"`
	Environment          string     `json:"environment"`
	Status               string     `json:"status"`
	GenerationCode       string     `json:"generation_code"`
	ControlNumber        string     `json:"control_number"`
	ReceiptSeal          string     `json:"receipt_seal"`
	ProviderStatus       string     `json:"provider_status"`
	ErrorCode            string     `json:"error_code"`
	ErrorMessage         string     `json:"error_message"`
	AttemptCount         int        `json:"attempt_count"`
	NextAttemptAt        *time.Time `json:"next_attempt_at,omitempty"`
	SubmittedAt          *time.Time `json:"submitted_at,omitempty"`
	AcceptedAt           *time.Time `json:"accepted_at,omitempty"`
	RejectedAt           *time.Time `json:"rejected_at,omitempty"`
	InvalidatedAt        *time.Time `json:"invalidated_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type DocumentDetail struct {
	DocumentSummary
	IDempotencyKey       string         `json:"idempotency_key"`
	Payload              map[string]any `json:"payload"`
	SignedDocument       string         `json:"signed_document"`
	ProviderRequest      map[string]any `json:"provider_request"`
	ProviderResponse     map[string]any `json:"provider_response"`
	InvalidationRequest  map[string]any `json:"invalidation_request"`
	InvalidationResponse map[string]any `json:"invalidation_response"`
	InvalidationReason   string         `json:"invalidation_reason"`
	Events               []Event        `json:"events"`
}

type InvoiceItemSnapshot struct {
	ID             string
	Description    string
	Quantity       float64
	UnitPrice      float64
	DiscountAmount float64
	NetAmount      float64
	TaxCategory    string
	TaxRate        float64
	TaxAmount      float64
	LineTotal      float64
	DTEItemType    int
	DTEUnitCode    int
	DTEProductCode string
}

type InvoiceSnapshot struct {
	ID                         string
	InvoiceNumber              int64
	InvoicePrefix              string
	Status                     string
	FiscalStatus               string
	IssueDate                  string
	DueDate                    string
	Currency                   string
	PricesIncludeTax           bool
	CustomerID                 string
	CustomerName               string
	CustomerTaxID              string
	CustomerRegistrationNumber string
	CustomerDocumentType       string
	CustomerTradeName          string
	CustomerEconomicActivity   string
	CustomerEconomicCode       string
	CustomerEmail              string
	CustomerPhone              string
	CustomerAddress            string
	CustomerDepartmentCode     string
	CustomerMunicipalityCode   string
	CustomerDistrictCode       string
	SellerLegalName            string
	SellerTradeName            string
	SellerTaxID                string
	SellerRegistrationNumber   string
	SellerEconomicActivity     string
	SellerEconomicCode         string
	SellerEmail                string
	SellerPhone                string
	SellerAddress              string
	SellerDepartmentCode       string
	SellerMunicipalityCode     string
	SellerDistrictCode         string
	TaxableAmount              float64
	ExemptAmount               float64
	NonTaxableAmount           float64
	TaxAmount                  float64
	TotalAmount                float64
	Notes                      string
	Terms                      string
	Items                      []InvoiceItemSnapshot
}

type ProviderSubmission struct {
	Settings Settings
	Document DocumentDetail
}

type ProviderResult struct {
	Accepted       bool
	Retryable      bool
	SignedDocument string
	ReceiptSeal    string
	ProviderStatus string
	ErrorCode      string
	ErrorMessage   string
	Request        map[string]any
	Response       map[string]any
}

type InvalidationResult struct {
	Accepted       bool
	Retryable      bool
	ProviderStatus string
	ErrorCode      string
	ErrorMessage   string
	Request        map[string]any
	Response       map[string]any
}
