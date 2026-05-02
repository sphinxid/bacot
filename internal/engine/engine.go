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

// vuEntry holds a running VU's cancel function and done channel.
type vuEntry struct {
	cancel context.CancelFunc
	done   chan struct{}
}

// Run executes the full test plan. It blocks until the test finishes or ctx is cancelled.
// All goroutines are guaranteed to exit before Run returns.
//
// During gradual ramps the engine adjusts the running VU pool incrementally:
// spawning additional VUs when the count increases and cancelling excess VUs
// when the count decreases. This avoids disrupting in-flight requests.
func (e *Engine) Run(ctx context.Context) {
	scheduler := NewScheduler(e.cfg)
	schedWg := &sync.WaitGroup{}
	schedWg.Add(1)

	schedCtx, schedCancel := context.WithCancel(ctx)
	defer schedCancel()

	go scheduler.Run(schedCtx, schedWg)

	// vuCtx / vuCancel is shared across all running VUs so we can kill them
	// all at once when the test ends.
	vuCtx, vuCancel := context.WithCancel(ctx)
	defer vuCancel()

	var vuWg sync.WaitGroup

	// Pool of currently running VUs, indexed sequentially.
	// We append when scaling up; pop+cancel when scaling down.
	pool := make([]vuEntry, 0, 64)
	nextID := 0

	spawnVU := func() {
		id := nextID
		nextID++
		vu := NewVU(id, e.cfg, e.collector)
		childCtx, childCancel := context.WithCancel(vuCtx)
		done := make(chan struct{})
		entry := vuEntry{cancel: childCancel, done: done}
		pool = append(pool, entry)
		vuWg.Add(1)
		go func() {
			defer vuWg.Done()
			defer close(done)
			vu.Run(childCtx)
		}()
	}

	cancelVU := func() {
		if len(pool) == 0 {
			return
		}
		last := pool[len(pool)-1]
		pool = pool[:len(pool)-1]
		last.cancel()
		// Wait for the VU to finish its current in-flight request before returning
		// so activeVUs stays accurate. Use a non-blocking drain in case it already
		// exited.
		<-last.done
	}

	for event := range scheduler.Events() {
		e.stageIdx.Store(int64(event.StageIndex))

		target := event.VUs
		current := len(pool)

		for current < target {
			spawnVU()
			current++
		}
		for current > target {
			cancelVU()
			current--
		}

		e.activeVUs.Store(int64(target))
	}

	// All stages done — cancel every remaining VU.
	for len(pool) > 0 {
		cancelVU()
	}

	// Wait for all VUs to finish draining in-flight requests.
	vuWg.Wait()
	schedWg.Wait()
	e.activeVUs.Store(0)
}
