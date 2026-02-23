package handler

import (
	"lb/auth"
	"lb/limiter"
	"net/http"

	"github.com/labstack/echo/v4"
)

const AdminCtxKey = "admin"

type setLimitsRequest struct {
	UserID          string `json:"user_id"`
	RPS             int    `json:"rps"`
	MaxTokens       int64  `json:"max_tokens"`
	MaxTokensPerReq int64  `json:"max_tokens_per_request"`
}

// SetLimits handles POST /admin/limits.
// Auth is enforced at the route-group level by AdminAuthMiddleware.
// Sets RPS, total token quota, and per-request token cap for a user.
func SetLimits(lim *limiter.Limiter) echo.HandlerFunc {
	return func(c echo.Context) error {
		// check cheaply if the viewer is an admin
		if !c.Get(auth.AdminCtxKey).(bool) {
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "unauthorized"})
		}
		var req setLimitsRequest
		if err := c.Bind(&req); err != nil || req.UserID == "" {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
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
