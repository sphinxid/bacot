# Output & Reports

bacot produces three types of output: the **live terminal dashboard**, the **final summary**, and optional **file reports** (JSON or HTML).

---

## Live terminal dashboard

While the test runs, bacot renders an in-place dashboard to `stderr` that refreshes every **500ms**.

```
──────────────────────────────────────────────────────────
  Stage 2/3  [████████████░░░░░░░░]  45s / 50s
  VUs: 50 active

  Requests      12,450   RPS: 245.3/s
  Failures      12       (0.09%)
  Duration      p50: 82ms   p95: 310ms   p99: 490ms
  Data          ↑ 4.2 MB   ↓ 18.7 MB

  Checks        ✓ 24,887   ✗ 13

──────────────────────────────────────────────────────────
  Scenarios
  ├─ GET homepage     8,715 reqs   p95: 298ms   ✓ 99.9%
  └─ POST login       3,735 reqs   p95: 345ms   ✓ 99.7%
```

### Dashboard sections

| Section | Description |
|---|---|
| **Stage progress bar** | Current stage index, animated progress bar, elapsed / total time |
| **VUs** | Number of active virtual users right now |
| **Requests** | Total completed requests + current average RPS |
| **Failures** | Count and percentage of failed requests |
| **Duration** | Rolling p50 / p95 / p99 latency |
| **Data** | Total bytes sent (↑) and received (↓) |
| **Checks** | Running total of passed ✓ and failed ✗ check assertions |
| **Scenarios** | Per-scenario request count, p95 latency, and check pass rate |

### Suppress the dashboard

Use `--quiet` to skip the live dashboard and only print the final summary:

```bash
bacot run script.yaml --quiet
```

### Disable colors

```bash
bacot run script.yaml --no-color
```

---

## Final summary

After all stages complete (or after a graceful interrupt), bacot prints a summary to `stdout`.

```
────────────────────── SUMMARY ──────────────────────

  http_reqs...............: 12,450   245.3/s
  http_req_duration.......: avg=95ms   min=12ms   p90=280ms   p95=310ms   p99=490ms
  http_req_failed.........: 0.09%   (12 of 12,450)
  data_sent...............: 4.2 MB   82 kB/s
  data_received...........: 18.7 MB  368 kB/s

  checks..................: 99.9%   ✓ 24887 / ✗ 13

  Thresholds
  ✓ http_req_duration_p95 < 500ms      (actual: 310ms)
  ✓ http_req_failed < 1%               (actual: 0.09%)
  ✓ http_req_duration_avg < 200ms      (actual: 95ms)

  Status Codes
  ├─ 200: 12,438  (99.9%)
  ├─ 401: 10      (0.08%)
  └─ 500: 2       (0.02%)

──────────────────────────────────────────────────────────
```

> The summary is always printed — even when `--quiet` is set or when the test is interrupted with Ctrl+C.

---

## JSON report

Export full structured results to a JSON file:

```bash
bacot run script.yaml --output json=result.json
```

### Top-level structure

```json
{
  "test_name": "API Load Test",
  "duration": "50.012s",
  "started_at": "2026-04-10T13:45:00+07:00",
  "summary": { ... },
  "scenarios": [ ... ],
  "thresholds": [ ... ],
  "status_codes": { "200": 12438, "401": 10, "500": 2 },
  "time_series": [ ... ]
}
```

### `summary` object

```json
{
  "total_requests": 12450,
  "total_failures": 12,
  "failure_rate_pct": 0.09,
  "rps": 245.3,
  "bytes_sent": 4404019,
  "bytes_received": 19609692,
  "checks_passed": 24887,
  "checks_failed": 13,
  "latency": {
    "min_ms": 12.0,
    "max_ms": 820.0,
    "avg_ms": 95.0,
    "p50_ms": 82.0,
    "p75_ms": 140.0,
    "p90_ms": 280.0,
    "p95_ms": 310.0,
    "p99_ms": 490.0
  }
}
```

### `scenarios` array

```json
[
  {
    "name": "GET homepage",
    "requests": 8715,
    "failures": 9,
    "failure_rate_pct": 0.103,
    "bytes_sent": 0,
    "bytes_received": 3547230,
    "checks_passed": 17421,
    "checks_failed": 3,
    "latency": {
      "avg_ms": 87.4,
      "p50_ms": 72.0,
      "p75_ms": 120.0,
      "p90_ms": 220.0,
      "p95_ms": 298.0,
      "p99_ms": 412.0
    }
  }
]
```

### `thresholds` array

```json
[
  {
    "name": "http_req_duration_p95",
    "expression": "< 500ms",
    "passed": true,
    "actual": 310.0,
    "threshold": 500.0,
    "unit": "ms"
  }
]
```

### `time_series` array

One entry per second of the test:

```json
[
  {
    "timestamp": 1744267500,
    "requests": 245,
    "failures": 0,
    "rps": 245.0,
    "error_rate_pct": 0.0,
    "p95_ms": 298.0,
    "avg_ms": 87.4,
    "bytes_sent": 0,
    "bytes_received": 99840
  }
]
```

---

## HTML report

Export a self-contained HTML file with interactive charts:

```bash
bacot run script.yaml --output html=report.html
```

Open `report.html` in any browser — no server required.

### Contents

| Section | Description |
|---|---|
| **Summary cards** | Total requests, failure rate, avg latency, p99 latency, data sent/received |
| **RPS over time** | Line chart of requests per second |
| **Latency over time (p95)** | Line chart of p95 latency by second |
| **Error rate over time** | Line chart of error percentage by second |
| **Latency distribution** | Bar chart of p50 / p75 / p90 / p95 / p99 |
| **Thresholds table** | Pass/fail badge, metric name, expression, actual vs threshold |
| **Scenarios table** | Per-scenario breakdown: requests, failures, latency percentiles |
| **Status codes table** | Count and percentage per HTTP status code |

Charts use [Chart.js](https://www.chartjs.org/) loaded from CDN. Tables and summary cards render fully offline.

---

## Using both outputs together

```bash
bacot run script.yaml \
  --output json=results/result.json \
  --output html=results/report.html
```

Both flags can be specified in the same command. The `--output` flag accepts multiple values.

---

## Redirecting output

The **live dashboard** and informational messages (including report paths) are written to `stderr`.  
The **summary** is written to `stdout`.

This means you can capture the summary separately:

```bash
# Capture summary to a file while dashboard still shows in terminal
bacot run script.yaml --quiet > summary.txt
```

Or silence all non-summary output:

```bash
bacot run script.yaml --quiet 2>/dev/null
```
