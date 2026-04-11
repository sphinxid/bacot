# bacot

A command-line tool for load testing HTTP APIs, built with Go — similar to k6.

## Features

- **Goroutine-based VU engine** — one goroutine per virtual user, scales to 500+ VUs
- **Staged load profiles** — ramp up/down VUs using `stages` in YAML
- **HTTP/1.1 & HTTP/2** — keep-alive, custom headers, request bodies, TLS skip verify
- **Accurate percentiles** — HDR histogram for p50/p90/p95/p99 latency
- **Check engine** — per-response assertions (`status == 200`, `duration < 500`)
- **Threshold engine** — post-test pass/fail criteria with exit code support
- **Live dashboard** — ANSI terminal dashboard refreshed every 500ms
- **Reports** — JSON and self-contained HTML reports with Chart.js charts
- **Signal handling** — SIGINT/SIGTERM triggers graceful shutdown with partial summary

---

## Installation

### From source

```bash
go install github.com/sphinxid/bacot@latest
```

### Build from repo

```bash
git clone https://github.com/sphinxid/bacot
cd bacot
make build
./bin/bacot version
```

### Pre-built binaries

```bash
make release
# Outputs: bin/bacot-linux-amd64, bin/bacot-darwin-arm64, etc.
```

---

## Quick Start

### Inline single-URL test

```bash
bacot run --url https://httpbin.org/get --vus 10 --duration 30s
```

### YAML script test

```bash
bacot run examples/api-load-test.yaml
```

### With reports

```bash
bacot run script.yaml --output json=result.json --output html=report.html
```

### With full per-scenario breakdown on stdout

```bash
bacot run script.yaml --full-report
```

---

## YAML Script Format

```yaml
name: "API Load Test"
target: "https://api.example.com"

# HTTP options (optional — defaults shown)
timeout: 30s
connect_timeout: 10s
insecure: false
max_redirects: 10
http2: false

stages:
  - duration: 10s
    vus: 5
  - duration: 30s
    vus: 50
  - duration: 10s
    vus: 5

scenarios:
  - name: "GET homepage"
    method: GET
    path: /
    weight: 70         # % of traffic (relative weight)
    headers:
      Accept: application/json
    checks:
      - status == 200
      - duration < 500

  - name: "POST login"
    method: POST
    path: /auth/login
    weight: 30
    body: '{"username":"test","password":"test"}'
    headers:
      Content-Type: application/json
    checks:
      - status == 200
      - status != 500

thresholds:
  http_req_duration_p95: "< 500ms"
  http_req_failed: "< 1%"
  http_req_duration_avg: "< 200ms"
```

---

## CLI Reference

### `bacot run`

```
bacot run [script.yaml] [flags]

Flags:
  --url string             target URL for inline test
  --vus int                number of virtual users (default 1)
  --duration string        test duration, e.g. 30s, 1m (default "30s")
  --timeout string         HTTP request timeout (default "30s")
  --connect-timeout string TCP connect timeout (default "10s")
  --insecure               skip TLS certificate verification
  --max-redirects int      maximum redirects to follow (default 10)
  --http2                  enable HTTP/2

Global Flags:
  --output stringArray     output format, e.g. --output json=result.json
  --no-color               disable terminal colors
  --quiet                  suppress live dashboard
  --full-report            print full per-scenario metrics report after the summary
```

### `bacot version`

```
bacot version
```

---

## Check Syntax

Checks are per-response assertions defined in scenario YAML:

| Expression | Meaning |
|---|---|
| `status == 200` | Status code must equal 200 |
| `status != 500` | Status code must not equal 500 |
| `status < 400` | Status code must be below 400 |
| `status >= 200` | Status code must be >= 200 |
| `duration < 500` | Response time must be < 500ms |
| `duration <= 1000` | Response time must be <= 1000ms |

**Supported fields:** `status`, `duration` (in milliseconds)  
**Supported operators:** `==`, `!=`, `<`, `<=`, `>`, `>=`

---

## Threshold Syntax

Thresholds are post-test pass/fail criteria defined at the script level:

```yaml
thresholds:
  http_req_duration_p95: "< 500ms"
  http_req_duration_avg: "< 200ms"
  http_req_failed:       "< 1%"
  http_reqs:             ">= 1000"
```

**Supported metrics:**

| Metric | Description |
|---|---|
| `http_req_duration_p50` | 50th percentile latency |
| `http_req_duration_p75` | 75th percentile latency |
| `http_req_duration_p90` | 90th percentile latency |
| `http_req_duration_p95` | 95th percentile latency |
| `http_req_duration_p99` | 99th percentile latency |
| `http_req_duration_avg` | Mean latency |
| `http_req_duration_min` | Minimum latency |
| `http_req_duration_max` | Maximum latency |
| `http_req_failed` | Failure rate (use `%` suffix for percentage) |
| `http_reqs` | Total request count |
| `http_reqs_per_sec` | Average RPS |

**Supported units:** `ms`, `s`, `%` (raw number if no unit)  
**Supported operators:** `<`, `<=`, `>`, `>=`

**Exit codes:** `0` = all thresholds pass, `1` = any threshold fails (or test error)

---

## Output Formats

### JSON report (`--output json=result.json`)

Full structured result including:
- Summary metrics (RPS, latency percentiles, failure rate, bytes)
- Per-scenario breakdown
- Threshold results
- Per-second timeseries (for charting)
- Status code distribution

### HTML report (`--output html=report.html`)

Self-contained single HTML file (no external dependencies needed) with:
- Summary stat cards
- RPS over time chart
- Latency (p95) over time chart
- Error rate over time chart
- Latency distribution bar chart
- Threshold pass/fail table
- Per-scenario breakdown table
- Status code distribution table

> Note: Charts use Chart.js loaded from CDN. For offline viewing, the page will still render tables and summary stats.

---

## Makefile Targets

```
make build    # build ./bin/bacot
make test     # go test ./... -race
make install  # go install
make release  # cross-compile for linux/darwin/windows
make clean    # remove bin/
make lint     # golangci-lint run
make tidy     # go mod tidy
```

---

## Architecture

```
bacot/
├── cmd/                    # CLI commands (Cobra)
│   ├── root.go
│   ├── run.go
│   └── version.go
├── internal/
│   ├── config/             # YAML parsing & validation
│   ├── engine/             # VU orchestrator + scheduler
│   ├── httpclient/         # HTTP client factory + request executor
│   ├── metrics/            # Thread-safe collector, HDR histogram, timeseries
│   ├── checks/             # Per-response check evaluator
│   ├── thresholds/         # Post-test threshold evaluator
│   ├── output/             # Terminal dashboard, summary, JSON, HTML
│   └── version/            # Build version info
├── examples/               # Example YAML scripts
├── main.go
├── go.mod
├── Makefile
└── README.md
```
