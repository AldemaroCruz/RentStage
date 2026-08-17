package identity

import (
	"errors"
	"net/http"
	"strings"

	"github.com/rentstage/rentstage/apps/api/internal/idutil"
	"github.com/rentstage/rentstage/apps/api/internal/webutil"
)

type Handler struct {
	repository *Repository
	service    *Service
}

func NewHandler(repository *Repository, service *Service) *Handler {
	return &Handler{repository: repository, service: service}
}

func (h *Handler) CreateOrganization(w http.ResponseWriter, r *http.Request) {
	var input CreateOrganizationInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	item, fields, err := h.service.CreateOrganization(r.Context(), webutil.UserID(r.Context()), input)
	if len(fields) > 0 {
		webutil.WriteValidationError(w, r, fields)
		return
	}
	if errors.Is(err, ErrSlugConflict) {
		webutil.WriteValidationError(w, r, map[string]string{"slug": "Ese identificador ya está en uso."})
		return
	}
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "organization_create_failed", "Could not create the organization.")
		return
	}
	webutil.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) UpdateOrganization(w http.ResponseWriter, r *http.Request) {
	var input UpdateOrganizationInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	item, fields, err := h.service.UpdateOrganization(r.Context(), webutil.TenantID(r.Context()), input)
	if len(fields) > 0 {
		webutil.WriteValidationError(w, r, fields)
		return
	}
	if errors.Is(err, ErrSlugConflict) {
		webutil.WriteValidationError(w, r, map[string]string{"slug": "Ese identificador ya está en uso."})
		return
	}
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "organization_update_failed", "Could not update the organization.")
		return
	}
	item.Role = Role(webutil.Role(r.Context()))
	item.MembershipStatus = "ACTIVE"
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) ListTeam(w http.ResponseWriter, r *http.Request) {
	members, err := h.repository.ListTeam(r.Context(), webutil.TenantID(r.Context()))
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "team_list_failed", "Could not load the team.")
		return
	}
	invitations, err := h.repository.ListInvitations(r.Context(), webutil.TenantID(r.Context()))
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "invitation_list_failed", "Could not load invitations.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, map[string]any{"members": members, "invitations": invitations})
}

func (h *Handler) CreateInvitation(w http.ResponseWriter, r *http.Request) {
	var input CreateInvitationInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	item, fields, err := h.service.CreateInvitation(
		r.Context(), webutil.TenantID(r.Context()), webutil.UserID(r.Context()), Role(webutil.Role(r.Context())), input,
	)
	if len(fields) > 0 {
		webutil.WriteValidationError(w, r, fields)
		return
	}
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "invitation_create_failed", "Could not create the invitation.")
		return
	}
	webutil.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) InvitationPreview(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.PathValue("token"))
	if token == "" {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_invitation_token", "Invitation token is required.")
		return
	}
	item, err := h.repository.GetInvitationByToken(r.Context(), token)
	if errors.Is(err, ErrInvitationNotFound) {
		webutil.WriteError(w, r, http.StatusNotFound, "invitation_not_found", "Invitation not found.")
		return
	}
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "invitation_load_failed", "Could not load the invitation.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, item)
}

func (h *Handler) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.PathValue("token"))
	user, err := h.repository.GetUser(r.Context(), webutil.UserID(r.Context()))
	if err != nil {
		webutil.WriteError(w, r, http.StatusUnauthorized, "user_not_found", "Authenticated user was not found.")
		return
	}
	item, err := h.service.AcceptInvitation(r.Context(), token, user)
	switch {
	case errors.Is(err, ErrInvitationNotFound):
		webutil.WriteError(w, r, http.StatusNotFound, "invitation_not_found", "Invitation is invalid, expired, or no longer available.")
	case errors.Is(err, ErrMembershipAlreadyExists):
		webutil.WriteError(w, r, http.StatusConflict, "membership_already_exists", "This account already belongs to the workspace.")
	case IsInvitationEmailMismatch(err):
		webutil.WriteError(w, r, http.StatusForbidden, "invitation_email_mismatch", "Sign in with the email address that received this invitation.")
	case err != nil:
		webutil.WriteError(w, r, http.StatusInternalServerError, "invitation_accept_failed", "Could not accept the invitation.")
	default:
		webutil.WriteJSON(w, http.StatusOK, item)
	}
}

func (h *Handler) RevokeInvitation(w http.ResponseWriter, r *http.Request) {
	invitationID := r.PathValue("invitationID")
	if !idutil.IsUUID(invitationID) {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_invitation_id", "Invitation ID is invalid.")
		return
	}
	if err := h.service.RevokeInvitation(r.Context(), webutil.TenantID(r.Context()), invitationID); err != nil {
		if errors.Is(err, ErrInvitationNotFound) {
			webutil.WriteError(w, r, http.StatusNotFound, "invitation_not_found", "Invitation not found.")
			return
		}
		webutil.WriteError(w, r, http.StatusInternalServerError, "invitation_revoke_failed", "Could not revoke the invitation.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) UpdateMember(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("userID")
	if !idutil.IsUUID(userID) {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_user_id", "User ID is invalid.")
		return
	}
	var input UpdateMemberInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	item, fields, err := h.service.UpdateMember(
		r.Context(), webutil.TenantID(r.Context()), webutil.UserID(r.Context()), userID,
		Role(webutil.Role(r.Context())), input,
	)
	if len(fields) > 0 {
		webutil.WriteValidationError(w, r, fields)
		return
	}
	switch {
	case errors.Is(err, ErrMembershipNotFound):
		webutil.WriteError(w, r, http.StatusNotFound, "membership_not_found", "Membership not found.")
	case errors.Is(err, ErrLastOwner):
		webutil.WriteError(w, r, http.StatusConflict, "last_owner_required", "The final active owner cannot be changed or suspended.")
	case errors.Is(err, ErrMembershipManagementDenied):
		webutil.WriteError(w, r, http.StatusForbidden, "membership_management_denied", "Your role cannot modify this membership.")
	case err != nil:
		webutil.WriteError(w, r, http.StatusInternalServerError, "membership_update_failed", "Could not update the membership.")
	default:
		webutil.WriteJSON(w, http.StatusOK, item)
	}
}
