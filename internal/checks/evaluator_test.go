package checks

import (
	"testing"
)

func TestEvaluator_StatusChecks(t *testing.T) {
	tests := []struct {
		expr   string
		resp   Response
		want   bool
	}{
		{"status == 200", Response{StatusCode: 200}, true},
		{"status == 200", Response{StatusCode: 404}, false},
		{"status != 500", Response{StatusCode: 200}, true},
		{"status != 500", Response{StatusCode: 500}, false},
		{"status >= 200", Response{StatusCode: 200}, true},
		{"status >= 200", Response{StatusCode: 199}, false},
		{"status < 400", Response{StatusCode: 200}, true},
		{"status < 400", Response{StatusCode: 500}, false},
	}
	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			e := NewEvaluator([]string{tc.expr})
			_, passed, failed := e.Evaluate(tc.resp)
			if tc.want && passed != 1 {
				t.Errorf("expected pass for %q with status %d", tc.expr, tc.resp.StatusCode)
			}
			if !tc.want && failed != 1 {
				t.Errorf("expected fail for %q with status %d", tc.expr, tc.resp.StatusCode)
			}
		})
	}
}

func TestEvaluator_DurationChecks(t *testing.T) {
	tests := []struct {
		expr   string
		resp   Response
		want   bool
	}{
		{"duration < 500", Response{DurationMs: 100}, true},
		{"duration < 500", Response{DurationMs: 600}, false},
		{"duration <= 500", Response{DurationMs: 500}, true},
		{"duration > 100", Response{DurationMs: 200}, true},
		{"duration > 100", Response{DurationMs: 50}, false},
	}
	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			e := NewEvaluator([]string{tc.expr})
			_, passed, failed := e.Evaluate(tc.resp)
			if tc.want && passed != 1 {
				t.Errorf("expected pass for %q with duration %.0fms", tc.expr, tc.resp.DurationMs)
			}
			if !tc.want && failed != 1 {
				t.Errorf("expected fail for %q with duration %.0fms", tc.expr, tc.resp.DurationMs)
			}
		})
	}
}

func TestEvaluator_MultipleChecks(t *testing.T) {
	e := NewEvaluator([]string{"status == 200", "duration < 500", "status != 404"})
	resp := Response{StatusCode: 200, DurationMs: 100}
	results, passed, failed := e.Evaluate(resp)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if passed != 3 {
		t.Errorf("expected 3 passed, got %d", passed)
	}
	if failed != 0 {
		t.Errorf("expected 0 failed, got %d", failed)
	}
}

func TestEvaluator_InvalidExpression(t *testing.T) {
	e := NewEvaluator([]string{"badexpr"})
	_, _, failed := e.Evaluate(Response{StatusCode: 200})
	if failed != 1 {
		t.Errorf("expected invalid expression to count as failed")
	}
}
