// Package engine provides the load test orchestration, VU lifecycle, and stage scheduling.
package engine

import (
	"context"
	"sync"
	"time"

	"github.com/sphinxid/bacot/internal/config"
)

// StageEvent describes a change in the number of active VUs.
type StageEvent struct {
	StageIndex int
	VUs        int
	Duration   time.Duration
}

// Scheduler manages VU ramp-up and ramp-down across test stages.
type Scheduler struct {
	cfg     *config.Config
	eventCh chan StageEvent
}

// NewScheduler creates a new Scheduler for the given config.
func NewScheduler(cfg *config.Config) *Scheduler {
	return &Scheduler{
		cfg:     cfg,
		eventCh: make(chan StageEvent, 1),
	}
}

// Run emits StageEvents for each stage in sequence. It blocks until all stages
// have completed or the context is cancelled. The events channel is always
// closed when Run returns, so range loops on Events() never deadlock.
func (s *Scheduler) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(s.eventCh)

	for i, stage := range s.cfg.Stages {
		select {
		case <-ctx.Done():
			return
		default:
		}

		select {
		case s.eventCh <- StageEvent{
			StageIndex: i,
			VUs:        stage.VUs,
			Duration:   stage.Duration.Duration,
		}:
		case <-ctx.Done():
			return
		}

		// Wait for the stage duration
		timer := time.NewTimer(stage.Duration.Duration)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return
		}
	}
}

// Events returns the channel on which StageEvents are published.
func (s *Scheduler) Events() <-chan StageEvent {
	return s.eventCh
}
