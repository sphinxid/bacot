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
		eventCh: make(chan StageEvent, 8),
	}
}

// Run emits StageEvents for each stage in sequence. It blocks until all stages
// have completed or the context is cancelled. The events channel is always
// closed when Run returns, so range loops on Events() never deadlock.
//
// For stages with start_vus set (and start_vus != vus), the scheduler emits
// one event per second that linearly interpolates the VU count, producing a
// smooth ramp rather than an instant step change.
func (s *Scheduler) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(s.eventCh)

	for i, stage := range s.cfg.Stages {
		select {
		case <-ctx.Done():
			return
		default:
		}

		isRamp := stage.StartVUs > 0 && stage.StartVUs != stage.VUs
		if !isRamp {
			// Instant transition: emit a single event and wait.
			if !s.emit(ctx, StageEvent{
				StageIndex: i,
				VUs:        stage.VUs,
				Duration:   stage.Duration.Duration,
			}) {
				return
			}
			if !s.sleep(ctx, stage.Duration.Duration) {
				return
			}
			continue
		}

		// Gradual ramp: tick every second, interpolating VU count.
		totalDur := stage.Duration.Duration
		startVUs := stage.StartVUs
		endVUs := stage.VUs
		tickInterval := time.Second

		stageStart := time.Now()
		lastVUs := -1 // track to avoid redundant events

		for {
			elapsed := time.Since(stageStart)
			if elapsed >= totalDur {
				break
			}

			// Linear interpolation: progress in [0,1]
			progress := float64(elapsed) / float64(totalDur)
			currentVUs := startVUs + int(progress*float64(endVUs-startVUs)+0.5)
			if currentVUs < 1 {
				currentVUs = 1
			}

			if currentVUs != lastVUs {
				if !s.emit(ctx, StageEvent{
					StageIndex: i,
					VUs:        currentVUs,
					Duration:   totalDur - elapsed,
				}) {
					return
				}
				lastVUs = currentVUs
			}

			// Sleep until next tick or stage end, whichever is sooner.
			remaining := totalDur - time.Since(stageStart)
			sleepDur := tickInterval
			if sleepDur > remaining {
				sleepDur = remaining
			}
			if sleepDur <= 0 {
				break
			}
			if !s.sleep(ctx, sleepDur) {
				return
			}
		}

		// Ensure we always end at the target VU count.
		if lastVUs != endVUs {
			if !s.emit(ctx, StageEvent{
				StageIndex: i,
				VUs:        endVUs,
				Duration:   0,
			}) {
				return
			}
		}
	}
}

// emit sends a StageEvent, returning false if ctx is cancelled.
func (s *Scheduler) emit(ctx context.Context, ev StageEvent) bool {
	select {
	case s.eventCh <- ev:
		return true
	case <-ctx.Done():
		return false
	}
}

// sleep waits for dur, returning false if ctx is cancelled.
func (s *Scheduler) sleep(ctx context.Context, dur time.Duration) bool {
	timer := time.NewTimer(dur)
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		timer.Stop()
		return false
	}
}

// Events returns the channel on which StageEvents are published.
func (s *Scheduler) Events() <-chan StageEvent {
	return s.eventCh
}
