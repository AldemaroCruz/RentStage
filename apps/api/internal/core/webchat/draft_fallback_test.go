package webchat

import (
	"context"
	"errors"
	"testing"
)

type stubDraftProvider struct {
	result DraftResult
	err    error
}

func (p stubDraftProvider) GenerateDraft(
	_ context.Context,
	_ DraftRequest,
) (DraftResult, error) {
	return p.result, p.err
}

func TestGenerateDraftUsesPrimaryProvider(t *testing.T) {
	service := NewServiceWithDraftProvider(
		nil,
		stubDraftProvider{
			result: DraftResult{
				Body:   " Borrador principal. ",
				Engine: " TEST_PROVIDER ",
				Model:  " TEST_MODEL ",
			},
		},
	)

	result, err := service.generateDraft(
		context.Background(),
		DraftRequest{Kind: DraftKindFollowUp},
	)
	if err != nil {
		t.Fatalf("generate primary draft: %v", err)
	}
	if result.UsedFallback {
		t.Fatal("did not expect fallback")
	}
	if result.Body != "Borrador principal." {
		t.Fatalf("unexpected body: %q", result.Body)
	}
	if result.Engine != "TEST_PROVIDER" {
		t.Fatalf("unexpected engine: %q", result.Engine)
	}
}

func TestGenerateDraftUsesFallbackOnProviderFailure(
	t *testing.T,
) {
	service := NewServiceWithDraftProvider(
		nil,
		stubDraftProvider{
			err: errors.New("provider failed"),
		},
	)

	result, err := service.generateDraft(
		context.Background(),
		DraftRequest{Kind: DraftKindFollowUp},
	)
	if err != nil {
		t.Fatalf("generate fallback draft: %v", err)
	}
	if !result.UsedFallback {
		t.Fatal("expected fallback draft")
	}
	if result.Engine != "WEB_CHAT_RULES" {
		t.Fatalf("unexpected engine: %q", result.Engine)
	}
	if result.Model != "DETERMINISTIC_V1" {
		t.Fatalf("unexpected model: %q", result.Model)
	}
	if result.FallbackReason != DraftFallbackReasonProviderError {
		t.Fatalf(
			"unexpected fallback reason: %q",
			result.FallbackReason,
		)
	}
}

func TestGenerateDraftUsesFallbackForInvalidResult(
	t *testing.T,
) {
	service := NewServiceWithDraftProvider(
		nil,
		stubDraftProvider{
			result: DraftResult{
				Body:   "   ",
				Engine: "TEST_PROVIDER",
				Model:  "TEST_MODEL",
			},
		},
	)

	result, err := service.generateDraft(
		context.Background(),
		DraftRequest{Kind: DraftKindFollowUp},
	)
	if err != nil {
		t.Fatalf("generate fallback draft: %v", err)
	}
	if !result.UsedFallback {
		t.Fatal("expected fallback for invalid result")
	}
	if result.FallbackReason != DraftFallbackReasonInvalidResponse {
		t.Fatalf(
			"unexpected fallback reason: %q",
			result.FallbackReason,
		)
	}
}

func TestGenerateDraftRecordsAllowlistedTimeoutReason(t *testing.T) {
	service := NewServiceWithDraftProvider(
		nil,
		stubDraftProvider{
			err: NewDraftProviderFailure(
				DraftFallbackReasonTimeout,
				context.DeadlineExceeded,
			),
		},
	)

	result, err := service.generateDraft(
		context.Background(),
		DraftRequest{Kind: DraftKindFollowUp},
	)
	if err != nil {
		t.Fatalf("generate timeout fallback draft: %v", err)
	}
	if result.FallbackReason != DraftFallbackReasonTimeout {
		t.Fatalf(
			"unexpected fallback reason: %q",
			result.FallbackReason,
		)
	}
}

func TestDraftProviderFailureRejectsUntrustedReason(t *testing.T) {
	err := NewDraftProviderFailure(
		DraftFallbackReason("UNTRUSTED_REASON"),
		errors.New("provider failed"),
	)

	reason := DraftFallbackReasonFromError(err)
	if reason != DraftFallbackReasonProviderError {
		t.Fatalf("unexpected fallback reason: %q", reason)
	}
}

func TestGenerateDraftRejectsProviderOwnedFallbackMetadata(
	t *testing.T,
) {
	service := NewServiceWithDraftProvider(
		nil,
		stubDraftProvider{
			result: DraftResult{
				Body:           "Borrador no confiable.",
				Engine:         "TEST_PROVIDER",
				Model:          "TEST_MODEL",
				UsedFallback:   true,
				FallbackReason: DraftFallbackReasonTimeout,
			},
		},
	)

	result, err := service.generateDraft(
		context.Background(),
		DraftRequest{Kind: DraftKindFollowUp},
	)
	if err != nil {
		t.Fatalf("generate protected fallback draft: %v", err)
	}
	if result.Engine != "WEB_CHAT_RULES" ||
		result.FallbackReason != DraftFallbackReasonInvalidResponse {
		t.Fatalf("unexpected protected fallback: %#v", result)
	}
}

func TestNewServiceWithNilProviderUsesRules(t *testing.T) {
	service := NewServiceWithDraftProvider(nil, nil)

	result, err := service.generateDraft(
		context.Background(),
		DraftRequest{Kind: DraftKindFollowUp},
	)
	if err != nil {
		t.Fatalf("generate rules draft: %v", err)
	}
	if result.UsedFallback {
		t.Fatal("did not expect fallback for default provider")
	}
	if result.Engine != "WEB_CHAT_RULES" {
		t.Fatalf("unexpected engine: %q", result.Engine)
	}
	if result.FallbackReason != "" {
		t.Fatalf(
			"default rules must not have a fallback reason: %q",
			result.FallbackReason,
		)
	}
}
