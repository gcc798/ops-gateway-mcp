package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManagerLoad(t *testing.T) {
	d := t.TempDir()
	if e := os.WriteFile(filepath.Join(d, "gateway.yaml"), []byte("gateway:\n  listen_address: 127.0.0.1:9191\n"), 0600); e != nil {
		t.Fatal(e)
	}
	if err := os.Mkdir(filepath.Join(d, "database"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "database", "dev.yaml"), []byte("databases:\n  - name: dev\n    environment: dev\n    driver: postgres\n    dsn_env: TEST_PG_DSN\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := &Manager{dir: d}
	s, e := m.Load()
	if e != nil {
		t.Fatal(e)
	}
	if s.Gateway.ListenAddress != "127.0.0.1:9191" {
		t.Fatalf("got %q", s.Gateway.ListenAddress)
	}
	if len(s.Databases) != 1 || s.Databases[0].Name != "dev" {
		t.Fatalf("databases: %#v", s.Databases)
	}
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()
	if got := ExpandPath("~/cluster.yaml"); got != filepath.Join(home, "cluster.yaml") {
		t.Fatalf("got %q", got)
	}
}
