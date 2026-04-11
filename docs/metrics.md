# Metrics Reference

bacot collects metrics per-request in a thread-safe, memory-efficient way using HDR histograms (no per-request storage — all data is aggregated on the fly).

---

## Summary metrics

These are printed in the final summary and included in all report formats.

### `http_reqs`

Total number of HTTP requests completed across all VUs and scenarios.

```
http_reqs...............: 12,450   245.3/s
```

The right-hand value is the **average RPS** over the entire test duration.

---

### `http_req_duration`

Full end-to-end request latency from the moment the request was sent until the full response body was read. Reported as percentiles.

```
http_req_duration.......: avg=95ms   min=12ms   p90=280ms   p95=310ms   p99=490ms
```

| Stat | Description |
|---|---|
| `avg` | Mean (arithmetic average) |
| `min` | Minimum observed latency |
| `p50` | 50th percentile (half of requests faster than this) |
| `p75` | 75th percentile |
| `p90` | 90th percentile |
| `p95` | 95th percentile — primary SLA indicator |
| `p99` | 99th percentile — tail latency |
| `max` | Maximum observed latency |

**Implementation:** bacot uses [HDR histogram](https://github.com/HdrHistogram/hdrhistogram-go) for percentile calculation — accurate to 3 significant figures across a 1µs–60s range, without storing individual samples.

---

### `http_req_failed`

The fraction of requests considered failed.

```
http_req_failed.........: 0.09%   (12 of 12,450)
```

**A request is counted as failed if:**
- A network error occurred (timeout, connection refused, DNS failure)
- The HTTP response code is `5xx`
- No response was received (zero status code)

> **Note:** `4xx` responses (e.g. 401, 404) are **not** counted as failures by default. Use [checks](checks.md) with `status < 400` if you want to catch them.

---

### `data_sent` / `data_received`

Total bytes transferred across all requests.

```
data_sent...............: 4.2 MB   82 kB/s
data_received...........: 18.7 MB  368 kB/s
```

- **`data_sent`**: request body bytes (headers are not counted)
- **`data_received`**: response body bytes read

---

### `checks`

Aggregated pass/fail count for all [check expressions](checks.md).

```
checks..................: 99.9%   ✓ 24887 / ✗ 13
```

---

## Per-scenario metrics

Every metric above is also broken down per scenario and shown in the dashboard and reports.

**Dashboard:**
```
  Scenarios
  ├─ GET homepage     8,715 reqs   p95: 298ms   ✓ 99.9%
  └─ POST login       3,735 reqs   p95: 345ms   ✓ 99.7%
```

**JSON report — per scenario:**
```json
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
    "p95_ms": 298.0,
    "p99_ms": 412.0
  }
}
```

---

## Status code distribution

bacot tracks the count of every HTTP status code returned during the test.

```
  Status Codes
  ├─ 200: 12,438  (99.9%)
  ├─ 401: 10      (0.08%)
  └─ 500: 2       (0.02%)
```

---

## Time-series data

bacot accumulates **per-second buckets** of:

| Field | Description |
|---|---|
| `requests` | Requests completed in this second |
| `failures` | Failed requests in this second |
| `rps` | Requests per second for this bucket |
| `error_rate_pct` | Error percentage for this second |
| `p95_ms` | p95 latency for requests in this second |
| `avg_ms` | Average latency for requests in this second |
| `bytes_sent` | Bytes sent in this second |
| `bytes_received` | Bytes received in this second |

Time-series data is included in the JSON report and drives the charts in the HTML report.

---

## Detailed timing breakdown

For each request, bacot records the following timing components via `net/http/httptrace`:

| Field | Description |
|---|---|
| `DurationMicros` | Total request time (connect → last byte) |
| `ConnectMicros` | DNS lookup + TCP connect time |
| `TLSMicros` | TLS handshake time |
| `TTFBMicros` | Time to first byte (TTFB) |

> These detailed timings are not currently surfaced in terminal/reports but are collected per-request and available for future extensions.

---

## Error classification

Network errors are classified into types:

| Type | Cause |
|---|---|
| `timeout` | `context.DeadlineExceeded` or `context.Canceled` |
| `dns` | DNS resolution failure |
| `refused` | TCP connection refused |
| `network` | Other `net.OpError` |
| `http` | Other HTTP-level errors |
| `request` | Request construction error |
| `io` | Body file read error |
