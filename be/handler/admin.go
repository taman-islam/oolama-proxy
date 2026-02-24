package handler

import (
	"fmt"
	"lb/auth"
	"lb/limiter"
	"lb/pb"
	"net/http"

	"github.com/labstack/echo/v4"
)

// validateLimits ensures all limit fields are explicitly set (> 0).
func validateLimits(r *pb.SetLimitsRequest) error {
	type field struct {
		name  string
		value int64
	}
	fields := []field{
		{"rps", int64(r.Rps)},
		{"max_tokens", r.MaxTokens},
		{"max_tokens_per_request", r.MaxTokensPerRequest},
	}
	for _, f := range fields {
		if f.value <= 0 {
			return fmt.Errorf("field %q must be > 0; got %d", f.name, f.value)
		}
	}
	if r.UserId == "" {
		return fmt.Errorf("field \"user_id\" is required")
	}
	return nil
}

// SetLimits handles POST /admin/limits.
// Auth is enforced at the route-group level by AdminAuthMiddleware.
func SetLimits(lim *limiter.Limiter) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Defense-in-depth: verify admin context key was set by AdminAuthMiddleware.
		if ok, isAdmin := c.Get(auth.AdminCtxKey).(bool); !ok || !isAdmin {
			return c.JSON(http.StatusForbidden, echo.Map{"error": "admin access required"})
		}
		var req pb.SetLimitsRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid JSON body"})
		}
		if err := validateLimits(&req); err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		lim.SetLimits(req.UserId, int(req.Rps), req.MaxTokens, req.MaxTokensPerRequest)

		return c.JSON(http.StatusOK, &pb.SetLimitsResponse{
			UserId:              req.UserId,
			Rps:                 req.Rps,
			MaxTokens:           req.MaxTokens,
			MaxTokensPerRequest: req.MaxTokensPerRequest,
		})
	}
}

func SuspendUser(lim *limiter.Limiter) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Defense-in-depth: verify admin context key was set by AdminAuthMiddleware.
		if ok, isAdmin := c.Get(auth.AdminCtxKey).(bool); !ok || !isAdmin {
			return c.JSON(http.StatusForbidden, echo.Map{"error": "admin access required"})
		}
		var req pb.SuspendUserRequest
		if err := c.Bind(&req); err != nil || req.UserId == "" {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "user_id is required"})
		}
		// rps=0 → rate.Limit(0) with burst 0: Allow() always returns false.
		// This hard-blocks the user on every incoming request.
		lim.SetLimits(req.UserId, 0, 0, 0)
		return c.JSON(http.StatusOK, &pb.SuspendUserResponse{
			UserId: req.UserId,
			Status: "suspended",
		})
	}
}
