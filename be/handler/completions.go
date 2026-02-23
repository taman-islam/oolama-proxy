package handler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"lb/auth"
	"lb/limiter"
	"lb/store"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"
)

const ollamaBase = "http://localhost:11434"

// usagePayload is the shape of the usage field in Ollama/OpenAI responses.
type usagePayload struct {
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// Completions handles POST /v1/chat/completions.
// It authenticates the caller, enforces rate/quota limits, proxies the request
// to Ollama, and accounts for token usage without blocking the inference path.
func Completions(s *store.Store, lim *limiter.Limiter) echo.HandlerFunc {
	upstream, _ := url.Parse(ollamaBase)

	proxy := httputil.NewSingleHostReverseProxy(upstream)
	// Disable response buffering — forward chunks immediately.
	proxy.FlushInterval = -1

	// Suppress default error handling so we can manage it ourselves.
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "upstream error: "+err.Error(), http.StatusBadGateway)
	}

	// ModifyResponse intercepts the upstream response for accounting.
	proxy.ModifyResponse = func(resp *http.Response) error {
		userKey := resp.Request.Context().Value(ctxKeyUser{}).(string)
		model := resp.Request.Context().Value(ctxKeyModel{}).(string)
		isStream := resp.Request.Context().Value(ctxKeyStream{}).(bool)

		if isStream {
			accountStream(resp, userKey, model, s, lim)
		} else {
			accountDirect(resp, userKey, model, s, lim)
		}
		return nil
	}

	return func(c echo.Context) error {
		key := auth.ExtractKey(c)
		if key == "" {
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "missing API key"})
		}

		// Admin key bypasses rate limits.
		if !auth.IsAdmin(key) {
			if err := lim.CheckRPS(key); err != nil {
				return c.JSON(http.StatusTooManyRequests, echo.Map{"error": "rate limit exceeded"})
			}
			if err := lim.CheckQuota(key); err != nil {
				return c.JSON(http.StatusForbidden, echo.Map{"error": "token quota exceeded"})
			}
		}

		// Peek at the body to detect streaming, model name, and max_tokens.
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "failed to read body"})
		}

		var peek struct {
			Model     string `json:"model"`
			Stream    *bool  `json:"stream"`
			MaxTokens *int64 `json:"max_tokens"`
		}
		_ = json.Unmarshal(body, &peek)

		model := peek.Model
		isStream := peek.Stream == nil || *peek.Stream // default true per OpenAI spec

		// Enforce per-request token cap: if the admin has set a limit and the
		// request asks for more tokens, clamp max_tokens before forwarding.
		// We only rewrite the body when necessary to avoid unnecessary allocs.
		if cap := lim.MaxTokensPerRequest(key); cap != limiter.INF_TOKEN_PER_REQ {
			if peek.MaxTokens == nil || *peek.MaxTokens > cap {
				// Unmarshal into a generic map so we preserve all other fields.
				var raw map[string]json.RawMessage
				if err := json.Unmarshal(body, &raw); err == nil {
					capBytes, _ := json.Marshal(cap)
					raw["max_tokens"] = capBytes
					if rewritten, err := json.Marshal(raw); err == nil {
						body = rewritten
					}
				}
			}
		}

		c.Request().Body = io.NopCloser(bytes.NewReader(body))
		c.Request().ContentLength = int64(len(body))

		// Attach values to request context so ModifyResponse can read them.
		req := c.Request().WithContext(
			contextWith(c.Request().Context(), key, model, isStream),
		)
		c.SetRequest(req)

		proxy.ServeHTTP(c.Response(), c.Request())
		return nil
	}
}

// accountDirect reads the full (non-streaming) response body, parses usage,
// restores the body for the client, and records accounting in the background.
func accountDirect(resp *http.Response, user, model string, s *store.Store, lim *limiter.Limiter) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))

	go func() {
		var p usagePayload
		if err := json.Unmarshal(body, &p); err != nil {
			return
		}
		total := p.Usage.PromptTokens + p.Usage.CompletionTokens
		s.Add(user, model, p.Usage.PromptTokens, p.Usage.CompletionTokens)
		lim.ConsumeTokens(user, total)
	}()
}

// accountStream wraps the streaming response body with an io.TeeReader that
// pipes bytes to a bufio.Scanner for incremental SSE frame parsing.
// Only the last usage-bearing frame (before [DONE]) is retained in memory.
// All other frames are forwarded immediately — no full-body buffering.
func accountStream(resp *http.Response, user, model string, s *store.Store, lim *limiter.Limiter) {
	pr, pw := io.Pipe()

	// TeeReader sends every byte to both the original resp.Body consumer
	// (the HTTP response writer) and our pipe writer.
	tee := io.TeeReader(resp.Body, pw)

	// Replace resp.Body with the tee so the proxy forwards the tee stream.
	resp.Body = io.NopCloser(tee)

	go func() {
		defer pw.Close()
		// We must drain the tee reader — this is driven by the proxy
		// copying resp.Body to the client. We only need to read from pr.
		scanner := bufio.NewScanner(pr)

		var lastUsageLine string
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}
			// Retain only lines that contain a usage field.
			if strings.Contains(data, `"usage"`) {
				lastUsageLine = data
			}
		}

		if lastUsageLine == "" {
			return
		}
		var p usagePayload
		if err := json.Unmarshal([]byte(lastUsageLine), &p); err != nil {
			return
		}
		total := p.Usage.PromptTokens + p.Usage.CompletionTokens
		s.Add(user, model, p.Usage.PromptTokens, p.Usage.CompletionTokens)
		lim.ConsumeTokens(user, total)
	}()
}
