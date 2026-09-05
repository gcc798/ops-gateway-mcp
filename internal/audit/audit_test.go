package audit

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestOperationFieldMapping(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "audit.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	want := Operation{
		ID: "all-fields", Timestamp: time.Now().UTC().Round(0),
		Client: "开发者", Tool: "db_query", Environment: "dev",
		ResourceType: "database", Resource: "示例库", Action: "sql",
		Target: "table", Risk: "low", Decision: "allow", Reason: "只读\n含引号'",
		Status: "succeeded", DurationMS: 123, AffectedRows: 7,
		StatementHash: "statement-hash", Error: "example error",
		ResourceRevision: "revision", RequestID: "request", ConfirmedBy: "reviewer",
	}
	ctx := context.Background()
	if err := store.Record(ctx, want); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(ctx, want.ID)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("field mapping mismatch: got=%+v err=%v", got, err)
	}
}

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
