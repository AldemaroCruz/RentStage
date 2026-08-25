package webchat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rentstage/rentstage/apps/api/internal/idutil"
)

type Service struct {
	repository *Repository
	now        func() time.Time
}

func NewService(repository *Repository) *Service {
	return &Service{
		repository: repository,
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

	item, err := s.repository.CreateSession(
		ctx,
		configuration,
		normalized,
		tokenHash,
		now.Add(SessionDuration),
		initialResponseDraft(normalized.ContactName),
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

	item, err := s.repository.AddInboundMessage(
		ctx,
		tenantSlug,
		sessionID,
		tokenHash,
		normalized,
		followUpResponseDraft(),
		s.now(),
	)
	if err != nil {
		return SessionView{}, nil, err
	}

	return item, nil, nil
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
