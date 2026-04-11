package metrics

import (
	"testing"
)

func TestCollector_RecordAndAggregate(t *testing.T) {
	c := NewCollector()

	c.Record(RequestResult{
		ScenarioName:   "GET /",
		StatusCode:     200,
		DurationMicros: 100_000,
		BytesSent:      512,
		BytesRecv:      2048,
		ChecksPassed:   2,
		ChecksFailed:   0,
	})
	c.Record(RequestResult{
		ScenarioName:   "GET /",
		StatusCode:     200,
		DurationMicros: 200_000,
		BytesSent:      512,
		BytesRecv:      2048,
		ChecksPassed:   1,
		ChecksFailed:   1,
	})
	c.Record(RequestResult{
		ScenarioName:   "POST /login",
		StatusCode:     500,
		DurationMicros: 50_000,
		BytesSent:      1024,
		BytesRecv:      256,
	})

	if got := c.TotalRequests.Load(); got != 3 {
		t.Errorf("TotalRequests: want 3, got %d", got)
	}
	if got := c.TotalFailures.Load(); got != 1 {
		t.Errorf("TotalFailures: want 1, got %d", got)
	}
	if got := c.TotalBytesSent.Load(); got != 2048 {
		t.Errorf("TotalBytesSent: want 2048, got %d", got)
	}
	if got := c.TotalBytesRecv.Load(); got != 4352 {
		t.Errorf("TotalBytesRecv: want 4352, got %d", got)
	}
	if got := c.ChecksPassed.Load(); got != 3 {
		t.Errorf("ChecksPassed: want 3, got %d", got)
	}
	if got := c.ChecksFailed.Load(); got != 1 {
		t.Errorf("ChecksFailed: want 1, got %d", got)
	}
}

func TestCollector_FailureRate(t *testing.T) {
	c := NewCollector()
	c.Record(RequestResult{StatusCode: 200, DurationMicros: 100_000})
	c.Record(RequestResult{StatusCode: 200, DurationMicros: 100_000})
	c.Record(RequestResult{StatusCode: 500, DurationMicros: 100_000})

	rate := c.FailureRate()
	if rate < 0.32 || rate > 0.34 {
		t.Errorf("FailureRate: want ~0.333, got %.4f", rate)
	}
}

func TestCollector_StatusCodes(t *testing.T) {
	c := NewCollector()
	c.Record(RequestResult{StatusCode: 200, DurationMicros: 10_000})
	c.Record(RequestResult{StatusCode: 200, DurationMicros: 10_000})
	c.Record(RequestResult{StatusCode: 404, DurationMicros: 10_000})

	codes := c.StatusCodes.Snapshot()
	if codes[200] != 2 {
		t.Errorf("want 2 for 200, got %d", codes[200])
	}
	if codes[404] != 1 {
		t.Errorf("want 1 for 404, got %d", codes[404])
	}
}

func TestCollector_ScenarioBreakdown(t *testing.T) {
	c := NewCollector()
	for i := 0; i < 5; i++ {
		c.Record(RequestResult{ScenarioName: "A", StatusCode: 200, DurationMicros: 100_000})
	}
	for i := 0; i < 3; i++ {
		c.Record(RequestResult{ScenarioName: "B", StatusCode: 200, DurationMicros: 200_000})
	}

	scenarios := c.ScenarioSnapshot()
	totals := make(map[string]int64)
	for _, sm := range scenarios {
		totals[sm.Name] = sm.Requests.Load()
	}
	if totals["A"] != 5 {
		t.Errorf("scenario A: want 5, got %d", totals["A"])
	}
	if totals["B"] != 3 {
		t.Errorf("scenario B: want 3, got %d", totals["B"])
	}
}

func TestCollector_RPS(t *testing.T) {
	c := NewCollector()
	c.Record(RequestResult{StatusCode: 200, DurationMicros: 10_000})
	rps := c.RPS()
	if rps <= 0 {
		t.Errorf("RPS should be > 0, got %.2f", rps)
	}
}
