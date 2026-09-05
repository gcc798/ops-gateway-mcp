package server

import (
	"database/sql"
	"errors"
	"time"

	"github.com/gcc798/ai-ops-gateway/internal/audit"
	"github.com/gcc798/ai-ops-gateway/internal/pagination"
	"github.com/labstack/echo/v5"
)

type AuditHandler struct{ D *Dependencies }

func (h *AuditHandler) Register(g *echo.Group) {
	g.GET("/operations", h.list)
	g.GET("/operations/:id", h.get)
	g.GET("/summary", h.summary)
}
func (h *AuditHandler) list(c *echo.Context) error {
	q, err := pagination.Parse(c.QueryParams())
	if err != nil {
		return c.JSON(400, errorBody("INVALID_FILTER", err.Error()))
	}
	f := audit.Filter{
		Query: q, Tool: c.QueryParam("tool"), ResourceType: c.QueryParam("resource_type"),
		Resource: c.QueryParam("resource"), Environment: c.QueryParam("environment"),
		Client: c.QueryParam("client"), Decision: c.QueryParam("decision"),
		Status: c.QueryParam("status"),
	}
	for key, dst := range map[string]*time.Time{"from": &f.From, "to": &f.To} {
		if value := c.QueryParam(key); value != "" {
			*dst, err = time.Parse(time.RFC3339Nano, value)
			if err != nil {
				return c.JSON(400, errorBody("INVALID_FILTER", "dates must use RFC3339 with timezone"))
			}
		}
	}
	if !f.From.IsZero() && !f.To.IsZero() && !f.From.Before(f.To) {
		return c.JSON(400, errorBody("INVALID_FILTER", "from must precede to"))
	}
	if h.D.Audits == nil {
		return c.JSON(503, errorBody("AUDIT_UNAVAILABLE", "audit store unavailable"))
	}
	v, e := h.D.Audits.Search(c.Request().Context(), f)
	if e != nil {
		return c.JSON(500, errorBody("AUDIT_READ_FAILED", "unable to read audit operations"))
	}
	return c.JSON(200, v)
}
func (h *AuditHandler) get(c *echo.Context) error {
	if h.D.Audits == nil {
		return c.JSON(503, errorBody("AUDIT_UNAVAILABLE", "audit store unavailable"))
	}
	v, e := h.D.Audits.Get(c.Request().Context(), c.Param("id"))
	if errors.Is(e, sql.ErrNoRows) {
		return c.JSON(404, errorBody("NOT_FOUND", "operation not found"))
	}
	if e != nil {
		return c.JSON(500, errorBody("AUDIT_READ_FAILED", "unable to read operation"))
	}
	detail := struct {
		audit.Operation
		Statement string     `json:"statement,omitempty"`
		ExpiresAt *time.Time `json:"expires_at,omitempty"`
	}{Operation: v}
	if v.Status == "pending" {
		if v.Action == "sql" {
			p, err := h.D.Audits.Pending(c.Request().Context(), v.ID)
			if err != nil {
				return c.JSON(500, errorBody("AUDIT_READ_FAILED", "unable to read pending operation"))
			}
			detail.Statement = p.Statement
			detail.ExpiresAt = &p.ExpiresAt
		} else {
			p, err := h.D.Audits.PendingAction(c.Request().Context(), v.ID)
			if err != nil {
				return c.JSON(500, errorBody("AUDIT_READ_FAILED", "unable to read pending operation"))
			}
			detail.ExpiresAt = &p.ExpiresAt
		}
	}
	return c.JSON(200, detail)
}
func (h *AuditHandler) summary(c *echo.Context) error {
	if h.D.Audits == nil {
		return c.JSON(503, errorBody("AUDIT_UNAVAILABLE", "audit store unavailable"))
	}
	v, err := h.D.Audits.Summary(c.Request().Context())
	if err != nil {
		return c.JSON(500, errorBody("AUDIT_READ_FAILED", "unable to read audit summary"))
	}
	return c.JSON(200, v)
}
