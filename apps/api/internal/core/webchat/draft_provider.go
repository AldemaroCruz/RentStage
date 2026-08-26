package webchat

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

type DraftKind string

type DraftMessageRole string

const (
	DraftKindInitial  DraftKind = "INITIAL"
	DraftKindFollowUp DraftKind = "FOLLOW_UP"

	DraftMessageRoleCustomer DraftMessageRole = "CUSTOMER"
	DraftMessageRoleTeam     DraftMessageRole = "TEAM"

	MaximumDraftContextMessages = 12
	MaximumDraftContextRunes    = 8000

	MaximumDraftSalesPackages         = 8
	MaximumDraftSalesResources        = 12
	MaximumDraftSalesNameRunes        = 180
	MaximumDraftSalesDescriptionRunes = 400
	MaximumDraftSalesMetadataRunes    = 180
	MaximumDraftSalesContextRunes     = 16000
)

var ErrInvalidDraft = errors.New("web chat draft is invalid")

type DraftRequest struct {
	Kind             DraftKind
	TenantName       string
	TenantSlug       string
	ContactName      string
	CustomerMessage  string
	PreviousMessages []DraftConversationMessage
	SalesContext     DraftSalesContext
}

type DraftConversationMessage struct {
	Role DraftMessageRole `json:"role"`
	Body string           `json:"body"`
}

type DraftSalesContext struct {
	Currency             string               `json:"currency"`
	ShowPrices           bool                 `json:"show_prices"`
	ShowResources        bool                 `json:"show_resources"`
	QuoteRequestsEnabled bool                 `json:"quote_requests_enabled"`
	Packages             []DraftSalesPackage  `json:"packages"`
	Resources            []DraftSalesResource `json:"resources"`
}

type DraftSalesPackage struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	GuestCapacity *int     `json:"guest_capacity,omitempty"`
	Price         *float64 `json:"price,omitempty"`
}

type DraftSalesResource struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category,omitempty"`
	Type        string   `json:"type"`
	PricingUnit string   `json:"pricing_unit"`
	Price       *float64 `json:"price,omitempty"`
}

type DraftResult struct {
	Body         string
	Engine       string
	Model        string
	UsedFallback bool
}

type DraftProvider interface {
	GenerateDraft(
		ctx context.Context,
		request DraftRequest,
	) (DraftResult, error)
}

type RulesDraftProvider struct{}

func NewRulesDraftProvider() *RulesDraftProvider {
	return &RulesDraftProvider{}
}

func NormalizeDraftConversation(
	messages []DraftConversationMessage,
) ([]DraftConversationMessage, error) {
	if len(messages) > MaximumDraftContextMessages {
		return nil, fmt.Errorf(
			"%w: conversation context exceeds %d messages",
			ErrInvalidDraft,
			MaximumDraftContextMessages,
		)
	}

	normalized := make(
		[]DraftConversationMessage,
		0,
		len(messages),
	)
	totalRunes := 0

	for _, message := range messages {
		message.Body = strings.TrimSpace(message.Body)

		switch message.Role {
		case DraftMessageRoleCustomer,
			DraftMessageRoleTeam:
		default:
			return nil, fmt.Errorf(
				"%w: unsupported conversation role %q",
				ErrInvalidDraft,
				message.Role,
			)
		}

		messageRunes := utf8.RuneCountInString(message.Body)
		if messageRunes == 0 ||
			messageRunes > MaximumMessageLength {
			return nil, fmt.Errorf(
				"%w: invalid conversation message body",
				ErrInvalidDraft,
			)
		}

		totalRunes += messageRunes
		if totalRunes > MaximumDraftContextRunes {
			return nil, fmt.Errorf(
				"%w: conversation context exceeds %d runes",
				ErrInvalidDraft,
				MaximumDraftContextRunes,
			)
		}

		normalized = append(normalized, message)
	}

	return normalized, nil
}

func NormalizeDraftSalesContext(
	context DraftSalesContext,
) (DraftSalesContext, error) {
	context.Currency = strings.ToUpper(
		strings.TrimSpace(context.Currency),
	)
	if !validDraftCurrency(context.Currency) {
		return DraftSalesContext{}, fmt.Errorf(
			"%w: invalid sales context currency",
			ErrInvalidDraft,
		)
	}
	if len(context.Packages) > MaximumDraftSalesPackages {
		return DraftSalesContext{}, fmt.Errorf(
			"%w: sales context exceeds %d packages",
			ErrInvalidDraft,
			MaximumDraftSalesPackages,
		)
	}
	if len(context.Resources) > MaximumDraftSalesResources {
		return DraftSalesContext{}, fmt.Errorf(
			"%w: sales context exceeds %d resources",
			ErrInvalidDraft,
			MaximumDraftSalesResources,
		)
	}
	if !context.ShowResources && len(context.Resources) > 0 {
		return DraftSalesContext{}, fmt.Errorf(
			"%w: hidden resources cannot enter sales context",
			ErrInvalidDraft,
		)
	}

	normalizedPackages := make(
		[]DraftSalesPackage,
		0,
		len(context.Packages),
	)
	normalizedResources := make(
		[]DraftSalesResource,
		0,
		len(context.Resources),
	)
	totalRunes := utf8.RuneCountInString(context.Currency)

	for _, item := range context.Packages {
		item.Name = strings.TrimSpace(item.Name)
		item.Description = strings.TrimSpace(item.Description)

		if err := validateDraftSalesText(
			item.Name,
			item.Description,
		); err != nil {
			return DraftSalesContext{}, err
		}
		if item.GuestCapacity != nil && *item.GuestCapacity <= 0 {
			return DraftSalesContext{}, fmt.Errorf(
				"%w: invalid package guest capacity",
				ErrInvalidDraft,
			)
		}
		if err := validateDraftSalesPrice(
			item.Price,
			context.ShowPrices,
		); err != nil {
			return DraftSalesContext{}, err
		}

		totalRunes += utf8.RuneCountInString(item.Name)
		totalRunes += utf8.RuneCountInString(item.Description)
		normalizedPackages = append(normalizedPackages, item)
	}

	for _, item := range context.Resources {
		item.Name = strings.TrimSpace(item.Name)
		item.Description = strings.TrimSpace(item.Description)
		item.Category = strings.TrimSpace(item.Category)
		item.Type = strings.TrimSpace(item.Type)
		item.PricingUnit = strings.TrimSpace(item.PricingUnit)

		if err := validateDraftSalesText(
			item.Name,
			item.Description,
		); err != nil {
			return DraftSalesContext{}, err
		}
		if item.Type == "" || item.PricingUnit == "" {
			return DraftSalesContext{}, fmt.Errorf(
				"%w: invalid resource sales metadata",
				ErrInvalidDraft,
			)
		}
		if utf8.RuneCountInString(item.Category) >
			MaximumDraftSalesMetadataRunes ||
			utf8.RuneCountInString(item.Type) >
				MaximumDraftSalesMetadataRunes ||
			utf8.RuneCountInString(item.PricingUnit) >
				MaximumDraftSalesMetadataRunes {
			return DraftSalesContext{}, fmt.Errorf(
				"%w: sales resource metadata is too long",
				ErrInvalidDraft,
			)
		}
		if err := validateDraftSalesPrice(
			item.Price,
			context.ShowPrices,
		); err != nil {
			return DraftSalesContext{}, err
		}

		totalRunes += utf8.RuneCountInString(item.Name)
		totalRunes += utf8.RuneCountInString(item.Description)
		totalRunes += utf8.RuneCountInString(item.Category)
		totalRunes += utf8.RuneCountInString(item.Type)
		totalRunes += utf8.RuneCountInString(item.PricingUnit)
		normalizedResources = append(normalizedResources, item)
	}

	if totalRunes > MaximumDraftSalesContextRunes {
		return DraftSalesContext{}, fmt.Errorf(
			"%w: sales context exceeds %d runes",
			ErrInvalidDraft,
			MaximumDraftSalesContextRunes,
		)
	}

	context.Packages = normalizedPackages
	context.Resources = normalizedResources

	return context, nil
}

func validDraftCurrency(value string) bool {
	if len(value) != 3 {
		return false
	}
	for _, character := range value {
		if character < 'A' || character > 'Z' {
			return false
		}
	}
	return true
}

func validateDraftSalesText(
	name string,
	description string,
) error {
	if name == "" || description == "" ||
		utf8.RuneCountInString(name) >
			MaximumDraftSalesNameRunes ||
		utf8.RuneCountInString(description) >
			MaximumDraftSalesDescriptionRunes {
		return fmt.Errorf(
			"%w: invalid sales context text",
			ErrInvalidDraft,
		)
	}
	return nil
}

func validateDraftSalesPrice(
	price *float64,
	showPrices bool,
) error {
	if !showPrices && price != nil {
		return fmt.Errorf(
			"%w: hidden prices cannot enter sales context",
			ErrInvalidDraft,
		)
	}
	if price != nil &&
		(*price < 0 || math.IsNaN(*price) || math.IsInf(*price, 0)) {
		return fmt.Errorf(
			"%w: invalid sales context price",
			ErrInvalidDraft,
		)
	}
	return nil
}

func (*RulesDraftProvider) GenerateDraft(
	_ context.Context,
	request DraftRequest,
) (DraftResult, error) {
	var body string

	switch request.Kind {
	case DraftKindInitial:
		body = initialResponseDraft(request.ContactName)
	case DraftKindFollowUp:
		body = followUpResponseDraft()
	default:
		return DraftResult{}, fmt.Errorf(
			"%w: unsupported kind %q",
			ErrInvalidDraft,
			request.Kind,
		)
	}

	return DraftResult{
		Body:   body,
		Engine: "WEB_CHAT_RULES",
		Model:  "DETERMINISTIC_V1",
	}, nil
}

func normalizeDraft(result DraftResult) (DraftResult, error) {
	result.Body = strings.TrimSpace(result.Body)
	result.Engine = strings.TrimSpace(result.Engine)
	result.Model = strings.TrimSpace(result.Model)

	if result.Body == "" ||
		result.Engine == "" ||
		result.Model == "" {
		return DraftResult{}, ErrInvalidDraft
	}

	if utf8.RuneCountInString(result.Body) >
		MaximumMessageLength {
		return DraftResult{}, ErrInvalidDraft
	}

	return result, nil
}
