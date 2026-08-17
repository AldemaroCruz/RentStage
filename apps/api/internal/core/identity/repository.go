package identity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUserNotFound               = errors.New("identity user not found")
	ErrMembershipNotFound         = errors.New("membership not found")
	ErrInvitationNotFound         = errors.New("invitation not found")
	ErrSlugConflict               = errors.New("organization slug already exists")
	ErrLastOwner                  = errors.New("the last active owner cannot be changed")
	ErrMembershipAlreadyExists    = errors.New("a membership already exists for this email")
	ErrPendingInvitation          = errors.New("a pending invitation already exists for this email")
	ErrMembershipManagementDenied = errors.New("the actor cannot manage this membership")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) SyncUser(ctx context.Context, profile IdentityProfile) (User, error) {
	email := strings.ToLower(strings.TrimSpace(profile.Email))
	displayName := strings.TrimSpace(profile.DisplayName)
	if displayName == "" {
		displayName = email
	}
	var avatar any
	if strings.TrimSpace(profile.AvatarURL) != "" {
		avatar = strings.TrimSpace(profile.AvatarURL)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return User{}, fmt.Errorf("begin sync user: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var userID string
	err = tx.QueryRow(ctx, `
		SELECT id
		FROM users
		WHERE identity_uid = $1 OR LOWER(email) = LOWER($2)
		ORDER BY (identity_uid = $1) DESC
		LIMIT 1
		FOR UPDATE
	`, profile.UID, email).Scan(&userID)
	if err == pgx.ErrNoRows {
		err = tx.QueryRow(ctx, `
			INSERT INTO users (
				identity_uid, email, display_name, avatar_url, email_verified,
				status, last_login_at
			) VALUES ($1, $2, $3, $4, $5, 'ACTIVE', NOW())
			RETURNING id
		`, profile.UID, email, displayName, avatar, profile.EmailVerified).Scan(&userID)
	} else if err == nil {
		_, err = tx.Exec(ctx, `
			UPDATE users
			SET identity_uid = $2,
			    email = $3,
			    display_name = $4,
			    avatar_url = $5,
			    email_verified = $6,
			    last_login_at = NOW()
			WHERE id = $1
		`, userID, profile.UID, email, displayName, avatar, profile.EmailVerified)
	}
	if err != nil {
		return User{}, fmt.Errorf("upsert identity user: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO user_preferences (user_id)
		VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING
	`, userID); err != nil {
		return User{}, fmt.Errorf("ensure user preferences: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return User{}, fmt.Errorf("commit sync user: %w", err)
	}
	return r.GetUser(ctx, userID)
}

func (r *Repository) GetUser(ctx context.Context, userID string) (User, error) {
	var item User
	err := r.pool.QueryRow(ctx, `
		SELECT id, COALESCE(identity_uid, ''), email, display_name, avatar_url,
		       email_verified, status, last_login_at, created_at, updated_at
		FROM users
		WHERE id = $1
	`, userID).Scan(
		&item.ID, &item.IdentityUID, &item.Email, &item.DisplayName, &item.AvatarURL,
		&item.EmailVerified, &item.Status, &item.LastLoginAt, &item.CreatedAt, &item.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("get identity user: %w", err)
	}
	return item, nil
}

func (r *Repository) GetUserByIdentityUID(ctx context.Context, uid string) (User, error) {
	var item User
	err := r.pool.QueryRow(ctx, `
		SELECT id, COALESCE(identity_uid, ''), email, display_name, avatar_url,
		       email_verified, status, last_login_at, created_at, updated_at
		FROM users
		WHERE identity_uid = $1
	`, uid).Scan(
		&item.ID, &item.IdentityUID, &item.Email, &item.DisplayName, &item.AvatarURL,
		&item.EmailVerified, &item.Status, &item.LastLoginAt, &item.CreatedAt, &item.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("get identity user by uid: %w", err)
	}
	return item, nil
}

func (r *Repository) ListWorkspaces(ctx context.Context, userID string) ([]Workspace, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT t.id, t.name, t.slug, t.logo_url, t.country_code, t.timezone,
		       t.currency, t.status, tm.role, tm.status
		FROM tenant_memberships tm
		JOIN tenants t ON t.id = tm.tenant_id
		WHERE tm.user_id = $1 AND tm.status = 'ACTIVE' AND t.status = 'ACTIVE'
		ORDER BY t.name
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	defer rows.Close()

	items := make([]Workspace, 0)
	for rows.Next() {
		var item Workspace
		if err := rows.Scan(
			&item.TenantID, &item.Name, &item.Slug, &item.LogoURL, &item.CountryCode,
			&item.Timezone, &item.Currency, &item.TenantStatus, &item.Role, &item.MembershipStatus,
		); err != nil {
			return nil, fmt.Errorf("scan workspace: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) GetMembership(ctx context.Context, userID, tenantID string) (Membership, error) {
	var item Membership
	err := r.pool.QueryRow(ctx, `
		SELECT tenant_id, user_id, role, status
		FROM tenant_memberships
		WHERE tenant_id = $1 AND user_id = $2
	`, tenantID, userID).Scan(&item.TenantID, &item.UserID, &item.Role, &item.Status)
	if err == pgx.ErrNoRows {
		return Membership{}, ErrMembershipNotFound
	}
	if err != nil {
		return Membership{}, fmt.Errorf("get membership: %w", err)
	}
	return item, nil
}

func (r *Repository) SetActiveTenant(ctx context.Context, userID, tenantID string) error {
	if _, err := r.GetMembership(ctx, userID, tenantID); err != nil {
		return err
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_preferences (user_id, active_tenant_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET
			active_tenant_id = EXCLUDED.active_tenant_id,
			updated_at = NOW()
	`, userID, tenantID)
	if err != nil {
		return fmt.Errorf("set active tenant: %w", err)
	}
	return nil
}

func (r *Repository) PreferredTenant(ctx context.Context, userID string) (string, error) {
	var tenantID *string
	err := r.pool.QueryRow(ctx, `
		SELECT up.active_tenant_id
		FROM user_preferences up
		JOIN tenant_memberships tm
		  ON tm.user_id = up.user_id
		 AND tm.tenant_id = up.active_tenant_id
		 AND tm.status = 'ACTIVE'
		JOIN tenants t ON t.id = up.active_tenant_id AND t.status = 'ACTIVE'
		WHERE up.user_id = $1
	`, userID).Scan(&tenantID)
	if err == pgx.ErrNoRows || tenantID == nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("load preferred tenant: %w", err)
	}
	return *tenantID, nil
}

func (r *Repository) CreateOrganization(ctx context.Context, userID string, input CreateOrganizationInput) (Workspace, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Workspace{}, fmt.Errorf("begin create organization: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var workspace Workspace
	err = tx.QueryRow(ctx, `
		INSERT INTO tenants (
			name, slug, legal_name, email, phone, country_code, timezone,
			currency, address
		) VALUES (
			$1, $2, NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''),
			$6, $7, $8, NULLIF($9, '')
		)
		RETURNING id, name, slug, logo_url, country_code, timezone, currency, status
	`, input.Name, input.Slug, input.LegalName, input.Email, input.Phone,
		input.CountryCode, input.Timezone, input.Currency, input.Address,
	).Scan(
		&workspace.TenantID, &workspace.Name, &workspace.Slug, &workspace.LogoURL,
		&workspace.CountryCode, &workspace.Timezone, &workspace.Currency, &workspace.TenantStatus,
	)
	if err != nil {
		if strings.Contains(err.Error(), "tenants_slug_key") {
			return Workspace{}, ErrSlugConflict
		}
		return Workspace{}, fmt.Errorf("insert organization: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO tenant_memberships (
			tenant_id, user_id, role, status, joined_at
		) VALUES ($1, $2, 'OWNER', 'ACTIVE', NOW())
	`, workspace.TenantID, userID); err != nil {
		return Workspace{}, fmt.Errorf("insert owner membership: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO user_preferences (user_id, active_tenant_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET active_tenant_id = $2, updated_at = NOW()
	`, userID, workspace.TenantID); err != nil {
		return Workspace{}, fmt.Errorf("set created organization active: %w", err)
	}
	workspace.Role = RoleOwner
	workspace.MembershipStatus = "ACTIVE"

	if err := tx.Commit(ctx); err != nil {
		return Workspace{}, fmt.Errorf("commit create organization: %w", err)
	}
	return workspace, nil
}

func (r *Repository) UpdateOrganization(ctx context.Context, tenantID string, input UpdateOrganizationInput) (Workspace, error) {
	var item Workspace
	err := r.pool.QueryRow(ctx, `
		UPDATE tenants
		SET name = $2,
		    slug = $3,
		    legal_name = NULLIF($4, ''),
		    email = NULLIF($5, ''),
		    phone = NULLIF($6, ''),
		    country_code = $7,
		    timezone = $8,
		    currency = $9,
		    address = NULLIF($10, '')
		WHERE id = $1
		RETURNING id, name, slug, logo_url, country_code, timezone, currency, status
	`, tenantID, input.Name, input.Slug, input.LegalName, input.Email, input.Phone,
		input.CountryCode, input.Timezone, input.Currency, input.Address,
	).Scan(
		&item.TenantID, &item.Name, &item.Slug, &item.LogoURL, &item.CountryCode,
		&item.Timezone, &item.Currency, &item.TenantStatus,
	)
	if err == pgx.ErrNoRows {
		return Workspace{}, ErrMembershipNotFound
	}
	if err != nil {
		if strings.Contains(err.Error(), "tenants_slug_key") {
			return Workspace{}, ErrSlugConflict
		}
		return Workspace{}, fmt.Errorf("update organization: %w", err)
	}
	return item, nil
}

func (r *Repository) ListTeam(ctx context.Context, tenantID string) ([]TeamMember, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT u.id, u.email, u.display_name, u.avatar_url, tm.role, tm.status,
		       u.email_verified, tm.joined_at, u.last_login_at, tm.created_at
		FROM tenant_memberships tm
		JOIN users u ON u.id = tm.user_id
		WHERE tm.tenant_id = $1 AND tm.status <> 'REMOVED'
		ORDER BY
		  CASE tm.role WHEN 'OWNER' THEN 1 WHEN 'ADMIN' THEN 2 WHEN 'MANAGER' THEN 3 ELSE 4 END,
		  u.display_name
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list team members: %w", err)
	}
	defer rows.Close()
	items := make([]TeamMember, 0)
	for rows.Next() {
		var item TeamMember
		if err := rows.Scan(
			&item.UserID, &item.Email, &item.DisplayName, &item.AvatarURL, &item.Role,
			&item.Status, &item.EmailVerified, &item.JoinedAt, &item.LastLoginAt, &item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan team member: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) EnsureInvitationAllowed(ctx context.Context, tenantID, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))

	// Persist logical expirations before relying on the partial unique index.
	if _, err := r.pool.Exec(ctx, `
		UPDATE tenant_invitations
		SET status = 'EXPIRED'
		WHERE tenant_id = $1
		  AND LOWER(email) = LOWER($2)
		  AND status = 'PENDING'
		  AND expires_at <= NOW()
	`, tenantID, email); err != nil {
		return fmt.Errorf("expire previous invitations: %w", err)
	}

	var memberExists bool
	var invitationExists bool
	if err := r.pool.QueryRow(ctx, `
		SELECT
		  EXISTS (
		    SELECT 1
		    FROM users u
		    JOIN tenant_memberships tm ON tm.user_id = u.id
		    WHERE tm.tenant_id = $1
		      AND LOWER(u.email) = LOWER($2)
		      AND tm.status <> 'REMOVED'
		  ),
		  EXISTS (
		    SELECT 1
		    FROM tenant_invitations i
		    WHERE i.tenant_id = $1
		      AND LOWER(i.email) = LOWER($2)
		      AND i.status = 'PENDING'
		      AND i.expires_at > NOW()
		  )
	`, tenantID, email).Scan(&memberExists, &invitationExists); err != nil {
		return fmt.Errorf("check invitation eligibility: %w", err)
	}
	if memberExists {
		return ErrMembershipAlreadyExists
	}
	if invitationExists {
		return ErrPendingInvitation
	}
	return nil
}

func InvitationHash(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

func (r *Repository) CreateInvitation(ctx context.Context, tenantID, actorUserID, email string, role Role, tokenHash string, expiresAt time.Time) (Invitation, error) {
	var item Invitation
	err := r.pool.QueryRow(ctx, `
		INSERT INTO tenant_invitations (
			tenant_id, email, role, token_hash, expires_at, invited_by
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, tenant_id, email, role, status, expires_at, invited_by,
		          accepted_by, accepted_at, created_at, updated_at
	`, tenantID, strings.ToLower(email), role, tokenHash, expiresAt, actorUserID).Scan(
		&item.ID, &item.TenantID, &item.Email, &item.Role, &item.Status, &item.ExpiresAt,
		&item.InvitedByID, &item.AcceptedBy, &item.AcceptedAt, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "idx_invitations_tenant_email_pending") {
			return Invitation{}, ErrPendingInvitation
		}
		return Invitation{}, fmt.Errorf("create invitation: %w", err)
	}
	return item, nil
}

func (r *Repository) ListInvitations(ctx context.Context, tenantID string) ([]Invitation, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT i.id, i.tenant_id, t.name, i.email, i.role,
		       CASE WHEN i.status = 'PENDING' AND i.expires_at <= NOW() THEN 'EXPIRED' ELSE i.status END,
		       i.expires_at, i.invited_by, u.display_name, i.accepted_by,
		       i.accepted_at, i.created_at, i.updated_at
		FROM tenant_invitations i
		JOIN tenants t ON t.id = i.tenant_id
		JOIN users u ON u.id = i.invited_by
		WHERE i.tenant_id = $1
		ORDER BY i.created_at DESC
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list invitations: %w", err)
	}
	defer rows.Close()
	items := make([]Invitation, 0)
	for rows.Next() {
		var item Invitation
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.TenantName, &item.Email, &item.Role, &item.Status,
			&item.ExpiresAt, &item.InvitedByID, &item.InvitedBy, &item.AcceptedBy,
			&item.AcceptedAt, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan invitation: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) GetInvitationByToken(ctx context.Context, token string) (Invitation, error) {
	var item Invitation
	err := r.pool.QueryRow(ctx, `
		SELECT i.id, i.tenant_id, t.name, i.email, i.role,
		       CASE WHEN i.status = 'PENDING' AND i.expires_at <= NOW() THEN 'EXPIRED' ELSE i.status END,
		       i.expires_at, i.invited_by, u.display_name, i.accepted_by,
		       i.accepted_at, i.created_at, i.updated_at
		FROM tenant_invitations i
		JOIN tenants t ON t.id = i.tenant_id
		JOIN users u ON u.id = i.invited_by
		WHERE i.token_hash = $1
	`, InvitationHash(token)).Scan(
		&item.ID, &item.TenantID, &item.TenantName, &item.Email, &item.Role, &item.Status,
		&item.ExpiresAt, &item.InvitedByID, &item.InvitedBy, &item.AcceptedBy,
		&item.AcceptedAt, &item.CreatedAt, &item.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return Invitation{}, ErrInvitationNotFound
	}
	if err != nil {
		return Invitation{}, fmt.Errorf("get invitation: %w", err)
	}
	return item, nil
}

func (r *Repository) AcceptInvitation(ctx context.Context, token, userID, userEmail string) (Workspace, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Workspace{}, fmt.Errorf("begin accept invitation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var invitation Invitation
	var lockedWorkspace Workspace
	err = tx.QueryRow(ctx, `
		SELECT i.id, i.tenant_id, t.name, t.slug, t.logo_url, t.country_code,
		       t.timezone, t.currency, t.status, i.email, i.role, i.status, i.expires_at
		FROM tenant_invitations i
		JOIN tenants t ON t.id = i.tenant_id
		WHERE i.token_hash = $1
		FOR UPDATE OF i
	`, InvitationHash(token)).Scan(
		&invitation.ID, &lockedWorkspace.TenantID, &lockedWorkspace.Name,
		&lockedWorkspace.Slug, &lockedWorkspace.LogoURL, &lockedWorkspace.CountryCode,
		&lockedWorkspace.Timezone, &lockedWorkspace.Currency, &lockedWorkspace.TenantStatus,
		&invitation.Email, &invitation.Role, &invitation.Status, &invitation.ExpiresAt,
	)
	if err == pgx.ErrNoRows {
		return Workspace{}, ErrInvitationNotFound
	}
	if err != nil {
		return Workspace{}, fmt.Errorf("lock invitation: %w", err)
	}
	invitation.TenantID = lockedWorkspace.TenantID
	invitation.TenantName = lockedWorkspace.Name
	if invitation.Status != "PENDING" || !invitation.ExpiresAt.After(time.Now()) {
		return Workspace{}, ErrInvitationNotFound
	}
	if !strings.EqualFold(invitation.Email, userEmail) {
		return Workspace{}, errors.New("invitation email does not match the signed-in user")
	}

	var existingStatus string
	err = tx.QueryRow(ctx, `
		SELECT status
		FROM tenant_memberships
		WHERE tenant_id = $1 AND user_id = $2
		FOR UPDATE
	`, invitation.TenantID, userID).Scan(&existingStatus)
	if err == nil && existingStatus != "REMOVED" {
		return Workspace{}, ErrMembershipAlreadyExists
	}
	if err != nil && err != pgx.ErrNoRows {
		return Workspace{}, fmt.Errorf("check existing membership: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO tenant_memberships (
			tenant_id, user_id, role, status, invited_by, joined_at
		)
		SELECT tenant_id, $2, role, 'ACTIVE', invited_by, NOW()
		FROM tenant_invitations WHERE id = $1
		ON CONFLICT (tenant_id, user_id) DO UPDATE SET
			role = EXCLUDED.role,
			status = 'ACTIVE',
			invited_by = EXCLUDED.invited_by,
			joined_at = COALESCE(tenant_memberships.joined_at, NOW()),
			updated_at = NOW()
	`, invitation.ID, userID); err != nil {
		return Workspace{}, fmt.Errorf("create invited membership: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE tenant_invitations
		SET status = 'ACCEPTED', accepted_by = $2, accepted_at = NOW()
		WHERE id = $1
	`, invitation.ID, userID); err != nil {
		return Workspace{}, fmt.Errorf("mark invitation accepted: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO user_preferences (user_id, active_tenant_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET active_tenant_id = $2, updated_at = NOW()
	`, userID, invitation.TenantID); err != nil {
		return Workspace{}, fmt.Errorf("activate invited workspace: %w", err)
	}

	var workspace Workspace
	err = tx.QueryRow(ctx, `
		SELECT t.id, t.name, t.slug, t.logo_url, t.country_code, t.timezone,
		       t.currency, t.status, tm.role, tm.status
		FROM tenants t
		JOIN tenant_memberships tm ON tm.tenant_id = t.id AND tm.user_id = $2
		WHERE t.id = $1
	`, invitation.TenantID, userID).Scan(
		&workspace.TenantID, &workspace.Name, &workspace.Slug, &workspace.LogoURL,
		&workspace.CountryCode, &workspace.Timezone, &workspace.Currency,
		&workspace.TenantStatus, &workspace.Role, &workspace.MembershipStatus,
	)
	if err != nil {
		return Workspace{}, fmt.Errorf("load accepted workspace: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Workspace{}, fmt.Errorf("commit invitation acceptance: %w", err)
	}
	return workspace, nil
}

func (r *Repository) RevokeInvitation(ctx context.Context, tenantID, invitationID string) error {
	command, err := r.pool.Exec(ctx, `
		UPDATE tenant_invitations
		SET status = 'REVOKED'
		WHERE id = $1 AND tenant_id = $2 AND status = 'PENDING'
	`, invitationID, tenantID)
	if err != nil {
		return fmt.Errorf("revoke invitation: %w", err)
	}
	if command.RowsAffected() == 0 {
		return ErrInvitationNotFound
	}
	return nil
}

func (r *Repository) UpdateMember(ctx context.Context, tenantID, targetUserID string, role *Role, status *string) (TeamMember, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return TeamMember{}, fmt.Errorf("begin update membership: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var currentRole Role
	var currentStatus string
	err = tx.QueryRow(ctx, `
		SELECT role, status
		FROM tenant_memberships
		WHERE tenant_id = $1 AND user_id = $2
		FOR UPDATE
	`, tenantID, targetUserID).Scan(&currentRole, &currentStatus)
	if err == pgx.ErrNoRows {
		return TeamMember{}, ErrMembershipNotFound
	}
	if err != nil {
		return TeamMember{}, fmt.Errorf("lock membership: %w", err)
	}

	if currentRole == RoleOwner {
		if (role != nil && *role != RoleOwner) || (status != nil && *status != "ACTIVE") {
			var owners int
			if err := tx.QueryRow(ctx, `
				SELECT COUNT(*)
				FROM tenant_memberships
				WHERE tenant_id = $1 AND role = 'OWNER' AND status = 'ACTIVE'
			`, tenantID).Scan(&owners); err != nil {
				return TeamMember{}, fmt.Errorf("count owners: %w", err)
			}
			if owners <= 1 {
				return TeamMember{}, ErrLastOwner
			}
		}
	}

	newRole := currentRole
	if role != nil {
		newRole = *role
	}
	newStatus := currentStatus
	if status != nil {
		newStatus = *status
	}
	if _, err := tx.Exec(ctx, `
		UPDATE tenant_memberships
		SET role = $3, status = $4
		WHERE tenant_id = $1 AND user_id = $2
	`, tenantID, targetUserID, newRole, newStatus); err != nil {
		return TeamMember{}, fmt.Errorf("update membership: %w", err)
	}

	var item TeamMember
	err = tx.QueryRow(ctx, `
		SELECT u.id, u.email, u.display_name, u.avatar_url, tm.role, tm.status,
		       u.email_verified, tm.joined_at, u.last_login_at, tm.created_at
		FROM tenant_memberships tm
		JOIN users u ON u.id = tm.user_id
		WHERE tm.tenant_id = $1 AND tm.user_id = $2
	`, tenantID, targetUserID).Scan(
		&item.UserID, &item.Email, &item.DisplayName, &item.AvatarURL, &item.Role,
		&item.Status, &item.EmailVerified, &item.JoinedAt, &item.LastLoginAt, &item.CreatedAt,
	)
	if err != nil {
		return TeamMember{}, fmt.Errorf("load updated member: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return TeamMember{}, fmt.Errorf("commit update member: %w", err)
	}
	return item, nil
}
