package server

import "github.com/labstack/echo/v5"

type DatabaseHandler struct{ D *Dependencies }

func (h *DatabaseHandler) Register(g *echo.Group) {
	g.GET("", h.list)
	g.POST("/test", h.test)
	g.POST("/:name/ping", h.ping)
	g.GET("/:name/tables", h.tables)
	g.GET("/:name/tables/:table", h.describe)
	g.POST("/:name/query", h.query)
}
func (h *DatabaseHandler) test(c *echo.Context) error {
	var in nameRequest
	if err := c.Bind(&in); err != nil {
		return c.JSON(400, errorBody("INVALID_REQUEST", "invalid request"))
	}
	db, err := h.D.App.Database(in.Name)
	if err != nil {
		return c.JSON(404, errorBody("DATABASE_NOT_FOUND", "database not found"))
	}
	if err := db.Ping(c.Request().Context()); err != nil {
		return c.JSON(502, errorBody("DATABASE_UNAVAILABLE", "database unavailable"))
	}
	return c.JSON(200, map[string]string{"status": "ok"})
}
func (h *DatabaseHandler) describe(c *echo.Context) error {
	db, e := h.D.App.Database(c.Param("name"))
	if e != nil {
		return c.JSON(404, errorBody("DATABASE_NOT_FOUND", "database not found"))
	}
	v, e := db.DescribeTable(c.Request().Context(), c.Param("table"))
	if e != nil {
		return c.JSON(502, errorBody("DATABASE_QUERY_FAILED", "unable to describe table"))
	}
	return c.JSON(200, v)
}
func (h *DatabaseHandler) list(c *echo.Context) error {
	if h.D.Databases == nil {
		return c.JSON(200, []string{})
	}
	return c.JSON(200, h.D.Databases.Names())
}
func (h *DatabaseHandler) ping(c *echo.Context) error {
	db, e := h.D.App.Database(c.Param("name"))
	if e != nil {
		return c.JSON(404, errorBody("DATABASE_NOT_FOUND", "database not found"))
	}
	if e = db.Ping(c.Request().Context()); e != nil {
		return c.JSON(502, errorBody("DATABASE_UNAVAILABLE", "database unavailable"))
	}
	return c.JSON(200, map[string]string{"status": "ok"})
}
func (h *DatabaseHandler) tables(c *echo.Context) error {
	db, e := h.D.App.Database(c.Param("name"))
	if e != nil {
		return c.JSON(404, errorBody("DATABASE_NOT_FOUND", "database not found"))
	}
	rows, e := db.ListTables(c.Request().Context())
	if e != nil {
		return c.JSON(502, errorBody("DATABASE_QUERY_FAILED", "unable to list tables"))
	}
	return c.JSON(200, rows)
}
func (h *DatabaseHandler) query(c *echo.Context) error {
	var in struct {
		Statement string `json:"statement"`
	}
	if e := c.Bind(&in); e != nil {
		return c.JSON(400, errorBody("INVALID_REQUEST", "invalid request"))
	}
	rows, e := h.D.App.QuerySQL(c.Request().Context(), c.Param("name"), in.Statement)
	if e != nil {
		return c.JSON(400, errorBody("QUERY_FAILED", e.Error()))
	}
	return c.JSON(200, rows)
}
