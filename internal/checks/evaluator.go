// Package checks provides check expression parsing and evaluation for bacot.
package checks

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// CheckResult records the outcome of evaluating a single check expression.
type CheckResult struct {
	Expression string
	Passed     bool
}

// Response holds the data available to check expressions.
type Response struct {
	StatusCode int
	DurationMs float64
	Body       []byte
	Headers    map[string]string
}

// Evaluator parses and evaluates check expressions against an HTTP response.
type Evaluator struct {
	expressions []string
}

// NewEvaluator creates a new Evaluator for the given check expressions.
func NewEvaluator(expressions []string) *Evaluator {
	return &Evaluator{expressions: expressions}
}

// Evaluate runs all checks against the given response, returning individual results.
// It also returns the total pass and fail counts.
func (e *Evaluator) Evaluate(resp Response) ([]CheckResult, int, int) {
	results := make([]CheckResult, 0, len(e.expressions))
	passed, failed := 0, 0
	for _, expr := range e.expressions {
		ok, err := evalExpression(expr, resp)
		if err != nil {
			ok = false
		}
		results = append(results, CheckResult{Expression: expr, Passed: ok})
		if ok {
			passed++
		} else {
			failed++
		}
	}
	return results, passed, failed
}

// NeedsBody returns true if any expression requires access to the response body.
// Used by the VU to decide whether to capture the response body.
func NeedsBody(expressions []string) bool {
	for _, expr := range expressions {
		trimmed := strings.TrimSpace(expr)
		if strings.HasPrefix(trimmed, "body_contains") ||
			strings.HasPrefix(trimmed, "body_matches") ||
			strings.HasPrefix(trimmed, "json_path") {
			return true
		}
	}
	return false
}

// NeedsHeaders returns true if any expression requires access to response headers.
func NeedsHeaders(expressions []string) bool {
	for _, expr := range expressions {
		if strings.HasPrefix(strings.TrimSpace(expr), "header ") {
			return true
		}
	}
	return false
}

// evalExpression parses a check expression.
//
// Supported forms:
//
//	status <op> <int>                     — HTTP status code comparison
//	duration <op> <float>                 — Total duration in ms
//	body_contains "<substring>"           — Response body substring match
//	body_matches "<regex>"                — Response body regex match
//	json_path "<path>" <op> <value>       — JSONPath-style value comparison
//	header "<name>" <op> "<value>"        — Response header comparison
//
// Supported operators for numeric comparisons: ==, !=, <, <=, >, >=
// Supported operators for string comparisons:  ==, !=
func evalExpression(expr string, resp Response) (bool, error) {
	expr = strings.TrimSpace(expr)

	switch {
	case strings.HasPrefix(expr, "body_contains "):
		return evalBodyContains(expr, resp)
	case strings.HasPrefix(expr, "body_matches "):
		return evalBodyMatches(expr, resp)
	case strings.HasPrefix(expr, "json_path "):
		return evalJSONPath(expr, resp)
	case strings.HasPrefix(expr, "header "):
		return evalHeader(expr, resp)
	default:
		return evalNumericField(expr, resp)
	}
}

// evalNumericField handles: status <op> <int>  and  duration <op> <float>
func evalNumericField(expr string, resp Response) (bool, error) {
	var field, op, rawValue string
	for _, operator := range []string{"==", "!=", "<=", ">=", "<", ">"} {
		if idx := strings.Index(expr, operator); idx != -1 {
			field = strings.TrimSpace(expr[:idx])
			op = operator
			rawValue = strings.TrimSpace(expr[idx+len(operator):])
			break
		}
	}

	if op == "" {
		return false, fmt.Errorf("invalid check expression: %q", expr)
	}

	switch field {
	case "status":
		target, err := strconv.Atoi(rawValue)
		if err != nil {
			return false, fmt.Errorf("invalid status value in check %q: %w", expr, err)
		}
		return compareFloat(float64(resp.StatusCode), op, float64(target)), nil

	case "duration":
		target, err := strconv.ParseFloat(rawValue, 64)
		if err != nil {
			return false, fmt.Errorf("invalid duration value in check %q: %w", expr, err)
		}
		return compareFloat(resp.DurationMs, op, target), nil

	default:
		return false, fmt.Errorf("unknown check field %q in expression %q", field, expr)
	}
}

// evalBodyContains checks whether the response body contains a literal substring.
//
//	body_contains "some text"
//	body_contains some_text          (unquoted also accepted)
func evalBodyContains(expr string, resp Response) (bool, error) {
	// Strip the "body_contains " prefix then unquote the remainder.
	rest := strings.TrimSpace(expr[len("body_contains "):])
	needle, err := unquoteOrRaw(rest)
	if err != nil {
		return false, fmt.Errorf("body_contains: %w", err)
	}
	return strings.Contains(string(resp.Body), needle), nil
}

// evalBodyMatches checks whether the response body matches a regular expression.
//
//	body_matches "^{.*}$"
func evalBodyMatches(expr string, resp Response) (bool, error) {
	rest := strings.TrimSpace(expr[len("body_matches "):])
	pattern, err := unquoteOrRaw(rest)
	if err != nil {
		return false, fmt.Errorf("body_matches: %w", err)
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, fmt.Errorf("body_matches: invalid regex %q: %w", pattern, err)
	}
	return re.Match(resp.Body), nil
}

// evalJSONPath evaluates a simple dot-notation JSON path expression.
//
//	json_path "$.user.id" == 42
//	json_path "$.status" == "ok"
//	json_path "$.count" > 0
//
// Supported path syntax: $.<key>.<key>... (dot-notation only, no array indexes).
func evalJSONPath(expr string, resp Response) (bool, error) {
	// Strip "json_path " prefix.
	rest := strings.TrimSpace(expr[len("json_path "):])

	// Extract quoted path.
	if len(rest) == 0 || rest[0] != '"' {
		return false, fmt.Errorf("json_path: path must be a quoted string, got %q", rest)
	}
	closeQ := strings.Index(rest[1:], "\"")
	if closeQ < 0 {
		return false, fmt.Errorf("json_path: unterminated quoted path in %q", expr)
	}
	path := rest[1 : closeQ+1]
	afterPath := strings.TrimSpace(rest[closeQ+2:])

	// Parse operator and value.
	var op, rawValue string
	for _, operator := range []string{"==", "!=", "<=", ">=", "<", ">"} {
		if strings.HasPrefix(afterPath, operator) {
			op = operator
			rawValue = strings.TrimSpace(afterPath[len(operator):])
			break
		}
	}
	if op == "" {
		return false, fmt.Errorf("json_path: missing operator in %q", expr)
	}

	// Resolve path in JSON body.
	actual, err := resolveJSONPath(path, resp.Body)
	if err != nil {
		return false, fmt.Errorf("json_path: %w", err)
	}

	// Try numeric comparison first, then string.
	targetFloat, numErr := strconv.ParseFloat(rawValue, 64)
	if numErr == nil {
		actualFloat, floatErr := toFloat(actual)
		if floatErr != nil {
			return false, fmt.Errorf("json_path: value at %q is not numeric (%T)", path, actual)
		}
		return compareFloat(actualFloat, op, targetFloat), nil
	}

	// String comparison — unquote expected value.
	targetStr, _ := unquoteOrRaw(rawValue)
	actualStr := fmt.Sprintf("%v", actual)
	switch op {
	case "==":
		return actualStr == targetStr, nil
	case "!=":
		return actualStr != targetStr, nil
	default:
		return false, fmt.Errorf("json_path: operator %q not supported for string values", op)
	}
}

// evalHeader checks a response header value.
//
//	header "Content-Type" == "application/json"
//	header "X-Rate-Limit-Remaining" > 0
func evalHeader(expr string, resp Response) (bool, error) {
	rest := strings.TrimSpace(expr[len("header "):])

	if len(rest) == 0 || rest[0] != '"' {
		return false, fmt.Errorf("header: name must be a quoted string in %q", expr)
	}
	closeQ := strings.Index(rest[1:], "\"")
	if closeQ < 0 {
		return false, fmt.Errorf("header: unterminated quoted name in %q", expr)
	}
	headerName := rest[1 : closeQ+1]
	afterName := strings.TrimSpace(rest[closeQ+2:])

	var op, rawValue string
	for _, operator := range []string{"==", "!=", "<=", ">=", "<", ">"} {
		if strings.HasPrefix(afterName, operator) {
			op = operator
			rawValue = strings.TrimSpace(afterName[len(operator):])
			break
		}
	}
	if op == "" {
		return false, fmt.Errorf("header: missing operator in %q", expr)
	}

	// Canonical header lookup (case-insensitive via map stored with canonical keys).
	actualVal := ""
	for k, v := range resp.Headers {
		if strings.EqualFold(k, headerName) {
			actualVal = v
			break
		}
	}

	// Numeric comparison.
	targetFloat, numErr := strconv.ParseFloat(rawValue, 64)
	if numErr == nil {
		actualFloat, floatErr := strconv.ParseFloat(actualVal, 64)
		if floatErr != nil {
			return false, fmt.Errorf("header: value of %q is not numeric", headerName)
		}
		return compareFloat(actualFloat, op, targetFloat), nil
	}

	// String comparison.
	targetStr, _ := unquoteOrRaw(rawValue)
	switch op {
	case "==":
		return actualVal == targetStr, nil
	case "!=":
		return actualVal != targetStr, nil
	default:
		return false, fmt.Errorf("header: operator %q not supported for string comparison", op)
	}
}

// resolveJSONPath resolves a simple dot-notation path like "$.user.id" in raw JSON.
func resolveJSONPath(path string, body []byte) (interface{}, error) {
	if len(body) == 0 {
		return nil, fmt.Errorf("response body is empty")
	}

	var root interface{}
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, fmt.Errorf("body is not valid JSON: %w", err)
	}

	// Strip leading "$." or "$"
	path = strings.TrimPrefix(path, "$")
	path = strings.TrimPrefix(path, ".")

	if path == "" {
		return root, nil
	}

	parts := strings.Split(path, ".")
	cur := root
	for _, part := range parts {
		m, ok := cur.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("path segment %q: not an object at this level", part)
		}
		val, exists := m[part]
		if !exists {
			return nil, fmt.Errorf("key %q not found in JSON", part)
		}
		cur = val
	}
	return cur, nil
}

// toFloat converts a JSON number/bool/string to float64.
func toFloat(v interface{}) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case int:
		return float64(val), nil
	case bool:
		if val {
			return 1, nil
		}
		return 0, nil
	case string:
		return strconv.ParseFloat(val, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to number", v)
	}
}

// unquoteOrRaw strips surrounding double-quotes if present, otherwise returns as-is.
func unquoteOrRaw(s string) (string, error) {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		unquoted, err := strconv.Unquote(s)
		if err != nil {
			return "", fmt.Errorf("invalid quoted string %q: %w", s, err)
		}
		return unquoted, nil
	}
	return s, nil
}

// compareFloat performs a numeric comparison using the given operator.
func compareFloat(actual float64, op string, target float64) bool {
	switch op {
	case "==":
		return actual == target
	case "!=":
		return actual != target
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
