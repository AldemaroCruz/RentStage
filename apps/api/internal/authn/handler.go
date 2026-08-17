package authn

import (
	"errors"
	"net/http"
	"strings"

	"github.com/rentstage/rentstage/apps/api/internal/config"
	"github.com/rentstage/rentstage/apps/api/internal/core/identity"
	"github.com/rentstage/rentstage/apps/api/internal/idutil"
	"github.com/rentstage/rentstage/apps/api/internal/webutil"
)

type Handler struct {
	service *Service
	cfg     config.Config
}

func NewHandler(service *Service, cfg config.Config) *Handler {
	return &Handler{service: service, cfg: cfg}
}

func (h *Handler) CSRF(w http.ResponseWriter, r *http.Request) {
	token, err := IssueCSRFToken(w, h.cfg)
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "csrf_generation_failed", "Could not initialize request protection.")
		return
	}
	webutil.WriteJSON(w, http.StatusOK, map[string]string{"csrf_token": token})
}

func (h *Handler) Session(w http.ResponseWriter, r *http.Request) {
	var input SessionInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	sessionCookie, user, err := h.service.CreateSession(r.Context(), input.IDToken)
	switch {
	case errors.Is(err, ErrInvalidSession):
		webutil.WriteError(w, r, http.StatusUnauthorized, "invalid_identity_token", "The sign-in token is invalid or expired.")
		return
	case errors.Is(err, ErrRecentLoginRequired):
		webutil.WriteError(w, r, http.StatusUnauthorized, "recent_login_required", "Please sign in again before creating a session.")
		return
	case errors.Is(err, ErrEmailNotVerified):
		webutil.WriteError(w, r, http.StatusForbidden, "email_not_verified", "Verify your email before signing in.")
		return
	case errors.Is(err, ErrUserDisabled):
		webutil.WriteError(w, r, http.StatusForbidden, "user_disabled", "This account is disabled.")
		return
	case err != nil:
		webutil.WriteError(w, r, http.StatusInternalServerError, "session_create_failed", "Could not create the session.")
		return
	}

	requestedTenantID := tenantCookieValue(r, h.cfg)
	me, err := h.service.BuildMe(r.Context(), user, requestedTenantID)
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "session_context_failed", "Could not load the workspace context.")
		return
	}
	SetSessionCookie(w, h.cfg, sessionCookie)
	if me.ActiveWorkspace != nil {
		SetTenantCookie(w, h.cfg, me.ActiveWorkspace.TenantID)
	}
	webutil.WriteJSON(w, http.StatusOK, me)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	ClearAuthCookies(w, h.cfg)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	user := identity.User{
		ID:          webutil.UserID(r.Context()),
		IdentityUID: webutil.IdentityUID(r.Context()),
		Email:       webutil.UserEmail(r.Context()),
		DisplayName: webutil.UserName(r.Context()),
		Status:      "ACTIVE",
	}
	stored, err := h.service.identity.GetUser(r.Context(), user.ID)
	if err == nil {
		user = stored
	}
	me, err := h.service.BuildMe(r.Context(), user, tenantCookieValue(r, h.cfg))
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "identity_context_failed", "Could not load the current user.")
		return
	}
	if me.ActiveWorkspace != nil {
		SetTenantCookie(w, h.cfg, me.ActiveWorkspace.TenantID)
	}
	webutil.WriteJSON(w, http.StatusOK, me)
}

func (h *Handler) SelectTenant(w http.ResponseWriter, r *http.Request) {
	var input SelectTenantInput
	if err := webutil.DecodeJSON(r, &input); err != nil {
		webutil.WriteError(w, r, http.StatusBadRequest, "invalid_json", "The request body is not valid JSON.")
		return
	}
	if !idutil.IsUUID(input.TenantID) {
		webutil.WriteValidationError(w, r, map[string]string{"tenant_id": "Workspace ID is invalid."})
		return
	}
	user, err := h.service.identity.GetUser(r.Context(), webutil.UserID(r.Context()))
	if err != nil {
		webutil.WriteError(w, r, http.StatusUnauthorized, "user_not_found", "Authenticated user was not found.")
		return
	}
	me, err := h.service.SelectTenant(r.Context(), user, input.TenantID)
	if errors.Is(err, ErrTenantAccessDenied) {
		webutil.WriteError(w, r, http.StatusForbidden, "tenant_access_denied", "You do not have access to this workspace.")
		return
	}
	if err != nil {
		webutil.WriteError(w, r, http.StatusInternalServerError, "tenant_select_failed", "Could not change the active workspace.")
		return
	}
	SetTenantCookie(w, h.cfg, input.TenantID)
	webutil.WriteJSON(w, http.StatusOK, me)
}

func tenantCookieValue(r *http.Request, cfg config.Config) string {
	cookie, err := r.Cookie(cfg.TenantCookieName)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cookie.Value)
}
