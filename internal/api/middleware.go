package api

import "github.com/gofiber/fiber/v2"

func adminAuth(token string) fiber.Handler {
	if token == "" {
		return func(c *fiber.Ctx) error { return c.Next() }
	}
	return func(c *fiber.Ctx) error {
		if c.Get("X-Admin-Token") != token {
			return fiber.ErrUnauthorized
		}
		return c.Next()
	}
}
