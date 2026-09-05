package server

import (
	"context"
	"embed"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gcc798/ops-gateway-mcp/internal/audit"
	"github.com/gcc798/ops-gateway-mcp/internal/auth"
	"github.com/gcc798/ops-gateway-mcp/internal/database"
	kube "github.com/gcc798/ops-gateway-mcp/internal/kubernetes"
	linux "github.com/gcc798/ops-gateway-mcp/internal/linux"
	"github.com/gcc798/ops-gateway-mcp/internal/resources"
	"github.com/gcc798/ops-gateway-mcp/internal/service"
	"github.com/gcc798/ops-gateway-mcp/internal/storage"
	"github.com/labstack/echo/v5"
)

// 手工浏览器回归专用，只使用临时库与本机不可连接地址。
func TestBrowserFixture(t *testing.T) {
	address := os.Getenv("GATEWAY_BROWSER_TEST_ADDR")
	if address == "" {
		t.Skip("未启用浏览器回归服务")
	}
	ctx := context.Background()
	db, err := storage.Open(ctx, filepath.Join(t.TempDir(), "gateway.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	catalog, audits, tokens := resources.New(db), audit.New(db), auth.New(db)
	if err := tokens.Upsert(ctx, "browser-test", "browser-test-token", auth.ScopeREST); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 65; i++ {
		name := fmt.Sprintf("db-%02d", i)
		if _, err := db.Exec("INSERT INTO database_resources(name,environment,driver,dsn) VALUES(?,'dev','postgres','postgres://test:test@127.0.0.1:1/test?connect_timeout=1')", name); err != nil {
			t.Fatal(err)
		}
		if err := audits.Record(ctx, audit.Operation{ID: name, Timestamp: time.Now().Add(-time.Duration(i) * time.Minute), Tool: "db_query", ResourceType: "database", Resource: name, Environment: "dev", Decision: "allow", Status: "succeeded"}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec("INSERT INTO linux_resources(name,environment,address,user,password,private_key) VALUES('dev-host','dev','127.0.0.1:1','developer','fixture-password','fixture-private-key'); INSERT INTO kubernetes_resources(name,environment,kubeconfig,context,token) VALUES('dev-cluster','dev','/tmp/test-config','test-context','fixture-token')"); err != nil {
		t.Fatal(err)
	}
	manager := database.NewManager()
	defer manager.Close()
	manager.SetResolver(func(name string) (*database.Adapter, error) {
		r, err := catalog.Database(ctx, name)
		if err != nil {
			return nil, err
		}
		return database.Open(database.Config{Name: r.Name, Driver: r.Driver, DSN: r.DSN})
	})
	app := &service.Service{Resources: catalog, Audit: audits, Databases: manager}
	for i := 0; i < 2; i++ {
		result, err := app.EvaluateSQL(auth.WithIdentity(ctx, "browser-test"), "", "", "db-00", "UPDATE test SET value=1 WHERE id=1")
		if err != nil {
			t.Fatal(err)
		}
		if i == 1 {
			if _, err := db.Exec("UPDATE pending_operations SET expires_at=? WHERE operation_id=?", time.Now().Add(-time.Hour).Format(time.RFC3339Nano), result.OperationID); err != nil {
				t.Fatal(err)
			}
		}
	}
	e := New(resources.Paths{Data: "临时回归数据库", Logs: "未写入日志"}, slog.New(slog.NewTextHandler(io.Discard, nil)), embed.FS{}, audits, tokens, manager, kube.NewManager(), linux.NewManager(), catalog)
	e.GET("/*", echo.WrapHandler(http.FileServer(http.Dir("../../web/dist"))))
	srv := &http.Server{Addr: address, Handler: e, ReadHeaderTimeout: 5 * time.Second}
	t.Log("浏览器回归地址: http://" + address)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		t.Fatal(err)
	}
}
