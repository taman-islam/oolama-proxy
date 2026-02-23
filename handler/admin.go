package handler

import (
	"fmt"
	"lb/auth"
	"lb/limiter"
	"net/http"

	"github.com/labstack/echo/v4"
)

type setLimitsRequest struct {
	UserID          string `json:"user_id"`
	RPS             int    `json:"rps"`
	MaxTokens       int64  `json:"max_tokens"`
	MaxTokensPerReq int64  `json:"max_tokens_per_request"`
}

// validateLimits ensures all limit fields are explicitly set (> 0).
// We reject zeros to avoid silent "keep current" semantics and reject
// negatives to prevent accidental use of internal INF_* sentinel values.
func (r setLimitsRequest) validate() error {
	type field struct {
		name  string
		value int64
	}
	fields := []field{
		{"rps", int64(r.RPS)},
		{"max_tokens", r.MaxTokens},
		{"max_tokens_per_request", r.MaxTokensPerReq},
	}
	for _, f := range fields {
		if f.value < 0 {
			return fmt.Errorf("field %q must not be negative; got %d", f.name, f.value)
		}
		if f.value == 0 {
			return fmt.Errorf("field %q must be explicitly set (> 0); use a positive value", f.name)
		}
	}
	if r.UserID == "" {
		return fmt.Errorf("field \"user_id\" is required")
	}
	return nil
}

// SetLimits handles POST /admin/limits.
// Auth is enforced at the route-group level by AdminAuthMiddleware.
// All limit fields (rps, max_tokens, max_tokens_per_request) must be > 0.
func SetLimits(lim *limiter.Limiter) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Defense-in-depth: verify admin context key was set by AdminAuthMiddleware.
		if ok, isAdmin := c.Get(auth.AdminCtxKey).(bool); !ok || !isAdmin {
			return c.JSON(http.StatusForbidden, echo.Map{"error": "admin access required"})
		}
		var req setLimitsRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid JSON body"})
		}
		if err := req.validate(); err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		lim.SetLimits(req.UserID, req.RPS, req.MaxTokens, req.MaxTokensPerReq)
		return c.JSON(http.StatusOK, echo.Map{
			"user_id":                req.UserID,
			"rps":                    req.RPS,
			"max_tokens":             req.MaxTokens,
			"max_tokens_per_request": req.MaxTokensPerReq,
		})
	}
}
