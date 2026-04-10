package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/tomfocker/lanmapper/internal/api"
	"github.com/tomfocker/lanmapper/internal/config"
	"github.com/tomfocker/lanmapper/internal/data"
	"github.com/tomfocker/lanmapper/internal/logger"
	"github.com/tomfocker/lanmapper/internal/report"
	"github.com/tomfocker/lanmapper/internal/scanner"
	"github.com/tomfocker/lanmapper/internal/topology"
)

func main() {
	log := logger.L()

	cfg, err := config.Load()
	if err != nil {
		log.Error("load config", "err", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		log.Error("create data dir", "err", err)
		os.Exit(1)
	}

	store, err := data.New(filepath.Join(cfg.DataDir, "lanmapper.db"))
	if err != nil {
		log.Error("open store", "err", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	runners := []scanner.Runner{
		scanner.NewARPNDRunner(log),
		scanner.NewICMPRunner(log),
		scanner.NewSNMPRunner(log, cfg.SNMPCommunities),
		scanner.NewServiceRunner(log),
	}
	mgr := scanner.NewManager(runners...)
	go mgr.Start(ctx)

	builder := topology.NewBuilder(store)
	gen := report.NewGenerator(store, cfg.DataDir)

	if err := api.Start(cfg, store, mgr, builder, gen); err != nil {
		log.Error("start api", "err", err)
		os.Exit(1)
	}
}
