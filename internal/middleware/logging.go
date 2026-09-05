package middleware

import (
	"github.com/gcc798/ai-ops-gateway/internal/audit"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

func RequestLogging(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			requestID := uuid.NewString()
			c.Response().Header().Set("X-Request-ID", requestID)
			c.SetRequest(c.Request().WithContext(audit.WithRequestID(c.Request().Context(), requestID)))
			started := time.Now()
			err := next(c)
			logger.Info("http request", "request_id", requestID, "method", c.Request().Method, "route", c.Path(), "status", ResponseStatus(c), "duration_ms", time.Since(started).Milliseconds())
			return err
		}
	}
}
