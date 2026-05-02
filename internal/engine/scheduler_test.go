package engine

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/sphinxid/bacot/internal/config"
)

func TestScheduler_EmitsStageEvents(t *testing.T) {
	cfg := &config.Config{
		Stages: []config.Stage{
			{Duration: config.Duration{Duration: 50 * time.Millisecond}, VUs: 5},
			{Duration: config.Duration{Duration: 50 * time.Millisecond}, VUs: 10},
		},
	}

	sched := NewScheduler(cfg)
	wg := &sync.WaitGroup{}
	wg.Add(1)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go sched.Run(ctx, wg)

	var events []StageEvent
	for e := range sched.Events() {
		events = append(events, e)
	}
	wg.Wait()

	if len(events) != 2 {
		t.Fatalf("expected 2 stage events, got %d", len(events))
	}
	if events[0].VUs != 5 {
		t.Errorf("stage 0: want 5 VUs, got %d", events[0].VUs)
	}
	if events[1].VUs != 10 {
		t.Errorf("stage 1: want 10 VUs, got %d", events[1].VUs)
	}
	if events[0].StageIndex != 0 {
		t.Errorf("stage 0: want index 0, got %d", events[0].StageIndex)
	}
	if events[1].StageIndex != 1 {
		t.Errorf("stage 1: want index 1, got %d", events[1].StageIndex)
	}
}

func TestScheduler_GradualRamp(t *testing.T) {
	// A 3-second ramp from 1 VU to 10 VUs; we use short durations for speed.
	cfg := &config.Config{
		Stages: []config.Stage{
			{
				Duration: config.Duration{Duration: 300 * time.Millisecond},
				VUs:      10,
				StartVUs: 1,
			},
		},
	}

	sched := NewScheduler(cfg)
	wg := &sync.WaitGroup{}
	wg.Add(1)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go sched.Run(ctx, wg)

	var events []StageEvent
	for e := range sched.Events() {
		events = append(events, e)
	}
	wg.Wait()

	// Must emit more than one event (ramp ticks).
	if len(events) < 2 {
		t.Fatalf("expected multiple events for gradual ramp, got %d", len(events))
	}

	// First event must be at (or near) StartVUs.
	if events[0].VUs < 1 || events[0].VUs > 3 {
		t.Errorf("first ramp event VUs out of expected range: %d", events[0].VUs)
	}

	// Last event must reach the target VU count.
	last := events[len(events)-1]
	if last.VUs != 10 {
		t.Errorf("final ramp event: want 10 VUs, got %d", last.VUs)
	}

	// VU count must be monotonically non-decreasing (ramp-up).
	for i := 1; i < len(events); i++ {
		if events[i].VUs < events[i-1].VUs {
			t.Errorf("VU count decreased during ramp: %d → %d at index %d",
				events[i-1].VUs, events[i].VUs, i)
		}
	}
}

func TestScheduler_GradualRampDown(t *testing.T) {
	cfg := &config.Config{
		Stages: []config.Stage{
			{
				Duration: config.Duration{Duration: 300 * time.Millisecond},
				VUs:      1,
				StartVUs: 10,
			},
		},
	}

	sched := NewScheduler(cfg)
	wg := &sync.WaitGroup{}
	wg.Add(1)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go sched.Run(ctx, wg)

	var events []StageEvent
	for e := range sched.Events() {
		events = append(events, e)
	}
	wg.Wait()

	if len(events) < 2 {
		t.Fatalf("expected multiple events for gradual ramp-down, got %d", len(events))
	}

	// VU count must be monotonically non-increasing (ramp-down).
	for i := 1; i < len(events); i++ {
		if events[i].VUs > events[i-1].VUs {
			t.Errorf("VU count increased during ramp-down: %d → %d at index %d",
				events[i-1].VUs, events[i].VUs, i)
		}
	}

	last := events[len(events)-1]
	if last.VUs != 1 {
		t.Errorf("final ramp-down event: want 1 VU, got %d", last.VUs)
	}
}

func TestScheduler_CancelMidway(t *testing.T) {
	cfg := &config.Config{
		Stages: []config.Stage{
			{Duration: config.Duration{Duration: 5 * time.Second}, VUs: 5},
			{Duration: config.Duration{Duration: 5 * time.Second}, VUs: 10},
		},
	}

	sched := NewScheduler(cfg)
	wg := &sync.WaitGroup{}
	wg.Add(1)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	go sched.Run(ctx, wg)

	var events []StageEvent
	for e := range sched.Events() {
		events = append(events, e)
	}
	wg.Wait()

	// Should have received at most 1 event before cancellation
	if len(events) > 1 {
		t.Errorf("expected at most 1 event before cancel, got %d", len(events))
	}
}
