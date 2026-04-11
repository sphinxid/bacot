// Package checks provides check expression parsing and evaluation for bacot.
package checks

import (
	"fmt"
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

// evalExpression parses a check expression in the form:
//
//	<field> <op> <value>
//
// Supported fields: status, duration
// Supported operators: ==, !=, <, <=, >, >=
func evalExpression(expr string, resp Response) (bool, error) {
	expr = strings.TrimSpace(expr)

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
		return compare(float64(resp.StatusCode), op, float64(target)), nil

	case "duration":
		target, err := strconv.ParseFloat(rawValue, 64)
		if err != nil {
			return false, fmt.Errorf("invalid duration value in check %q: %w", expr, err)
		}
		return compare(resp.DurationMs, op, target), nil

	default:
		return false, fmt.Errorf("unknown check field %q in expression %q", field, expr)
	}
}

// compare performs a numeric comparison using the given operator.
func compare(actual float64, op string, target float64) bool {
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
