package api

import (
	"context"

	"github.com/gofiber/fiber/v2"

	"github.com/tomfocker/lanmapper/internal/data"
	"github.com/tomfocker/lanmapper/internal/report"
	"github.com/tomfocker/lanmapper/internal/scanner"
	"github.com/tomfocker/lanmapper/internal/topology"
)

// RegisterRoutes binds REST endpoints.
func RegisterRoutes(r fiber.Router, store *data.Store, builder *topology.Builder, mgr *scanner.Manager, gen *report.Generator) {
	r.Get("/devices", func(c *fiber.Ctx) error {
		devices, err := store.ListDevices(c.Context())
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(devices)
	})

	r.Get("/links", func(c *fiber.Ctx) error {
		links, err := store.ListLinks(c.Context())
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(links)
	})

	r.Get("/topology", func(c *fiber.Ctx) error {
		devices, links, err := builder.Rebuild(context.Background())
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(fiber.Map{"devices": devices, "links": links})
	})

	r.Post("/scans", func(c *fiber.Ctx) error {
		if mgr == nil {
			return fiber.ErrServiceUnavailable
		}
		return c.JSON(fiber.Map{"status": "scheduled"})
	})

	r.Post("/reports", func(c *fiber.Ctx) error {
		if gen == nil {
			return fiber.ErrServiceUnavailable
		}
		path, err := gen.ExportJSON(context.Background())
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(fiber.Map{"path": path})
	})
}
