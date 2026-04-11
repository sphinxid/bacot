# bacot Documentation

**bacot** is a production-grade HTTP performance testing CLI tool written in Go — similar to k6.

## Documentation Index

| Document | Description |
|---|---|
| [Getting Started](getting-started.md) | Installation, first run, quick examples |
| [CLI Reference](cli-reference.md) | All commands and flags |
| [YAML Script Reference](yaml-reference.md) | Complete YAML test script format |
| [Checks](checks.md) | Per-response assertion syntax |
| [Thresholds](thresholds.md) | Post-test pass/fail criteria |
| [Metrics Reference](metrics.md) | All collected metrics explained |
| [Output & Reports](output-reports.md) | Terminal dashboard, JSON, HTML reports |
| [Examples](examples.md) | Real-world test script examples |

---

## What is bacot?

bacot runs HTTP load tests against your APIs and services. You define **virtual users (VUs)**, **load stages**, and **scenarios** — bacot drives them, collects latency/error metrics, evaluates assertions, and produces rich reports.

```
bacot v1.0.0 — API Load Test
──────────────────────────────────────────────────────────
  Stage 2/3  [████████████░░░░░░░░]  32s / 50s
  VUs: 20 active

  Requests      1,245   RPS: 38.2/s
  Failures      12       (0.96%)
  Duration      p50: 182ms   p95: 641ms   p99: 980ms
  Data          ↑ 16.1 kB   ↓ 745.9 kB

  Checks        ✓ 2,479   ✗ 11

──────────────────────────────────────────────────────────
  Scenarios
  ├─ GET anything         871 reqs   p95: 598ms   ✓ 99.1%
  └─ POST data            374 reqs   p95: 710ms   ✓ 98.9%
```

## Key features

- **Goroutine-based VU engine** — scales to 500+ VUs without deadlock
- **Staged load profiles** — ramp up, sustain, ramp down
- **HDR histogram** — accurate p50/p90/p95/p99 latency percentiles
- **Checks** — per-response assertions (`status == 200`, `duration < 500`)
- **Thresholds** — pass/fail criteria that control exit code
- **Live ANSI dashboard** — refreshes every 500ms in-place
- **JSON & HTML reports** — structured data and interactive charts
- **Signal handling** — Ctrl+C triggers graceful shutdown with partial summary
