package handler

import (
	"lb/pb"
	"lb/store"
	"net/http"

	"github.com/labstack/echo/v4"
)

// AllUsage handles GET /admin/usage.
// Auth is enforced at the route-group level by AdminAuthMiddleware.
// Returns token usage for every user, keyed by user → model.
func AllUsage(s *store.Store) echo.HandlerFunc {
	return func(c echo.Context) error {
		usage := s.GetAll()
		resp := &pb.AllUsageResponse{
			UsageByUser: make(map[string]*pb.UsageResponse, len(usage)),
		}

		for user, models := range usage {
			userResp := &pb.UsageResponse{
				UsageByModel: make(map[string]*pb.ModelUsage, len(models)),
			}
			for model, u := range models {
				userResp.UsageByModel[model] = &pb.ModelUsage{
					PromptTokens:     int32(u.PromptTokens),
					CompletionTokens: int32(u.CompletionTokens),
				}
			}
			resp.UsageByUser[user] = userResp
		}

		return c.JSON(http.StatusOK, resp)
	}
}
