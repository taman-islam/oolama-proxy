package handler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"lb/auth"
	"lb/limiter"
	"lb/store"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/labstack/echo/v4"
)

const ollamaBase = "http://localhost:11434"

var reqCount uint64

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

	// Create a custom director to enforce the correct upstream Host
	// while stripping headers that Ollama might reject (like Origin from extensions)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = upstream.Host
		req.Header.Del("Origin")
	}

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
		count := atomic.AddUint64(&reqCount, 1)
		// log 10% of requests for debugging
		if rand.Intn(10) == 0 {
			log.Printf("[Req #%d] ==> Incoming /v1/chat/completions request (IP: %s)", count, c.RealIP())
		}

		userID := c.Get(auth.UserIDKey).(string)
		admin := c.Get(auth.AdminCtxKey).(bool)

		if !admin {
			if err := lim.CheckRPS(userID); err != nil {
				return c.JSON(http.StatusTooManyRequests, echo.Map{"error": "rate limit exceeded"})
			}
			if err := lim.CheckQuota(userID); err != nil {
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

		var requestedMaxTokens string
		if peek.MaxTokens != nil {
			requestedMaxTokens = strconv.FormatInt(*peek.MaxTokens, 10)
		} else {
			requestedMaxTokens = "nil"
		}
		// log 10% of requests for debugging
		if rand.Intn(10) == 0 {
			log.Printf("    Parsed Model: %s, Stream: %t, Requested MaxTokens: %s", peek.Model, isStream, requestedMaxTokens)
		}

		// Enforce per-request token cap and stream_options for accounting.
		// We use a generic map to preserve all other fields exactly as provided.
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(body, &raw); err == nil {
			modified := false

			// 1. Enforce max_tokens
			cap := lim.MaxTokensPerRequest(userID)
			if cap != limiter.INF_TOKEN_PER_REQ && (peek.MaxTokens == nil || *peek.MaxTokens > cap) {
				capBytes, _ := json.Marshal(cap)
				raw["max_tokens"] = capBytes
				modified = true
			}

			// 2. Enforce stream_options: { include_usage: true }
			if isStream {
				var opts map[string]interface{}
				if val, ok := raw["stream_options"]; ok {
					_ = json.Unmarshal(val, &opts)
				}
				if opts == nil {
					opts = make(map[string]interface{})
				}
				if include, _ := opts["include_usage"].(bool); !include {
					opts["include_usage"] = true
					optsBytes, _ := json.Marshal(opts)
					raw["stream_options"] = optsBytes
					modified = true
				}
			}

			if modified {
				if rewritten, err := json.Marshal(raw); err == nil {
					body = rewritten
				}
			}
		}

		c.Request().Body = io.NopCloser(bytes.NewReader(body))
		c.Request().ContentLength = int64(len(body))
		c.Request().Header.Set("Content-Length", strconv.Itoa(len(body)))

		// Attach values to request context so ModifyResponse can read them.
		req := c.Request().WithContext(
			contextWith(c.Request().Context(), userID, model, isStream),
		)
		c.SetRequest(req)

		if rand.Intn(10) == 0 {
			log.Printf("    Forwarding to upstream proxy...")
		}
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

	// We must close the pipe writer when the original body is closed,
	// otherwise the scanner goroutine below will hang forever waiting for EOF.
	resp.Body = &teeReadCloser{
		Reader: tee,
		Closer: resp.Body,
		pw:     pw,
	}

	go func() {
		// Ensure we always drain the pipe entirely, even if the scanner fails or exits early.
		// If the pipe isn't drained, TeeReader's pw.Write will block forever and freeze the proxy stream.
		defer func() {
			io.Copy(io.Discard, pr)
		}()

		// scanner reads from the read-half of our pipe.
		scanner := bufio.NewScanner(pr)
		// Provide a larger buffer (up to 1MB) for exceptionally large SSE JSON chunks so it doesn't ErrTooLong
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		var lastUsageLine string
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
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

type teeReadCloser struct {
	io.Reader
	io.Closer
	pw *io.PipeWriter
}

func (t *teeReadCloser) Close() error {
	err := t.Closer.Close()
	// Signal to the scanner that no more bytes are coming.
	t.pw.Close()
	return err
}
