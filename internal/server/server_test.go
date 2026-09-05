package server

import (
	"context"
	"embed"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gcc798/ops-gateway-mcp/internal/auth"
	"github.com/gcc798/ops-gateway-mcp/internal/database"
	kube "github.com/gcc798/ops-gateway-mcp/internal/kubernetes"
	linux "github.com/gcc798/ops-gateway-mcp/internal/linux"
	"github.com/gcc798/ops-gateway-mcp/internal/resources"
)

func TestReadRoutes(t *testing.T) {
	e := New(resources.Paths{}, slog.New(slog.NewTextHandler(io.Discard, nil)), embed.FS{}, nil, nil, database.NewManager(), kube.NewManager(), linux.NewManager(), nil)
	for _, path := range []string{"/healthz", "/api/v1/databases", "/api/v1/linux/hosts", "/api/v1/kubernetes/clusters", "/api/v1/policies"} {
		recorder := httptest.NewRecorder()
		e.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s: status %d body %s", path, recorder.Code, recorder.Body.String())
		}
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/config/paths", nil)
	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, request)
	if recorder.Header().Get("X-Request-ID") == "" {
		t.Fatal("request id missing")
	}
}

func TestAuthScopes(t *testing.T) {
	store, err := auth.Open(filepath.Join(t.TempDir(), "auth.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Upsert(context.Background(), "mcp", "mcp-secret", auth.ScopeMCP); err != nil {
		t.Fatal(err)
	}
	e := New(resources.Paths{}, slog.New(slog.NewTextHandler(io.Discard, nil)), embed.FS{}, nil, store, database.NewManager(), kube.NewManager(), linux.NewManager(), nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/databases", nil)
	rr := httptest.NewRecorder()
	e.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("REST without token: %d", rr.Code)
	}
	req = httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr = httptest.NewRecorder()
	e.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("healthz: %d", rr.Code)
	}
}
