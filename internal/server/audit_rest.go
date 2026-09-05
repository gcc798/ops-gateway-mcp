package server

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/gcc798/ops-gateway-mcp/internal/audit"
	"github.com/gcc798/ops-gateway-mcp/internal/auth"
	"github.com/gcc798/ops-gateway-mcp/internal/middleware"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

func (d *Dependencies) auditREST(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		route := c.Path()
		kind := ""
		switch {
		case strings.HasPrefix(route, "/api/v1/databases"):
			kind = "database"
		case strings.HasPrefix(route, "/api/v1/linux"):
			kind = "linux"
		case strings.HasPrefix(route, "/api/v1/kubernetes"):
			kind = "kubernetes"
		}
		// 提交与确认操作由应用服务记录完整生命周期，避免重复审计。
		if kind == "" || strings.HasSuffix(route, "/restart") || strings.HasSuffix(route, "/query") || d.Audits == nil {
			c.SetRequest(c.Request().WithContext(audit.WithTool(c.Request().Context(), c.Request().Method+" "+route)))
			return next(c)
		}
		ctx := c.Request().Context()
		name := c.Param("name")
		if name == "" {
			name = c.Param("cluster")
		}
		environment := ""
		if name != "" && d.App.Resources != nil {
			if r, err := d.App.Resource(ctx, kind, name); err == nil {
				environment = r.Environment
			}
		}
		started := time.Now()
		op := audit.Operation{
			ID: uuid.NewString(), Timestamp: started, RequestID: audit.RequestID(ctx),
			Client: auth.Identity(ctx), Tool: c.Request().Method + " " + route,
			ResourceType: kind, Resource: name, Environment: environment,
			Action: route, Status: "executing", Decision: "allow", Risk: "low",
		}
		if err := d.Audits.Record(ctx, op); err != nil {
			return c.JSON(503, errorBody("AUDIT_UNAVAILABLE", "audit store unavailable"))
		}
		err := next(c)
		status, errorMessage := "succeeded", ""
		if err != nil || middleware.ResponseStatus(c) >= 400 {
			status, errorMessage = "failed", "request failed"
		}
		finishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if auditErr := d.Audits.CompleteResult(finishCtx, op.ID, status, time.Since(started).Milliseconds(), 0, errorMessage); auditErr != nil {
			slog.Error("REST audit completion failed", "operation_id", op.ID)
		}
		return err
	}
}
