# Ollama / OpenAI-Compatible API Proxy

A high-performance reverse proxy for [Ollama](https://ollama.com) that sits between your users and the inference engine. It injects authentication, usage accounting, rate limiting, and an admin dashboard—all while supporting zero-buffered streaming Server-Sent Events (SSE).

The repository is split into two halves:

- `be/` - High-performance Go Reverse Proxy
- `fe/` - Modern Next.js Admin Dashboard & Chat UI

## Features

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
