# Thresholds

Thresholds are **post-test pass/fail criteria** evaluated after all stages complete. If any threshold fails, bacot exits with code `1` — making it suitable for CI/CD pipelines.

---

## Syntax

```yaml
thresholds:
  <metric_name>: "<operator> <value><unit>"
```

```yaml
thresholds:
  http_req_duration_p95: "< 500ms"
  http_req_failed:       "< 1%"
  http_req_duration_avg: "< 200ms"
  http_reqs:             ">= 10000"
```

---

## Operators

| Operator | Meaning |
|---|---|
| `<` | Metric must be less than value |
| `<=` | Metric must be less than or equal to value |
| `>` | Metric must be greater than value |
| `>=` | Metric must be greater than or equal to value |

---

## Units

Append a unit suffix directly to the value (no space):

| Suffix | Meaning | Applies to |
|---|---|---|
| `ms` | milliseconds | Duration metrics |
| `s` | seconds | Duration metrics |
| `%` | percentage | `http_req_failed` |
| *(none)* | raw number | `http_reqs`, `http_reqs_per_sec` |

---

## Available metrics

### Latency metrics

| Metric name | Description |
|---|---|
| `http_req_duration_p50` | 50th percentile (median) latency |
| `http_req_duration_p75` | 75th percentile latency |
| `http_req_duration_p90` | 90th percentile latency |
| `http_req_duration_p95` | 95th percentile latency |
| `http_req_duration_p99` | 99th percentile latency |
| `http_req_duration_avg` | Mean (average) latency |
| `http_req_duration_min` | Minimum latency observed |
| `http_req_duration_max` | Maximum latency observed |

Latency values are stored internally in **microseconds** and converted to the unit you specify (`ms` or `s`). If no unit is given, the raw microsecond value is used.

### Error and traffic metrics

| Metric name | Description | Unit |
|---|---|---|
| `http_req_failed` | Fraction of failed requests | Use `%` suffix for percentage |
| `http_reqs` | Total number of requests completed | raw integer |
| `http_reqs_per_sec` | Average requests per second over the test | raw float |

---

## Examples

### Latency SLA

```yaml
thresholds:
  http_req_duration_p95: "< 500ms"   # 95th percentile under 500ms
  http_req_duration_p99: "< 1s"      # 99th percentile under 1 second
  http_req_duration_avg: "< 200ms"   # average under 200ms
```

### Error rate

```yaml
thresholds:
  http_req_failed: "< 1%"    # less than 1% failures
```

Using a fraction instead of percentage:

```yaml
thresholds:
  http_req_failed: "< 0.01"  # same as < 1% but without the % unit
```

> When `%` is **not** used, the value is compared against the raw failure rate (0.0–1.0). `< 0.01` means less than 1%.

### Minimum throughput

```yaml
thresholds:
  http_reqs:             ">= 5000"    # at least 5000 total requests
  http_reqs_per_sec:     ">= 100"     # at least 100 RPS average
```

### Combined SLA

```yaml
thresholds:
  http_req_duration_p95: "< 500ms"
  http_req_duration_avg: "< 200ms"
  http_req_failed:       "< 1%"
  http_reqs:             ">= 10000"
```

---

## Terminal output

Passing thresholds are shown in **green**, failing ones in **red**:

```
  Thresholds
  ✓ http_req_duration_p95 < 500ms      (actual: 310ms)
  ✓ http_req_failed < 1%               (actual: 0.09%)
  ✗ http_req_duration_avg < 200ms      (actual: 382ms)
```

---

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | All thresholds passed (or no thresholds defined) |
| `1` | One or more thresholds failed |

### CI/CD usage

```bash
#!/bin/bash
bin/bacot run perf-test.yaml --quiet
if [ $? -ne 0 ]; then
  echo "❌ Performance regression — thresholds failed"
  exit 1
fi
echo "✅ All performance thresholds passed"
```

---

## Threshold results in reports

### JSON report

```json
{
  "thresholds": [
    {
      "name": "http_req_duration_p95",
      "expression": "< 500ms",
      "passed": true,
      "actual": 310.0,
      "threshold": 500.0,
      "unit": "ms"
    },
    {
      "name": "http_req_failed",
      "expression": "< 1%",
      "passed": true,
      "actual": 0.09,
      "threshold": 1.0,
      "unit": "%"
    }
  ]
}
```

### HTML report

The HTML report includes a **Thresholds table** with pass/fail badges, actual values, and threshold values.

---

## Error handling

| Situation | Behaviour |
|---|---|
| Unknown metric name | Threshold treated as **failed** |
| Unparseable expression | Threshold treated as **failed** |
| No thresholds defined | Ignored; exit code is always `0` |
