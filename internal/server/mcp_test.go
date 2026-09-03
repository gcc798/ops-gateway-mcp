package server

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gcc798/ai-ops-gateway/internal/audit"
	"github.com/gcc798/ai-ops-gateway/internal/database"
	kube "github.com/gcc798/ai-ops-gateway/internal/kubernetes"
	linux "github.com/gcc798/ai-ops-gateway/internal/linux"
	"github.com/gcc798/ai-ops-gateway/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPListsTools(t *testing.T) {
	db, k8s, hosts := database.NewManager(), kube.NewManager(), linux.NewManager()
	audits, err := audit.Open(filepath.Join(t.TempDir(), "audit.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer audits.Close()
	d := &Dependencies{Audits: audits, Databases: db, Kubernetes: k8s, Linux: hosts, App: &service.Service{Audit: audits, Databases: db, Kubernetes: k8s, Linux: hosts}}
	httpServer := httptest.NewServer(newMCPHandler(d))
	defer httpServer.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v1"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: httpServer.URL}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools.Tools) != 28 {
		t.Fatalf("got %d tools, want 28", len(tools.Tools))
	}
	if _, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "db_list_connections"}); err != nil {
		t.Fatal(err)
	}
	operations, err := audits.List(context.Background(), 10)
	if err != nil || len(operations) != 1 || operations[0].Tool != "db_list_connections" {
		t.Fatalf("operations=%#v err=%v", operations, err)
	}
}
