package server

import (
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	httpRequests = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "ai_ops_gateway_http_requests_total", Help: "HTTP requests handled by the gateway."}, []string{"method", "route", "status"})
	httpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "ai_ops_gateway_http_request_duration_seconds", Help: "HTTP request duration."}, []string{"method", "route"})
)

func init() { prometheus.MustRegister(httpRequests, httpDuration) }

func responseStatus(c *echo.Context) int {
	if response, err := echo.UnwrapResponse(c.Response()); err == nil {
		return response.Status
	}
	return 200
}

func metricsMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		started := time.Now()
		err := next(c)
		route := c.Path()
		if route == "" {
			route = "unmatched"
		}
		status := responseStatus(c)
		httpRequests.WithLabelValues(c.Request().Method, route, strconv.Itoa(status)).Inc()
		httpDuration.WithLabelValues(c.Request().Method, route).Observe(time.Since(started).Seconds())
		return err
	}
}

func requestLoggingMiddleware(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			requestID := uuid.NewString()
			c.Response().Header().Set("X-Request-ID", requestID)
			started := time.Now()
			err := next(c)
			logger.Info("http request", "request_id", requestID, "method", c.Request().Method, "route", c.Path(), "status", responseStatus(c), "duration_ms", time.Since(started).Milliseconds())
			return err
		}
	}
}
