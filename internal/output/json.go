package output

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/sphinxid/bacot/internal/metrics"
	"github.com/sphinxid/bacot/internal/thresholds"
)

// JSONReport is the top-level structure for the JSON output file.
type JSONReport struct {
	TestName         string                    `json:"test_name"`
	Duration         string                    `json:"duration"`
	StartedAt        time.Time                 `json:"started_at"`
	Summary          JSONSummary               `json:"summary"`
	Scenarios        []JSONScenarioSummary     `json:"scenarios"`
	ThresholdResults []JSONThresholdResult     `json:"thresholds"`
	StatusCodes      map[string]int64          `json:"status_codes"`
	TimeSeries       []JSONTimeSeriesBucket    `json:"time_series"`
}

// JSONSummary holds the top-level metric aggregates.
type JSONSummary struct {
	TotalRequests  int64   `json:"total_requests"`
	TotalFailures  int64   `json:"total_failures"`
	FailureRatePct float64 `json:"failure_rate_pct"`
	RPS            float64 `json:"rps"`
	BytesSent      int64   `json:"bytes_sent"`
	BytesRecv      int64   `json:"bytes_received"`
	ChecksPassed   int64   `json:"checks_passed"`
	ChecksFailed   int64   `json:"checks_failed"`
	LatencyStats   JSONLatency `json:"latency"`
}

// JSONLatency holds latency percentile statistics (in ms).
type JSONLatency struct {
	MinMs  float64 `json:"min_ms"`
	MaxMs  float64 `json:"max_ms"`
	AvgMs  float64 `json:"avg_ms"`
	P50Ms  float64 `json:"p50_ms"`
	P75Ms  float64 `json:"p75_ms"`
	P90Ms  float64 `json:"p90_ms"`
	P95Ms  float64 `json:"p95_ms"`
	P99Ms  float64 `json:"p99_ms"`
}

// JSONScenarioSummary holds per-scenario aggregated metrics.
type JSONScenarioSummary struct {
	Name           string      `json:"name"`
	Requests       int64       `json:"requests"`
	Failures       int64       `json:"failures"`
	FailureRatePct float64     `json:"failure_rate_pct"`
	BytesSent      int64       `json:"bytes_sent"`
	BytesRecv      int64       `json:"bytes_received"`
	ChecksPassed   int64       `json:"checks_passed"`
	ChecksFailed   int64       `json:"checks_failed"`
	Latency        JSONLatency `json:"latency"`
}

// JSONThresholdResult holds the evaluation result for a threshold.
type JSONThresholdResult struct {
	Name       string  `json:"name"`
	Expression string  `json:"expression"`
	Passed     bool    `json:"passed"`
	Actual     float64 `json:"actual"`
	Threshold  float64 `json:"threshold"`
	Unit       string  `json:"unit"`
}

// JSONTimeSeriesBucket holds aggregated metrics for a single second.
type JSONTimeSeriesBucket struct {
	TimestampUnix int64   `json:"timestamp"`
	Requests      int64   `json:"requests"`
	Failures      int64   `json:"failures"`
	RPS           float64 `json:"rps"`
	ErrorRatePct  float64 `json:"error_rate_pct"`
	P95Ms         float64 `json:"p95_ms"`
	AvgMs         float64 `json:"avg_ms"`
	BytesSent     int64   `json:"bytes_sent"`
	BytesRecv     int64   `json:"bytes_received"`
}

// WriteJSON writes a full JSON report to the given file path.
func WriteJSON(path string, testName string, collector *metrics.Collector, thresholdResults []thresholds.Result, elapsed time.Duration, startedAt time.Time) error {
	report := buildJSONReport(testName, collector, thresholdResults, elapsed, startedAt)

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON report: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing JSON report to %s: %w", path, err)
	}
	return nil
}

func buildJSONReport(testName string, collector *metrics.Collector, thresholdResults []thresholds.Result, elapsed time.Duration, startedAt time.Time) JSONReport {
	total := collector.TotalRequests.Load()
	failures := collector.TotalFailures.Load()
	failPct := float64(0)
	if total > 0 {
		failPct = float64(failures) / float64(total) * 100
	}

	lat := collector.Latency.Snapshot()

	summary := JSONSummary{
		TotalRequests:  total,
		TotalFailures:  failures,
		FailureRatePct: failPct,
		RPS:            collector.RPS(),
		BytesSent:      collector.TotalBytesSent.Load(),
		BytesRecv:      collector.TotalBytesRecv.Load(),
		ChecksPassed:   collector.ChecksPassed.Load(),
		ChecksFailed:   collector.ChecksFailed.Load(),
		LatencyStats: JSONLatency{
			MinMs: float64(lat.Min) / 1000.0,
			MaxMs: float64(lat.Max) / 1000.0,
			AvgMs: lat.Mean / 1000.0,
			P50Ms: float64(lat.P50) / 1000.0,
			P75Ms: float64(lat.P75) / 1000.0,
			P90Ms: float64(lat.P90) / 1000.0,
			P95Ms: float64(lat.P95) / 1000.0,
			P99Ms: float64(lat.P99) / 1000.0,
		},
	}

	// Scenarios
	scenarioSnapshots := collector.ScenarioSnapshot()
	scenarioSummaries := make([]JSONScenarioSummary, 0, len(scenarioSnapshots))
	for _, sm := range scenarioSnapshots {
		reqs := sm.Requests.Load()
		smFail := sm.Failures.Load()
		smFailPct := float64(0)
		if reqs > 0 {
			smFailPct = float64(smFail) / float64(reqs) * 100
		}
		smLat := sm.Latency.Snapshot()
		scenarioSummaries = append(scenarioSummaries, JSONScenarioSummary{
			Name:           sm.Name,
			Requests:       reqs,
			Failures:       smFail,
			FailureRatePct: smFailPct,
			BytesSent:      sm.BytesSent.Load(),
			BytesRecv:      sm.BytesRecv.Load(),
			ChecksPassed:   sm.ChecksPassed.Load(),
			ChecksFailed:   sm.ChecksFailed.Load(),
			Latency: JSONLatency{
				MinMs: float64(smLat.Min) / 1000.0,
				MaxMs: float64(smLat.Max) / 1000.0,
				AvgMs: smLat.Mean / 1000.0,
				P50Ms: float64(smLat.P50) / 1000.0,
				P75Ms: float64(smLat.P75) / 1000.0,
				P90Ms: float64(smLat.P90) / 1000.0,
				P95Ms: float64(smLat.P95) / 1000.0,
				P99Ms: float64(smLat.P99) / 1000.0,
			},
		})
	}

	// Thresholds
	threshJSON := make([]JSONThresholdResult, 0, len(thresholdResults))
	for _, tr := range thresholdResults {
		threshJSON = append(threshJSON, JSONThresholdResult{
			Name:       tr.Name,
			Expression: tr.Expression,
			Passed:     tr.Passed,
			Actual:     tr.Actual,
			Threshold:  tr.Threshold,
			Unit:       tr.Unit,
		})
	}

	// Status codes
	rawCodes := collector.StatusCodes.Snapshot()
	statusCodesStr := make(map[string]int64, len(rawCodes))
	for code, cnt := range rawCodes {
		statusCodesStr[fmt.Sprintf("%d", code)] = cnt
	}

	// TimeSeries
	buckets := collector.TimeSeries.Flush()
	tsBuckets := make([]JSONTimeSeriesBucket, 0, len(buckets))
	for _, b := range buckets {
		rps := float64(0)
		errPct := float64(0)
		if b.Requests > 0 {
			errPct = float64(b.Failures) / float64(b.Requests) * 100
			rps = float64(b.Requests)
		}
		tsBuckets = append(tsBuckets, JSONTimeSeriesBucket{
			TimestampUnix: b.Timestamp.Unix(),
			Requests:      b.Requests,
			Failures:      b.Failures,
			RPS:           rps,
			ErrorRatePct:  errPct,
			P95Ms:         float64(b.P95Micros) / 1000.0,
			AvgMs:         b.AvgMicros / 1000.0,
			BytesSent:     b.BytesSent,
			BytesRecv:     b.BytesRecv,
		})
	}

	return JSONReport{
		TestName:         testName,
		Duration:         elapsed.String(),
		StartedAt:        startedAt,
		Summary:          summary,
		Scenarios:        scenarioSummaries,
		ThresholdResults: threshJSON,
		StatusCodes:      statusCodesStr,
		TimeSeries:       tsBuckets,
	}
}
