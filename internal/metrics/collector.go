package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// RequestResult holds the result of a single HTTP request.
type RequestResult struct {
	ScenarioName   string
	StatusCode     int
	DurationMicros int64
	DNSMicros      int64
	TCPMicros      int64
	TLSMicros      int64
	TTFBMicros     int64
	BytesSent      int64
	BytesRecv      int64
	Error          error
	ErrorType      string // "timeout", "refused", "dns", "http"
	ChecksPassed   int
	ChecksFailed   int
	ResponseBody    []byte            // Populated only when scenario has body checks
	ResponseHeaders map[string]string // Populated only when scenario has header checks
}

// ScenarioMetrics holds per-scenario aggregated metrics.
type ScenarioMetrics struct {
	Name         string
	Requests     atomic.Int64
	Failures     atomic.Int64
	ChecksPassed atomic.Int64
	ChecksFailed atomic.Int64
	BytesSent    atomic.Int64
	BytesRecv    atomic.Int64
	Latency      *LatencyHistogram
}

// NewScenarioMetrics creates a new ScenarioMetrics for the named scenario.
func NewScenarioMetrics(name string) *ScenarioMetrics {
	return &ScenarioMetrics{
		Name:    name,
		Latency: NewLatencyHistogram(),
	}
}

// StatusCodeCount tracks counts per HTTP status code.
type StatusCodeCount struct {
	mu     sync.Mutex
	counts map[int]int64
}

// NewStatusCodeCount creates a new StatusCodeCount.
func NewStatusCodeCount() *StatusCodeCount {
	return &StatusCodeCount{counts: make(map[int]int64)}
}

// Inc increments the count for a status code.
func (s *StatusCodeCount) Inc(code int) {
	s.mu.Lock()
	s.counts[code]++
	s.mu.Unlock()
}

// Snapshot returns a copy of the status code counts.
func (s *StatusCodeCount) Snapshot() map[int]int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[int]int64, len(s.counts))
	for k, v := range s.counts {
		out[k] = v
	}
	return out
}

// Collector aggregates all metrics across all VUs.
type Collector struct {
	startTime time.Time

	TotalRequests  atomic.Int64
	TotalFailures  atomic.Int64
	TotalBytesSent atomic.Int64
	TotalBytesRecv atomic.Int64
	ChecksPassed   atomic.Int64
	ChecksFailed   atomic.Int64

	Latency     *LatencyHistogram
	DNSLatency  *LatencyHistogram
	TCPLatency  *LatencyHistogram
	TLSLatency  *LatencyHistogram
	TTFBLatency *LatencyHistogram
	StatusCodes *StatusCodeCount
	TimeSeries  *TimeSeries

	scenarioMu sync.RWMutex
	scenarios  map[string]*ScenarioMetrics
}

// NewCollector creates a new Collector.
func NewCollector() *Collector {
	return &Collector{
		startTime:   time.Now(),
		Latency:     NewLatencyHistogram(),
		DNSLatency:  NewLatencyHistogram(),
		TCPLatency:  NewLatencyHistogram(),
		TLSLatency:  NewLatencyHistogram(),
		TTFBLatency: NewLatencyHistogram(),
		StatusCodes: NewStatusCodeCount(),
		TimeSeries:  NewTimeSeries(),
		scenarios:   make(map[string]*ScenarioMetrics),
	}
}

// Record records a single request result into all relevant aggregators.
func (c *Collector) Record(r RequestResult) {
	failed := r.Error != nil || r.StatusCode >= 500 || r.StatusCode == 0
	if r.StatusCode >= 400 {
		failed = r.Error != nil || r.StatusCode >= 500
	}

	c.TotalRequests.Add(1)
	if failed {
		c.TotalFailures.Add(1)
	}
	c.TotalBytesSent.Add(r.BytesSent)
	c.TotalBytesRecv.Add(r.BytesRecv)
	c.ChecksPassed.Add(int64(r.ChecksPassed))
	c.ChecksFailed.Add(int64(r.ChecksFailed))

	if r.DurationMicros > 0 {
		c.Latency.RecordMicros(r.DurationMicros)
	}
	if r.DNSMicros > 0 {
		c.DNSLatency.RecordMicros(r.DNSMicros)
	}
	if r.TCPMicros > 0 {
		c.TCPLatency.RecordMicros(r.TCPMicros)
	}
	if r.TLSMicros > 0 {
		c.TLSLatency.RecordMicros(r.TLSMicros)
	}
	if r.TTFBMicros > 0 {
		c.TTFBLatency.RecordMicros(r.TTFBMicros)
	}
	if r.StatusCode > 0 {
		c.StatusCodes.Inc(r.StatusCode)
	}
	c.TimeSeries.Record(r.DurationMicros, failed, r.BytesSent, r.BytesRecv)

	// Per-scenario metrics
	sm := c.getOrCreateScenario(r.ScenarioName)
	sm.Requests.Add(1)
	if failed {
		sm.Failures.Add(1)
	}
	sm.BytesSent.Add(r.BytesSent)
	sm.BytesRecv.Add(r.BytesRecv)
	sm.ChecksPassed.Add(int64(r.ChecksPassed))
	sm.ChecksFailed.Add(int64(r.ChecksFailed))
	if r.DurationMicros > 0 {
		sm.Latency.RecordMicros(r.DurationMicros)
	}
}

func (c *Collector) getOrCreateScenario(name string) *ScenarioMetrics {
	c.scenarioMu.RLock()
	sm, ok := c.scenarios[name]
	c.scenarioMu.RUnlock()
	if ok {
		return sm
	}

	c.scenarioMu.Lock()
	defer c.scenarioMu.Unlock()
	if sm, ok = c.scenarios[name]; ok {
		return sm
	}
	sm = NewScenarioMetrics(name)
	c.scenarios[name] = sm
	return sm
}

// ScenarioSnapshot returns a snapshot of per-scenario metrics.
func (c *Collector) ScenarioSnapshot() []*ScenarioMetrics {
	c.scenarioMu.RLock()
	defer c.scenarioMu.RUnlock()
	out := make([]*ScenarioMetrics, 0, len(c.scenarios))
	for _, sm := range c.scenarios {
		out = append(out, sm)
	}
	return out
}

// Elapsed returns the time since the collector was started.
func (c *Collector) Elapsed() time.Duration {
	return time.Since(c.startTime)
}

// RPS returns the current average requests per second.
func (c *Collector) RPS() float64 {
	elapsed := c.Elapsed().Seconds()
	if elapsed <= 0 {
		return 0
	}
	return float64(c.TotalRequests.Load()) / elapsed
}

// FailureRate returns the failure rate as a fraction (0.0 to 1.0).
func (c *Collector) FailureRate() float64 {
	total := c.TotalRequests.Load()
	if total == 0 {
		return 0
	}
	return float64(c.TotalFailures.Load()) / float64(total)
}
