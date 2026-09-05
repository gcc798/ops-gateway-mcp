package middleware

import (
	"strconv"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	httpRequests = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "ai_ops_gateway_http_requests_total", Help: "HTTP requests handled by the gateway."}, []string{"method", "route", "status"})
	httpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "ai_ops_gateway_http_request_duration_seconds", Help: "HTTP request duration."}, []string{"method", "route"})
)

func init() { prometheus.MustRegister(httpRequests, httpDuration) }

func ResponseStatus(c *echo.Context) int {
	if response, err := echo.UnwrapResponse(c.Response()); err == nil {
		return response.Status
	}
	return 200
}

func Metrics(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		started := time.Now()
		err := next(c)
		route := c.Path()
		if route == "" {
			route = "unmatched"
		}
		httpRequests.WithLabelValues(c.Request().Method, route, strconv.Itoa(ResponseStatus(c))).Inc()
		httpDuration.WithLabelValues(c.Request().Method, route).Observe(time.Since(started).Seconds())
		return err
	}
}
