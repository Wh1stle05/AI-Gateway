# AI Gateway

OpenAI-compatible LLM gateway for multi-provider routing, reliability, and observability — built for **AI Infra / SRE / DevOps** portfolio work.

## Features (Phase 1)

- **OpenAI-compatible API** — `POST /v1/chat/completions` (JSON + SSE streaming)
- **Multi-provider routing** — route by model name with optional fallback
- **Providers** — OpenAI API, Ollama (any OpenAI-compatible upstream)
- **Health probes** — `GET /health`, `GET /ready`
- **Structured logging** — JSON logs with request ID
- **Optional gateway auth** — `Authorization: Bearer <GATEWAY_API_KEY>`

## Quick Start

### 1. Config

```bash
cp config.example.yaml config.yaml
export OPENAI_API_KEY=sk-...   # optional if using Ollama only
```

### 2. Run locally

```bash
make build
./bin/gateway -config config.yaml
```

### 3. Test

```bash
# Health
curl http://localhost:8080/health
curl http://localhost:8080/ready

# Chat (Ollama — start Ollama first)
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3.2",
    "messages": [{"role": "user", "content": "Hello"}]
  }'

# Streaming
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3.2",
    "stream": true,
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

Use any OpenAI SDK by pointing `base_url` to `http://localhost:8080/v1`.

## Configuration

See [`config.example.yaml`](config.example.yaml).

| Section | Purpose |
|---------|---------|
| `server.addr` | Listen address (default `:8080`) |
| `gateway.api_key` | Optional client auth for the gateway |
| `providers` | Upstream LLM backends |
| `routing` | Model → provider (+ optional fallback) |

Environment variables in config use `${VAR}` syntax.

## Project Layout

```text
cmd/gateway/           Entrypoint
internal/config/       YAML config loading
internal/provider/     Upstream adapters
internal/router/       Model routing + fallback
internal/handler/      HTTP handlers
internal/middleware/   Logging, request ID
internal/gateway/      HTTP server wiring
deploy/                Docker Compose
docs/                  Architecture & runbooks
```

## Docker

```bash
cp config.example.yaml config.yaml
docker compose -f deploy/docker-compose.yml up --build
```

## Roadmap

| Phase | Focus |
|-------|--------|
| **1** ✅ | Core gateway, routing, streaming, health |
| **2** | Redis rate limit, circuit breaker, load test |
| **3** | Prometheus metrics + Grafana dashboard |
| **4** | CI/CD, K8s manifests, runbooks |

## License

MIT
