package authn

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	firebase "firebase.google.com/go/v4"
	firebaseauth "firebase.google.com/go/v4/auth"
	"github.com/rentstage/rentstage/apps/api/internal/config"
	"github.com/rentstage/rentstage/apps/api/internal/core/identity"
)

var (
	ErrInvalidSession      = errors.New("invalid session")
	ErrRecentLoginRequired = errors.New("recent login required")
	ErrEmailNotVerified    = errors.New("email is not verified")
	ErrUserDisabled        = errors.New("user is disabled")
	ErrTenantAccessDenied  = errors.New("tenant access denied")
)

type Service struct {
	client   *firebaseauth.Client
	identity *identity.Repository
	cfg      config.Config
}

func New(ctx context.Context, cfg config.Config, identityRepository *identity.Repository) (*Service, error) {
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: cfg.FirebaseProjectID})
	if err != nil {
		return nil, fmt.Errorf("initialize firebase app: %w", err)
	}
	client, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("initialize firebase auth: %w", err)
	}
	return &Service{client: client, identity: identityRepository, cfg: cfg}, nil
}

func (s *Service) BootstrapLocalOwner(ctx context.Context) error {
	if !s.cfg.LocalAuthBootstrap {
		return nil
	}
	record, err := s.client.GetUserByEmail(ctx, s.cfg.LocalOwnerEmail)
	if err != nil {
		if !firebaseauth.IsUserNotFound(err) {
			return fmt.Errorf("lookup local owner in auth emulator: %w", err)
		}
		input := (&firebaseauth.UserToCreate{}).
			UID("rentstage-local-owner").
			Email(s.cfg.LocalOwnerEmail).
			Password(s.cfg.LocalOwnerPassword).
			DisplayName(s.cfg.LocalOwnerDisplayName).
			EmailVerified(true)
		record, err = s.client.CreateUser(ctx, input)
		if err != nil {
			return fmt.Errorf("create local owner in auth emulator: %w", err)
		}
	}
	_, err = s.identity.SyncUser(ctx, identityProfile(record))
	if err != nil {
		return fmt.Errorf("link local owner to rentstage user: %w", err)
	}
	return nil
}

func (s *Service) CreateSession(ctx context.Context, idToken string) (string, identity.User, error) {
	idToken = strings.TrimSpace(idToken)
	if idToken == "" {
		return "", identity.User{}, ErrInvalidSession
	}
	decoded, err := s.client.VerifyIDTokenAndCheckRevoked(ctx, idToken)
	if err != nil {
		return "", identity.User{}, ErrInvalidSession
	}
	authenticatedAt := time.Unix(decoded.AuthTime, 0)
	authenticationAge := time.Since(authenticatedAt)
	if decoded.AuthTime == 0 || authenticationAge < -time.Minute || authenticationAge > 5*time.Minute {
		return "", identity.User{}, ErrRecentLoginRequired
	}
	record, err := s.client.GetUser(ctx, decoded.UID)
	if err != nil {
		return "", identity.User{}, ErrInvalidSession
	}
	if record.Disabled {
		return "", identity.User{}, ErrUserDisabled
	}
	if s.cfg.RequireVerifiedEmail && !record.EmailVerified {
		return "", identity.User{}, ErrEmailNotVerified
	}
	user, err := s.identity.SyncUser(ctx, identityProfile(record))
	if err != nil {
		return "", identity.User{}, err
	}
	if user.Status != "ACTIVE" {
		return "", identity.User{}, ErrUserDisabled
	}
	cookie, err := s.client.SessionCookie(ctx, idToken, s.cfg.SessionDuration)
	if err != nil {
		return "", identity.User{}, fmt.Errorf("create firebase session cookie: %w", err)
	}
	return cookie, user, nil
}

func (s *Service) VerifySession(ctx context.Context, sessionCookie string) (identity.User, error) {
	if strings.TrimSpace(sessionCookie) == "" {
		return identity.User{}, ErrInvalidSession
	}
	decoded, err := s.client.VerifySessionCookieAndCheckRevoked(ctx, sessionCookie)
	if err != nil {
		return identity.User{}, ErrInvalidSession
	}
	user, err := s.identity.GetUserByIdentityUID(ctx, decoded.UID)
	if errors.Is(err, identity.ErrUserNotFound) {
		record, lookupErr := s.client.GetUser(ctx, decoded.UID)
		if lookupErr != nil {
			return identity.User{}, ErrInvalidSession
		}
		user, err = s.identity.SyncUser(ctx, identityProfile(record))
	}
	if err != nil {
		return identity.User{}, err
	}
	if user.Status != "ACTIVE" {
		return identity.User{}, ErrUserDisabled
	}
	return user, nil
}

func (s *Service) BuildMe(ctx context.Context, user identity.User, requestedTenantID string) (Me, error) {
	workspaces, err := s.identity.ListWorkspaces(ctx, user.ID)
	if err != nil {
		return Me{}, err
	}
	activeID := strings.TrimSpace(requestedTenantID)
	if activeID == "" {
		activeID, err = s.identity.PreferredTenant(ctx, user.ID)
		if err != nil {
			return Me{}, err
		}
	}
	var active *identity.Workspace
	for index := range workspaces {
		if workspaces[index].TenantID == activeID {
			candidate := workspaces[index]
			active = &candidate
			break
		}
	}
	if active == nil && len(workspaces) > 0 {
		candidate := workspaces[0]
		active = &candidate
		_ = s.identity.SetActiveTenant(ctx, user.ID, candidate.TenantID)
	}
	result := Me{User: user, Workspaces: workspaces, ActiveWorkspace: active}
	if active != nil {
		result.Permissions = identity.PermissionsForRole(active.Role)
	}
	return result, nil
}

func (s *Service) SelectTenant(ctx context.Context, user identity.User, tenantID string) (Me, error) {
	membership, err := s.identity.GetMembership(ctx, user.ID, tenantID)
	if err != nil || membership.Status != "ACTIVE" {
		return Me{}, ErrTenantAccessDenied
	}
	if err := s.identity.SetActiveTenant(ctx, user.ID, tenantID); err != nil {
		return Me{}, err
	}
	return s.BuildMe(ctx, user, tenantID)
}

func identityProfile(record *firebaseauth.UserRecord) identity.IdentityProfile {
	return identity.IdentityProfile{
		UID:           record.UID,
		Email:         record.Email,
		DisplayName:   record.DisplayName,
		AvatarURL:     record.PhotoURL,
		EmailVerified: record.EmailVerified,
	}
}
