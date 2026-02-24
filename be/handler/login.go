package handler

import (
	"lb/users"
	"net/http"

	"github.com/labstack/echo/v4"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login handles POST /auth/login.
// Validates username + password and returns the user's API key on success.
// This is a simulation endpoint — never return raw API keys in production.
func Login() echo.HandlerFunc {
	return func(c echo.Context) error {
		var req loginRequest
		if err := c.Bind(&req); err != nil || req.Username == "" || req.Password == "" {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "username and password required"})
		}
		u, ok := users.Login(req.Username, req.Password)
		if !ok {
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "invalid credentials"})
		}
		return c.JSON(http.StatusOK, echo.Map{
			"user_id":  u.ID,
			"api_key":  u.Key,
			"is_admin": u.IsAdmin,
		})
	}
}
