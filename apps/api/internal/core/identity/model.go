package identity

import "time"

type Role string

const (
	RoleOwner   Role = "OWNER"
	RoleAdmin   Role = "ADMIN"
	RoleManager Role = "MANAGER"
	RoleStaff   Role = "STAFF"
)

type Permission string

const (
	PermissionTenantRead          Permission = "tenant.read"
	PermissionTenantManage        Permission = "tenant.manage"
	PermissionTeamManage          Permission = "team.manage"
	PermissionAuditRead           Permission = "audit.read"
	PermissionCatalogRead         Permission = "catalog.read"
	PermissionCatalogManage       Permission = "catalog.manage"
	PermissionPackageRead         Permission = "package.read"
	PermissionPackageManage       Permission = "package.manage"
	PermissionPublicCatalogRead   Permission = "public_catalog.read"
	PermissionPublicCatalogManage Permission = "public_catalog.manage"
	PermissionQuoteRequestRead    Permission = "quote_request.read"
	PermissionQuoteRequestManage  Permission = "quote_request.manage"
	PermissionInventoryRead       Permission = "inventory.read"
	PermissionInventoryManage     Permission = "inventory.manage"
	PermissionCustomerRead        Permission = "customer.read"
	PermissionCustomerManage      Permission = "customer.manage"
	PermissionQuoteRead           Permission = "quote.read"
	PermissionQuoteManage         Permission = "quote.manage"
	PermissionBillingRead         Permission = "billing.read"
	PermissionBillingManage       Permission = "billing.manage"
	PermissionPaymentRead         Permission = "payment.read"
	PermissionPaymentManage       Permission = "payment.manage"
	PermissionFiscalRead          Permission = "fiscal.read"
	PermissionFiscalManage        Permission = "fiscal.manage"
	PermissionReservationRead     Permission = "reservation.read"
	PermissionReservationManage   Permission = "reservation.manage"
	PermissionWarehouseOperate    Permission = "warehouse.operate"
	PermissionOperationsRead      Permission = "operations.read"
)

var allPermissions = []Permission{
	PermissionTenantRead,
	PermissionTenantManage,
	PermissionTeamManage,
	PermissionAuditRead,
	PermissionCatalogRead,
	PermissionCatalogManage,
	PermissionPackageRead,
	PermissionPackageManage,
	PermissionPublicCatalogRead,
	PermissionPublicCatalogManage,
	PermissionQuoteRequestRead,
	PermissionQuoteRequestManage,
	PermissionInventoryRead,
	PermissionInventoryManage,
	PermissionCustomerRead,
	PermissionCustomerManage,
	PermissionQuoteRead,
	PermissionQuoteManage,
	PermissionBillingRead,
	PermissionBillingManage,
	PermissionPaymentRead,
	PermissionPaymentManage,
	PermissionFiscalRead,
	PermissionFiscalManage,
	PermissionReservationRead,
	PermissionReservationManage,
	PermissionWarehouseOperate,
	PermissionOperationsRead,
}

func PermissionsForRole(role Role) []Permission {
	switch role {
	case RoleOwner, RoleAdmin:
		return append([]Permission(nil), allPermissions...)
	case RoleManager:
		return []Permission{
			PermissionTenantRead,
			PermissionCatalogRead,
			PermissionPackageRead,
			PermissionPublicCatalogRead,
			PermissionQuoteRequestRead,
			PermissionQuoteRequestManage,
			PermissionInventoryRead,
			PermissionInventoryManage,
			PermissionCustomerRead,
			PermissionCustomerManage,
			PermissionQuoteRead,
			PermissionQuoteManage,
			PermissionBillingRead,
			PermissionBillingManage,
			PermissionPaymentRead,
			PermissionPaymentManage,
			PermissionFiscalRead,
			PermissionFiscalManage,
			PermissionReservationRead,
			PermissionReservationManage,
			PermissionWarehouseOperate,
			PermissionOperationsRead,
		}
	case RoleStaff:
		return []Permission{
			PermissionTenantRead,
			PermissionCatalogRead,
			PermissionPackageRead,
			PermissionPublicCatalogRead,
			PermissionQuoteRequestRead,
			PermissionInventoryRead,
			PermissionCustomerRead,
			PermissionCustomerManage,
			PermissionQuoteRead,
			PermissionBillingRead,
			PermissionPaymentRead,
			PermissionFiscalRead,
			PermissionReservationRead,
			PermissionWarehouseOperate,
			PermissionOperationsRead,
		}
	default:
		return nil
	}
}

func HasPermission(role Role, permission Permission) bool {
	for _, candidate := range PermissionsForRole(role) {
		if candidate == permission {
			return true
		}
	}
	return false
}

// CanManageMember centralizes hierarchy rules in addition to endpoint permissions.
// Owners can manage any membership; administrators can only manage managers and staff.
func CanManageMember(actorRole, targetRole Role) bool {
	switch actorRole {
	case RoleOwner:
		return true
	case RoleAdmin:
		return targetRole == RoleManager || targetRole == RoleStaff
	default:
		return false
	}
}

func ValidRole(value string) bool {
	switch Role(value) {
	case RoleOwner, RoleAdmin, RoleManager, RoleStaff:
		return true
	default:
		return false
	}
}

type User struct {
	ID            string     `json:"id"`
	IdentityUID   string     `json:"identity_uid"`
	Email         string     `json:"email"`
	DisplayName   string     `json:"display_name"`
	AvatarURL     *string    `json:"avatar_url,omitempty"`
	EmailVerified bool       `json:"email_verified"`
	Status        string     `json:"status"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type IdentityProfile struct {
	UID           string
	Email         string
	DisplayName   string
	AvatarURL     string
	EmailVerified bool
}

type Workspace struct {
	TenantID         string  `json:"tenant_id"`
	Name             string  `json:"name"`
	Slug             string  `json:"slug"`
	LogoURL          *string `json:"logo_url,omitempty"`
	CountryCode      string  `json:"country_code"`
	Timezone         string  `json:"timezone"`
	Currency         string  `json:"currency"`
	TenantStatus     string  `json:"tenant_status"`
	Role             Role    `json:"role"`
	MembershipStatus string  `json:"membership_status"`
}

type Membership struct {
	TenantID string
	UserID   string
	Role     Role
	Status   string
}

type TeamMember struct {
	UserID        string     `json:"user_id"`
	Email         string     `json:"email"`
	DisplayName   string     `json:"display_name"`
	AvatarURL     *string    `json:"avatar_url,omitempty"`
	Role          Role       `json:"role"`
	Status        string     `json:"status"`
	EmailVerified bool       `json:"email_verified"`
	JoinedAt      *time.Time `json:"joined_at,omitempty"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type Invitation struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	TenantName  string     `json:"tenant_name"`
	Email       string     `json:"email"`
	Role        Role       `json:"role"`
	Status      string     `json:"status"`
	ExpiresAt   time.Time  `json:"expires_at"`
	InvitedByID string     `json:"invited_by_id"`
	InvitedBy   string     `json:"invited_by"`
	AcceptedBy  *string    `json:"accepted_by,omitempty"`
	AcceptedAt  *time.Time `json:"accepted_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	AcceptURL   string     `json:"accept_url,omitempty"`
}

type CreateOrganizationInput struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	LegalName   string `json:"legal_name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	CountryCode string `json:"country_code"`
	Timezone    string `json:"timezone"`
	Currency    string `json:"currency"`
	Address     string `json:"address"`
}

type UpdateOrganizationInput = CreateOrganizationInput

type CreateInvitationInput struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type UpdateMemberInput struct {
	Role   *string `json:"role"`
	Status *string `json:"status"`
}
