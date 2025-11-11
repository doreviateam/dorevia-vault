package auth

import (
	"fmt"
	"strings"
)

// Role représente un rôle utilisateur
type Role string

const (
	RoleAdmin    Role = "admin"
	RoleAuditor  Role = "auditor"
	RoleOperator Role = "operator"
	RoleViewer   Role = "viewer"
)

// Permission représente une permission
type Permission string

const (
	PermissionReadDocuments    Permission = "documents:read"
	PermissionWriteDocuments   Permission = "documents:write"
	PermissionReadAudit        Permission = "audit:read"
	PermissionReadLedger       Permission = "ledger:read"
	PermissionVerifyDocuments  Permission = "documents:verify"
	PermissionReconcile        Permission = "reconcile:execute"
	PermissionManageUsers      Permission = "users:manage"
)

// RBACService gère les autorisations basées sur les rôles
type RBACService struct {
	rolePermissions map[Role][]Permission
}

// NewRBACService crée un nouveau service RBAC
func NewRBACService() *RBACService {
	// Définir les permissions par rôle
	rolePermissions := map[Role][]Permission{
		RoleAdmin: {
			PermissionReadDocuments,
			PermissionWriteDocuments,
			PermissionReadAudit,
			PermissionReadLedger,
			PermissionVerifyDocuments,
			PermissionReconcile,
			PermissionManageUsers,
		},
		RoleAuditor: {
			PermissionReadDocuments,
			PermissionReadAudit,
			PermissionReadLedger,
			PermissionVerifyDocuments,
		},
		RoleOperator: {
			PermissionReadDocuments,
			PermissionWriteDocuments,
			PermissionReadAudit,
		},
		RoleViewer: {
			PermissionReadDocuments,
		},
	}

	return &RBACService{
		rolePermissions: rolePermissions,
	}
}

// HasPermission vérifie si un rôle a une permission
func (r *RBACService) HasPermission(role Role, permission Permission) bool {
	permissions, ok := r.rolePermissions[role]
	if !ok {
		return false
	}

	for _, p := range permissions {
		if p == permission {
			return true
		}
	}

	return false
}

// RequirePermission vérifie si un utilisateur a une permission (lève une erreur si non)
func (r *RBACService) RequirePermission(userInfo *UserInfo, permission Permission) error {
	if userInfo == nil {
		return fmt.Errorf("user not authenticated")
	}

	role := Role(userInfo.Role)
	if role == "" {
		return fmt.Errorf("user role not specified")
	}

	if !r.HasPermission(role, permission) {
		return fmt.Errorf("permission denied: role %s does not have permission %s", role, permission)
	}

	return nil
}

// GetRolePermissions retourne toutes les permissions d'un rôle
func (r *RBACService) GetRolePermissions(role Role) []Permission {
	permissions, ok := r.rolePermissions[role]
	if !ok {
		return []Permission{}
	}
	return permissions
}

// IsValidRole vérifie si un rôle est valide
func IsValidRole(role string) bool {
	validRoles := []Role{RoleAdmin, RoleAuditor, RoleOperator, RoleViewer}
	for _, validRole := range validRoles {
		if Role(role) == validRole {
			return true
		}
	}
	return false
}

// ParseRole parse une chaîne en Role
func ParseRole(roleStr string) (Role, error) {
	roleStr = strings.ToLower(strings.TrimSpace(roleStr))
	if !IsValidRole(roleStr) {
		return "", fmt.Errorf("invalid role: %s", roleStr)
	}
	return Role(roleStr), nil
}

// EndpointPermission mappe les endpoints aux permissions requises
var EndpointPermission = map[string]Permission{
	"/api/v1/invoices":        PermissionWriteDocuments,
	"/api/v1/ledger/export":   PermissionReadLedger,
	"/audit/export":           PermissionReadAudit,
	"/api/v1/ledger/verify/:id": PermissionVerifyDocuments,
	"/documents":              PermissionReadDocuments,
	"/download/:id":           PermissionReadDocuments,
}

// GetRequiredPermission retourne la permission requise pour un endpoint
func GetRequiredPermission(endpoint string) (Permission, error) {
	// Normaliser l'endpoint (enlever les paramètres dynamiques)
	normalized := normalizeEndpoint(endpoint)

	permission, ok := EndpointPermission[normalized]
	if !ok {
		// Par défaut, nécessite la permission de lecture
		return PermissionReadDocuments, nil
	}

	return permission, nil
}

// normalizeEndpoint normalise un endpoint (remplace :id, :document_id, etc. par des placeholders)
func normalizeEndpoint(endpoint string) string {
	// Remplacer les paramètres dynamiques
	endpoint = strings.ReplaceAll(endpoint, ":id", ":id")
	endpoint = strings.ReplaceAll(endpoint, ":document_id", ":id")
	
	// Pour la correspondance exacte, on garde tel quel
	// Mais on peut aussi faire une correspondance par pattern
	return endpoint
}

