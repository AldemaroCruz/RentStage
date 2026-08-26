package webchat

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

type DraftKind string

const (
	DraftKindInitial  DraftKind = "INITIAL"
	DraftKindFollowUp DraftKind = "FOLLOW_UP"
)

var ErrInvalidDraft = errors.New("web chat draft is invalid")

type DraftRequest struct {
	Kind            DraftKind
	TenantName      string
	TenantSlug      string
	ContactName     string
	CustomerMessage string
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
