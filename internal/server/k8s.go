package server

import (
	kube "github.com/gcc798/ai-ops-gateway/internal/kubernetes"
	"github.com/labstack/echo/v5"
)

type KubernetesHandler struct{ D *Dependencies }

func (h *KubernetesHandler) Register(g *echo.Group) {
	g.GET("/clusters", h.clusters)
	g.GET("/clusters/:name", func(c *echo.Context) error { return h.D.resourceDetail(c, "kubernetes") })
	g.POST("/clusters/test", h.test)
	g.GET("/:cluster/namespaces/:namespace/pods", h.pods)
	g.GET("/:cluster/namespaces/:namespace/pods/:pod", h.pod)
	g.GET("/:cluster/namespaces/:namespace/deployments", h.deployments)
	g.GET("/:cluster/namespaces/:namespace/deployments/:deployment", h.deployment)
	g.POST("/:cluster/namespaces/:namespace/deployments/:deployment/restart", h.restart)
	g.GET("/:cluster/namespaces/:namespace/services", h.services)
	g.GET("/:cluster/namespaces/:namespace/pods/:pod/logs", h.logs)
	g.POST("/:cluster/namespaces/:namespace/pods/:pod/download", h.download)
}
func (h *KubernetesHandler) test(c *echo.Context) error {
	var in nameRequest
	if err := c.Bind(&in); err != nil {
		return c.JSON(400, errorBody("INVALID_REQUEST", "invalid request"))
	}
	client, err := h.D.App.KubernetesClient(in.Name)
	if err != nil {
		return c.JSON(404, errorBody("CLUSTER_NOT_FOUND", "cluster not found"))
	}
	if err := client.Ping(c.Request().Context()); err != nil {
		return c.JSON(502, errorBody("K8S_UNAVAILABLE", "cluster unavailable"))
	}
	return c.JSON(200, map[string]string{"status": "ok"})
}
func (h *KubernetesHandler) deployment(c *echo.Context) error {
	x, err := h.client(c)
	if err != nil {
		return c.JSON(404, errorBody("CLUSTER_NOT_FOUND", "cluster not found"))
	}
	deployment, err := x.Deployment(c.Request().Context(), c.Param("namespace"), c.Param("deployment"))
	if err != nil {
		return c.JSON(502, errorBody("K8S_QUERY_FAILED", "unable to get deployment"))
	}
	return c.JSON(200, map[string]any{"name": deployment.Name, "replicas": deployment.Status.Replicas, "ready_replicas": deployment.Status.ReadyReplicas, "available_replicas": deployment.Status.AvailableReplicas, "containers": deployment.Spec.Template.Spec.Containers})
}
func (h *KubernetesHandler) restart(c *echo.Context) error {
	var in struct {
		Client      string `json:"client"`
		Environment string `json:"environment"`
	}
	if err := c.Bind(&in); err != nil {
		return c.JSON(400, errorBody("INVALID_REQUEST", "invalid request"))
	}
	target := c.Param("namespace") + "/" + c.Param("deployment")
	result, err := h.D.App.PrepareAction(c.Request().Context(), in.Client, in.Environment, "kubernetes", c.Param("cluster"), "rollout_restart", target)
	if err != nil {
		return c.JSON(400, errorBody("RESTART_REJECTED", err.Error()))
	}
	return c.JSON(202, result)
}
func (h *KubernetesHandler) client(c *echo.Context) (*kube.Client, error) {
	return h.D.App.KubernetesClient(c.Param("cluster"))
}
func (h *KubernetesHandler) clusters(c *echo.Context) error {
	return h.D.resourceList(c, "kubernetes")
}
func (h *KubernetesHandler) pods(c *echo.Context) error {
	x, e := h.client(c)
	if e != nil {
		return c.JSON(404, errorBody("CLUSTER_NOT_FOUND", "cluster not found"))
	}
	v, e := x.ListPods(c.Request().Context(), c.Param("namespace"))
	if e != nil {
		return c.JSON(502, errorBody("K8S_QUERY_FAILED", "unable to list pods"))
	}
	return c.JSON(200, v)
}
func (h *KubernetesHandler) pod(c *echo.Context) error {
	x, e := h.client(c)
	if e != nil {
		return c.JSON(404, errorBody("CLUSTER_NOT_FOUND", "cluster not found"))
	}
	p, e := x.Pod(c.Request().Context(), c.Param("namespace"), c.Param("pod"))
	if e != nil {
		return c.JSON(502, errorBody("K8S_QUERY_FAILED", "unable to get pod"))
	}
	return c.JSON(200, map[string]any{"name": p.Name, "phase": p.Status.Phase, "node": p.Spec.NodeName, "containers": p.Spec.Containers})
}
func (h *KubernetesHandler) deployments(c *echo.Context) error {
	x, e := h.client(c)
	if e != nil {
		return c.JSON(404, errorBody("CLUSTER_NOT_FOUND", "cluster not found"))
	}
	v, e := x.ListDeployments(c.Request().Context(), c.Param("namespace"))
	if e != nil {
		return c.JSON(502, errorBody("K8S_QUERY_FAILED", "unable to list deployments"))
	}
	return c.JSON(200, v)
}
func (h *KubernetesHandler) services(c *echo.Context) error {
	x, e := h.client(c)
	if e != nil {
		return c.JSON(404, errorBody("CLUSTER_NOT_FOUND", "cluster not found"))
	}
	v, e := x.ListServices(c.Request().Context(), c.Param("namespace"))
	if e != nil {
		return c.JSON(502, errorBody("K8S_QUERY_FAILED", "unable to list services"))
	}
	return c.JSON(200, v)
}
func (h *KubernetesHandler) logs(c *echo.Context) error {
	x, e := h.client(c)
	if e != nil {
		return c.JSON(404, errorBody("CLUSTER_NOT_FOUND", "cluster not found"))
	}
	v, e := x.Logs(c.Request().Context(), c.Param("namespace"), c.Param("pod"), c.QueryParam("container"), 0)
	if e != nil {
		return c.JSON(502, errorBody("K8S_LOGS_FAILED", "unable to read logs"))
	}
	return c.String(200, v)
}
func (h *KubernetesHandler) download(c *echo.Context) error {
	var in struct {
		Container string `json:"container"`
		Path      string `json:"path"`
	}
	if e := c.Bind(&in); e != nil {
		return c.JSON(400, errorBody("INVALID_REQUEST", "invalid request"))
	}
	x, e := h.client(c)
	if e != nil {
		return c.JSON(404, errorBody("CLUSTER_NOT_FOUND", "cluster not found"))
	}
	v, e := x.DownloadFile(c.Request().Context(), c.Param("namespace"), c.Param("pod"), in.Container, in.Path, 0)
	if e != nil {
		return c.JSON(400, errorBody("DOWNLOAD_FAILED", e.Error()))
	}
	return c.Blob(200, "application/octet-stream", v)
}
