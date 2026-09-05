package audit

import (
	"context"
	"fmt"
	"github.com/gcc798/ops-gateway-mcp/internal/pagination"
	"path/filepath"
	"testing"
	"time"
)

func TestSearchBeyondFiftyAndCombinedFilters(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "audit.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	emptyOptions, err := s.FilterOptions(ctx)
	if err != nil || emptyOptions.Tools == nil || emptyOptions.Clients == nil {
		t.Fatalf("empty options: %+v %v", emptyOptions, err)
	}
	start := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 75; i++ {
		kind, tool := "database", "db_query"
		if i%2 == 1 {
			kind, tool = "linux", "linux_read_file"
		}
		if err := s.Record(ctx, Operation{ID: fmt.Sprintf("op-%03d", i), Timestamp: start.Add(time.Duration(i) * time.Minute), Tool: tool, ResourceType: kind, Resource: "test", Environment: "dev", Decision: "allow", Status: "succeeded"}); err != nil {
			t.Fatal(err)
		}
	}
	out, err := s.Search(ctx, Filter{Query: pagination.Query{Page: 4, PageSize: 20}})
	if err != nil || out.Total != 75 || len(out.Items) != 15 || out.Items[0].ID != "op-014" {
		t.Fatalf("pagination: %+v %v", out, err)
	}
	out, err = s.Search(ctx, Filter{From: start.Add(10 * time.Minute), To: start.Add(20 * time.Minute), Tool: "db_query", ResourceType: "database", Environment: "dev", Decision: "allow", Status: "succeeded", Resource: "test"})
	if err != nil || out.Total != 5 || len(out.Items) != 5 || out.Items[0].ID != "op-018" || out.Items[4].ID != "op-010" {
		t.Fatalf("filters: %+v %v", out, err)
	}
	summary, err := s.Summary(ctx)
	options, optionsErr := s.FilterOptions(ctx)
	if optionsErr != nil || len(options.Tools) != 2 || options.Tools[0] != "db_query" || options.Tools[1] != "linux_read_file" || len(options.Clients) != 0 {
		t.Fatalf("distinct options: %+v %v", options, optionsErr)
	}
	if err != nil || summary.Total != 75 || summary.Allow != 75 {
		t.Fatalf("summary: %+v %v", summary, err)
	}
	out, err = s.Search(ctx, Filter{Tool: "' OR 1=1 --"})
	if err != nil || out.Total != 0 || out.Items == nil {
		t.Fatal("unsafe or null empty query", err)
	}
}
