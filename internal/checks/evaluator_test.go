package checks

import (
	"testing"
)

func TestEvaluator_StatusChecks(t *testing.T) {
	tests := []struct {
		expr string
		resp Response
		want bool
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
		expr string
		resp Response
		want bool
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

// ── body_contains ────────────────────────────────────────────────────────────

func TestEvaluator_BodyContains(t *testing.T) {
	body := []byte(`{"status":"ok","token":"abc123"}`)
	tests := []struct {
		expr string
		want bool
	}{
		{`body_contains "ok"`, true},
		{`body_contains "abc123"`, true},
		{`body_contains "missing"`, false},
		{`body_contains ""`, true}, // empty needle always matches
	}
	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			e := NewEvaluator([]string{tc.expr})
			_, passed, failed := e.Evaluate(Response{Body: body})
			if tc.want && passed != 1 {
				t.Errorf("expected pass for %q", tc.expr)
			}
			if !tc.want && failed != 1 {
				t.Errorf("expected fail for %q", tc.expr)
			}
		})
	}
}

func TestEvaluator_BodyContains_EmptyBody(t *testing.T) {
	e := NewEvaluator([]string{`body_contains "something"`})
	_, _, failed := e.Evaluate(Response{Body: nil})
	if failed != 1 {
		t.Errorf("expected fail when body is nil and needle is non-empty")
	}
}

// ── body_matches ─────────────────────────────────────────────────────────────

func TestEvaluator_BodyMatches(t *testing.T) {
	body := []byte(`{"id":42,"name":"Alice"}`)
	tests := []struct {
		expr string
		want bool
	}{
		{`body_matches "\"id\":[0-9]+"`, true},
		{`body_matches "\"name\":\"[A-Z][a-z]+"`, true},
		{`body_matches "^\\{.*\\}$"`, true},
		{`body_matches "\"missing\""`, false},
	}
	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			e := NewEvaluator([]string{tc.expr})
			_, passed, failed := e.Evaluate(Response{Body: body})
			if tc.want && passed != 1 {
				t.Errorf("expected pass for %q", tc.expr)
			}
			if !tc.want && failed != 1 {
				t.Errorf("expected fail for %q", tc.expr)
			}
		})
	}
}

func TestEvaluator_BodyMatches_InvalidRegex(t *testing.T) {
	e := NewEvaluator([]string{`body_matches "[invalid"`})
	_, _, failed := e.Evaluate(Response{Body: []byte("anything")})
	if failed != 1 {
		t.Errorf("expected fail for invalid regex")
	}
}

// ── json_path ─────────────────────────────────────────────────────────────────

func TestEvaluator_JSONPath_NumericComparison(t *testing.T) {
	body := []byte(`{"user":{"id":42},"count":10}`)
	tests := []struct {
		expr string
		want bool
	}{
		{`json_path "$.user.id" == 42`, true},
		{`json_path "$.user.id" != 99`, true},
		{`json_path "$.user.id" > 40`, true},
		{`json_path "$.user.id" < 40`, false},
		{`json_path "$.count" >= 10`, true},
		{`json_path "$.count" <= 9`, false},
	}
	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			e := NewEvaluator([]string{tc.expr})
			_, passed, failed := e.Evaluate(Response{Body: body})
			if tc.want && passed != 1 {
				t.Errorf("expected pass for %q", tc.expr)
			}
			if !tc.want && failed != 1 {
				t.Errorf("expected fail for %q", tc.expr)
			}
		})
	}
}

func TestEvaluator_JSONPath_StringComparison(t *testing.T) {
	body := []byte(`{"status":"ok","env":"prod"}`)
	tests := []struct {
		expr string
		want bool
	}{
		{`json_path "$.status" == "ok"`, true},
		{`json_path "$.status" != "error"`, true},
		{`json_path "$.env" == "staging"`, false},
	}
	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			e := NewEvaluator([]string{tc.expr})
			_, passed, failed := e.Evaluate(Response{Body: body})
			if tc.want && passed != 1 {
				t.Errorf("expected pass for %q", tc.expr)
			}
			if !tc.want && failed != 1 {
				t.Errorf("expected fail for %q", tc.expr)
			}
		})
	}
}

func TestEvaluator_JSONPath_InvalidJSON(t *testing.T) {
	e := NewEvaluator([]string{`json_path "$.key" == 1`})
	_, _, failed := e.Evaluate(Response{Body: []byte("not json")})
	if failed != 1 {
		t.Errorf("expected fail for invalid JSON body")
	}
}

func TestEvaluator_JSONPath_MissingKey(t *testing.T) {
	e := NewEvaluator([]string{`json_path "$.missing" == 1`})
	_, _, failed := e.Evaluate(Response{Body: []byte(`{"key":1}`)})
	if failed != 1 {
		t.Errorf("expected fail for missing JSON key")
	}
}

// ── header ────────────────────────────────────────────────────────────────────

func TestEvaluator_Header_StringComparison(t *testing.T) {
	hdrs := map[string]string{"Content-Type": "application/json", "X-Custom": "hello"}
	tests := []struct {
		expr string
		want bool
	}{
		{`header "Content-Type" == "application/json"`, true},
		{`header "Content-Type" != "text/html"`, true},
		{`header "content-type" == "application/json"`, true}, // case-insensitive
		{`header "X-Custom" == "world"`, false},
		{`header "X-Missing" == "value"`, false}, // missing header → empty string
	}
	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			e := NewEvaluator([]string{tc.expr})
			_, passed, failed := e.Evaluate(Response{Headers: hdrs})
			if tc.want && passed != 1 {
				t.Errorf("expected pass for %q", tc.expr)
			}
			if !tc.want && failed != 1 {
				t.Errorf("expected fail for %q", tc.expr)
			}
		})
	}
}

func TestEvaluator_Header_NumericComparison(t *testing.T) {
	hdrs := map[string]string{"X-Rate-Limit-Remaining": "42"}
	tests := []struct {
		expr string
		want bool
	}{
		{`header "X-Rate-Limit-Remaining" > 0`, true},
		{`header "X-Rate-Limit-Remaining" == 42`, true},
		{`header "X-Rate-Limit-Remaining" < 10`, false},
	}
	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			e := NewEvaluator([]string{tc.expr})
			_, passed, failed := e.Evaluate(Response{Headers: hdrs})
			if tc.want && passed != 1 {
				t.Errorf("expected pass for %q", tc.expr)
			}
			if !tc.want && failed != 1 {
				t.Errorf("expected fail for %q", tc.expr)
			}
		})
	}
}

// ── NeedsBody / NeedsHeaders ──────────────────────────────────────────────────

func TestNeedsBody(t *testing.T) {
	if !NeedsBody([]string{`body_contains "x"`}) {
		t.Error("expected NeedsBody true for body_contains")
	}
	if !NeedsBody([]string{`body_matches "x"`}) {
		t.Error("expected NeedsBody true for body_matches")
	}
	if !NeedsBody([]string{`json_path "$.x" == 1`}) {
		t.Error("expected NeedsBody true for json_path")
	}
	if NeedsBody([]string{"status == 200", "duration < 500"}) {
		t.Error("expected NeedsBody false for status/duration checks")
	}
}

func TestNeedsHeaders(t *testing.T) {
	if !NeedsHeaders([]string{`header "Content-Type" == "application/json"`}) {
		t.Error("expected NeedsHeaders true for header check")
	}
	if NeedsHeaders([]string{"status == 200", `body_contains "x"`}) {
		t.Error("expected NeedsHeaders false for non-header checks")
	}
}
