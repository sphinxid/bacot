// Package metrics provides thread-safe metrics collection and aggregation for bacot.
package metrics

import (
	"sync"

	hdrhistogram "github.com/HdrHistogram/hdrhistogram-go"
)

// LatencyHistogram wraps an HDR histogram for latency measurements (in microseconds).
type LatencyHistogram struct {
	mu   sync.Mutex
	hist *hdrhistogram.Histogram
}

// NewLatencyHistogram creates a new LatencyHistogram with 1µs to 60s range and 3 significant figures.
func NewLatencyHistogram() *LatencyHistogram {
	return &LatencyHistogram{
		hist: hdrhistogram.New(1, 60_000_000, 3),
	}
}

// RecordMicros records a latency value in microseconds.
func (h *LatencyHistogram) RecordMicros(us int64) {
	h.mu.Lock()
	_ = h.hist.RecordValue(us)
	h.mu.Unlock()
}

// Percentile returns the value at the given percentile (0-100).
func (h *LatencyHistogram) Percentile(p float64) int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.hist.ValueAtQuantile(p)
}

// Min returns the minimum recorded value.
func (h *LatencyHistogram) Min() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.hist.Min()
}

// Max returns the maximum recorded value.
func (h *LatencyHistogram) Max() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.hist.Max()
}

// Mean returns the mean of all recorded values.
func (h *LatencyHistogram) Mean() float64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.hist.Mean()
}

// TotalCount returns the total number of recorded values.
func (h *LatencyHistogram) TotalCount() int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.hist.TotalCount()
}

// Merge merges another histogram into this one.
func (h *LatencyHistogram) Merge(other *LatencyHistogram) {
	other.mu.Lock()
	otherHist := other.hist
	other.mu.Unlock()

	h.mu.Lock()
	defer h.mu.Unlock()
	h.hist.Merge(otherHist)
}

// Snapshot returns a Stats snapshot of the current histogram data.
func (h *LatencyHistogram) Snapshot() Stats {
	h.mu.Lock()
	defer h.mu.Unlock()
	return Stats{
		Min:   h.hist.Min(),
		Max:   h.hist.Max(),
		Mean:  h.hist.Mean(),
		P50:   h.hist.ValueAtQuantile(50),
		P75:   h.hist.ValueAtQuantile(75),
		P90:   h.hist.ValueAtQuantile(90),
		P95:   h.hist.ValueAtQuantile(95),
		P99:   h.hist.ValueAtQuantile(99),
		Count: h.hist.TotalCount(),
	}
}

// Stats holds aggregated latency statistics (values in microseconds).
type Stats struct {
	Min   int64
	Max   int64
	Mean  float64
	P50   int64
	P75   int64
	P90   int64
	P95   int64
	P99   int64
	Count int64
}
