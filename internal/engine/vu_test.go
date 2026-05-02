package engine

import (
	"math/rand"
	"testing"
	"time"

	"github.com/sphinxid/bacot/internal/config"
)

func TestThinkTimeDuration_Nil(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	if got := thinkTimeDuration(nil, rng); got != 0 {
		t.Errorf("nil think_time: want 0, got %s", got)
	}
}

func TestThinkTimeDuration_ZeroMin(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	tt := &config.ThinkTime{MinMs: 0, MaxMs: 0}
	if got := thinkTimeDuration(tt, rng); got != 0 {
		t.Errorf("zero think_time: want 0, got %s", got)
	}
}

func TestThinkTimeDuration_Constant(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	tt := &config.ThinkTime{MinMs: 250, MaxMs: 0}
	got := thinkTimeDuration(tt, rng)
	if got != 250*time.Millisecond {
		t.Errorf("constant think_time: want 250ms, got %s", got)
	}
}

func TestThinkTimeDuration_ConstantEqualMinMax(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	tt := &config.ThinkTime{MinMs: 100, MaxMs: 100}
	got := thinkTimeDuration(tt, rng)
	if got != 100*time.Millisecond {
		t.Errorf("min==max think_time: want 100ms, got %s", got)
	}
}

func TestThinkTimeDuration_Range(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	tt := &config.ThinkTime{MinMs: 100, MaxMs: 500}
	for i := 0; i < 100; i++ {
		got := thinkTimeDuration(tt, rng)
		if got < 100*time.Millisecond || got > 500*time.Millisecond {
			t.Errorf("think_time out of range [100ms,500ms]: got %s", got)
		}
	}
}
