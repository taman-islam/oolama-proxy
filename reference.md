# API Reference

The Proxy Server exposes an OpenAI-compatible API for accessing LLMs securely, with built-in token accounting and rate limiting.

## Base URL

All requests should be routed to the API proxy base URL:

```
http://localhost:8000
```

_(In production, this should be replaced with the deployed proxy hostname)._

## Authentication

All API requests must include an `Authorization` header containing your `Bearer` token (API Key).

```http
Authorization: Bearer <your_api_key>
```

Testing keys currently available:

- `sk-alice-001`
- `sk-bob-001`
- `sk-charlie-001`
- `sk-admin-001` [admin]

---

## Endpoints

### 1. Chat Completions

Generates a model response for the given chat conversation. Fully compatible with OpenAI's `v1/chat/completions` spec.

**Endpoint:** `POST /v1/chat/completions`
**Headers:**

- `Authorization: Bearer <your_api_key>`
- `Content-Type: application/json`

**Request Body (JSON):**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `model` | string | Yes | ID of the model to use (e.g. `llama3.2`). |
| `messages` | array | Yes | A list of messages comprising the conversation so far. |
| `stream` | boolean | No | If set, partial message deltas will be sent, like in ChatGPT. Tokens will be sent as data-only [server-sent events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events). |
| `max_tokens` | integer | No | The maximum number of tokens to generate in the completion. |

**Example (Non-Streaming):**

```bash
curl -X POST http://localhost:8000/v1/chat/completions \
  -H "Authorization: Bearer sk-alice-001" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3.2",
    "messages": [
      {
        "role": "user",
        "content": "Hello!"
      }
    ]
  }'
```

**Example (Streaming):**

```bash
curl -N -X POST http://localhost:8000/v1/chat/completions \
  -H "Authorization: Bearer sk-alice-001" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3.2",
    "stream": true,
    "messages": [{"role": "user", "content": "Count to 5"}]
  }'
```

_(Note: Usage statistics for streaming requests are automatically captured by the proxy)._

---

### 2. View Token Usage

Returns the total consumed `prompt_tokens` and `completion_tokens` for the authenticated user, aggregated by model.

**Endpoint:** `GET /v1/usage`
**Headers:**

- `Authorization: Bearer <your_api_key>`

**Example Request:**

```bash
curl -H "Authorization: Bearer sk-alice-001" http://localhost:8000/v1/usage
```

**Example Response:**

```json
{
  "llama3.2": {
    "prompt_tokens": 145,
    "completion_tokens": 402
  },
  "moondream": {
    "prompt_tokens": 10,
    "completion_tokens": 12
  }
}
```

---

## Errors and Rate Limiting

The API will return standard HTTP status codes depending on the violation:

- **`401 Unauthorized`**: Missing or invalid API Key.
- **`403 Forbidden`**: Token quota exceeded. You have utilized all allocated tokens for your account.
- **`429 Too Many Requests`**: Rate limit exceeded (RPS threshold hit). Please back off and try again later.
- **`502 Bad Gateway`**: Upstream inference engine (Ollama) is offline or unreachable.
