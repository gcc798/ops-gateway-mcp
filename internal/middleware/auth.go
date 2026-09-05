package middleware

import (
	"net/http"
	"strings"

	"github.com/gcc798/ops-gateway-mcp/internal/auth"
	"github.com/labstack/echo/v5"
)

func Auth(store *auth.Store) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			path := c.Request().URL.Path
			if strings.HasPrefix(path, "/api/v1/") {
				name, ok := store.Authenticate(c.Request().Context(), auth.Bearer(c.Request().Header.Get("Authorization")), auth.ScopeREST)
				if !ok {
					return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
				}
				c.SetRequest(c.Request().WithContext(auth.WithIdentity(c.Request().Context(), name)))
				c.Response().Header().Set("Cache-Control", "no-store")
			}
			return next(c)
		}
	}
}
