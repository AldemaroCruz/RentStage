package vertexai

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/rentstage/rentstage/apps/api/internal/core/webchat"
)

func TestNormalizeMaxOutputTokens(t *testing.T) {
	tests := []struct {
		name    string
		value   int
		want    int32
		wantErr bool
	}{
		{name: "minimum", value: minimumOutputTokens, want: 64},
		{name: "maximum", value: maximumOutputTokens, want: 2048},
		{name: "below minimum", value: minimumOutputTokens - 1, wantErr: true},
		{name: "above maximum", value: maximumOutputTokens + 1, wantErr: true},
		{name: "platform maximum", value: math.MaxInt, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeMaxOutputTokens(test.value)
			if test.wantErr {
				if !errors.Is(err, ErrInvalidResponse) {
					t.Fatalf("expected invalid response error, got %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("normalize token limit: %v", err)
			}
			if got != test.want {
				t.Fatalf("normalized token limit = %d, want %d", got, test.want)
			}
		})
	}
}

type stubGenerator struct {
	response            string
	err                 error
	request             generationRequest
	calls               int
	waitForCancellation bool
}

func (g *stubGenerator) Generate(
	ctx context.Context,
	request generationRequest,
) (string, error) {
	g.calls++
	g.request = request

	if g.waitForCancellation {
		<-ctx.Done()
		return "", ctx.Err()
	}

	return g.response, g.err
}

func TestDraftProviderGeneratesStructuredDraft(t *testing.T) {
	generator := &stubGenerator{
		response: `{"reply":"Gracias por la información. El equipo confirmará los detalles."}`,
	}
	provider := newDraftProvider(
		generator,
		"gemini-2.5-flash",
		time.Second,
		512,
	)

	result, err := provider.GenerateDraft(
		context.Background(),
		webchat.DraftRequest{
			Kind:            webchat.DraftKindInitial,
			TenantName:      "AudioPro Demo",
			ContactName:     "Aldemaro",
			CustomerMessage: "Necesito sonido para 150 personas.",
		},
	)
	if err != nil {
		t.Fatalf("generate draft: %v", err)
	}

	if result.Body !=
		"Gracias por la información. El equipo confirmará los detalles." {
		t.Fatalf("unexpected body: %q", result.Body)
	}
	if result.Engine != engineName {
		t.Fatalf("unexpected engine: %q", result.Engine)
	}
	if result.Model != "gemini-2.5-flash" {
		t.Fatalf("unexpected model: %q", result.Model)
	}
	if result.UsedFallback {
		t.Fatal("Vertex provider must not mark its own result as fallback")
	}
	if generator.calls != 1 {
		t.Fatalf("unexpected generator calls: %d", generator.calls)
	}
	if !strings.Contains(
		generator.request.Prompt,
		`"customer_message":"Necesito sonido para 150 personas."`,
	) {
		t.Fatalf(
			"prompt does not contain encoded customer data: %q",
			generator.request.Prompt,
		)
	}
	if generator.request.MaxOutputTokens != 512 {
		t.Fatalf(
			"unexpected max output tokens: %d",
			generator.request.MaxOutputTokens,
		)
	}
}

func TestDraftProviderRejectsInvalidResponses(t *testing.T) {
	responses := []string{
		"",
		"not-json",
		`{"reply":""}`,
		`{"reply":"válido","extra":true}`,
		`{"reply":"válido"} trailing`,
	}

	for _, response := range responses {
		t.Run(response, func(t *testing.T) {
			provider := newDraftProvider(
				&stubGenerator{response: response},
				"gemini-2.5-flash",
				time.Second,
				512,
			)

			_, err := provider.GenerateDraft(
				context.Background(),
				webchat.DraftRequest{
					Kind: webchat.DraftKindFollowUp,
				},
			)
			if !errors.Is(err, ErrInvalidResponse) {
				t.Fatalf(
					"expected invalid response error, got %v",
					err,
				)
			}
		})
	}
}

func TestDraftProviderPropagatesGeneratorFailure(t *testing.T) {
	providerFailure := errors.New("provider unavailable")
	provider := newDraftProvider(
		&stubGenerator{err: providerFailure},
		"gemini-2.5-flash",
		time.Second,
		512,
	)

	_, err := provider.GenerateDraft(
		context.Background(),
		webchat.DraftRequest{
			Kind: webchat.DraftKindFollowUp,
		},
	)
	if !errors.Is(err, providerFailure) {
		t.Fatalf("expected provider failure, got %v", err)
	}
}

func TestDraftProviderEnforcesTimeout(t *testing.T) {
	provider := newDraftProvider(
		&stubGenerator{waitForCancellation: true},
		"gemini-2.5-flash",
		10*time.Millisecond,
		512,
	)

	_, err := provider.GenerateDraft(
		context.Background(),
		webchat.DraftRequest{
			Kind: webchat.DraftKindFollowUp,
		},
	)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
}

func TestDraftProviderRejectsUnsupportedKindBeforeGeneration(
	t *testing.T,
) {
	generator := &stubGenerator{}
	provider := newDraftProvider(
		generator,
		"gemini-2.5-flash",
		time.Second,
		512,
	)

	_, err := provider.GenerateDraft(
		context.Background(),
		webchat.DraftRequest{
			Kind: webchat.DraftKind("UNKNOWN"),
		},
	)
	if !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("expected invalid response error, got %v", err)
	}
	if generator.calls != 0 {
		t.Fatalf("unexpected generator calls: %d", generator.calls)
	}
}

func TestGenerationConfigRequiresStructuredJSON(t *testing.T) {
	config := generateContentConfig(512)

	if config.ResponseMIMEType != "application/json" {
		t.Fatalf(
			"unexpected MIME type: %q",
			config.ResponseMIMEType,
		)
	}
	if config.MaxOutputTokens != 512 {
		t.Fatalf(
			"unexpected token limit: %d",
			config.MaxOutputTokens,
		)
	}
	if config.SystemInstruction == nil {
		t.Fatal("system instruction is required")
	}
	if config.ResponseSchema == nil {
		t.Fatal("response schema is required")
	}

	replySchema := config.ResponseSchema.Properties["reply"]
	if replySchema == nil {
		t.Fatal("reply schema is required")
	}
}
