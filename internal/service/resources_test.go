package service

import (
	"context"
	"database/sql"
	"github.com/gcc798/ops-gateway-mcp/internal/audit"
	"github.com/gcc798/ops-gateway-mcp/internal/auth"
	"github.com/gcc798/ops-gateway-mcp/internal/database"
	"github.com/gcc798/ops-gateway-mcp/internal/policy"
	"github.com/gcc798/ops-gateway-mcp/internal/resources"
	"path/filepath"
	"strings"
	"testing"
)

func TestTrustedEnvironmentAndChangedResources(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gateway.db")
	ctx := auth.WithIdentity(context.Background(), "developer")
	catalog, err := resources.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer catalog.Close()
	audits, err := audit.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer audits.Close()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`INSERT INTO database_resources(name,environment,driver,dsn) VALUES('prod-db','prod','postgres','postgres://u:p@localhost/app')`); err != nil {
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
	s := &Service{Resources: catalog, Databases: manager, Audit: audits}
	result, err := s.EvaluateSQL(ctx, "forged-user", "dev", "prod-db", "INSERT INTO t(v) VALUES(1)")
	if err != nil || result.Policy.Decision != policy.DecisionDeny {
		t.Fatalf("environment bypass: %+v %v", result, err)
	}
	op, err := audits.Get(ctx, result.OperationID)
	if err != nil || op.Environment != "prod" || op.Client != "developer" {
		t.Fatalf("trusted metadata: %+v %v", op, err)
	}
	result, err = s.EvaluateSQL(ctx, "forged", "dev", "prod-db", "UPDATE t SET v=1 WHERE id=1")
	if err != nil {
		t.Fatal(err)
	}
	first, err := s.Database("prod-db")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE database_resources SET dsn='postgres://u:new@localhost/app' WHERE name='prod-db'`); err != nil {
		t.Fatal(err)
	}
	if err := s.ConfirmSQL(ctx, result.OperationID); err == nil || !strings.Contains(err.Error(), "resource changed") {
		t.Fatalf("confirmed changed resource: %v", err)
	}
	second, err := s.Database("prod-db")
	if err != nil || second == first {
		t.Fatalf("stale client: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM database_resources WHERE name='prod-db'`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Database("prod-db"); err == nil {
		t.Fatal("deleted resource still accessible")
	}
}
