package unit

import (
	"testing"

	"github.com/doreviateam/dorevia-vault/internal/auth"
	"github.com/stretchr/testify/assert"
)

// TestNewRBACService teste la création d'un RBACService
func TestNewRBACService(t *testing.T) {
	rbac := auth.NewRBACService()
	assert.NotNil(t, rbac)
}

// TestRBACService_HasPermission teste HasPermission
func TestRBACService_HasPermission(t *testing.T) {
	rbac := auth.NewRBACService()

	// Admin a toutes les permissions
	assert.True(t, rbac.HasPermission(auth.RoleAdmin, auth.PermissionReadDocuments))
	assert.True(t, rbac.HasPermission(auth.RoleAdmin, auth.PermissionWriteDocuments))
	assert.True(t, rbac.HasPermission(auth.RoleAdmin, auth.PermissionReadAudit))
	assert.True(t, rbac.HasPermission(auth.RoleAdmin, auth.PermissionManageUsers))

	// Auditor a certaines permissions
	assert.True(t, rbac.HasPermission(auth.RoleAuditor, auth.PermissionReadAudit))
	assert.True(t, rbac.HasPermission(auth.RoleAuditor, auth.PermissionReadDocuments))
	assert.False(t, rbac.HasPermission(auth.RoleAuditor, auth.PermissionWriteDocuments))
	assert.False(t, rbac.HasPermission(auth.RoleAuditor, auth.PermissionManageUsers))

	// Operator a permissions limitées
	assert.True(t, rbac.HasPermission(auth.RoleOperator, auth.PermissionReadDocuments))
	assert.True(t, rbac.HasPermission(auth.RoleOperator, auth.PermissionWriteDocuments))
	assert.False(t, rbac.HasPermission(auth.RoleOperator, auth.PermissionReadLedger))

	// Viewer a seulement lecture
	assert.True(t, rbac.HasPermission(auth.RoleViewer, auth.PermissionReadDocuments))
	assert.False(t, rbac.HasPermission(auth.RoleViewer, auth.PermissionWriteDocuments))
}

// TestRBACService_RequirePermission teste RequirePermission
func TestRBACService_RequirePermission(t *testing.T) {
	rbac := auth.NewRBACService()

	// Test avec permission valide
	userInfo := &auth.UserInfo{
		UserID: "user-123",
		Role:   "admin",
	}

	err := rbac.RequirePermission(userInfo, auth.PermissionReadDocuments)
	assert.NoError(t, err)

	// Test avec permission invalide
	userInfo.Role = "viewer"
	err = rbac.RequirePermission(userInfo, auth.PermissionWriteDocuments)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}

// TestRBACService_RequirePermission_NoUser teste sans utilisateur
func TestRBACService_RequirePermission_NoUser(t *testing.T) {
	rbac := auth.NewRBACService()

	err := rbac.RequirePermission(nil, auth.PermissionReadDocuments)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not authenticated")
}

// TestRBACService_GetRolePermissions teste GetRolePermissions
func TestRBACService_GetRolePermissions(t *testing.T) {
	rbac := auth.NewRBACService()

	// Admin devrait avoir toutes les permissions
	adminPerms := rbac.GetRolePermissions(auth.RoleAdmin)
	assert.Greater(t, len(adminPerms), 5)

	// Viewer devrait avoir peu de permissions
	viewerPerms := rbac.GetRolePermissions(auth.RoleViewer)
	assert.Equal(t, 1, len(viewerPerms))
	assert.Contains(t, viewerPerms, auth.PermissionReadDocuments)
}

// TestIsValidRole teste IsValidRole
func TestIsValidRole(t *testing.T) {
	assert.True(t, auth.IsValidRole("admin"))
	assert.True(t, auth.IsValidRole("auditor"))
	assert.True(t, auth.IsValidRole("operator"))
	assert.True(t, auth.IsValidRole("viewer"))
	assert.False(t, auth.IsValidRole("invalid"))
	assert.False(t, auth.IsValidRole(""))
}

// TestParseRole teste ParseRole
func TestParseRole(t *testing.T) {
	role, err := auth.ParseRole("admin")
	assert.NoError(t, err)
	assert.Equal(t, auth.RoleAdmin, role)

	role, err = auth.ParseRole("AUDITOR")
	assert.NoError(t, err)
	assert.Equal(t, auth.RoleAuditor, role)

	_, err = auth.ParseRole("invalid")
	assert.Error(t, err)
}

// TestGetRequiredPermission teste GetRequiredPermission
func TestGetRequiredPermission(t *testing.T) {
	permission, err := auth.GetRequiredPermission("/api/v1/invoices")
	assert.NoError(t, err)
	assert.Equal(t, auth.PermissionWriteDocuments, permission)

	permission, err = auth.GetRequiredPermission("/audit/export")
	assert.NoError(t, err)
	assert.Equal(t, auth.PermissionReadAudit, permission)

	permission, err = auth.GetRequiredPermission("/api/v1/ledger/export")
	assert.NoError(t, err)
	assert.Equal(t, auth.PermissionReadLedger, permission)
}

