package config

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestManagerLoad(t *testing.T) {
	d := t.TempDir()
	if e := os.WriteFile(filepath.Join(d, "gateway.yaml"), []byte("gateway:\n  listen_address: 127.0.0.1:9191\n"), 0600); e != nil {
		t.Fatal(e)
	}
	data := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(data, "ai-ops-gateway.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE database_resources(name TEXT PRIMARY KEY,environment TEXT NOT NULL DEFAULT '',driver TEXT NOT NULL,dsn_env TEXT NOT NULL);INSERT INTO database_resources VALUES('dev','dev','postgres','TEST_PG_DSN')`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	m := NewManager(Paths{Conf: d, Data: data})
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
