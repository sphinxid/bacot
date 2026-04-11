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
