package server

import (
	linux "github.com/gcc798/ai-ops-gateway/internal/linux"
	"github.com/labstack/echo/v5"
	"net/http"
)

type LinuxHandler struct{ D *Dependencies }

func (h *LinuxHandler) Register(g *echo.Group) {
	g.GET("/hosts", h.list)
	g.GET("/hosts/:name", func(c *echo.Context) error { return h.D.resourceDetail(c, "linux") })
	g.POST("/hosts/test", h.test)
	g.GET("/hosts/:name/system-info", h.system)
	g.GET("/hosts/:name/disk-usage", h.disk)
	g.GET("/hosts/:name/memory-usage", h.memory)
	g.GET("/hosts/:name/processes", h.processes)
	g.GET("/hosts/:name/services/:service/status", h.serviceStatus)
	g.GET("/hosts/:name/services/:service/logs", h.serviceLogs)
	g.POST("/hosts/:name/services/:service/restart", h.restartService)
	g.GET("/hosts/:name/directory", h.directory)
	g.GET("/hosts/:name/file", h.file)
}
func (h *LinuxHandler) test(c *echo.Context) error {
	var in nameRequest
	if err := c.Bind(&in); err != nil {
		return c.JSON(400, errorBody("INVALID_REQUEST", "invalid request"))
	}
	host, err := h.D.App.LinuxClient(in.Name)
	if err != nil {
		return c.JSON(404, errorBody("HOST_NOT_FOUND", "host not found"))
	}
	if _, err := host.SystemInfo(c.Request().Context()); err != nil {
		return c.JSON(502, errorBody("SSH_FAILED", "host unavailable"))
	}
	return c.JSON(200, map[string]string{"status": "ok"})
}
func (h *LinuxHandler) serviceStatus(c *echo.Context) error {
	return h.text(c, func(x *linux.Client) (string, error) {
		return x.ServiceStatus(c.Request().Context(), c.Param("service"))
	})
}
func (h *LinuxHandler) serviceLogs(c *echo.Context) error {
	return h.text(c, func(x *linux.Client) (string, error) {
		return x.ServiceLogs(c.Request().Context(), c.Param("service"), 0)
	})
}
func (h *LinuxHandler) restartService(c *echo.Context) error {
	var in struct {
		Client      string `json:"client"`
		Environment string `json:"environment"`
	}
	if err := c.Bind(&in); err != nil {
		return c.JSON(400, errorBody("INVALID_REQUEST", "invalid request"))
	}
	result, err := h.D.App.PrepareAction(c.Request().Context(), in.Client, in.Environment, "linux", c.Param("name"), "restart_service", c.Param("service"))
	if err != nil {
		return c.JSON(400, errorBody("RESTART_REJECTED", err.Error()))
	}
	return c.JSON(202, result)
}
func (h *LinuxHandler) list(c *echo.Context) error {
	return h.D.resourceList(c, "linux")
}
func (h *LinuxHandler) host(c *echo.Context) (*linux.Client, error) {
	return h.D.App.LinuxClient(c.Param("name"))
}
func (h *LinuxHandler) text(c *echo.Context, run func(*linux.Client) (string, error)) error {
	x, e := h.host(c)
	if e != nil {
		return c.JSON(404, errorBody("HOST_NOT_FOUND", "host not found"))
	}
	out, e := run(x)
	if e != nil {
		return c.JSON(502, errorBody("SSH_FAILED", "ssh operation failed"))
	}
	return c.String(http.StatusOK, out)
}
func (h *LinuxHandler) system(c *echo.Context) error {
	return h.text(c, func(x *linux.Client) (string, error) { return x.SystemInfo(c.Request().Context()) })
}
func (h *LinuxHandler) disk(c *echo.Context) error {
	return h.text(c, func(x *linux.Client) (string, error) { return x.DiskUsage(c.Request().Context()) })
}
func (h *LinuxHandler) memory(c *echo.Context) error {
	return h.text(c, func(x *linux.Client) (string, error) { return x.MemoryUsage(c.Request().Context()) })
}
func (h *LinuxHandler) processes(c *echo.Context) error {
	return h.text(c, func(x *linux.Client) (string, error) { return x.Processes(c.Request().Context()) })
}
func (h *LinuxHandler) directory(c *echo.Context) error {
	return h.text(c, func(x *linux.Client) (string, error) {
		return x.ListDirectory(c.Request().Context(), c.QueryParam("path"))
	})
}
func (h *LinuxHandler) file(c *echo.Context) error {
	return h.text(c, func(x *linux.Client) (string, error) { return x.ReadFile(c.Request().Context(), c.QueryParam("path")) })
}
