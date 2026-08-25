package webchat

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRulesDraftProvider(t *testing.T) {
	provider := NewRulesDraftProvider()

	result, err := provider.GenerateDraft(
		context.Background(),
		DraftRequest{
			Kind:        DraftKindInitial,
			ContactName: "Ana",
		},
	)
	if err != nil {
		t.Fatalf("generate rules draft: %v", err)
	}

	if result.Engine != "WEB_CHAT_RULES" {
		t.Fatalf("unexpected engine: %q", result.Engine)
	}
	if result.Model != "DETERMINISTIC_V1" {
		t.Fatalf("unexpected model: %q", result.Model)
	}
	if result.Body != initialResponseDraft("Ana") {
		t.Fatalf("unexpected body: %q", result.Body)
	}
}

func TestRulesDraftProviderGeneratesFollowUp(t *testing.T) {
	result, err := NewRulesDraftProvider().GenerateDraft(
		context.Background(),
		DraftRequest{
			Kind: DraftKindFollowUp,
		},
	)
	if err != nil {
		t.Fatalf("generate follow-up draft: %v", err)
	}

	if result.Body != followUpResponseDraft() {
		t.Fatalf("unexpected body: %q", result.Body)
	}
}

func TestRulesDraftProviderRejectsUnknownKind(t *testing.T) {
	_, err := NewRulesDraftProvider().GenerateDraft(
		context.Background(),
		DraftRequest{
			Kind: "UNKNOWN",
		},
	)

	if !errors.Is(err, ErrInvalidDraft) {
		t.Fatalf(
			"expected invalid draft error, got %v",
			err,
		)
	}
}

func TestNormalizeDraft(t *testing.T) {
	result, err := normalizeDraft(DraftResult{
		Body:   "  Respuesta de prueba.  ",
		Engine: "  TEST_ENGINE  ",
		Model:  "  TEST_MODEL  ",
	})
	if err != nil {
		t.Fatalf("normalize draft: %v", err)
	}

	if result.Body != "Respuesta de prueba." {
		t.Fatalf("unexpected body: %q", result.Body)
	}
	if result.Engine != "TEST_ENGINE" {
		t.Fatalf("unexpected engine: %q", result.Engine)
	}
	if result.Model != "TEST_MODEL" {
		t.Fatalf("unexpected model: %q", result.Model)
	}
}

func TestNormalizeDraftRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name   string
		result DraftResult
	}{
		{
			name: "empty body",
			result: DraftResult{
				Engine: "TEST",
				Model:  "TEST_MODEL",
			},
		},
		{
			name: "empty engine",
			result: DraftResult{
				Body:  "Respuesta",
				Model: "TEST_MODEL",
			},
		},
		{
			name: "empty model",
			result: DraftResult{
				Body:   "Respuesta",
				Engine: "TEST",
			},
		},
		{
			name: "message too long",
			result: DraftResult{
				Body: strings.Repeat(
					"a",
					MaximumMessageLength+1,
				),
				Engine: "TEST",
				Model:  "TEST_MODEL",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := normalizeDraft(test.result)

			if !errors.Is(err, ErrInvalidDraft) {
				t.Fatalf(
					"expected invalid draft error, got %v",
					err,
				)
			}
		})
	}
}
