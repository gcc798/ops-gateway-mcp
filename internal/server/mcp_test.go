package server

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gcc798/ops-gateway-mcp/internal/audit"
	"github.com/gcc798/ops-gateway-mcp/internal/database"
	kube "github.com/gcc798/ops-gateway-mcp/internal/kubernetes"
	linux "github.com/gcc798/ops-gateway-mcp/internal/linux"
	"github.com/gcc798/ops-gateway-mcp/internal/service"
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
	if len(tools.Tools) != 26 {
		t.Fatalf("got %d tools, want 26", len(tools.Tools))
	}
	names := map[string]bool{}
	for _, tool := range tools.Tools {
		if tool.Name == "" || names[tool.Name] {
			t.Fatalf("invalid tool name %q", tool.Name)
		}
		names[tool.Name] = true
	}
	if !names["db_list_tables"] || names["db_confirm_execute"] || names["ops_confirm"] {
		t.Fatal("missing list tables or exposed confirmation")
	}
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "db_list_connections", Arguments: map[string]any{"page": 2, "page_size": 10},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("list failed: %#v", result.Content)
	}
	content, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var output struct {
		Value struct {
			Items    []json.RawMessage `json:"items"`
			Total    int               `json:"total"`
			Page     int               `json:"page"`
			PageSize int               `json:"page_size"`
		} `json:"value"`
	}
	if err := json.Unmarshal(content, &output); err != nil {
		t.Fatal(err)
	}
	if output.Value.Items == nil || output.Value.Total != 0 || output.Value.Page != 2 || output.Value.PageSize != 10 {
		t.Fatalf("invalid pagination response: %s", content)
	}
	operations, err := audits.List(context.Background(), 10)
	if err != nil || len(operations) != 1 || operations[0].Tool != "db_list_connections" {
		t.Fatalf("operations=%#v err=%v", operations, err)
	}
}
