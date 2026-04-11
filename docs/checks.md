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
```

---

## Fields

| Field | Description | Unit |
|---|---|---|
| `status` | HTTP response status code | integer |
| `duration` | Total request duration (TTFB + transfer) | milliseconds |

---

## Operators

| Operator | Meaning |
|---|---|
| `==` | Equal to |
| `!=` | Not equal to |
| `<` | Less than |
| `<=` | Less than or equal to |
| `>` | Greater than |
| `>=` | Greater than or equal to |

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

### Combining status and latency

```yaml
checks:
  - status == 200
  - status != 502
  - duration < 1000
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

---

## Invalid expression behaviour

| Expression | Outcome |
|---|---|
| `status == 200` | Evaluated normally |
| `badfield == 200` | Counts as **failed** (unknown field) |
| `status` | Counts as **failed** (no operator) |
| `duration < abc` | Counts as **failed** (non-numeric value) |
