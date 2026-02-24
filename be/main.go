package main

import (
	"lb/auth"
	"lb/handler"
	"lb/limiter"
	"lb/store"
	"lb/ui"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	s := store.New()
	lim := limiter.New()

	e := echo.New()
	e.HideBanner = true

	// Middleware: recover from panics, basic request logging, and CORS
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:3000", "http://127.0.0.1:3000"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))

	// Inference
	e.POST("/v1/chat/completions", handler.Completions(s, lim))

	// User API
	e.GET("/v1/usage", handler.Usage(s))

	// Auth
	e.POST("/auth/login", handler.Login())

	// Admin APIs — auth enforced at group level
	admin := e.Group("/admin", auth.AdminAuthMiddleware)
	admin.POST("/limits", handler.SetLimits(lim))
	admin.POST("/suspend", handler.SuspendUser(lim))
	admin.GET("/usage", handler.AllUsage(s))
	admin.GET("/limits", handler.AllLimits(lim))
	admin.GET("/ui", ui.Dashboard(s, lim))

	// Catch-all: explicit 404
	e.Any("/*", func(c echo.Context) error {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "not found"})
	})

	log.Println("Proxy listening on :8000  →  Ollama at http://localhost:11434")
	if err := e.Start(":8000"); err != nil {
		log.Fatal(err)
	}
}
