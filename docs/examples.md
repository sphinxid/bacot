# Examples

Real-world test script examples covering common scenarios.

---

## 1. Simple smoke test (CLI inline)

Test that an endpoint is up and responding fast:

```bash
bin/bacot run --url https://api.example.com/health --vus 1 --duration 10s
```

---

## 2. Load ramp — single endpoint

```yaml
# ramp-test.yaml
name: "Ramp Load Test"
target: "https://api.example.com"

stages:
  - duration: 30s
    vus: 10
  - duration: 1m
    vus: 50
  - duration: 30s
    vus: 100
  - duration: 30s
    vus: 10

scenarios:
  - name: "GET /products"
    method: GET
    path: /products
    checks:
      - status == 200
      - duration < 1000

thresholds:
  http_req_duration_p95: "< 800ms"
  http_req_failed:       "< 1%"
```

```bash
bin/bacot run ramp-test.yaml --output html=ramp-report.html
```

---

## 3. Multi-scenario weighted traffic

Simulate realistic API traffic with different endpoint weights:

```yaml
# api-mixed.yaml
name: "Mixed API Traffic"
target: "https://api.example.com"

stages:
  - duration: 10s
    vus: 5
  - duration: 2m
    vus: 30
  - duration: 10s
    vus: 5

scenarios:
  - name: "List products"
    method: GET
    path: /products
    weight: 50
    headers:
      Accept: application/json
    checks:
      - status == 200

  - name: "Get product detail"
    method: GET
    path: /products/1
    weight: 30
    headers:
      Accept: application/json
    checks:
      - status == 200
      - duration < 300

  - name: "Create cart item"
    method: POST
    path: /cart
    weight: 20
    body: '{"product_id": 1, "quantity": 2}'
    headers:
      Content-Type: application/json
      Authorization: "Bearer test-token"
    checks:
      - status == 201
      - status != 400
      - status != 500

thresholds:
  http_req_duration_p95: "< 500ms"
  http_req_duration_avg: "< 200ms"
  http_req_failed:       "< 2%"
```

---

## 4. Authentication flow

Test login and protected endpoint together:

```yaml
# auth-test.yaml
name: "Auth Flow Test"
target: "https://api.example.com"

stages:
  - duration: 30s
    vus: 20

scenarios:
  - name: "POST /auth/login"
    method: POST
    path: /auth/login
    weight: 20
    body: '{"email":"test@example.com","password":"secret"}'
    headers:
      Content-Type: application/json
    checks:
      - status == 200
      - status != 401
      - status != 500

  - name: "GET /me (authenticated)"
    method: GET
    path: /me
    weight: 80
    headers:
      Authorization: "Bearer static-test-token"
      Accept: application/json
    checks:
      - status == 200
      - duration < 200

thresholds:
  http_req_duration_p99: "< 1s"
  http_req_failed:       "< 0.5%"
```

---

## 5. Spike test

Test behaviour under sudden traffic spikes:

```yaml
# spike-test.yaml
name: "Spike Test"
target: "https://api.example.com"

stages:
  - duration: 30s    # Normal baseline
    vus: 5
  - duration: 10s    # Sudden spike
    vus: 200
  - duration: 30s    # Recovery
    vus: 5

scenarios:
  - name: "GET /search"
    method: GET
    path: /search?q=test
    checks:
      - status == 200
      - status != 503
      - duration < 3000

thresholds:
  http_req_duration_p95: "< 3000ms"
  http_req_failed:       "< 5%"
```

---

## 6. Soak test (long duration)

Detect memory leaks and gradual degradation over time:

```yaml
# soak-test.yaml
name: "Soak Test — 30 minutes"
target: "https://api.example.com"

stages:
  - duration: 5m
    vus: 20
  - duration: 20m
    vus: 20
  - duration: 5m
    vus: 20

scenarios:
  - name: "GET /feed"
    method: GET
    path: /feed
    headers:
      Authorization: "Bearer test-token"
    checks:
      - status == 200

thresholds:
  http_req_duration_p95: "< 500ms"
  http_req_duration_avg: "< 200ms"
  http_req_failed:       "< 0.1%"
  http_reqs:             ">= 10000"
```

```bash
bin/bacot run soak-test.yaml \
  --output json=soak-result.json \
  --output html=soak-report.html
```

---

## 7. POST with file body

Send a large JSON payload stored in a file:

```yaml
# file-body-test.yaml
name: "Large Payload Test"
target: "https://api.example.com"

stages:
  - duration: 30s
    vus: 10

scenarios:
  - name: "POST large payload"
    method: POST
    path: /import
    body: "@/path/to/payload.json"
    headers:
      Content-Type: application/json
      Authorization: "Bearer token"
    checks:
      - status == 200
      - status == 202
```

---

## 8. Testing against localhost

Test a local service with relaxed timeouts:

```yaml
# local-test.yaml
name: "Local Service Test"
target: "http://localhost:8080"

timeout: 5s
connect_timeout: 1s

stages:
  - duration: 30s
    vus: 10

scenarios:
  - name: "GET /api/v1/users"
    method: GET
    path: /api/v1/users
    checks:
      - status == 200
      - duration < 100

thresholds:
  http_req_duration_p95: "< 100ms"
  http_req_failed:       "< 0%"
```

---

## 9. HTTPS with self-signed certificate

```yaml
# internal-test.yaml
name: "Internal Service Test"
target: "https://internal.company.local"

insecure: true
timeout: 10s

stages:
  - duration: 1m
    vus: 20

scenarios:
  - name: "GET /status"
    method: GET
    path: /status
    checks:
      - status == 200
```

Or override from CLI without changing the YAML:

```bash
bin/bacot run internal-test.yaml --insecure
```

---

## 10. CI/CD pipeline usage

```bash
#!/bin/bash
set -e

echo "Running performance tests..."
bin/bacot run perf-test.yaml \
  --quiet \
  --output json=reports/perf-result.json \
  --output html=reports/perf-report.html

EXIT_CODE=$?

if [ $EXIT_CODE -ne 0 ]; then
  echo "❌ Performance thresholds failed — see reports/perf-report.html"
  exit 1
fi

echo "✅ All performance thresholds passed"
```

---

## 11. HTTP/2 endpoint

```yaml
# http2-test.yaml
name: "HTTP/2 Endpoint Test"
target: "https://api.example.com"

http2: true

stages:
  - duration: 30s
    vus: 20

scenarios:
  - name: "GET /stream"
    method: GET
    path: /stream
    checks:
      - status == 200
```

Or from CLI:

```bash
bin/bacot run --url https://api.example.com/stream \
  --vus 20 --duration 30s --http2
```

---

## 12. Full example — production API

The file at `examples/api-load-test.yaml` demonstrates a complete production-style test:

```bash
bin/bacot run examples/api-load-test.yaml \
  --output json=result.json \
  --output html=report.html
```

This runs against `https://httpbin.org` with:
- 3-stage ramp: 5 → 20 → 5 VUs over 50 seconds
- 70/30 weighted split between GET and POST scenarios
- Per-response checks on status code and latency
- Three thresholds: p95 latency, error rate, average latency
