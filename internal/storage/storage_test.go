package storage

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
)

func TestFutureVersionRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gateway.db")
	db, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO goose_db_version(version_id,is_applied) VALUES(999,1)`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if db, err = Open(context.Background(), path); err == nil {
		db.Close()
		t.Fatal("newer database accepted")
	} else if !strings.Contains(err.Error(), "exceeds supported version") {
		t.Fatal(err)
	}
}

func TestFreshDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gateway.db")
	for i := 0; i < 2; i++ {
		db, err := Open(context.Background(), path)
		if err != nil {
			t.Fatal(err)
		}
		var count int
		err = db.Get(&count, `SELECT count(*) FROM sqlite_master WHERE type='table'
			AND name IN ('database_resources','linux_resources','kubernetes_resources',
			'operations','pending_operations','pending_actions','api_tokens')`)
		if err != nil || count != 7 {
			t.Fatalf("schema incomplete: %d %v", count, err)
		}
		if err := db.Get(&count, `SELECT count(*) FROM goose_db_version WHERE is_applied=1 AND version_id>0`); err != nil || count != 2 {
			t.Fatalf("migration history: %d %v", count, err)
		}
		db.Close()
	}
	backups, _ := filepath.Glob(path + ".pre-migrate-*.db")
	if len(backups) != 0 {
		t.Fatal("fresh or unchanged database should not create a backup")
	}
}

func TestSQLMigrationRollback(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, filepath.Join(t.TempDir(), "gateway.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	initial, err := migrationFiles.ReadFile("migrations/00001_initial.sql")
	if err != nil {
		t.Fatal(err)
	}
	indexes, err := migrationFiles.ReadFile("migrations/00002_indexes.sql")
	if err != nil {
		t.Fatal(err)
	}
	files := fstest.MapFS{
		"00001_initial.sql": {Data: initial},
		"00002_indexes.sql": {Data: indexes},
		"00003_failure.sql": {Data: []byte("-- +goose Up\nCREATE TABLE partial_upgrade(id INTEGER);\nINSERT INTO missing_table VALUES(1);\n")},
	}
	p, err := goose.NewProvider(goose.DialectSQLite3, db.DB, files,
		goose.WithDisableGlobalRegistry(true),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Up(ctx); err == nil {
		t.Fatal("invalid migration succeeded")
	}
	var count int
	if err := db.Get(&count, `SELECT count(*) FROM sqlite_master WHERE name='partial_upgrade'`); err != nil || count != 0 {
		t.Fatalf("partial schema persisted: %d %v", count, err)
	}
	if err := db.Get(&count, `SELECT max(version_id) FROM goose_db_version WHERE is_applied=1`); err != nil || count != 2 {
		t.Fatalf("failed version recorded: %d %v", count, err)
	}
}

func TestCanceledStartup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if db, err := Open(ctx, filepath.Join(t.TempDir(), "gateway.db")); err == nil {
		db.Close()
		t.Fatal("canceled startup succeeded")
	}
}

func TestVersionedUpgrade(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "gateway.db")
	db, err := sqlx.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	p, err := provider(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.UpTo(ctx, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO kubernetes_resources(name,kubeconfig,token) VALUES('cluster','config','test-token')`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	db, err = Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	var version int
	if err := db.Get(&version, `SELECT max(version_id) FROM goose_db_version WHERE is_applied=1`); err != nil || version != 2 {
		t.Fatalf("upgrade failed: %d %v", version, err)
	}
	var token string
	if err := db.Get(&token, `SELECT token FROM kubernetes_resources`); err != nil || token != "test-token" {
		t.Fatalf("upgrade changed token: %v", err)
	}
	backups, _ := filepath.Glob(path + ".pre-migrate-*.db")
	if len(backups) != 1 {
		t.Fatal("versioned upgrade must create a backup")
	}
	info, err := os.Stat(backups[0])
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("backup permissions: %v", err)
	}
	snapshot, err := sqlx.Open("sqlite", backups[0])
	if err != nil {
		t.Fatal(err)
	}
	defer snapshot.Close()
	if err := snapshot.Get(&version, "SELECT max(version_id) FROM goose_db_version WHERE is_applied=1"); err != nil || version != 1 {
		t.Fatalf("backup version=%d err=%v", version, err)
	}
	if err := snapshot.Get(&token, "SELECT token FROM kubernetes_resources"); err != nil || token != "test-token" {
		t.Fatalf("backup data lost: %v", err)
	}
	db.Close()
	db, err = Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	backups, _ = filepath.Glob(path + ".pre-migrate-*.db")
	if len(backups) != 1 {
		t.Fatal("repeated startup created another backup")
	}
}
