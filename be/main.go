package main

import (
	"encoding/json"
	"lb/auth"
	"lb/handler"
	"lb/limiter"
	"lb/store"
	"lb/ui"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	var config struct {
		OllamaURL string `json:"ollama_url"`
		Port      string `json:"port"`
	}
	// Fallback defaults
	config.OllamaURL = "http://localhost:11434"
	config.Port = ":8000"

	if b, err := os.ReadFile("config.json"); err == nil {
		json.Unmarshal(b, &config)
	} else {
		log.Println("warn: config.json not found, using defaults")
	}

	s := store.New()
	lim := limiter.New()

	e := echo.New()
	e.HideBanner = true

	// Middleware: recover from panics, basic request logging, and CORS
	e.Use(middleware.Recover())
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogValuesFunc: logRequestValues,
	}))
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:3000", "http://127.0.0.1:3000", "*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))

	// Allow CORS preflights to succeed gracefully instead of hitting the catch-all 404
	e.OPTIONS("/*", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	// Inference
	e.POST("/v1/chat/completions", handler.Completions(config.OllamaURL, s, lim), auth.AuthMiddleware)

	// User API
	e.GET("/v1/usage", handler.Usage(s), auth.AuthMiddleware)

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

	log.Printf("Proxy listening on %s  →  Ollama at %s\n", config.Port, config.OllamaURL)
	if err := e.Start(config.Port); err != nil {
		log.Fatal(err)
	}
}

func logRequestValues(c echo.Context, v middleware.RequestLoggerValues) error {
	userID, ok := c.Get(auth.UserIDKey).(string)
	if !ok {
		userID = "anonymous"
	}

	c.Logger().Infof(
		"%s %d %s %s (User: %s) %v",
		v.StartTime.Format(time.RFC3339),
		v.Status,
		v.Method,
		v.URI,
		userID,
		v.Latency,
	)
	return nil
}
