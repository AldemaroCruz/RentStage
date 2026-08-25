package httpapi

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rentstage/rentstage/apps/api/internal/config"
	"github.com/rentstage/rentstage/apps/api/internal/core/webchat"
)

type stubWebChatDraftProvider struct{}

func (*stubWebChatDraftProvider) GenerateDraft(
	_ context.Context,
	_ webchat.DraftRequest,
) (webchat.DraftResult, error) {
	return webchat.DraftResult{
		Body:   "Borrador de prueba.",
		Engine: "TEST",
		Model:  "TEST",
	}, nil
}

func TestResolveWebChatDraftProviderUsesRulesByDefault(
	t *testing.T,
) {
	factoryCalled := false

	provider, err := resolveWebChatDraftProvider(
		context.Background(),
		config.Config{
			AssistantAIMode: "rules",
		},
		func(
			context.Context,
			string,
			string,
			string,
			time.Duration,
			int,
		) (webchat.DraftProvider, error) {
			factoryCalled = true
			return nil, errors.New("must not be called")
		},
	)
	if err != nil {
		t.Fatalf("resolve rules provider: %v", err)
	}
	if factoryCalled {
		t.Fatal("Vertex factory must not run in rules mode")
	}

	result, err := provider.GenerateDraft(
		context.Background(),
		webchat.DraftRequest{
			Kind:        webchat.DraftKindInitial,
			ContactName: "Aldemaro",
		},
	)
	if err != nil {
		t.Fatalf("generate rules draft: %v", err)
	}
	if result.Engine != "WEB_CHAT_RULES" {
		t.Fatalf("unexpected engine: %q", result.Engine)
	}
}

func TestResolveWebChatDraftProviderUsesVertexConfiguration(
	t *testing.T,
) {
	expectedProvider := &stubWebChatDraftProvider{}
	var capturedProject string
	var capturedLocation string
	var capturedModel string
	var capturedTimeout time.Duration
	var capturedTokens int

	provider, err := resolveWebChatDraftProvider(
		context.Background(),
		config.Config{
			AssistantAIMode:            "vertex",
			AssistantAIProjectID:       "rentstage-staging",
			AssistantAILocation:        "us-central1",
			AssistantAIModel:           "gemini-2.5-flash",
			AssistantAITimeout:         8 * time.Second,
			AssistantAIMaxOutputTokens: 512,
		},
		func(
			_ context.Context,
			projectID string,
			location string,
			model string,
			timeout time.Duration,
			maxOutputTokens int,
		) (webchat.DraftProvider, error) {
			capturedProject = projectID
			capturedLocation = location
			capturedModel = model
			capturedTimeout = timeout
			capturedTokens = maxOutputTokens
			return expectedProvider, nil
		},
	)
	if err != nil {
		t.Fatalf("resolve Vertex provider: %v", err)
	}
	if provider != expectedProvider {
		t.Fatal("unexpected Vertex provider")
	}
	if capturedProject != "rentstage-staging" ||
		capturedLocation != "us-central1" ||
		capturedModel != "gemini-2.5-flash" ||
		capturedTimeout != 8*time.Second ||
		capturedTokens != 512 {
		t.Fatal("Vertex configuration was not forwarded correctly")
	}
}

func TestResolveWebChatDraftProviderWrapsVertexFailure(
	t *testing.T,
) {
	providerFailure := errors.New("ADC unavailable")

	_, err := resolveWebChatDraftProvider(
		context.Background(),
		config.Config{
			AssistantAIMode: "vertex",
		},
		func(
			context.Context,
			string,
			string,
			string,
			time.Duration,
			int,
		) (webchat.DraftProvider, error) {
			return nil, providerFailure
		},
	)
	if !errors.Is(err, providerFailure) {
		t.Fatalf("expected wrapped provider failure, got %v", err)
	}
}

func TestResolveWebChatDraftProviderRejectsUnknownMode(
	t *testing.T,
) {
	_, err := resolveWebChatDraftProvider(
		context.Background(),
		config.Config{
			AssistantAIMode: "unknown",
		},
		nil,
	)
	if err == nil {
		t.Fatal("expected unsupported mode error")
	}
}
