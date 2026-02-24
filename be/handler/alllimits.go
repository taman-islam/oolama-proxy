package handler

import (
	"lb/limiter"
	"net/http"

	"github.com/labstack/echo/v4"
)

// AllLimits handles GET /admin/limits.
// Returns current RPS, token quota, per-request cap, and usage for every user
// the limiter knows about.
func AllLimits(lim *limiter.Limiter) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, lim.GetAllLimits())
	}
}
