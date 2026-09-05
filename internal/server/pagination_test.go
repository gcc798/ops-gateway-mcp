package server

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"github.com/gcc798/ai-ops-gateway/internal/audit"
	"github.com/gcc798/ai-ops-gateway/internal/auth"
	"github.com/gcc798/ai-ops-gateway/internal/database"
	kube "github.com/gcc798/ai-ops-gateway/internal/kubernetes"
	linux "github.com/gcc798/ai-ops-gateway/internal/linux"
	"github.com/gcc798/ai-ops-gateway/internal/pagination"
	"github.com/gcc798/ai-ops-gateway/internal/resources"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAuthenticatedCatalogAndAudit(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "gateway.db")
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
	tokens, err := auth.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer tokens.Close()
	if err := tokens.Upsert(ctx, "developer", "test-rest-token", auth.ScopeREST); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for i := 0; i < 65; i++ {
		name := fmt.Sprintf("db-%02d", i)
		if _, err := db.Exec(`INSERT INTO database_resources(name,environment,driver,dsn) VALUES(?,'dev','postgres','postgres://u:visible-password@example.invalid/db')`, name); err != nil {
			t.Fatal(err)
		}
		if err := audits.Record(ctx, audit.Operation{ID: name, Timestamp: time.Date(2026, 9, 5, 0, i, 0, 0, time.UTC), Tool: "db_query", ResourceType: "database", Decision: "allow", Status: "succeeded"}); err != nil {
			t.Fatal(err)
		}
	}
	e := New(resources.Paths{}, slog.New(slog.NewTextHandler(io.Discard, nil)), embed.FS{}, audits, tokens, database.NewManager(), kube.NewManager(), linux.NewManager(), catalog)
	get := func(path, token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", path, nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rr := httptest.NewRecorder()
		e.ServeHTTP(rr, req)
		return rr
	}
	if rr := get("/api/v1/databases/db-00", ""); rr.Code != 401 || strings.Contains(rr.Body.String(), "visible-password") {
		t.Fatal("unauthenticated credentials", rr.Code)
	}
	rr := get("/api/v1/databases?page=3&page_size=20&environment=dev&driver=postgres&name=db-", "test-rest-token")
	var page pagination.Result[resources.Resource]
	if err := json.Unmarshal(rr.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if rr.Code != 200 || page.Total != 65 || len(page.Items) != 20 || page.Items[0].Name != "db-40" {
		t.Fatalf("list: %s", rr.Body)
	}
	rr = get("/api/v1/databases/db-00", "test-rest-token")
	if rr.Code != 200 || !strings.Contains(rr.Body.String(), "visible-password") || rr.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("detail: %s", rr.Body)
	}
	for _, path := range []string{"/api/v1/databases?page=0", "/api/v1/linux/hosts?page_size=101", "/api/v1/kubernetes/clusters?page=x", "/api/v1/databases?address=abc", "/api/v1/audit/operations?from=invalid", "/api/v1/audit/operations?from=2026-09-06T00:00:00Z&to=2026-09-05T00:00:00Z"} {
		if rr := get(path, "test-rest-token"); rr.Code != http.StatusBadRequest {
			t.Fatalf("%s: %d %s", path, rr.Code, rr.Body)
		}
	}
	rr = get("/api/v1/audit/operations?page=4&page_size=20&tool=db_query&resource_type=database&from=2026-09-05T00:00:00Z&to=2026-09-06T00:00:00Z", "test-rest-token")
	var ops pagination.Result[audit.Operation]
	if err := json.Unmarshal(rr.Body.Bytes(), &ops); err != nil {
		t.Fatal(err)
	}
	if rr.Code != 200 || ops.Total != 65 || len(ops.Items) != 5 {
		t.Fatalf("audit page: %s", rr.Body)
	}
	rr = get("/api/v1/audit/operations?tool=GET%20%2Fapi%2Fv1%2Fdatabases%2F%3Aname", "test-rest-token")
	if err := json.Unmarshal(rr.Body.Bytes(), &ops); err != nil {
		t.Fatal(err)
	}
	if len(ops.Items) != 1 || ops.Items[0].Client != "developer" || ops.Items[0].RequestID == "" || ops.Items[0].Resource != "db-00" {
		t.Fatalf("REST audit: %s", rr.Body)
	}
}
