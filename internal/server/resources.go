package server

import (
	"database/sql"
	"errors"

	"github.com/gcc798/ai-ops-gateway/internal/pagination"
	"github.com/gcc798/ai-ops-gateway/internal/resources"
	"github.com/labstack/echo/v5"
)

func (d *Dependencies) resourceList(c *echo.Context, kind string) error {
	q, err := pagination.Parse(c.QueryParams())
	if err != nil {
		return c.JSON(400, errorBody("INVALID_FILTER", err.Error()))
	}
	f := resources.Filter{
		Query: q, Name: c.QueryParam("name"), Environment: c.QueryParam("environment"),
		Driver: c.QueryParam("driver"), Address: c.QueryParam("address"),
		User: c.QueryParam("user"), Context: c.QueryParam("context"),
	}
	if (f.Driver != "" && kind != "database") || ((f.Address != "" || f.User != "") && kind != "linux") || (f.Context != "" && kind != "kubernetes") {
		return c.JSON(400, errorBody("INVALID_FILTER", "filter not supported for resource type"))
	}
	v, err := d.App.ListResources(c.Request().Context(), kind, f)
	if err != nil {
		return c.JSON(500, errorBody("RESOURCE_READ_FAILED", "unable to list resources"))
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.JSON(200, v)
}
func (d *Dependencies) resourceDetail(c *echo.Context, kind string) error {
	r, err := d.App.Resource(c.Request().Context(), kind, c.Param("name"))
	if errors.Is(err, sql.ErrNoRows) {
		return c.JSON(404, errorBody("NOT_FOUND", "resource not found"))
	}
	if err != nil {
		return c.JSON(500, errorBody("RESOURCE_READ_FAILED", "unable to read resource"))
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.JSON(200, r)
}
