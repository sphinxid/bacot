package engine

import (
	"context"
	"math/rand"
	"net/http"
	"time"

	"github.com/sphinxid/bacot/internal/checks"
	"github.com/sphinxid/bacot/internal/config"
	"github.com/sphinxid/bacot/internal/httpclient"
	"github.com/sphinxid/bacot/internal/metrics"
)

// thinkTimeDuration computes the think-time sleep for a scenario, using the
// VU's own RNG for randomness. Returns 0 if no think time is configured.
func thinkTimeDuration(tt *config.ThinkTime, rng *rand.Rand) time.Duration {
	if tt == nil || tt.MinMs <= 0 {
		return 0
	}
	if tt.MaxMs <= 0 || tt.MaxMs == tt.MinMs {
		return time.Duration(tt.MinMs) * time.Millisecond
	}
	rangeMs := tt.MaxMs - tt.MinMs
	return time.Duration(tt.MinMs+rng.Intn(rangeMs+1)) * time.Millisecond //nolint:gosec
}

// VU is a virtual user that executes HTTP requests in a loop.
type VU struct {
	id        int
	cfg       *config.Config
	client    *http.Client
	collector *metrics.Collector
	rng       *rand.Rand
}

// NewVU creates a new VU with its own HTTP client and random source.
func NewVU(id int, cfg *config.Config, collector *metrics.Collector) *VU {
	clientOpts := httpclient.Options{
		Timeout:        cfg.Timeout,
		ConnectTimeout: cfg.ConnectTimeout,
		Insecure:       cfg.Insecure,
		MaxRedirects:   cfg.MaxRedirects,
		HTTP2:          cfg.HTTP2,
		KeepAlive:      cfg.KeepAlive,
		EnableCookies:  cfg.Cookies,
	}
	return &VU{
		id:        id,
		cfg:       cfg,
		client:    httpclient.New(clientOpts),
		collector: collector,
		rng:       rand.New(rand.NewSource(time.Now().UnixNano() + int64(id))), //nolint:gosec
	}
}

// Run executes the VU request loop until the context is cancelled.
func (v *VU) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		scenario := v.pickScenario()
		if scenario == nil {
			return
		}

		url := v.cfg.Target + scenario.Path

		spec := httpclient.RequestSpec{
			Method:         scenario.Method,
			URL:            url,
			Headers:        scenario.Headers,
			Body:           scenario.Body,
			Name:           scenario.Name,
			CaptureBody:    checks.NeedsBody(scenario.Checks),
			CaptureHeaders: checks.NeedsHeaders(scenario.Checks),
		}

		result := httpclient.Execute(ctx, v.client, spec)

		// Evaluate checks
		if len(scenario.Checks) > 0 {
			eval := checks.NewEvaluator(scenario.Checks)
			durationMs := float64(result.DurationMicros) / 1000.0
			_, passed, failed := eval.Evaluate(checks.Response{
				StatusCode: result.StatusCode,
				DurationMs: durationMs,
				Body:       result.ResponseBody,
				Headers:    result.ResponseHeaders,
			})
			result.ChecksPassed = passed
			result.ChecksFailed = failed
		}

		v.collector.Record(result)

		// Think time: optional pause between requests, respects context cancellation.
		if sleep := thinkTimeDuration(scenario.ThinkTime, v.rng); sleep > 0 {
			timer := time.NewTimer(sleep)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		} else {
			// Brief yield to avoid busy-spinning on very fast endpoints.
			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}
}

// pickScenario picks a scenario based on weighted random selection.
func (v *VU) pickScenario() *config.Scenario {
	if len(v.cfg.Scenarios) == 0 {
		return nil
	}
	if len(v.cfg.Scenarios) == 1 {
		return &v.cfg.Scenarios[0]
	}

	total := v.cfg.WeightTotal()
	if total <= 0 {
		return &v.cfg.Scenarios[v.rng.Intn(len(v.cfg.Scenarios))]
	}

	r := v.rng.Intn(total)
	cumulative := 0
	for i := range v.cfg.Scenarios {
		cumulative += v.cfg.Scenarios[i].Weight
		if r < cumulative {
			return &v.cfg.Scenarios[i]
		}
	}
	return &v.cfg.Scenarios[len(v.cfg.Scenarios)-1]
}
