package thresholds

import (
	"testing"
)

func makeSnap() MetricsSnapshot {
	return MetricsSnapshot{
		DurationP50Micros: 80_000,   // 80ms
		DurationP75Micros: 150_000,  // 150ms
		DurationP90Micros: 280_000,  // 280ms
		DurationP95Micros: 310_000,  // 310ms
		DurationP99Micros: 490_000,  // 490ms
		DurationAvgMicros: 95_000.0, // 95ms
		DurationMinMicros: 12_000,   // 12ms
		DurationMaxMicros: 600_000,  // 600ms
		FailureRate:       0.0009,   // 0.09%
		TotalRequests:     12450,
		TotalFailures:     12,
		RPS:               245.3,
	}
}

func TestEvaluateAll_AllPass(t *testing.T) {
	thresh := map[string]string{
		"http_req_duration_p95": "< 500ms",
		"http_req_failed":       "< 1%",
		"http_req_duration_avg": "< 200ms",
	}
	results, allPassed := EvaluateAll(thresh, makeSnap())
	if !allPassed {
		t.Errorf("expected all thresholds to pass")
	}
	for _, r := range results {
		if !r.Passed {
			t.Errorf("threshold %q should pass (actual: %.2f)", r.Name, r.Actual)
		}
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}

func TestEvaluateAll_Fail(t *testing.T) {
	thresh := map[string]string{
		"http_req_duration_p95": "< 200ms", // 310ms > 200ms — should FAIL
	}
	results, allPassed := EvaluateAll(thresh, makeSnap())
	if allPassed {
		t.Errorf("expected threshold to fail")
	}
	if len(results) != 1 || results[0].Passed {
		t.Errorf("expected result to be failed")
	}
}

func TestEvaluate_Percentage(t *testing.T) {
	thresh := map[string]string{
		"http_req_failed": "< 1%",
	}
	results, allPassed := EvaluateAll(thresh, makeSnap())
	if !allPassed {
		t.Errorf("expected 0.09%% < 1%% to pass")
	}
	if len(results) != 1 || results[0].Unit != "%" {
		t.Errorf("expected unit to be '%%'")
	}
}

func TestEvaluate_SecondsUnit(t *testing.T) {
	thresh := map[string]string{
		"http_req_duration_p99": "< 1s", // 490ms < 1s
	}
	results, allPassed := EvaluateAll(thresh, makeSnap())
	if !allPassed {
		t.Errorf("expected 490ms < 1s to pass")
	}
	_ = results
}

func TestEvaluate_UnknownMetric(t *testing.T) {
	thresh := map[string]string{
		"unknown_metric": "< 100",
	}
	results, allPassed := EvaluateAll(thresh, makeSnap())
	if allPassed {
		t.Errorf("expected unknown metric to fail")
	}
	if len(results) == 0 {
		t.Error("expected at least one result")
	}
}

func TestEvaluate_GTE(t *testing.T) {
	thresh := map[string]string{
		"http_reqs": ">= 1000",
	}
	results, allPassed := EvaluateAll(thresh, makeSnap())
	if !allPassed {
		t.Errorf("expected 12450 >= 1000 to pass")
	}
	_ = results
}
