# Ollama / OpenAI-Compatible API Proxy (Single Node)

A high-performance reverse proxy for [Ollama](https://ollama.com) that sits between your users and the inference engine. It injects authentication, usage accounting, rate limiting, and an admin dashboard—all while supporting streaming Server-Sent Events (SSE) without buffering the full response.

The repository is split into two halves:

- `be/` - High-performance Go Reverse Proxy
- `fe/` - Modern Next.js Admin Dashboard & Chat UI

## Features

### Documentation References

For deep dives into the API specs and architectural decisions of each stack, please see the dedicated reference documents:

- [Backend Proxy Reference](reference.md)
- [Frontend architecture Reference](fe/reference.md)

### Backend (`be/`)

- **Streaming Reverse Proxy:** Fully supports `stream: true` OpenAI completions to Ollama. SSE frames are streamed line-by-line natively without buffering the full response, ensuring ultra-low latency.
- **Accounting:** Usage tokens (`prompt_tokens`, `completion_tokens`) are parsed dynamically from the final streaming SSE frame or non-streaming JSON body.
- **Rate Limiting (RPS):** Token-bucket RPS limiting using `golang.org/x/time/rate`, configurable per user via the admin panel.
- **Token Quotas:** Enforces hard upper bounds on total token consumption. Users exceeding their quota receive a `403 Forbidden` response.
- **Per-Request Caps:** Imposes limits on `max_tokens` per request to prevent single long-running queries from monopolizing the GPU.
- **Role-Based Auth & Mocking:** In-memory user registry (`users.go`) supporting both API `Bearer` keys and username/password pairs for simulated login.

### Frontend (`fe/`)

- **Admin Dashboard:** A real-time control panel built in Next.js.
  - Live-updating table of registered users across all models.
  - Form inputs to hot-reload rate limits (RPS, Max Tokens, Per-Request limit) without restarting the Go proxy.
  - One-click suspend action.
- **Chat Simulator (`/chat`):** A working chat playground restricted to authenticated non-admin users. Connects directly to the proxy using the user's stored API key and renders SSE streaming bubbles in real-time.
- **Role-Based Routing:** The `/login` page routes Admin credentials directly to the dashboard, and standard users to their chat environments natively.

### API Contracts (`proto/`)

- **Strict Type Definitions:** Shared Protocol Buffer definitions (`api.proto`) establish a strict, language-agnostic contract between the frontend and backend for authentication schemas, admin telemetry payloads, and moderation features.
- **Auto-Generated Bindings:** Standalone bash scripts (`generate-gopb.sh`, `generate-proto-unix.sh`) seamlessly compile `.proto` files into Go structs (`be/pb`) and TypeScript hydrators (`fe/src/generated`).

## Getting Started

### 1. Start Ollama

Ensure you have [Ollama](https://ollama.com/) running locally on the default port (`11434`):

```bash
ollama serve
```

### 2. Start the Go Proxy

```bash
cd be/
go run .
```

The Go proxy starts on `http://localhost:8000`.

- Proxy Endpoint: `POST http://localhost:8000/v1/chat/completions`

### 3. Start the Next.js UI

```bash
cd fe/
npm install
npm run dev
```

The frontend starts on `http://localhost:3000`.

## Load Testing

The repository includes a standalone Go load-testing script `test_load.go` to benchmark the proxy's throughput and rate-limiting concurrently, without needing to install third-party tools like `hey` or `vegeta`.

To run a load test (e.g., 100 total requests, 10 at a time):

```bash
go run test_load.go -n 100 -c 10 -m "llama3.2:1b"
```

The script will output:

- Requests/second
- Fast/Average/Slowest latency
- p50/p90/p99 latency distribution
- A distribution of HTTP status codes (useful for verifying `429 Too Many Requests` when testing your rate limit configurations).

## Testing the Proxy

You can query the Go proxy directly using standard OpenAI clients, pointing them at port `8000`.

### Non-Streaming

```bash
curl -X POST http://localhost:8000/v1/chat/completions \
  -H "Authorization: Bearer sk-alice-123" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3.2:1b",
    "messages": [{"role": "user", "content": "Why is the sky blue?"}]
  }'
```

### Streaming

```bash
curl -X POST http://localhost:8000/v1/chat/completions \
  -H "Authorization: Bearer sk-bob-456" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3.2:1b",
    "stream": true,
    "messages": [{"role": "user", "content": "Count to 10"}]
  }'
```

## Simulated Users

The system currently exposes mock users for testing out the UI and rate limits:

| Username  | Password   | API Key        | Role      |
| --------- | ---------- | -------------- | --------- |
| `admin`   | `admin123` | `sk-admin-001` | **Admin** |
| `alice`   | `alice123` | `sk-alice-123` | User      |
| `bob`     | `bob123`   | `sk-bob-456`   | User      |
| `charlie` | `char123`  | `sk-char-789`  | User      |

## Out of Scope

The following capabilities were intentionally excluded from this version to keep the scope focused, the system easy to evaluate, and the demo friction-free. Each item is well understood and designed for, but not required to meet the goals of this exercise.

### Horizontal Load Balancing

The proxy is currently deployed as a single stateless instance.
The Go proxy comfortably handles tens of thousands of requests per second; in practice, model inference (Ollama) is the dominant bottleneck.

Horizontal scaling would involve running multiple proxy replicas behind a standard L4/L7 load balancer (e.g. NGINX, Envoy) with a shared Redis backend for usage and rate limits. This was omitted to avoid unnecessary distributed-system complexity for a single-node demo.

### Multi-Node Ollama / Inference Scheduling

Requests are forwarded to a single Ollama instance. Support for multiple inference backends (e.g. GPU pool, weighted routing, health-based failover) is a natural extension, but was intentionally excluded. The proxy interface and rate-limiting model are already compatible with this design.

### Persistent Storage

Usage accounting and limits are stored in memory for simplicity. A Redis-backed implementation is planned and would enable:

- Persistence across restarts
- Multi-proxy deployments
- Stronger rate-limit guarantees

In-memory storage was chosen to minimize setup overhead for reviewers.

### Authentication & Secrets Management

API keys and users are statically defined for demonstration purposes. Production deployments would integrate with:

- Secure secret storage
- OAuth / SSO
- Rotatable API keys

These were excluded to keep the authentication flow transparent and easy to inspect.

### Model Lifecycle Management

Models are assumed to be preloaded in Ollama. Dynamic model downloads, eviction, and version pinning are out of scope for this submission.
