package api

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	fiberlog "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	websocket "github.com/gofiber/websocket/v2"

	"github.com/tomfocker/lanmapper/internal/config"
	"github.com/tomfocker/lanmapper/internal/data"
	"github.com/tomfocker/lanmapper/internal/logger"
	"github.com/tomfocker/lanmapper/internal/report"
	"github.com/tomfocker/lanmapper/internal/scanner"
	"github.com/tomfocker/lanmapper/internal/topology"
	"github.com/tomfocker/lanmapper/ui"
)

// Start launches HTTP + WS API server.
func Start(cfg *config.Config, store *data.Store, mgr *scanner.Manager, builder *topology.Builder, gen *report.Generator, sched *scanner.Scheduler, defaultTargets []scanner.DetectedCIDR) error {
	app := fiber.New()
	app.Use(recover.New())
	app.Use(fiberlog.New())

	log := logger.L()
	app.Get("/health", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) })

	apiGroup := app.Group("/api/v1", adminAuth(cfg.AdminToken))
	RegisterRoutes(apiGroup, store, builder, mgr, gen, sched, defaultTargets)

	app.Get("/ws", websocket.New(func(conn *websocket.Conn) {
		defer conn.Close()
		conn.WriteMessage(websocket.TextMessage, []byte("lanmapper ws connected"))
	}))
	app.Use("/", filesystem.New(filesystem.Config{
		Root:       ui.StaticFS(),
		Browse:     false,
		PathPrefix: "",
	}))

	addr := fmt.Sprintf(":%d", cfg.HTTPPort)
	log.Info("http server listening", "addr", addr)
	return app.Listen(addr)
}
