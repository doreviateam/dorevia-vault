package unit

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAuthMiddleware_MissingHeader teste le middleware avec header manquant
func TestAuthMiddleware_MissingHeader(t *testing.T) {
	cfg := auth.AuthConfig{
		JWTEnabled:    true,
		APIKeyEnabled: true,
		Logger:        zerolog.Nop(),
	}

	authService := auth.NewAuthService(cfg)
	app := fiber.New()
	app.Use(auth.AuthMiddleware(authService, zerolog.Nop()))

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

// TestAuthMiddleware_InvalidToken teste le middleware avec token invalide
func TestAuthMiddleware_InvalidToken(t *testing.T) {
	cfg := auth.AuthConfig{
		JWTEnabled:    true,
		APIKeyEnabled: true,
		Logger:        zerolog.Nop(),
	}

	authService := auth.NewAuthService(cfg)
	app := fiber.New()
	app.Use(auth.AuthMiddleware(authService, zerolog.Nop()))

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid.token")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

// TestAuthMiddleware_ValidJWT teste le middleware avec JWT valide
func TestAuthMiddleware_ValidJWT(t *testing.T) {
	// Créer une paire de clés RSA
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	cfg := auth.AuthConfig{
		JWTPublicKey: &privateKey.PublicKey,
		JWTEnabled:   true,
		Logger:       zerolog.Nop(),
	}

	authService := auth.NewAuthService(cfg)
	app := fiber.New()
	app.Use(auth.AuthMiddleware(authService, zerolog.Nop()))

	app.Get("/test", func(c *fiber.Ctx) error {
		userInfo, err := auth.GetUserInfo(c)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}
		return c.JSON(fiber.Map{"user_id": userInfo.UserID, "role": userInfo.Role})
	})

	// Créer un JWT valide
	claims := jwt.MapClaims{
		"sub":   "user-123",
		"role":  "operator",
		"email": "user@example.com",
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(1 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(privateKey)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

// TestRequirePermission_Allowed teste RequirePermission avec permission accordée
func TestRequirePermission_Allowed(t *testing.T) {
	rbacService := auth.NewRBACService()
	app := fiber.New()

	// Simuler un utilisateur authentifié avec rôle admin
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user", &auth.UserInfo{
			UserID: "user-123",
			Role:   "admin",
		})
		return c.Next()
	})

	app.Use(auth.RequirePermission(rbacService, auth.PermissionReadDocuments, zerolog.Nop()))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

// TestRequirePermission_Denied teste RequirePermission avec permission refusée
func TestRequirePermission_Denied(t *testing.T) {
	rbacService := auth.NewRBACService()
	app := fiber.New()

	// Simuler un utilisateur authentifié avec rôle viewer (pas de permission write)
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user", &auth.UserInfo{
			UserID: "user-123",
			Role:   "viewer",
		})
		return c.Next()
	})

	app.Use(auth.RequirePermission(rbacService, auth.PermissionWriteDocuments, zerolog.Nop()))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

// TestRequireRole_Allowed teste RequireRole avec rôle autorisé
func TestRequireRole_Allowed(t *testing.T) {
	rbacService := auth.NewRBACService()
	app := fiber.New()

	// Simuler un utilisateur authentifié avec rôle auditor
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user", &auth.UserInfo{
			UserID: "user-123",
			Role:   "auditor",
		})
		return c.Next()
	})

	app.Use(auth.RequireRole(rbacService, auth.RoleAuditor, zerolog.Nop()))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

// TestRequireRole_Denied teste RequireRole avec rôle non autorisé
func TestRequireRole_Denied(t *testing.T) {
	rbacService := auth.NewRBACService()
	app := fiber.New()

	// Simuler un utilisateur authentifié avec rôle viewer (pas admin)
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user", &auth.UserInfo{
			UserID: "user-123",
			Role:   "viewer",
		})
		return c.Next()
	})

	app.Use(auth.RequireRole(rbacService, auth.RoleAdmin, zerolog.Nop()))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

// TestGetUserInfo teste GetUserInfo
func TestGetUserInfo(t *testing.T) {
	app := fiber.New()

	app.Get("/test", func(c *fiber.Ctx) error {
		userInfo, err := auth.GetUserInfo(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
		}
		return c.JSON(fiber.Map{"user_id": userInfo.UserID, "role": userInfo.Role})
	})

	// Test sans utilisateur
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)

	// Test avec utilisateur
	app.Get("/test2", func(c *fiber.Ctx) error {
		c.Locals("user", &auth.UserInfo{
			UserID: "user-123",
			Role:   "operator",
		})
		userInfo, err := auth.GetUserInfo(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
		}
		return c.JSON(fiber.Map{"user_id": userInfo.UserID, "role": userInfo.Role})
	})

	req2 := httptest.NewRequest("GET", "/test2", nil)
	resp2, err := app.Test(req2)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp2.StatusCode)
}

