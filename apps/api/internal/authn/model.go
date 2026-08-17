package authn

import "github.com/rentstage/rentstage/apps/api/internal/core/identity"

type Me struct {
	User            identity.User         `json:"user"`
	Workspaces      []identity.Workspace  `json:"workspaces"`
	ActiveWorkspace *identity.Workspace   `json:"active_workspace,omitempty"`
	Permissions     []identity.Permission `json:"permissions"`
}

type SessionInput struct {
	IDToken string `json:"id_token"`
}

type SelectTenantInput struct {
	TenantID string `json:"tenant_id"`
}
