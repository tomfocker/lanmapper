package scanner

import (
	"context"
	"time"

	"github.com/grandcat/zeroconf"
)

// ServiceRunner listens to mDNS/SSDP broadcasts (placeholder mDNS only).
type ServiceRunner struct {
	log Logger
}

func NewServiceRunner(log Logger) *ServiceRunner {
	return &ServiceRunner{log: log}
}

func (r *ServiceRunner) Name() string { return "services" }

func (r *ServiceRunner) Run(job Job) error {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return err
	}
	entries := make(chan *zeroconf.ServiceEntry)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go func() {
		for entry := range entries {
			r.log.Info("mdns service", "instance", entry.Instance, "addr", entry.AddrIPv4)
		}
	}()
	return resolver.Browse(ctx, "_services._dns-sd._udp", "local.", entries)
}
