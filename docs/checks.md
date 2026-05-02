# Checks

Checks are **per-response assertions** defined inside a scenario. bacot evaluates them against every HTTP response and tracks pass/fail counts across the entire test.

---

## Syntax

```
<field> <operator> <value>
```

```yaml
checks:
  - status == 200
  - status != 500
  - status < 400
  - status >= 200
  - duration < 500
  - duration <= 1000
  - body_contains "access_token"
  - body_matches "\"id\":[0-9]+"
  - json_path "$.status" == "ok"
  - json_path "$.count" > 0
  - header "Content-Type" == "application/json"
```

---

## Fields

### Numeric fields

| Field | Description | Unit |
|---|---|---|
| `status` | HTTP response status code | integer |
| `duration` | Total request duration (TTFB + transfer) | milliseconds |

**Operators:** `==`, `!=`, `<`, `<=`, `>`, `>=`

---

### Body fields

These fields require bacot to read and buffer the response body. They are automatically detected — no extra configuration is needed.

#### `body_contains "<substring>"`

Passes if the response body contains the given literal string.

```yaml
checks:
  - body_contains "access_token"
  - body_contains "\"success\":true"
```

#### `body_matches "<regex>"`

Passes if the response body matches the given regular expression (Go `regexp` syntax).

```yaml
checks:
  - body_matches "\"id\":[0-9]+"
  - body_matches "^\\{.*\\}$"
```

#### `json_path "<path>" <op> <value>`

Evaluates a dot-notation JSON path against the parsed response body and compares the result.

- **Path syntax:** `$.<key>.<key>...` (dot notation, no array indexes)
- **Numeric comparison:** if `<value>` parses as a number, a numeric comparison is performed
- **String comparison:** otherwise a string comparison is performed (operators `==` and `!=` only)

```yaml
checks:
  - json_path "$.status" == "ok"
  - json_path "$.user.id" > 0
  - json_path "$.count" >= 10
  - json_path "$.error" != "null"
```

---

### Header field

#### `header "<name>" <op> <value>`

Checks a response header value. Header name matching is **case-insensitive**.

- Numeric comparison if `<value>` parses as a number
- String comparison otherwise (`==` and `!=` only)

```yaml
checks:
  - header "Content-Type" == "application/json"
  - header "X-Rate-Limit-Remaining" > 0
  - header "Cache-Control" != "no-store"
```

---

## Operators

| Operator | Meaning | Applicable to |
|---|---|---|
| `==` | Equal to | numeric, string |
| `!=` | Not equal to | numeric, string |
| `<` | Less than | numeric only |
| `<=` | Less than or equal to | numeric only |
| `>` | Greater than | numeric only |
| `>=` | Greater than or equal to | numeric only |

---

## Examples

### Assert successful response

```yaml
checks:
  - status == 200
```

### Assert not a server error

```yaml
checks:
  - status != 500
  - status < 500
```

### Assert 2xx range

```yaml
checks:
  - status >= 200
  - status < 300
```

### Assert latency under SLA

```yaml
checks:
  - duration < 500    # must complete in under 500ms
```

### Assert response body contains a token

```yaml
checks:
  - status == 200
  - body_contains "access_token"
```

### Assert JSON response structure

```yaml
checks:
  - status == 200
  - json_path "$.status" == "ok"
  - json_path "$.user.id" > 0
```

### Assert response header

```yaml
checks:
  - header "Content-Type" == "application/json"
  - header "X-Rate-Limit-Remaining" > 0
```

### Combining all check types

```yaml
checks:
  - status == 200
  - duration < 1000
  - body_contains "data"
  - json_path "$.success" == true
  - header "Content-Type" == "application/json"
```

---

## Check results in the dashboard

During the test the dashboard shows a running total:

```
  Checks        ✓ 24,887   ✗ 13
```

In the final summary:

```
  checks..................: 99.9%   ✓ 24887 / ✗ 13
```

Per-scenario check pass rate is shown in the scenario breakdown:

```
  Scenarios
  ├─ GET homepage     8,715 reqs   p95: 298ms   ✓ 99.9%
  └─ POST login       3,735 reqs   p95: 345ms   ✓ 99.7%
```

---

## Check results in reports

In the JSON report, check counts are included per scenario:

```json
{
  "scenarios": [
    {
      "name": "GET homepage",
      "checks_passed": 8712,
      "checks_failed": 3
    }
  ]
}
```

---

## Important notes

- A check **failure does not stop the test** — it only increments the fail counter.
- Checks are evaluated in order; all checks run regardless of earlier failures.
- An invalid check expression (e.g. unrecognised field, missing operator) counts as a **failed** check.
- Check counts contribute to `checks_passed` / `checks_failed` in the JSON report but do **not** directly influence threshold evaluation (thresholds operate on separate metrics like `http_req_failed`).
- `body_contains`, `body_matches`, and `json_path` checks require buffering the response body in memory. bacot does this automatically only for scenarios that use these check types.

---

## Invalid expression behaviour

| Expression | Outcome |
|---|---|
| `status == 200` | Evaluated normally |
| `badfield == 200` | Counts as **failed** (unknown field) |
| `status` | Counts as **failed** (no operator) |
| `duration < abc` | Counts as **failed** (non-numeric value) |
| `json_path "$.x" == 1` (body not JSON) | Counts as **failed** |
| `body_matches "[invalid"` | Counts as **failed** (invalid regex) |
