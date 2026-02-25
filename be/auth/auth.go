package auth

import (
	"lb/users"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// TODO(Taman / critical): Move to vault and make this configurable.
const (
	AdminCtxKey = "adminCtxKey"
	UserIDKey   = "user_id"
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

// IsAdmin returns true if the key belongs to an admin user.
func IsAdmin(key string) bool {
	if u, ok := users.Lookup(key); ok {
		return u.IsAdmin
	}
	return false
}

// ResolveUser validates the Bearer key and returns the resolved user ID.
// Admin key returns "admin" and bypasses the user registry.
// Unknown keys return empty string and false.
func ResolveUser(key string) (userID string, ok bool) {
	if IsAdmin(key) {
		return "admin", true
	}
	u, ok := users.Lookup(key)
	if !ok {
		return "", false
	}
	return u.ID, true
}

// AdminAuthMiddleware is an Echo middleware that rejects non-admin requests.
// Apply it to the /admin route group so individual handlers stay clean.
func AdminAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !IsAdmin(ExtractKey(c)) {
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "admin access required"})
		}
		c.Set(UserIDKey, "admin")
		c.Set(AdminCtxKey, true)
		return next(c)
	}
}

// AuthMiddleware is an Echo middleware that requires a valid API key.
// It resolves the user ID and injects it into the context for downstream handlers.
func AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		key := ExtractKey(c)
		if key == "" {
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "missing API key"})
		}

		userID, ok := ResolveUser(key)
		if !ok {
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "unknown API key"})
		}

		c.Set(UserIDKey, userID)
		c.Set(AdminCtxKey, IsAdmin(key))

		return next(c)
	}
}
