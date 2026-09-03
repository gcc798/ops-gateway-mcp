package server

import (
	"embed"
	"github.com/gcc798/ai-ops-gateway/internal/audit"
	"github.com/gcc798/ai-ops-gateway/internal/config"
	"github.com/gcc798/ai-ops-gateway/internal/database"
	kube "github.com/gcc798/ai-ops-gateway/internal/kubernetes"
	linux "github.com/gcc798/ai-ops-gateway/internal/linux"
	"github.com/gcc798/ai-ops-gateway/internal/service"
	"github.com/labstack/echo/v5"
	"io/fs"
	"log/slog"
	"net/http"
)

type Dependencies struct {
	Paths      config.Paths
	Audits     *audit.Store
	Databases  *database.Manager
	Kubernetes *kube.Manager
	Linux      *linux.Manager
	App        *service.Service
}

func New(paths config.Paths, logger *slog.Logger, web embed.FS, audits *audit.Store, databases *database.Manager, kubernetes *kube.Manager, linuxHosts *linux.Manager) *echo.Echo {
	e := echo.New()
	e.Use(metricsMiddleware)
	e.Use(requestLoggingMiddleware(logger))
	d := &Dependencies{Paths: paths, Audits: audits, Databases: databases, Kubernetes: kubernetes, Linux: linuxHosts, App: &service.Service{Audit: audits, Databases: databases, Kubernetes: kubernetes, Linux: linuxHosts}}
	e.GET("/healthz", d.health)
	e.GET("/api/v1/info", d.info)
	e.GET("/api/v1/config/paths", d.paths)
	api := e.Group("/api/v1")
	(&DatabaseHandler{d}).Register(api.Group("/databases"))
	(&LinuxHandler{d}).Register(api.Group("/linux"))
	(&KubernetesHandler{d}).Register(api.Group("/kubernetes"))
	(&AuditHandler{d}).Register(api.Group("/audit"))
	(&SQLHandler{d}).Register(api.Group("/database"))
	(&ActionHandler{d}).Register(api.Group("/operations"))
	(&PolicyHandler{d}).Register(api.Group("/policies"))
	e.Any("/mcp", echo.WrapHandler(newMCPHandler(d)))
	if sub, err := fs.Sub(web, "web/dist"); err == nil {
		e.GET("/*", echo.WrapHandler(http.FileServer(http.FS(sub))))
	}
	return e
}

func (d *Dependencies) health(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
func (d *Dependencies) info(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"name": "ai-ops-gateway", "version": "0.1.0"})
}
func (d *Dependencies) paths(c *echo.Context) error { return c.JSON(http.StatusOK, d.Paths) }
