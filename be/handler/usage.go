package handler

import (
	"lb/auth"
	"lb/store"
	"net/http"

	"github.com/labstack/echo/v4"
)

// Usage handles GET /v1/usage.
// Returns token usage for the authenticated user, keyed by model.
func Usage(s *store.Store) echo.HandlerFunc {
	return func(c echo.Context) error {
		key := auth.ExtractKey(c)
		if key == "" {
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "missing API key"})
		}
		return c.JSON(http.StatusOK, s.Get(key))
	}
}
