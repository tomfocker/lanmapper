package scanner

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/tomfocker/lanmapper/internal/logger"
)

// ScanRecorder persists scan metadata. Implemented by data.Store.
type ScanRecorder interface {
	InsertScan(ctx context.Context, scanID string, targets []string) error
	FinishScan(ctx context.Context, scanID string, status string) error
}

// Scheduler handles scan enqueueing and optional periodic execution.
type Scheduler struct {
	mgr      *Manager
	recorder ScanRecorder
	ticker   *time.Ticker
	log      Logger
	mu       sync.Mutex
}

func NewScheduler(mgr *Manager, recorder ScanRecorder) *Scheduler {
	return &Scheduler{
		mgr:      mgr,
		recorder: recorder,
		log:      logger.L(),
	}
}

// Trigger enqueues jobs for each target CIDR.
func (s *Scheduler) Trigger(ctx context.Context, targets []DetectedCIDR) (string, error) {
	if len(targets) == 0 {
		return "", errors.New("scheduler: no targets provided")
	}
	scanID := uuid.NewString()
	if s.recorder != nil {
		if err := s.recorder.InsertScan(ctx, scanID, targetsToStrings(targets)); err != nil {
			s.log.Error("record scan", "err", err)
		}
	}
	for _, target := range targets {
		s.mgr.Enqueue(Job{
			CIDR:      target.CIDR,
			Interface: target.Interface,
			ScanID:    scanID,
		})
	}
	return scanID, nil
}

// StartInterval triggers scans periodically if interval > 0.
func (s *Scheduler) StartInterval(ctx context.Context, targets []DetectedCIDR, interval time.Duration) {
	if interval <= 0 {
		return
	}
	s.mu.Lock()
	if s.ticker != nil {
		s.ticker.Stop()
	}
	s.ticker = time.NewTicker(interval)
	s.mu.Unlock()

	go func() {
		for {
			select {
			case <-ctx.Done():
				s.Stop()
				return
			case <-s.ticker.C:
				if _, err := s.Trigger(ctx, targets); err != nil {
					s.log.Error("interval scan failed", "err", err)
				}
			}
		}
	}()
}

// Stop halts periodic scanning.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ticker != nil {
		s.ticker.Stop()
		s.ticker = nil
	}
}

func targetsToStrings(targets []DetectedCIDR) []string {
	var out []string
	for _, t := range targets {
		out = append(out, t.CIDR.String())
	}
	return out
}

// MergeTargets combines detected and configured CIDRs.
func MergeTargets(auto []DetectedCIDR, configured []string) ([]DetectedCIDR, error) {
	targets := append([]DetectedCIDR{}, auto...)
	defaultIface := ""
	if len(auto) > 0 {
		defaultIface = auto[0].Interface
	}
	for _, cidrStr := range configured {
		_, ipnet, err := net.ParseCIDR(strings.TrimSpace(cidrStr))
		if err != nil {
			return nil, fmt.Errorf("parse configured cidr %s: %w", cidrStr, err)
		}
		ipnet = canonicalize(ipnet)
		targets = append(targets, DetectedCIDR{
			CIDR:      ipnet,
			Interface: defaultIface,
		})
	}
	return dedupeCIDRs(targets), nil
}
