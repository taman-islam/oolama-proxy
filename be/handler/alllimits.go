package handler

import (
	"lb/limiter"
	"lb/pb"
	"net/http"

	"github.com/labstack/echo/v4"
)

// AllLimits handles GET /admin/limits.
// Returns current RPS, token quota, per-request cap, and usage for every user
// the limiter knows about.
func AllLimits(lim *limiter.Limiter) echo.HandlerFunc {
	return func(c echo.Context) error {
		limits := lim.GetAllLimits()
		resp := &pb.AllLimitsResponse{
			Limits: make(map[string]*pb.LimitInfo, len(limits)),
		}
		for userID, info := range limits {
			resp.Limits[userID] = &pb.LimitInfo{
				MaxTokens:       info.MaxTokens,
				MaxTokensPerReq: info.MaxTokensPerReq,
				UsedTokens:      info.UsedTokens,
				Rps:             info.RPS,
			}
		}
		return c.JSON(http.StatusOK, resp)
	}
}
