package scanner

import (
	"context"
	"runtime"
	"sync"

	"github.com/tomfocker/lanmapper/internal/logger"
)

// Manager schedules scanning jobs across runners.
type Manager struct {
	runners  []Runner
	jobs     chan Job
	wg       sync.WaitGroup
	log      Logger
	recorder Recorder
}

// Logger minimal logging interface to decouple from slog.
type Logger interface {
	Error(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
}

// NewManager constructs manager with provided runners.
func NewManager(recorder Recorder, runners ...Runner) *Manager {
	return &Manager{
		runners:  runners,
		jobs:     make(chan Job, 64),
		log:      logger.L(),
		recorder: recorder,
	}
}

// Start launches worker goroutines.
func (m *Manager) Start(ctx context.Context) {
	workerCount := runtime.NumCPU() * 5
	for i := 0; i < workerCount; i++ {
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case job, ok := <-m.jobs:
					if !ok {
						return
					}
					for _, r := range m.runners {
						if err := r.Run(job, m.recorder); err != nil {
							m.log.Error("runner failed", "runner", r.Name(), "err", err)
						}
					}
				}
			}
		}()
	}
}

// Enqueue queues a job for execution.
func (m *Manager) Enqueue(job Job) {
	m.jobs <- job
}

// Stop waits for workers to finish.
func (m *Manager) Stop() {
	close(m.jobs)
	m.wg.Wait()
}
