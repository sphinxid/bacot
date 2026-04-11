# Getting Started

## Installation

### Build from source (recommended)

```bash
git clone https://github.com/sphinxid/bacot
cd bacot
make build
# Binary is at bin/bacot
```

Add to your PATH:
```bash
export PATH="$PATH:/path/to/bacot/bin"
```

### Go install

```bash
CGO_ENABLED=0 go install github.com/sphinxid/bacot@latest
```

### Pre-built cross-platform binaries

```bash
make release
# Produces:
#   bin/bacot-linux-amd64
#   bin/bacot-darwin-amd64
#   bin/bacot-darwin-arm64
#   bin/bacot-windows-amd64.exe
```

> **Note (macOS Tahoe / macOS 26+):** Build with `CGO_ENABLED=0` to avoid a known `dyld: missing LC_UUID` issue with Go 1.22.x on macOS 26. All `make` targets already include this flag.

---

## Verify installation

```bash
bin/bacot version
# bacot v1.0.0 (commit: abc1234, built: 2026-04-10T13:00:00Z)
```

```bash
bin/bacot --help
```

---

## Your first test

### Option 1: Inline — no config file needed

Run a quick test against any URL:

```bash
bin/bacot run --url https://httpbin.org/get --vus 5 --duration 10s
```

This starts **5 virtual users** sending GET requests for **10 seconds**, then prints a summary.

### Option 2: YAML script

Create `my-test.yaml`:

```yaml
name: "My First Test"
target: "https://httpbin.org"

stages:
  - duration: 10s
    vus: 5

scenarios:
  - name: "GET /get"
    method: GET
    path: /get
    checks:
      - status == 200
```

Run it:

```bash
bin/bacot run my-test.yaml
```

---

## Understanding the output

During the test you see a **live dashboard** (refreshes every 500ms):

```
──────────────────────────────────────────────────────────
  Stage 1/1  [████████░░░░░░░░░░░░]  8s / 10s
  VUs: 5 active

  Requests      87    RPS: 10.9/s
  Failures      0     (0.00%)
  Duration      p50: 215ms   p95: 890ms   p99: 920ms
  Data          ↑ 0 B   ↓ 21.2 kB

  Checks        ✓ 87   ✗ 0
```

After the test a **summary** is printed:

```
────────────────────── SUMMARY ──────────────────────

  http_reqs...............: 109   10.9/s
  http_req_duration.......: avg=432ms   min=198ms   p90=840ms   p95=890ms   p99=920ms
  http_req_failed.........: 0.00%   (0 of 109)
  data_sent...............: 0 B   0 B/s
  data_received...........: 26.6 kB   2.7 kB/s

  checks..................: 100.0%   ✓ 109 / ✗ 0

  Status Codes
  └─ 200: 109 (100.00%)
```

---

## Save results to a file

```bash
# JSON report
bin/bacot run my-test.yaml --output json=result.json

# HTML report (self-contained, open in browser)
bin/bacot run my-test.yaml --output html=report.html

# Both at once
bin/bacot run my-test.yaml --output json=result.json --output html=report.html
```

---

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Test completed, all thresholds passed (or no thresholds defined) |
| `1` | One or more thresholds failed, or a fatal error occurred |

This makes bacot suitable for use in CI/CD pipelines:

```bash
bin/bacot run perf-test.yaml || echo "Performance regression detected!"
```
