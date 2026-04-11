package api

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/tomfocker/lanmapper/internal/data"
	"github.com/tomfocker/lanmapper/internal/report"
	"github.com/tomfocker/lanmapper/internal/scanner"
	"github.com/tomfocker/lanmapper/internal/topology"
)

type scanRequest struct {
	CIDR      []string `json:"cidr"`
	Interface string   `json:"interface"`
}

// RegisterRoutes binds REST endpoints.
func RegisterRoutes(r fiber.Router, store *data.Store, builder *topology.Builder, mgr *scanner.Manager, gen *report.Generator, sched *scanner.Scheduler, defaultTargets []scanner.DetectedCIDR) {
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
		if sched == nil {
			return fiber.ErrServiceUnavailable
		}
		var req scanRequest
		if len(c.Body()) > 0 {
			if err := c.BodyParser(&req); err != nil {
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			}
		}
		targets, err := requestTargets(req, defaultTargets)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		scanID, err := sched.Trigger(c.Context(), targets)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(fiber.Map{
			"scan_id": scanID,
			"targets": targetsToStrings(targets),
		})
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

func requestTargets(req scanRequest, defaults []scanner.DetectedCIDR) ([]scanner.DetectedCIDR, error) {
	if len(req.CIDR) == 0 {
		if len(defaults) == 0 {
			return nil, fmt.Errorf("no default targets configured")
		}
		return defaults, nil
	}
	iface := req.Interface
	if iface == "" && len(defaults) > 0 {
		iface = defaults[0].Interface
	}
	var out []scanner.DetectedCIDR
	for _, raw := range req.CIDR {
		_, ipnet, err := net.ParseCIDR(strings.TrimSpace(raw))
		if err != nil {
			return nil, fmt.Errorf("invalid cidr %s: %w", raw, err)
		}
		ipnet = canonicalizeForAPI(ipnet)
		out = append(out, scanner.DetectedCIDR{CIDR: ipnet, Interface: iface})
	}
	return out, nil
}

func canonicalizeForAPI(ipnet *net.IPNet) *net.IPNet {
	maskOnes, bits := ipnet.Mask.Size()
	if bits != 32 {
		return ipnet
	}
	if maskOnes == 32 {
		maskOnes = 24
	}
	mask := net.CIDRMask(maskOnes, 32)
	ip := ipnet.IP.Mask(mask)
	ipCopy := make(net.IP, len(ip))
	copy(ipCopy, ip)
	return &net.IPNet{IP: ipCopy, Mask: mask}
}

func targetsToStrings(targets []scanner.DetectedCIDR) []string {
	var out []string
	for _, t := range targets {
		out = append(out, t.CIDR.String())
	}
	return out
}
