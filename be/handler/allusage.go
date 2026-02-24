package handler

import (
	"lb/store"
	"net/http"

	"github.com/labstack/echo/v4"
)

// AllUsage handles GET /admin/usage.
// Auth is enforced at the route-group level by AdminAuthMiddleware.
// Returns token usage for every user, keyed by user → model.
func AllUsage(s *store.Store) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, s.GetAll())
	}
}
