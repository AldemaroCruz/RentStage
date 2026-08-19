package identity

import "testing"

func TestRolePermissions(t *testing.T) {
	tests := []struct {
		name       string
		role       Role
		permission Permission
		want       bool
	}{
		{"owner manages team", RoleOwner, PermissionTeamManage, true},
		{"admin reads audit", RoleAdmin, PermissionAuditRead, true},
		{"manager manages reservations", RoleManager, PermissionReservationManage, true},
		{"manager reads packages", RoleManager, PermissionPackageRead, true},
		{"manager reads public catalog", RoleManager, PermissionPublicCatalogRead, true},
		{"manager manages quote requests", RoleManager, PermissionQuoteRequestManage, true},
		{"manager cannot publish catalog", RoleManager, PermissionPublicCatalogManage, false},
		{"manager cannot manage packages", RoleManager, PermissionPackageManage, false},
		{"manager manages billing", RoleManager, PermissionBillingManage, true},
		{"manager manages payments", RoleManager, PermissionPaymentManage, true},
		{"manager manages DTE", RoleManager, PermissionFiscalManage, true},
		{"manager approves assistant proposals", RoleManager, PermissionAssistantManage, true},
		{"manager cannot manage team", RoleManager, PermissionTeamManage, false},
		{"staff operates warehouse", RoleStaff, PermissionWarehouseOperate, true},
		{"staff reads packages", RoleStaff, PermissionPackageRead, true},
		{"staff reads billing", RoleStaff, PermissionBillingRead, true},
		{"staff cannot manage billing", RoleStaff, PermissionBillingManage, false},
		{"staff reads payments", RoleStaff, PermissionPaymentRead, true},
		{"staff reads DTE", RoleStaff, PermissionFiscalRead, true},
		{"staff cannot manage DTE", RoleStaff, PermissionFiscalManage, false},
		{"staff reads assistant conversations", RoleStaff, PermissionAssistantRead, true},
		{"staff cannot approve assistant proposals", RoleStaff, PermissionAssistantManage, false},
		{"staff cannot manage payments", RoleStaff, PermissionPaymentManage, false},
		{"staff reads quote requests", RoleStaff, PermissionQuoteRequestRead, true},
		{"staff cannot manage quote requests", RoleStaff, PermissionQuoteRequestManage, false},
		{"staff cannot change catalog", RoleStaff, PermissionCatalogManage, false},
		{"unknown role has no access", Role("UNKNOWN"), PermissionTenantRead, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := HasPermission(test.role, test.permission); got != test.want {
				t.Fatalf("HasPermission(%s, %s) = %v, want %v", test.role, test.permission, got, test.want)
			}
		})
	}
}

func TestPermissionsForRoleReturnsCopy(t *testing.T) {
	first := PermissionsForRole(RoleOwner)
	if len(first) == 0 {
		t.Fatal("owner permissions must not be empty")
	}
	first[0] = Permission("mutated")
	second := PermissionsForRole(RoleOwner)
	if second[0] == Permission("mutated") {
		t.Fatal("PermissionsForRole exposed its shared backing array")
	}
}

func TestCanManageMemberHierarchy(t *testing.T) {
	tests := []struct {
		name   string
		actor  Role
		target Role
		want   bool
	}{
		{"owner manages owner", RoleOwner, RoleOwner, true},
		{"owner manages admin", RoleOwner, RoleAdmin, true},
		{"admin cannot manage owner", RoleAdmin, RoleOwner, false},
		{"admin cannot manage admin", RoleAdmin, RoleAdmin, false},
		{"admin manages manager", RoleAdmin, RoleManager, true},
		{"admin manages staff", RoleAdmin, RoleStaff, true},
		{"manager cannot manage staff", RoleManager, RoleStaff, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := CanManageMember(test.actor, test.target); got != test.want {
				t.Fatalf("CanManageMember(%s, %s) = %v, want %v", test.actor, test.target, got, test.want)
			}
		})
	}
}
