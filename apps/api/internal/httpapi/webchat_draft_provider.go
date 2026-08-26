package httpapi

import (
	"context"
	"fmt"
	"time"

	"github.com/rentstage/rentstage/apps/api/internal/config"
	"github.com/rentstage/rentstage/apps/api/internal/core/webchat"
	vertexintegration "github.com/rentstage/rentstage/apps/api/internal/integrations/vertexai"
)

type vertexDraftProviderFactory func(
	ctx context.Context,
	projectID string,
	location string,
	model string,
	timeout time.Duration,
	maxOutputTokens int,
) (webchat.DraftProvider, error)

func newVertexDraftProvider(
	ctx context.Context,
	projectID string,
	location string,
	model string,
	timeout time.Duration,
	maxOutputTokens int,
) (webchat.DraftProvider, error) {
	return vertexintegration.NewDraftProvider(
		ctx,
		projectID,
		location,
		model,
		timeout,
		maxOutputTokens,
	)
}

func resolveWebChatDraftProvider(
	ctx context.Context,
	cfg config.Config,
	vertexFactory vertexDraftProviderFactory,
) (webchat.DraftProvider, error) {
	switch cfg.AssistantAIMode {
	case "rules":
		return webchat.NewRulesDraftProvider(), nil

	case "vertex":
		if vertexFactory == nil {
			return nil, fmt.Errorf(
				"initialize Vertex AI web chat draft provider: factory is nil",
			)
		}

		provider, err := vertexFactory(
			ctx,
			cfg.AssistantAIProjectID,
			cfg.AssistantAILocation,
			cfg.AssistantAIModel,
			cfg.AssistantAITimeout,
			cfg.AssistantAIMaxOutputTokens,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"initialize Vertex AI web chat draft provider: %w",
				err,
			)
		}

		return provider, nil

	default:
		return nil, fmt.Errorf(
			"unsupported ASSISTANT_AI_MODE %q",
			cfg.AssistantAIMode,
		)
	}
}
