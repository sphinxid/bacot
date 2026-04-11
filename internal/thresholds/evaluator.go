// Package thresholds provides threshold expression parsing and evaluation for bacot.
package thresholds

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Result holds the evaluation result for a single threshold.
type Result struct {
	Name       string
	Expression string
	Actual     float64
	Threshold  float64
	Operator   string
	Passed     bool
	Unit       string
}

// MetricsSnapshot holds the final metrics needed for threshold evaluation.
type MetricsSnapshot struct {
	DurationP50Micros int64
	DurationP75Micros int64
	DurationP90Micros int64
	DurationP95Micros int64
	DurationP99Micros int64
	DurationAvgMicros float64
	DurationMinMicros int64
	DurationMaxMicros int64
	FailureRate       float64 // 0.0 to 1.0
	TotalRequests     int64
	TotalFailures     int64
	RPS               float64
}

// EvaluateAll evaluates all threshold expressions against the provided metrics snapshot.
// Returns individual results and whether all thresholds passed.
func EvaluateAll(thresholds map[string]string, snap MetricsSnapshot) ([]Result, bool) {
	allPassed := true
	results := make([]Result, 0, len(thresholds))

	for name, expr := range thresholds {
		result, err := evaluate(name, expr, snap)
		if err != nil {
			result = Result{
				Name:       name,
				Expression: expr,
				Passed:     false,
				Unit:       "error",
			}
		}
		if !result.Passed {
			allPassed = false
		}
		results = append(results, result)
	}

	return results, allPassed
}

// evaluate parses and evaluates a single threshold expression.
// Supported formats:
//
//	"< 500ms"   (duration in ms)
//	"< 1%"      (percentage)
//	"< 100"     (raw float)
func evaluate(name, expr string, snap MetricsSnapshot) (Result, error) {
	expr = strings.TrimSpace(expr)

	var op, rawValue string
	for _, operator := range []string{"<=", ">=", "<", ">"} {
		if strings.HasPrefix(expr, operator) {
			op = operator
			rawValue = strings.TrimSpace(expr[len(operator):])
			break
		}
	}

	if op == "" {
		return Result{}, fmt.Errorf("cannot parse threshold expression %q: missing operator", expr)
	}

	unit := ""
	if strings.HasSuffix(rawValue, "ms") {
		unit = "ms"
		rawValue = strings.TrimSuffix(rawValue, "ms")
	} else if strings.HasSuffix(rawValue, "s") {
		unit = "s"
		rawValue = strings.TrimSuffix(rawValue, "s")
	} else if strings.HasSuffix(rawValue, "%") {
		unit = "%"
		rawValue = strings.TrimSuffix(rawValue, "%")
	}

	threshold, err := strconv.ParseFloat(strings.TrimSpace(rawValue), 64)
	if err != nil {
		return Result{}, fmt.Errorf("cannot parse threshold value %q: %w", rawValue, err)
	}

	actual, err := resolveMetric(name, snap, unit)
	if err != nil {
		return Result{}, err
	}

	passed := compareFloat(actual, op, threshold)

	return Result{
		Name:       name,
		Expression: expr,
		Actual:     actual,
		Threshold:  threshold,
		Operator:   op,
		Passed:     passed,
		Unit:       unit,
	}, nil
}

// resolveMetric maps a threshold name to the corresponding metric value.
func resolveMetric(name string, snap MetricsSnapshot, unit string) (float64, error) {
	switch name {
	case "http_req_duration_p50":
		return microsToUnit(snap.DurationP50Micros, unit), nil
	case "http_req_duration_p75":
		return microsToUnit(snap.DurationP75Micros, unit), nil
	case "http_req_duration_p90":
		return microsToUnit(snap.DurationP90Micros, unit), nil
	case "http_req_duration_p95":
		return microsToUnit(snap.DurationP95Micros, unit), nil
	case "http_req_duration_p99":
		return microsToUnit(snap.DurationP99Micros, unit), nil
	case "http_req_duration_avg":
		return microsToUnit(int64(snap.DurationAvgMicros), unit), nil
	case "http_req_duration_min":
		return microsToUnit(snap.DurationMinMicros, unit), nil
	case "http_req_duration_max":
		return microsToUnit(snap.DurationMaxMicros, unit), nil
	case "http_req_failed":
		if unit == "%" {
			return snap.FailureRate * 100.0, nil
		}
		return snap.FailureRate, nil
	case "http_reqs":
		return float64(snap.TotalRequests), nil
	case "http_reqs_per_sec":
		return snap.RPS, nil
	default:
		return 0, fmt.Errorf("unknown threshold metric %q", name)
	}
}

// microsToUnit converts microseconds to the target unit (ms, s, or raw microseconds).
func microsToUnit(us int64, unit string) float64 {
	switch unit {
	case "ms":
		return float64(us) / float64(time.Millisecond/time.Microsecond)
	case "s":
		return float64(us) / float64(time.Second/time.Microsecond)
	default:
		return float64(us)
	}
}

// compareFloat performs a numeric comparison.
func compareFloat(actual float64, op string, target float64) bool {
	switch op {
	case "<":
		return actual < target
	case "<=":
		return actual <= target
	case ">":
		return actual > target
	case ">=":
		return actual >= target
	}
	return false
}
