package handlers

import (
	"github.com/doreviateam/dorevia-vault/internal/buildinfo"
	"github.com/gofiber/fiber/v2"
)

// Version retourne la version enrichie du service
func Version(c *fiber.Ctx) error {
	return c.JSON(buildinfo.VersionPayload{
		Version: buildinfo.Version,
		Commit:  buildinfo.Commit,
		BuiltAt: buildinfo.BuiltAt,
		Schema:  buildinfo.Schema,
	})
}

