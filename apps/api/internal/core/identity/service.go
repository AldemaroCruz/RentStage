package identity

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/rentstage/rentstage/apps/api/internal/core/audit"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type Service struct {
	repository *Repository
	audit      *audit.Repository
	webBaseURL string
}

func NewService(repository *Repository, auditRepository *audit.Repository, webBaseURL string) *Service {
	return &Service{repository: repository, audit: auditRepository, webBaseURL: strings.TrimRight(webBaseURL, "/")}
}

func NormalizeOrganization(input CreateOrganizationInput) (CreateOrganizationInput, map[string]string) {
	input.Name = strings.TrimSpace(input.Name)
	input.Slug = normalizeSlug(input.Slug)
	if input.Slug == "" {
		input.Slug = normalizeSlug(input.Name)
	}
	input.LegalName = strings.TrimSpace(input.LegalName)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.Phone = strings.TrimSpace(input.Phone)
	input.CountryCode = strings.ToUpper(strings.TrimSpace(input.CountryCode))
	input.Timezone = strings.TrimSpace(input.Timezone)
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	input.Address = strings.TrimSpace(input.Address)
	if input.CountryCode == "" {
		input.CountryCode = "SV"
	}
	if input.Timezone == "" {
		input.Timezone = "America/El_Salvador"
	}
	if input.Currency == "" {
		input.Currency = "USD"
	}

	fields := map[string]string{}
	if input.Name == "" || len(input.Name) > 160 {
		fields["name"] = "El nombre es obligatorio y debe tener 160 caracteres o menos."
	}
	if !slugPattern.MatchString(input.Slug) || len(input.Slug) > 80 {
		fields["slug"] = "Usa letras minúsculas, números y guiones; máximo 80 caracteres."
	}
	if input.Email != "" {
		parsed, err := mail.ParseAddress(input.Email)
		if err != nil || !strings.EqualFold(parsed.Address, input.Email) {
			fields["email"] = "Ingresa un correo válido."
		}
	}
	if len(input.CountryCode) != 2 {
		fields["country_code"] = "Usa un código ISO de dos letras."
	}
	if len(input.Currency) != 3 {
		fields["currency"] = "Usa un código de moneda de tres letras."
	}
	if input.Timezone == "" || len(input.Timezone) > 100 {
		fields["timezone"] = "Ingresa una zona horaria válida."
	}
	return input, fields
}

func (s *Service) CreateOrganization(ctx context.Context, userID string, input CreateOrganizationInput) (Workspace, map[string]string, error) {
	normalized, fields := NormalizeOrganization(input)
	if len(fields) > 0 {
		return Workspace{}, fields, nil
	}
	workspace, err := s.repository.CreateOrganization(ctx, userID, normalized)
	if err != nil {
		return Workspace{}, nil, err
	}
	_ = s.audit.Record(ctx, workspace.TenantID, "TENANT_CREATED", "tenant", &workspace.TenantID, map[string]any{
		"name": workspace.Name,
		"slug": workspace.Slug,
	})
	return workspace, nil, nil
}

func (s *Service) UpdateOrganization(ctx context.Context, tenantID string, input UpdateOrganizationInput) (Workspace, map[string]string, error) {
	normalized, fields := NormalizeOrganization(input)
	if len(fields) > 0 {
		return Workspace{}, fields, nil
	}
	workspace, err := s.repository.UpdateOrganization(ctx, tenantID, normalized)
	if err != nil {
		return Workspace{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "TENANT_UPDATED", "tenant", &tenantID, map[string]any{
		"name": workspace.Name,
		"slug": workspace.Slug,
	})
	return workspace, nil, nil
}

func (s *Service) CreateInvitation(ctx context.Context, tenantID, actorUserID string, actorRole Role, input CreateInvitationInput) (Invitation, map[string]string, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	role := Role(strings.ToUpper(strings.TrimSpace(input.Role)))
	fields := map[string]string{}
	parsed, err := mail.ParseAddress(email)
	if email == "" || err != nil || !strings.EqualFold(parsed.Address, email) {
		fields["email"] = "Ingresa un correo válido."
	}
	if role != RoleAdmin && role != RoleManager && role != RoleStaff {
		fields["role"] = "Selecciona ADMIN, MANAGER o STAFF."
	}
	if actorRole == RoleAdmin && role == RoleAdmin {
		fields["role"] = "Un administrador no puede invitar a otro administrador."
	}
	if len(fields) > 0 {
		return Invitation{}, fields, nil
	}
	if err := s.repository.EnsureInvitationAllowed(ctx, tenantID, email); err != nil {
		switch {
		case errors.Is(err, ErrMembershipAlreadyExists):
			fields["email"] = "Ese correo ya pertenece al equipo de esta organización."
		case errors.Is(err, ErrPendingInvitation):
			fields["email"] = "Ya existe una invitación pendiente para ese correo."
		default:
			return Invitation{}, nil, err
		}
		return Invitation{}, fields, nil
	}

	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return Invitation{}, nil, err
	}
	token := base64.RawURLEncoding.EncodeToString(random)
	item, err := s.repository.CreateInvitation(ctx, tenantID, actorUserID, email, role, InvitationHash(token), time.Now().Add(7*24*time.Hour))
	if errors.Is(err, ErrPendingInvitation) {
		return Invitation{}, map[string]string{"email": "Ya existe una invitación pendiente para ese correo."}, nil
	}
	if err != nil {
		return Invitation{}, nil, err
	}
	item.AcceptURL = s.webBaseURL + "/invites/" + token
	_ = s.audit.Record(ctx, tenantID, "MEMBERSHIP_INVITATION_CREATED", "tenant_invitation", &item.ID, map[string]any{
		"email": item.Email,
		"role":  item.Role,
	})
	return item, nil, nil
}

func (s *Service) AcceptInvitation(ctx context.Context, token string, user User) (Workspace, error) {
	workspace, err := s.repository.AcceptInvitation(ctx, strings.TrimSpace(token), user.ID, user.Email)
	if err != nil {
		return Workspace{}, err
	}
	_ = s.audit.Record(ctx, workspace.TenantID, "MEMBERSHIP_INVITATION_ACCEPTED", "tenant", &workspace.TenantID, map[string]any{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    workspace.Role,
	})
	return workspace, nil
}

func (s *Service) RevokeInvitation(ctx context.Context, tenantID, invitationID string) error {
	if err := s.repository.RevokeInvitation(ctx, tenantID, invitationID); err != nil {
		return err
	}
	_ = s.audit.Record(ctx, tenantID, "MEMBERSHIP_INVITATION_REVOKED", "tenant_invitation", &invitationID, nil)
	return nil
}

func (s *Service) UpdateMember(ctx context.Context, tenantID, actorUserID, targetUserID string, actorRole Role, input UpdateMemberInput) (TeamMember, map[string]string, error) {
	targetMembership, err := s.repository.GetMembership(ctx, targetUserID, tenantID)
	if err != nil {
		return TeamMember{}, nil, err
	}
	if !CanManageMember(actorRole, targetMembership.Role) {
		return TeamMember{}, nil, ErrMembershipManagementDenied
	}

	fields := map[string]string{}
	var role *Role
	if input.Role != nil {
		value := Role(strings.ToUpper(strings.TrimSpace(*input.Role)))
		if value != RoleAdmin && value != RoleManager && value != RoleStaff {
			fields["role"] = "Selecciona ADMIN, MANAGER o STAFF."
		} else if actorRole == RoleAdmin && value == RoleAdmin {
			fields["role"] = "Un administrador no puede promover a otro administrador."
		} else {
			role = &value
		}
	}
	var status *string
	if input.Status != nil {
		value := strings.ToUpper(strings.TrimSpace(*input.Status))
		if value != "ACTIVE" && value != "SUSPENDED" {
			fields["status"] = "El estado debe ser ACTIVE o SUSPENDED."
		} else {
			status = &value
		}
	}
	if targetUserID == actorUserID && status != nil && *status == "SUSPENDED" {
		fields["status"] = "No puedes suspender tu propia membresía."
	}
	if len(fields) > 0 {
		return TeamMember{}, fields, nil
	}
	item, err := s.repository.UpdateMember(ctx, tenantID, targetUserID, role, status)
	if err != nil {
		return TeamMember{}, nil, err
	}
	_ = s.audit.Record(ctx, tenantID, "MEMBERSHIP_UPDATED", "user", &targetUserID, map[string]any{
		"role":   item.Role,
		"status": item.Status,
	})
	return item, nil, nil
}

func normalizeSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, char := range value {
		valid := (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')
		if valid {
			builder.WriteRune(char)
			lastDash = false
			continue
		}
		if !lastDash && builder.Len() > 0 {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

func IsInvitationEmailMismatch(err error) bool {
	return err != nil && strings.Contains(err.Error(), "does not match")
}

func IsNotFound(err error) bool {
	return errors.Is(err, ErrUserNotFound) || errors.Is(err, ErrMembershipNotFound) || errors.Is(err, ErrInvitationNotFound)
}
