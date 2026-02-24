package handler

import (
	"lb/auth"
	"lb/pb"
	"lb/store"
	"net/http"

	"github.com/labstack/echo/v4"
)

// Usage handles GET /v1/usage.
// Returns token usage for the authenticated user, keyed by model.
func Usage(s *store.Store) echo.HandlerFunc {
	return func(c echo.Context) error {
		userID, ok := auth.ResolveUser(auth.ExtractKey(c))
		if !ok || userID == "" {
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "invalid API key"})
		}

		usage := s.Get(userID)
		resp := &pb.UsageResponse{
			UsageByModel: make(map[string]*pb.ModelUsage, len(usage)),
		}
		for model, u := range usage {
			resp.UsageByModel[model] = &pb.ModelUsage{
				PromptTokens:     int32(u.PromptTokens),
				CompletionTokens: int32(u.CompletionTokens),
			}
		}

		return c.JSON(http.StatusOK, resp)
	}
}
