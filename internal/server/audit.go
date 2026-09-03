package server

import "github.com/labstack/echo/v5"

type AuditHandler struct{ D *Dependencies }

func (h *AuditHandler) Register(g *echo.Group) {
	g.GET("/operations", h.list)
	g.GET("/operations/:id", h.get)
}
func (h *AuditHandler) list(c *echo.Context) error {
	v, e := h.D.Audits.List(c.Request().Context(), 0)
	if e != nil {
		return c.JSON(500, errorBody("AUDIT_READ_FAILED", "unable to read audit operations"))
	}
	return c.JSON(200, v)
}
func (h *AuditHandler) get(c *echo.Context) error {
	v, e := h.D.Audits.Get(c.Request().Context(), c.Param("id"))
	if e != nil {
		return c.JSON(404, errorBody("NOT_FOUND", "operation not found"))
	}
	return c.JSON(200, v)
}
