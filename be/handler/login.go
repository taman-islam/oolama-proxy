package handler

import (
	"lb/pb"
	"lb/users"
	"net/http"

	"github.com/labstack/echo/v4"
)

// Login handles POST /auth/login.
// Validates username + password and returns the user's API key on success.
// This is a simulation endpoint — never return raw API keys in production.
func Login() echo.HandlerFunc {
	return func(c echo.Context) error {
		var req pb.LoginRequest
		if err := c.Bind(&req); err != nil || req.Username == "" || req.Password == "" {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "username and password required"})
		}
		u, ok := users.Login(req.Username, req.Password)
		if !ok {
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "invalid credentials"})
		}
		return c.JSON(http.StatusOK, &pb.LoginResponse{
			UserId:  u.ID,
			ApiKey:  u.Key,
			IsAdmin: u.IsAdmin,
		})
	}
}
