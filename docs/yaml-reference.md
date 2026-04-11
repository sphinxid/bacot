# YAML Script Reference

A bacot YAML script defines the complete test plan: target, load profile, request scenarios, checks, and thresholds.

## Full annotated example

```yaml
# Human-readable test name shown in dashboard and reports
name: "API Load Test"

# Base URL — all scenario paths are appended to this
target: "https://api.example.com"

# ── HTTP options (optional — defaults shown) ─────────────
timeout: 30s           # Overall request timeout per request
connect_timeout: 10s   # TCP connection timeout
insecure: false        # true = skip TLS certificate verification
max_redirects: 10      # 0 = never follow redirects
http2: false           # true = enable HTTP/2

# ── Load stages ──────────────────────────────────────────
# Stages run in sequence. VU count transitions instantly
# between stages. The test ends when all stages complete.
stages:
  - duration: 10s   # Ramp-up: 5 VUs for 10 seconds
    vus: 5
  - duration: 30s   # Sustained load: 50 VUs for 30 seconds
    vus: 50
  - duration: 10s   # Ramp-down: 5 VUs for 10 seconds
    vus: 5

# ── Scenarios ─────────────────────────────────────────────
# Each VU picks a scenario on every iteration using weighted
# random selection. Weights are relative (not percentages).
scenarios:
  - name: "GET homepage"
    method: GET
    path: /                 # Appended to target URL
    weight: 70              # 70% of traffic
    headers:
      Accept: application/json
      Authorization: "Bearer mytoken"
    checks:
      - status == 200
      - duration < 500      # Response time in ms

  - name: "POST login"
    method: POST
    path: /auth/login
    weight: 30              # 30% of traffic
    body: '{"username":"test","password":"test"}'
    headers:
      Content-Type: application/json
    checks:
      - status == 200
      - status != 500

# ── Thresholds ────────────────────────────────────────────
# Evaluated after the test. If any threshold fails,
# bacot exits with code 1.
thresholds:
  http_req_duration_p95: "< 500ms"
  http_req_failed:       "< 1%"
  http_req_duration_avg: "< 200ms"
```

---

## Top-level fields

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | No | Display name for the test |
| `target` | string | **Yes** | Base URL (scheme + host). Scenario `path` values are appended |
| `stages` | list | **Yes** | One or more load stages |
| `scenarios` | list | No* | Request scenarios. If omitted, a single GET to `target` is used |
| `thresholds` | map | No | Pass/fail criteria evaluated after the test |
| `timeout` | duration | No | HTTP request timeout (default: `30s`) |
| `connect_timeout` | duration | No | TCP connect timeout (default: `10s`) |
| `insecure` | bool | No | Skip TLS verification (default: `false`) |
| `max_redirects` | int | No | Max redirects to follow (default: `10`) |
| `http2` | bool | No | Enable HTTP/2 (default: `false`) |

---

## `stages`

Each stage sets a VU count that persists for its duration. Stages run sequentially; VU transitions are instantaneous.

```yaml
stages:
  - duration: 10s
    vus: 5
  - duration: 60s
    vus: 100
  - duration: 10s
    vus: 5
```

| Field | Type | Required | Description |
|---|---|---|---|
| `duration` | duration string | **Yes** | How long this stage lasts. Must be `> 0`. |
| `vus` | int | **Yes** | Number of virtual users active during this stage. Must be `>= 1`. |

**Duration format:** Go duration strings — `10s`, `1m`, `1m30s`, `2h`, `500ms`.

**Single-stage (constant load):**
```yaml
stages:
  - duration: 60s
    vus: 50
```

**Multi-stage (ramp pattern):**
```yaml
stages:
  - duration: 30s   # warm-up
    vus: 10
  - duration: 2m    # peak
    vus: 200
  - duration: 30s   # cool-down
    vus: 10
```

---

## `scenarios`

Scenarios define the HTTP requests VUs execute. On every iteration a VU picks one scenario using weighted random selection.

```yaml
scenarios:
  - name: "GET user profile"
    method: GET
    path: /users/me
    weight: 80
    headers:
      Authorization: "Bearer token123"
      Accept: application/json
    checks:
      - status == 200
      - duration < 300

  - name: "POST create order"
    method: POST
    path: /orders
    weight: 20
    body: '{"product_id": 42, "qty": 1}'
    headers:
      Content-Type: application/json
    checks:
      - status == 201
```

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | **Yes** | Display name used in dashboard and reports |
| `method` | string | No | HTTP method (default: `GET`). Case-insensitive. |
| `path` | string | No | Path appended to `target`. Can be empty. |
| `weight` | int | No | Relative selection weight (default: `1`). Higher = more frequent. |
| `headers` | map | No | HTTP headers sent with every request in this scenario |
| `body` | string | No | Request body. String literal or `@/path/to/file` for file content. |
| `checks` | list | No | Per-response assertions. See [Checks](checks.md). |

### Supported HTTP methods

`GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `HEAD`, `OPTIONS`

### Weight and traffic distribution

Weights are **relative** — only their ratios matter:

```yaml
# These two configs produce the same traffic split (70/30):
scenarios:
  - weight: 70    # 70%
  - weight: 30    # 30%

scenarios:
  - weight: 7     # 70%
  - weight: 3     # 30%
```

If all weights are `0` or unset, bacot assigns equal weight to all scenarios.

### Request body from file

Prefix the `body` value with `@` to read from a file:

```yaml
body: "@/path/to/payload.json"
```

### URL construction

The final request URL is: `target` + `path`

```yaml
target: "https://api.example.com"

scenarios:
  - path: /users       # → https://api.example.com/users
  - path: /orders/123  # → https://api.example.com/orders/123
  - path: ""           # → https://api.example.com
```

---

## `thresholds`

See the full [Thresholds reference](thresholds.md).

```yaml
thresholds:
  http_req_duration_p95: "< 500ms"
  http_req_failed:       "< 1%"
  http_req_duration_avg: "< 200ms"
  http_reqs:             ">= 10000"
```

---

## Duration string format

All duration fields (`timeout`, `connect_timeout`, stage `duration`) use Go's duration syntax:

| String | Meaning |
|---|---|
| `500ms` | 500 milliseconds |
| `10s` | 10 seconds |
| `1m` | 1 minute |
| `1m30s` | 1 minute 30 seconds |
| `2h` | 2 hours |
| `90s` | 90 seconds |

---

## Validation rules

bacot validates the script at load time and will exit with an error if:

- `target` is missing or empty (inline mode: `--url` is required instead)
- Any stage has `duration: 0` or `vus: 0`
- Any scenario has an empty `name`
- Any scenario uses an unsupported HTTP method
- Any duration field cannot be parsed
