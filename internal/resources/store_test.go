package resources

import (
	"context"
	"fmt"
	"github.com/gcc798/ai-ops-gateway/internal/pagination"
	"path/filepath"
	"testing"
)

func TestResourceQueries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gateway.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	for i := 0; i < 65; i++ {
		name := fmt.Sprintf("db-%02d", i)
		if _, err := s.db.Exec(`INSERT INTO database_resources(name,environment,driver,dsn) VALUES(?,'dev','mysql','user:pw@tcp(localhost)/db')`, name); err != nil {
			t.Fatal(err)
		}
	}
	page, err := s.List(ctx, "database", Filter{Query: pagination.Query{Page: 3, PageSize: 20}, Environment: "dev", Driver: "mysql", Name: "db-"})
	if err != nil || page.Total != 65 || len(page.Items) != 20 || page.Items[0].Name != "db-40" {
		t.Fatalf("page: %+v %v", page, err)
	}
	empty, err := s.List(ctx, "database", Filter{Name: "%' OR 1=1 --"})
	if err != nil || empty.Total != 0 || empty.Items == nil {
		t.Fatalf("literal filter: %+v %v", empty, err)
	}
	if _, err := s.List(ctx, "database", Filter{Query: pagination.Query{PageSize: 101}}); err == nil {
		t.Fatal("accepted unbounded page")
	}
	if _, err := s.db.Exec(`INSERT INTO linux_resources(name,environment,address,user,password,private_key) VALUES('host','dev','10.0.0.1:22','ops','linux-password','private-key');INSERT INTO kubernetes_resources(name,environment,kubeconfig,context,token) VALUES('cluster','prod','/tmp/kubeconfig','production','k8s-token')`); err != nil {
		t.Fatal(err)
	}
	hosts, err := s.List(ctx, "linux", Filter{Address: "10.0.", User: "ops", Environment: "dev"})
	if err != nil || hosts.Total != 1 || hosts.Items[0].Password != "linux-password" || hosts.Items[0].PrivateKey != "private-key" {
		t.Fatalf("hosts: %+v %v", hosts, err)
	}
	clusters, err := s.List(ctx, "kubernetes", Filter{Context: "duct", Environment: "prod"})
	if err != nil || clusters.Total != 1 || clusters.Items[0].Token != "k8s-token" {
		t.Fatalf("clusters: %+v %v", clusters, err)
	}
	if err := s.Observe(ctx); err != nil {
		t.Fatal(err)
	}
	events := []string{}
	s.SetObserver(func(kind, name, action string) error { events = append(events, kind+":"+name+":"+action); return nil })
	if _, err := s.db.Exec(`UPDATE linux_resources SET password='new' WHERE name='host';DELETE FROM kubernetes_resources WHERE name='cluster'`); err != nil {
		t.Fatal(err)
	}
	if err := s.Observe(ctx); err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("change audit: %v", events)
	}
	s.Close()
	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	r, err := s2.Linux(ctx, "host")
	if err != nil || r.Password != "new" {
		t.Fatalf("reopen lost config: %+v %v", r, err)
	}
}
