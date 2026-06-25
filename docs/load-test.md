# Load Test

Phase 2 load testing uses [k6](https://k6.io/) against the mock provider so results do not depend on external LLM APIs.

## Prerequisites

- Gateway running with `config.loadtest.yaml`
- Redis running (required when rate limiting is enabled)
- [k6 installed](https://grafana.com/docs/k6/latest/set-up/install-k6/)

## Quick run

```bash
# Terminal 1: Redis
docker run --rm -p 6379:6379 redis:7-alpine

# Terminal 2: Gateway
./bin/gateway -config config.loadtest.yaml

# Terminal 3: Load test
k6 run scripts/loadtest.js
```

Or with Docker Compose:

```bash
cp config.loadtest.yaml config.yaml
docker compose -f deploy/docker-compose.yml up --build -d
k6 run scripts/loadtest.js
```

## Scenarios

| Scenario | VUs | Duration | Target |
|----------|-----|----------|--------|
| `healthCheck` | 20 | 30s | `GET /health` |
| `chatCompletion` | ramp 0→20 | 40s | `POST /v1/chat/completions` (mock-model) |

## Thresholds

- Error rate `< 1%`
- p99 latency `< 500ms` (mock provider, local)

## Sample results (fill after running)

Record your environment and paste k6 summary here:

```text
Environment: WSL2 / local, mock provider, Redis 7
Gateway: Go 1.26, rate_limit burst=50, rpm=600

http_reqs ..............: ~XXXX
http_req_failed ........: ~0.X%
http_req_duration p(99).: ~XX ms
```

## Rate limit verification

To verify 429 responses:

```bash
# Temporarily lower burst in config.loadtest.yaml, restart gateway, then:
for i in $(seq 1 100); do
  curl -s -o /dev/null -w "%{http_code}\n" \
    http://127.0.0.1:8080/v1/chat/completions \
    -H 'Content-Type: application/json' \
    -d '{"model":"mock-model","messages":[{"role":"user","content":"hi"}]}'
done | sort | uniq -c
```

Expect a mix of `200` and `429` when burst is exceeded.

## Circuit breaker verification

1. Point primary provider to a failing upstream (or use routing with bad base URL).
2. Send repeated requests until `failure_threshold` is reached.
3. Confirm fallback provider serves traffic without hitting the failing upstream.
4. Check logs for `circuit breaker open` / fast fallback behavior.
