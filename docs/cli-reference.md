# CLI Reference

## Global synopsis

```
bacot [command] [flags]
```

---

## Commands

### `bacot run`

Runs a load test. Accepts either a YAML script path or inline flags.

```
bacot run [script.yaml] [flags]
```

**Arguments:**

| Argument | Description |
|---|---|
| `script.yaml` | Optional path to a YAML test script. If omitted, `--url` is required. |

**Run flags:**

| Flag | Type | Default | Description |
|---|---|---|---|
| `--url` | string | — | Target URL for inline (flag-only) tests |
| `--vus` | int | `1` | Number of virtual users for inline test |
| `--duration` | string | `30s` | Test duration for inline test (e.g. `10s`, `1m`, `1m30s`) |
| `--timeout` | string | `30s` | Overall HTTP request timeout per request |
| `--connect-timeout` | string | `10s` | TCP connection timeout |
| `--insecure` | bool | `false` | Skip TLS certificate verification (`-k` equivalent) |
| `--max-redirects` | int | `10` | Maximum number of HTTP redirects to follow (`0` = no redirects) |
| `--http2` | bool | `false` | Enable HTTP/2 support |

**Global flags (apply to all commands):**

| Flag | Type | Default | Description |
|---|---|---|---|
| `--output` | stringArray | — | Output destination(s). See [Output & Reports](output-reports.md) |
| `--no-color` | bool | `false` | Disable ANSI color in terminal output |
| `--quiet` | bool | `false` | Suppress live dashboard; only print final summary |
| `--full-report` | bool | `false` | Print full per-scenario metrics breakdown to stdout after the summary. Does not affect JSON/HTML output (those always include per-scenario data). |

---

### `bacot version`

Prints version information.

```
bacot version
```

**Example output:**
```
bacot v1.0.0 (commit: abc1234, built: 2026-04-10T13:00:00Z)
```

---

### `bacot help`

Prints help for any command.

```
bacot help
bacot help run
bacot run --help
```

---

## Usage examples

### Inline quick test

```bash
# 10 VUs for 30 seconds
bacot run --url https://api.example.com/health --vus 10 --duration 30s
```

### Inline with custom timeout

```bash
bacot run --url https://api.example.com/slow \
  --vus 5 --duration 20s \
  --timeout 5s --connect-timeout 2s
```

### Inline against self-signed TLS

```bash
bacot run --url https://localhost:8443/api \
  --vus 3 --duration 10s \
  --insecure
```

### YAML script run

```bash
bacot run perf-test.yaml
```

### YAML script with JSON + HTML reports

```bash
bacot run perf-test.yaml \
  --output json=results/result.json \
  --output html=results/report.html
```

### YAML script, quiet mode (CI-friendly)

```bash
bacot run perf-test.yaml --quiet
echo "Exit code: $?"
```

### Full per-scenario stdout report

```bash
bacot run perf-test.yaml --full-report
```

Prints a `SCENARIO REPORT` section after the summary with per-scenario latency percentiles (p50/p75/p90/p95/p99), failure rate, data transfer, and checks. This is a **stdout-only** flag — JSON and HTML reports always include full per-scenario data regardless.

### Override YAML HTTP settings from CLI

CLI flags take precedence over YAML HTTP settings:

```bash
# Use the YAML script but force HTTP/2 and skip TLS verify
bacot run perf-test.yaml --http2 --insecure
```

### Disable redirects

```bash
bacot run --url https://example.com --vus 1 --duration 5s --max-redirects 0
```

---

## Flag precedence

When running a YAML script, CLI flags **override** the equivalent YAML fields:

| CLI flag | Overrides YAML field |
|---|---|
| `--timeout` | `timeout` |
| `--connect-timeout` | `connect_timeout` |
| `--insecure` | `insecure` |
| `--http2` | `http2` |
| `--max-redirects` | `max_redirects` |

`--vus` and `--duration` only apply to inline (no-script) runs.
