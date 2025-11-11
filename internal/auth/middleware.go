package auth

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

// AuthMiddleware crée un middleware d'authentification
func AuthMiddleware(authService *AuthService, log zerolog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Récupérer le header Authorization
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "authorization header missing",
			})
		}

		// Authentifier
		userInfo, err := authService.Authenticate(c.Context(), authHeader)
		if err != nil {
			log.Warn().Err(err).Msg("Authentication failed")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "authentication failed",
			})
		}

		// Stocker les informations utilisateur dans le contexte
		c.Locals("user", userInfo)

		return c.Next()
	}
}

// RequireRole crée un middleware qui requiert un rôle spécifique
func RequireRole(rbacService *RBACService, requiredRole Role, log zerolog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Récupérer les informations utilisateur depuis le contexte
		userInfo, ok := c.Locals("user").(*UserInfo)
		if !ok || userInfo == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "user not authenticated",
			})
		}

		// Vérifier le rôle
		userRole, err := ParseRole(userInfo.Role)
		if err != nil {
			log.Warn().Err(err).Str("role", userInfo.Role).Msg("Invalid user role")
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "invalid user role",
			})
		}

		if userRole != requiredRole {
			log.Warn().
				Str("user_role", string(userRole)).
				Str("required_role", string(requiredRole)).
				Msg("Role mismatch")
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "insufficient permissions",
			})
		}

		return c.Next()
	}
}

// RequirePermission crée un middleware qui requiert une permission spécifique
func RequirePermission(rbacService *RBACService, permission Permission, log zerolog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Récupérer les informations utilisateur depuis le contexte
		userInfo, ok := c.Locals("user").(*UserInfo)
		if !ok || userInfo == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "user not authenticated",
			})
		}

		// Vérifier la permission
		err := rbacService.RequirePermission(userInfo, permission)
		if err != nil {
			log.Warn().Err(err).
				Str("user_id", userInfo.UserID).
				Str("permission", string(permission)).
				Msg("Permission denied")
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "permission denied",
			})
		}

		return c.Next()
	}
}

// RequireEndpointPermission crée un middleware qui vérifie la permission basée sur l'endpoint
func RequireEndpointPermission(rbacService *RBACService, log zerolog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Récupérer les informations utilisateur depuis le contexte
		userInfo, ok := c.Locals("user").(*UserInfo)
		if !ok || userInfo == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "user not authenticated",
			})
		}

		// Obtenir la permission requise pour cet endpoint
		endpoint := c.Path()
		permission, err := GetRequiredPermission(endpoint)
		if err != nil {
			log.Warn().Err(err).Str("endpoint", endpoint).Msg("Failed to get required permission")
			// Par défaut, permettre l'accès si on ne peut pas déterminer la permission
			return c.Next()
		}

		// Vérifier la permission
		err = rbacService.RequirePermission(userInfo, permission)
		if err != nil {
			log.Warn().Err(err).
				Str("user_id", userInfo.UserID).
				Str("endpoint", endpoint).
				Str("permission", string(permission)).
				Msg("Permission denied for endpoint")
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "permission denied",
			})
		}

		return c.Next()
	}
}

// GetUserInfo récupère les informations utilisateur depuis le contexte Fiber
func GetUserInfo(c *fiber.Ctx) (*UserInfo, error) {
	userInfo, ok := c.Locals("user").(*UserInfo)
	if !ok || userInfo == nil {
		return nil, fmt.Errorf("user not authenticated")
	}
	return userInfo, nil
}

