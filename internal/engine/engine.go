package engine

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/sphinxid/bacot/internal/config"
	"github.com/sphinxid/bacot/internal/metrics"
)

// Engine orchestrates the load test: creates VUs, drives stages, and collects metrics.
type Engine struct {
	cfg       *config.Config
	collector *metrics.Collector
	activeVUs atomic.Int64
	stageIdx  atomic.Int64
}

// New creates a new Engine for the given config and collector.
func New(cfg *config.Config, collector *metrics.Collector) *Engine {
	return &Engine{
		cfg:       cfg,
		collector: collector,
	}
}

// ActiveVUs returns the current number of active virtual users.
func (e *Engine) ActiveVUs() int64 {
	return e.activeVUs.Load()
}

// CurrentStage returns the index of the currently executing stage (0-based).
func (e *Engine) CurrentStage() int64 {
	return e.stageIdx.Load()
}

// Run executes the full test plan. It blocks until the test finishes or ctx is cancelled.
// All goroutines are guaranteed to exit before Run returns.
func (e *Engine) Run(ctx context.Context) {
	scheduler := NewScheduler(e.cfg)
	schedWg := &sync.WaitGroup{}
	schedWg.Add(1)

	schedCtx, schedCancel := context.WithCancel(ctx)
	defer schedCancel()

	go scheduler.Run(schedCtx, schedWg)

	var vuWg sync.WaitGroup
	var vuCancel context.CancelFunc
	var vuCtx context.Context

	// Track current VUs so we can cancel old ones on stage transitions
	type vuPool struct {
		cancel context.CancelFunc
		wg     *sync.WaitGroup
	}
	var currentPool *vuPool

	for event := range scheduler.Events() {
		e.stageIdx.Store(int64(event.StageIndex))

		// Cancel previous stage VUs
		if currentPool != nil {
			currentPool.cancel()
			currentPool.wg.Wait()
		}

		// Create stage context that lives for the stage duration
		vuCtx, vuCancel = context.WithCancel(ctx)
		stageWg := &sync.WaitGroup{}
		currentPool = &vuPool{cancel: vuCancel, wg: stageWg}

		vus := event.VUs
		e.activeVUs.Store(int64(vus))

		for i := 0; i < vus; i++ {
			vu := NewVU(i, e.cfg, e.collector)
			stageWg.Add(1)
			vuWg.Add(1)
			go func(v *VU) {
				defer stageWg.Done()
				defer vuWg.Done()
				v.Run(vuCtx)
			}(vu)
		}
	}

	// Cancel the last stage's VUs once the scheduler channel closes (all stages done).
	if currentPool != nil {
		currentPool.cancel()
		currentPool.wg.Wait()
	}

	// Wait for all VUs to finish draining in-flight requests.
	vuWg.Wait()
	// schedWg is implicitly done: scheduler closed its channel before we got here.
	schedWg.Wait()
	e.activeVUs.Store(0)
}
