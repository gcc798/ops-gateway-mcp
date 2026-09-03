package server

import "github.com/labstack/echo/v5"

import "strings"

type nameRequest struct {
	Name string `json:"name"`
}

type SQLHandler struct{ D *Dependencies }
type ActionHandler struct{ D *Dependencies }

func (h *SQLHandler) Register(g *echo.Group) {
	g.POST("/evaluate", h.evaluate)
	g.POST("/confirm", h.confirm)
}
func (h *ActionHandler) Register(g *echo.Group) { g.POST("/:id/confirm", h.confirm) }
func (h *ActionHandler) confirm(c *echo.Context) error {
	if err := h.D.App.ConfirmOperation(c.Request().Context(), c.Param("id")); err != nil {
		return c.JSON(409, errorBody("CONFIRM_FAILED", err.Error()))
	}
	return c.JSON(200, map[string]string{"status": "confirmed"})
}
func (h *SQLHandler) evaluate(c *echo.Context) error {
	var in struct {
		Client      string `json:"client"`
		Environment string `json:"environment"`
		Resource    string `json:"resource"`
		Statement   string `json:"statement"`
	}
	if e := c.Bind(&in); e != nil {
		return c.JSON(400, errorBody("INVALID_REQUEST", "invalid request"))
	}
	v, e := h.D.App.EvaluateSQL(c.Request().Context(), in.Client, in.Environment, in.Resource, in.Statement)
	if e != nil {
		return c.JSON(400, errorBody("INVALID_STATEMENT", e.Error()))
	}
	return c.JSON(200, v)
}
func (h *SQLHandler) confirm(c *echo.Context) error {
	var in struct {
		OperationID string `json:"operation_id"`
	}
	if e := c.Bind(&in); e != nil {
		return c.JSON(400, errorBody("INVALID_REQUEST", "invalid request"))
	}
	if e := h.D.App.ConfirmSQL(c.Request().Context(), in.OperationID); e != nil {
		return c.JSON(409, errorBody("CONFIRM_FAILED", e.Error()))
	}
	return c.JSON(200, map[string]string{"status": "confirmed"})
}
func errorBody(code, message string) map[string]string {
	message = sanitizeError(message)
	return map[string]string{"code": code, "message": message}
}
func sanitizeError(message string) string {
	for _, marker := range []string{"postgres://", "mysql://", "password=", "PASSWORD=", "token=", "TOKEN="} {
		if strings.Contains(message, marker) {
			return "request failed"
		}
	}
	return message
}
