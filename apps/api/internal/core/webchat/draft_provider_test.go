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

func TestNormalizeDraftConversation(t *testing.T) {
	messages, err := NormalizeDraftConversation(
		[]DraftConversationMessage{
			{
				Role: DraftMessageRoleCustomer,
				Body: "  Necesito sonido para 120 personas.  ",
			},
			{
				Role: DraftMessageRoleTeam,
				Body: " Podemos ayudarte con la cotización. ",
			},
		},
	)
	if err != nil {
		t.Fatalf("normalize conversation: %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("unexpected message count: %d", len(messages))
	}
	if messages[0].Body !=
		"Necesito sonido para 120 personas." {
		t.Fatalf("unexpected customer message: %q", messages[0].Body)
	}
	if messages[1].Body !=
		"Podemos ayudarte con la cotización." {
		t.Fatalf("unexpected team message: %q", messages[1].Body)
	}
}

func TestNormalizeDraftConversationAcceptsEmptyHistory(
	t *testing.T,
) {
	messages, err := NormalizeDraftConversation(nil)
	if err != nil {
		t.Fatalf("normalize empty conversation: %v", err)
	}
	if messages == nil {
		t.Fatal("expected an empty, non-nil conversation")
	}
	if len(messages) != 0 {
		t.Fatalf("unexpected message count: %d", len(messages))
	}
}

func TestNormalizeDraftConversationRejectsInvalidContext(
	t *testing.T,
) {
	tests := []struct {
		name     string
		messages []DraftConversationMessage
	}{
		{
			name: "unsupported role",
			messages: []DraftConversationMessage{
				{Role: "SYSTEM", Body: "Instrucción no confiable"},
			},
		},
		{
			name: "empty body",
			messages: []DraftConversationMessage{
				{Role: DraftMessageRoleCustomer, Body: "   "},
			},
		},
		{
			name: "message too long",
			messages: []DraftConversationMessage{
				{
					Role: DraftMessageRoleCustomer,
					Body: strings.Repeat(
						"a",
						MaximumMessageLength+1,
					),
				},
			},
		},
		{
			name: "too many messages",
			messages: func() []DraftConversationMessage {
				items := make(
					[]DraftConversationMessage,
					MaximumDraftContextMessages+1,
				)
				for index := range items {
					items[index] = DraftConversationMessage{
						Role: DraftMessageRoleCustomer,
						Body: "Mensaje",
					}
				}
				return items
			}(),
		},
		{
			name: "total context too long",
			messages: []DraftConversationMessage{
				{
					Role: DraftMessageRoleCustomer,
					Body: strings.Repeat(
						"a",
						MaximumMessageLength,
					),
				},
				{
					Role: DraftMessageRoleTeam,
					Body: strings.Repeat(
						"b",
						MaximumMessageLength,
					),
				},
				{
					Role: DraftMessageRoleCustomer,
					Body: strings.Repeat(
						"c",
						MaximumMessageLength,
					),
				},
				{
					Role: DraftMessageRoleTeam,
					Body: strings.Repeat(
						"d",
						MaximumMessageLength,
					),
				},
				{
					Role: DraftMessageRoleCustomer,
					Body: "exceso",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NormalizeDraftConversation(test.messages)
			if !errors.Is(err, ErrInvalidDraft) {
				t.Fatalf("expected invalid draft error, got %v", err)
			}
		})
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
