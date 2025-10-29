package auth

import "errors"

// RBAC роли и разрешения
const (
	RoleAdmin     = "admin"
	RoleUser      = "user"
	RoleModerator = "moderator"
)

// Permissions список разрешений
var Permissions = map[string][]string{
	RoleAdmin: {
		"users:read",
		"users:write",
		"users:delete",
		"content:read",
		"content:write",
		"content:delete",
		"system:admin",
	},
	RoleModerator: {
		"users:read",
		"content:read",
		"content:write",
		"content:delete",
	},
	RoleUser: {
		"users:read:self",
		"users:write:self",
		"content:read",
		"content:write:self",
		"content:delete:self",
	},
}

// HasPermission проверяет есть ли у роли указанное разрешение
func HasPermission(role, permission string) bool {
	permissions, exists := Permissions[role]
	if !exists {
		return false
	}

	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// CanPerformAction проверяет может ли пользователь выполнить действие
func CanPerformAction(claims *Claims, permission string) bool {
	return HasPermission(claims.Role, permission)
}

// IsAdmin проверяет является ли пользователь администратором
func IsAdmin(claims *Claims) bool {
	return claims.Role == RoleAdmin
}

// IsModeratorOrHigher проверяет является ли пользователь модератором или выше
func IsModeratorOrHigher(claims *Claims) bool {
	return claims.Role == RoleModerator || claims.Role == RoleAdmin
}

// ValidateRole проверяет валидность роли
func ValidateRole(role string) error {
	switch role {
	case RoleAdmin, RoleUser, RoleModerator:
		return nil
	default:
		return errors.New("invalid role")
	}
}
