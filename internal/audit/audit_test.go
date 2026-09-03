package audit

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

func TestPendingLifecycle(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "audit.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	op := Operation{ID: "op-1", Status: "pending", Timestamp: time.Now()}
	pending := Pending{OperationID: op.ID, Resource: "db", Dialect: "postgres", Statement: "UPDATE t SET v=1 WHERE id=1", StatementHash: "hash", ExpiresAt: time.Now().Add(time.Minute)}
	if err := store.Prepare(ctx, op, pending); err != nil {
		t.Fatal(err)
	}
	if got, err := store.Pending(ctx, op.ID); err != nil || got.Statement != pending.Statement {
		t.Fatalf("pending=%#v err=%v", got, err)
	}
	if err := store.Claim(ctx, op.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.Claim(ctx, op.ID); err == nil {
		t.Fatal("operation must only be claimed once")
	}
	if err := store.Complete(ctx, op.ID, "succeeded", 7); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(ctx, op.ID)
	if err != nil || got.Status != "succeeded" || got.DurationMS != 7 {
		t.Fatalf("operation=%#v err=%v", got, err)
	}
	if err := store.Complete(ctx, op.ID, "succeeded", 8); err == nil {
		t.Fatal("completed operation must not execute twice")
	}
}

func TestOpenMigratesLegacyOperations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE operations(operation_id TEXT PRIMARY KEY,timestamp TEXT NOT NULL,client TEXT,environment TEXT,resource TEXT,action TEXT,risk TEXT,policy_decision TEXT,status TEXT,duration_ms INTEGER);INSERT INTO operations VALUES('old','2026-01-01T00:00:00Z','','','','','','','succeeded',0)`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if operations, err := store.List(context.Background(), 1); err != nil || len(operations) != 1 {
		t.Fatalf("operations=%#v err=%v", operations, err)
	}
}

func TestPendingActionLifecycle(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "audit.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	op := Operation{ID: "restart-1", Resource: "dev", Action: "rollout_restart", Status: "pending", Timestamp: time.Now()}
	pending := PendingAction{OperationID: op.ID, ResourceType: "kubernetes", Resource: "dev", Action: op.Action, Target: "default/api", ExpiresAt: time.Now().Add(time.Minute)}
	if err := store.PrepareAction(ctx, op, pending); err != nil {
		t.Fatal(err)
	}
	if got, err := store.PendingAction(ctx, op.ID); err != nil || got.Target != pending.Target {
		t.Fatalf("pending=%#v err=%v", got, err)
	}
	if err := store.Claim(ctx, op.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.Complete(ctx, op.ID, "succeeded", 2); err != nil {
		t.Fatal(err)
	}
}
