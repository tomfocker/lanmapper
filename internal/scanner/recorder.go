package scanner

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/tomfocker/lanmapper/internal/data"
	"github.com/tomfocker/lanmapper/internal/models"
	"github.com/tomfocker/lanmapper/internal/scanner/oui"
)

// DeviceObservation captures a single device discovery event.
type DeviceObservation struct {
	ID          string
	IPv4        string
	MAC         string
	Hostname    string
	Interface   string
	Source      string
	Vendor      string
	SysObjectID string
	Services    []string
	TypeHint    string
	Confidence  float64
	ObservedAt  time.Time
}

// LinkObservation captures an inferred connection between two devices.
type LinkObservation struct {
	ID         string
	ADevice    string
	AInterface string
	BDevice    string
	BInterface string
	Media      string
	Source     string
	Kind       string
	Confidence float64
	ObservedAt time.Time
}

// Recorder persists discovery observations.
type Recorder interface {
	RecordDevice(context.Context, DeviceObservation)
	RecordLink(context.Context, LinkObservation)
	SetGateway(string)
	Close()
}

type recorder struct {
	store    *data.Store
	log      Logger
	devCh    chan DeviceObservation
	linkCh   chan LinkObservation
	stopCh   chan struct{}
	wg       sync.WaitGroup
	cacheTTL time.Duration
	devCache map[string]time.Time
	mu       sync.Mutex
	lookup   *oui.Lookup
	gatewayID string
}

// NewRecorder builds a Recorder backed by data.Store.
func NewRecorder(store *data.Store, log Logger) Recorder {
	r := &recorder{
		store:    store,
		log:      log,
		devCh:    make(chan DeviceObservation, 256),
		linkCh:   make(chan LinkObservation, 256),
		stopCh:   make(chan struct{}),
		cacheTTL: 5 * time.Second,
		devCache: make(map[string]time.Time),
		lookup:   oui.NewLookup(),
	}
	r.start()
	return r
}

func (r *recorder) start() {
	r.wg.Add(2)
	go r.consumeDevices()
	go r.consumeLinks()
}

func (r *recorder) Close() {
	close(r.stopCh)
	r.wg.Wait()
}

func (r *recorder) RecordDevice(ctx context.Context, obs DeviceObservation) {
	select {
	case r.devCh <- obs:
	default:
		r.log.Warn("device observation backlog", "id", obs.ID)
	}
}

func (r *recorder) RecordLink(ctx context.Context, obs LinkObservation) {
	select {
	case r.linkCh <- obs:
	default:
		r.log.Warn("link observation backlog", "id", obs.ID)
	}
}

func (r *recorder) SetGateway(id string) {
	if id == "" {
		return
	}
	r.mu.Lock()
	r.gatewayID = strings.ToLower(strings.TrimSpace(id))
	r.mu.Unlock()
}

func (r *recorder) consumeDevices() {
	defer r.wg.Done()
	for {
		select {
		case <-r.stopCh:
			return
		case obs := <-r.devCh:
			r.handleDevice(obs)
		}
	}
}

func (r *recorder) consumeLinks() {
	defer r.wg.Done()
	for {
		select {
		case <-r.stopCh:
			return
		case obs := <-r.linkCh:
			r.handleLink(obs)
		}
	}
}

func (r *recorder) handleDevice(obs DeviceObservation) {
	id := strings.ToLower(strings.TrimSpace(obs.ID))
	if id == "" {
		id = strings.ToLower(strings.TrimSpace(obs.MAC))
	}
	if id == "" {
		id = strings.TrimSpace(obs.IPv4)
	}
	if id == "" {
		r.log.Warn("skip device observation without identifier", "source", obs.Source)
		return
	}
	if obs.ObservedAt.IsZero() {
		obs.ObservedAt = time.Now().UTC()
	}
	if r.isCached(id, obs.ObservedAt) {
		return
	}
	vendor := obs.Vendor
	typeHint := obs.TypeHint
	if vendor == "" || typeHint == "" {
		if entry, ok := r.lookup.Lookup(obs.MAC); ok {
			if vendor == "" {
				vendor = entry.Vendor
			}
			if typeHint == "" {
				typeHint = entry.TypeHint
			}
		}
	}

	deviceType := typeHint
	if deviceType == "" {
		deviceType = obs.Source
	}
	device := &models.Device{
		ID:          id,
		IPv4:        obs.IPv4,
		MAC:         strings.ToLower(obs.MAC),
		Vendor:      vendor,
		Type:        deviceType,
		Hostname:    obs.Hostname,
		SysObjectID: obs.SysObjectID,
		LastSeen:    obs.ObservedAt,
		Confidence:  obs.Confidence,
	}
	if err := r.store.UpsertDevice(context.Background(), device); err != nil {
		r.log.Error("upsert device failed", "id", id, "err", err)
	}
	r.maybeEmitGatewayLink(id, obs.Source)
}

func (r *recorder) handleLink(obs LinkObservation) {
	if obs.ADevice == "" || obs.BDevice == "" {
		r.log.Warn("skip link with missing device", "source", obs.Source)
		return
	}
	if obs.ID == "" {
		ids := []string{obs.ADevice, obs.BDevice}
		if ids[0] > ids[1] {
			ids[0], ids[1] = ids[1], ids[0]
		}
		obs.ID = strings.ToLower(strings.Join(ids, "-"))
	}
	link := &models.Link{
		ID:         obs.ID,
		ADevice:    obs.ADevice,
		AInterface: obs.AInterface,
		BDevice:    obs.BDevice,
		BInterface: obs.BInterface,
		Media:      obs.Media,
		Source:     obs.Source,
		Kind:       obs.Kind,
		Confidence: obs.Confidence,
	}
	if err := r.store.UpsertLink(context.Background(), link); err != nil {
		r.log.Error("upsert link failed", "id", obs.ID, "err", err)
	}
}

func (r *recorder) isCached(id string, ts time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if last, ok := r.devCache[id]; ok {
		if ts.Sub(last) < r.cacheTTL {
			return true
		}
	}
	r.devCache[id] = ts
	return false
}

func (r *recorder) maybeEmitGatewayLink(deviceID, source string) {
	r.mu.Lock()
	gateway := r.gatewayID
	r.mu.Unlock()
	if gateway == "" || deviceID == gateway {
		return
	}
	if source != "arp_nd" && source != "icmp" {
		return
	}
	r.RecordLink(context.Background(), LinkObservation{
		ADevice:    deviceID,
		BDevice:    gateway,
		Kind:       "gateway",
		Source:     "gateway",
		Confidence: 0.2,
	})
}
