package metrics

import (
	"sync"
	"time"
)

// Bucket holds aggregated metrics for a single second.
type Bucket struct {
	Timestamp   time.Time
	Requests    int64
	Failures    int64
	BytesSent   int64
	BytesRecv   int64
	P95Micros   int64
	AvgMicros   float64
}

// TimeSeries holds per-second metric buckets for chart generation.
type TimeSeries struct {
	mu      sync.Mutex
	buckets []Bucket
	current Bucket
	hist    *LatencyHistogram
}

// NewTimeSeries creates a new TimeSeries starting at now.
func NewTimeSeries() *TimeSeries {
	return &TimeSeries{
		hist: NewLatencyHistogram(),
		current: Bucket{
			Timestamp: time.Now().Truncate(time.Second),
		},
	}
}

// Record adds a data point to the current second bucket.
func (ts *TimeSeries) Record(durationMicros int64, failed bool, bytesSent, bytesRecv int64) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	now := time.Now().Truncate(time.Second)
	if now.After(ts.current.Timestamp) {
		// Flush current bucket
		ts.current.P95Micros = ts.hist.hist.ValueAtQuantile(95)
		if ts.hist.hist.TotalCount() > 0 {
			ts.current.AvgMicros = ts.hist.hist.Mean()
		}
		ts.buckets = append(ts.buckets, ts.current)

		// Reset
		ts.hist = NewLatencyHistogram()
		ts.current = Bucket{Timestamp: now}
	}

	ts.current.Requests++
	if failed {
		ts.current.Failures++
	}
	ts.current.BytesSent += bytesSent
	ts.current.BytesRecv += bytesRecv
	_ = ts.hist.hist.RecordValue(durationMicros)
}

// Flush finalizes the current bucket and returns all buckets.
func (ts *TimeSeries) Flush() []Bucket {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.current.P95Micros = ts.hist.hist.ValueAtQuantile(95)
	if ts.hist.hist.TotalCount() > 0 {
		ts.current.AvgMicros = ts.hist.hist.Mean()
	}
	all := make([]Bucket, len(ts.buckets)+1)
	copy(all, ts.buckets)
	all[len(ts.buckets)] = ts.current
	return all
}

// Buckets returns a snapshot of completed buckets (not the current partial bucket).
func (ts *TimeSeries) Buckets() []Bucket {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	result := make([]Bucket, len(ts.buckets))
	copy(result, ts.buckets)
	return result
}
