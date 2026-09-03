package server

import "github.com/labstack/echo/v5"

type PolicyHandler struct{ D *Dependencies }

func (h *PolicyHandler) Register(g *echo.Group) { g.GET("", h.list) }
func (h *PolicyHandler) list(c *echo.Context) error {
	return c.JSON(200, []map[string]string{
		{"domain": "database", "action": "read-only SQL", "decision": "allow"},
		{"domain": "database", "action": "scoped mutation / schema change", "decision": "confirm"},
		{"domain": "database", "action": "destructive or unscoped mutation", "decision": "deny"},
		{"domain": "linux", "action": "fixed read tools", "decision": "allow"},
		{"domain": "linux", "action": "restart service", "decision": "confirm"},
		{"domain": "kubernetes", "action": "fixed read tools / file download", "decision": "allow"},
		{"domain": "kubernetes", "action": "rollout restart", "decision": "confirm"},
	})
}
