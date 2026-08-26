package webchat

import (
	"context"
	"errors"
	"fmt"
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
)

var ErrInvalidDraft = errors.New("web chat draft is invalid")

type DraftRequest struct {
	Kind             DraftKind
	TenantName       string
	TenantSlug       string
	ContactName      string
	CustomerMessage  string
	PreviousMessages []DraftConversationMessage
}

type DraftConversationMessage struct {
	Role DraftMessageRole `json:"role"`
	Body string           `json:"body"`
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
