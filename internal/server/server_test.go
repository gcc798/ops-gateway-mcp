package server

import (
	"embed"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gcc798/ai-ops-gateway/internal/config"
	"github.com/gcc798/ai-ops-gateway/internal/database"
	kube "github.com/gcc798/ai-ops-gateway/internal/kubernetes"
	linux "github.com/gcc798/ai-ops-gateway/internal/linux"
)

func TestReadRoutes(t *testing.T) {
	e := New(config.Paths{}, slog.New(slog.NewTextHandler(io.Discard, nil)), embed.FS{}, nil, database.NewManager(), kube.NewManager(), linux.NewManager())
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
