package auth

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// TODO(Taman / critical): Move to vault and make this configurable.
const (
	AdminKey    = "sk-admin"
	AdminCtxKey = "adminCtxKey"
)

// ExtractKey pulls the Bearer token from the Authorization header.
// Returns empty string if missing or malformed.
func ExtractKey(c echo.Context) string {
	h := c.Request().Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(h, "Bearer ")
}

// IsAdmin returns true if the key is the fixed admin key.
func IsAdmin(key string) bool {
	return key == AdminKey
}

// AdminAuthMiddleware is an Echo middleware that rejects non-admin requests.
// Apply it to the /admin route group so individual handlers stay clean.
func AdminAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !IsAdmin(ExtractKey(c)) {
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "admin access required"})
		}
		c.Set(AdminCtxKey, true)
		return next(c)
	}
}
