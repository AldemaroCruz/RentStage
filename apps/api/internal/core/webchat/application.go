package webchat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rentstage/rentstage/apps/api/internal/idutil"
)

type Service struct {
	repository            *Repository
	draftProvider         DraftProvider
	fallbackDraftProvider DraftProvider
	now                   func() time.Time
}

func NewService(repository *Repository) *Service {
	return NewServiceWithDraftProvider(
		repository,
		NewRulesDraftProvider(),
	)
}

func NewServiceWithDraftProvider(
	repository *Repository,
	draftProvider DraftProvider,
) *Service {
	fallbackProvider := NewRulesDraftProvider()
	if draftProvider == nil {
		draftProvider = fallbackProvider
	}

	return &Service{
		repository:            repository,
		draftProvider:         draftProvider,
		fallbackDraftProvider: fallbackProvider,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *Service) CreateSession(
	ctx context.Context,
	tenantSlug string,
	input CreateSessionInput,
) (CreateSessionResult, map[string]string, error) {
	configuration, err := s.repository.ResolveConfiguration(
		ctx,
		tenantSlug,
	)
	if err != nil {
		return CreateSessionResult{}, nil, err
	}

	normalized, fields := normalizeCreateSession(
		input,
		configuration,
	)
	if len(fields) > 0 {
		return CreateSessionResult{}, fields, nil
	}

	rawToken, tokenHash, err := generateSessionToken()
	if err != nil {
		return CreateSessionResult{}, nil, err
	}

	now := s.now()
	salesContext, err := s.repository.LoadDraftSalesContext(
		ctx,
		configuration.TenantID,
	)
	if err != nil {
		return CreateSessionResult{}, nil, err
	}

	draft, err := s.generateDraft(
		ctx,
		DraftRequest{
			Kind:            DraftKindInitial,
			TenantName:      configuration.TenantName,
			TenantSlug:      configuration.TenantSlug,
			ContactName:     normalized.ContactName,
			CustomerMessage: normalized.Message,
			SalesContext:    salesContext,
		},
	)
	if err != nil {
		return CreateSessionResult{}, nil, fmt.Errorf(
			"generate initial web chat draft: %w",
			err,
		)
	}

	item, err := s.repository.CreateSession(
		ctx,
		configuration,
		normalized,
		tokenHash,
		now.Add(SessionDuration),
		draft,
		now,
	)
	if err != nil {
		return CreateSessionResult{}, nil, err
	}

	return CreateSessionResult{
		Session: item,
		Token:   rawToken,
	}, nil, nil
}

func (s *Service) GetSession(
	ctx context.Context,
	tenantSlug string,
	sessionID string,
	rawToken string,
) (SessionView, error) {
	tokenHash, err := normalizeSessionAccess(
		sessionID,
		rawToken,
	)
	if err != nil {
		return SessionView{}, err
	}

	return s.repository.GetSession(
		ctx,
		tenantSlug,
		sessionID,
		tokenHash,
		s.now(),
	)
}

func (s *Service) SendMessage(
	ctx context.Context,
	tenantSlug string,
	sessionID string,
	rawToken string,
	input SendMessageInput,
) (SessionView, map[string]string, error) {
	tokenHash, err := normalizeSessionAccess(
		sessionID,
		rawToken,
	)
	if err != nil {
		return SessionView{}, nil, err
	}

	normalized, fields := normalizeSendMessage(input)
	if len(fields) > 0 {
		return SessionView{}, fields, nil
	}

	now := s.now()

	preparation, err := s.repository.PrepareInboundDraft(
		ctx,
		tenantSlug,
		sessionID,
		tokenHash,
		normalized,
		now,
	)
	if err != nil {
		return SessionView{}, nil, err
	}

	var draft DraftResult
	if !preparation.Duplicate {
		draft, err = s.generateDraft(ctx, preparation.Request)
		if err != nil {
			return SessionView{}, nil, fmt.Errorf(
				"generate follow-up web chat draft: %w",
				err,
			)
		}
	}

	item, err := s.repository.AddInboundMessage(
		ctx,
		tenantSlug,
		sessionID,
		tokenHash,
		normalized,
		draft,
		now,
	)

	return item, nil, nil
}

func (s *Service) generateDraft(
	ctx context.Context,
	request DraftRequest,
) (DraftResult, error) {
	fallbackReason := DraftFallbackReasonProviderError
	result, err := s.draftProvider.GenerateDraft(ctx, request)
	if err == nil {
		normalized, normalizeErr := normalizeDraft(result)
		if normalizeErr == nil {
			normalized.GroundingReferences, normalizeErr =
				NormalizeDraftGroundingReferences(
					normalized.GroundingReferences,
					request.SalesContext,
				)
			if normalizeErr == nil {
				normalized.SalesBrief, normalizeErr =
					NormalizeDraftSalesBrief(
						normalized.SalesBrief,
						request,
					)
			}
			if normalizeErr == nil {
				return normalized, nil
			}
		}
		fallbackReason = DraftFallbackReasonInvalidResponse
	} else {
		fallbackReason = DraftFallbackReasonFromError(err)
	}

	fallback, err := s.fallbackDraftProvider.GenerateDraft(
		ctx,
		request,
	)
	if err != nil {
		return DraftResult{}, fmt.Errorf(
			"generate web chat fallback draft: %w",
			err,
		)
	}

	fallback, err = normalizeDraft(fallback)
	if err != nil {
		return DraftResult{}, fmt.Errorf(
			"normalize web chat fallback draft: %w",
			err,
		)
	}

	fallback.UsedFallback = true
	fallback.FallbackReason = fallbackReason
	fallback.GroundingReferences = []DraftGroundingReference{}
	fallback.SalesBrief = emptyDraftSalesBrief()

	return fallback, nil
}

func normalizeSessionAccess(
	sessionID string,
	rawToken string,
) (string, error) {
	sessionID = strings.TrimSpace(sessionID)
	rawToken = strings.TrimSpace(rawToken)

	if !idutil.IsUUID(sessionID) ||
		len(rawToken) != 43 {
		return "", ErrInvalidToken
	}

	return hashSessionToken(rawToken), nil
}

func initialResponseDraft(contactName string) string {
	return fmt.Sprintf(
		"¡Hola, %s! Gracias por escribirnos. "+
			"Recibimos tu consulta y estamos revisando las opciones "+
			"disponibles. Un miembro del equipo confirmará los "+
			"detalles antes de preparar una cotización.",
		strings.TrimSpace(contactName),
	)
}

func followUpResponseDraft() string {
	return "Gracias por la información adicional. " +
		"La incorporaremos a tu solicitud y un miembro " +
		"del equipo confirmará los siguientes pasos."
}
