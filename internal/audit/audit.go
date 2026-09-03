package audit

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type Operation struct {
	ID            string    `json:"operation_id"`
	Timestamp     time.Time `json:"timestamp"`
	Client        string    `json:"client"`
	Tool          string    `json:"tool"`
	Environment   string    `json:"environment"`
	ResourceType  string    `json:"resource_type"`
	Resource      string    `json:"resource"`
	Action        string    `json:"action"`
	Target        string    `json:"target"`
	Risk          string    `json:"risk"`
	Decision      string    `json:"decision"`
	Reason        string    `json:"reason"`
	Status        string    `json:"status"`
	DurationMS    int64     `json:"duration_ms"`
	AffectedRows  int64     `json:"affected_rows"`
	StatementHash string    `json:"statement_hash,omitempty"`
	Error         string    `json:"error,omitempty"`
}
type Pending struct {
	OperationID, Resource, Dialect, Statement, StatementHash string
	ExpiresAt                                                time.Time
}
type PendingAction struct {
	OperationID, ResourceType, Resource, Action, Target string
	ExpiresAt                                           time.Time
}
type Store struct{ db *sql.DB }

const operationInsert = `INSERT INTO operations(operation_id,timestamp,client,tool,environment,resource_type,resource,action,target,risk,policy_decision,policy_reason,status,duration_ms,affected_rows,statement_hash,error) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`
const operationSelect = `SELECT operation_id,timestamp,client,tool,environment,resource_type,resource,action,target,risk,policy_decision,policy_reason,status,duration_ms,affected_rows,statement_hash,error FROM operations`

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open gateway db: %w", err)
	}
	db.SetMaxOpenConns(1)
	schema := `CREATE TABLE IF NOT EXISTS operations(operation_id TEXT PRIMARY KEY,timestamp TEXT NOT NULL,client TEXT,tool TEXT,environment TEXT,resource_type TEXT,resource TEXT,action TEXT,target TEXT,risk TEXT,policy_decision TEXT,policy_reason TEXT,status TEXT,duration_ms INTEGER,affected_rows INTEGER,statement_hash TEXT,error TEXT);CREATE TABLE IF NOT EXISTS pending_operations(operation_id TEXT PRIMARY KEY,resource TEXT NOT NULL,dialect TEXT NOT NULL,statement TEXT NOT NULL,statement_hash TEXT NOT NULL,expires_at TEXT NOT NULL,FOREIGN KEY(operation_id) REFERENCES operations(operation_id));CREATE TABLE IF NOT EXISTS pending_actions(operation_id TEXT PRIMARY KEY,resource_type TEXT NOT NULL,resource TEXT NOT NULL,action TEXT NOT NULL,target TEXT NOT NULL,expires_at TEXT NOT NULL,FOREIGN KEY(operation_id) REFERENCES operations(operation_id));`
	if _, err = db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("init audit schema: %w", err)
	}
	if err = ensureOperationColumns(db); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db}, nil
}

func (s *Store) Close() error { return s.db.Close() }
func (s *Store) Record(ctx context.Context, operation Operation) error {
	if _, err := s.db.ExecContext(ctx, operationInsert, operationValues(operation)...); err != nil {
		return fmt.Errorf("record audit operation: %w", err)
	}
	return nil
}
func (s *Store) Prepare(ctx context.Context, operation Operation, pending Pending) error {
	return s.prepare(ctx, operation, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO pending_operations(operation_id,resource,dialect,statement,statement_hash,expires_at) VALUES(?,?,?,?,?,?)`, pending.OperationID, pending.Resource, pending.Dialect, pending.Statement, pending.StatementHash, pending.ExpiresAt.UTC().Format(time.RFC3339Nano))
		return err
	})
}
func (s *Store) PrepareAction(ctx context.Context, operation Operation, pending PendingAction) error {
	return s.prepare(ctx, operation, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO pending_actions(operation_id,resource_type,resource,action,target,expires_at) VALUES(?,?,?,?,?,?)`, pending.OperationID, pending.ResourceType, pending.Resource, pending.Action, pending.Target, pending.ExpiresAt.UTC().Format(time.RFC3339Nano))
		return err
	})
}
func (s *Store) prepare(ctx context.Context, operation Operation, freeze func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, operationInsert, operationValues(operation)...); err != nil {
		return fmt.Errorf("record pending operation: %w", err)
	}
	if err = freeze(tx); err != nil {
		return fmt.Errorf("freeze pending operation: %w", err)
	}
	return tx.Commit()
}
func (s *Store) Pending(ctx context.Context, id string) (Pending, error) {
	var pending Pending
	var expires string
	err := s.db.QueryRowContext(ctx, `SELECT operation_id,resource,dialect,statement,statement_hash,expires_at FROM pending_operations WHERE operation_id=?`, id).Scan(&pending.OperationID, &pending.Resource, &pending.Dialect, &pending.Statement, &pending.StatementHash, &expires)
	if err != nil {
		return pending, fmt.Errorf("get pending operation: %w", err)
	}
	pending.ExpiresAt, err = time.Parse(time.RFC3339Nano, expires)
	return pending, err
}
func (s *Store) PendingAction(ctx context.Context, id string) (PendingAction, error) {
	var pending PendingAction
	var expires string
	err := s.db.QueryRowContext(ctx, `SELECT operation_id,resource_type,resource,action,target,expires_at FROM pending_actions WHERE operation_id=?`, id).Scan(&pending.OperationID, &pending.ResourceType, &pending.Resource, &pending.Action, &pending.Target, &expires)
	if err != nil {
		return pending, fmt.Errorf("get pending action: %w", err)
	}
	pending.ExpiresAt, err = time.Parse(time.RFC3339Nano, expires)
	return pending, err
}
func (s *Store) Claim(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `UPDATE operations SET status='executing' WHERE operation_id=? AND status='pending'`, id)
	if err != nil {
		return fmt.Errorf("claim operation: %w", err)
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		return fmt.Errorf("claim operation: operation is not pending")
	}
	return nil
}
func (s *Store) Complete(ctx context.Context, id, status string, durationMS int64) error {
	return s.CompleteResult(ctx, id, status, durationMS, 0, "")
}
func (s *Store) CompleteResult(ctx context.Context, id, status string, durationMS, affectedRows int64, errorMessage string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE operations SET status=?,duration_ms=?,affected_rows=?,error=? WHERE operation_id=? AND status IN ('pending','executing')`, status, durationMS, affectedRows, errorMessage, id)
	if err != nil {
		return fmt.Errorf("complete audit operation: %w", err)
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		return fmt.Errorf("complete audit operation: operation is not pending")
	}
	for _, table := range []string{"pending_operations", "pending_actions"} {
		if _, err = tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE operation_id=?`, id); err != nil {
			return fmt.Errorf("remove pending operation: %w", err)
		}
	}
	return tx.Commit()
}
func (s *Store) List(ctx context.Context, limit int) ([]Operation, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, operationSelect+` ORDER BY timestamp DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list audit operations: %w", err)
	}
	defer rows.Close()
	operations := []Operation{}
	for rows.Next() {
		operation, err := scanOperation(rows)
		if err != nil {
			return nil, err
		}
		operations = append(operations, operation)
	}
	return operations, rows.Err()
}
func (s *Store) Get(ctx context.Context, id string) (Operation, error) {
	return scanOperation(s.db.QueryRowContext(ctx, operationSelect+` WHERE operation_id=?`, id))
}

type scanner interface{ Scan(...any) error }

func scanOperation(row scanner) (Operation, error) {
	var operation Operation
	var timestamp string
	if err := row.Scan(&operation.ID, &timestamp, &operation.Client, &operation.Tool, &operation.Environment, &operation.ResourceType, &operation.Resource, &operation.Action, &operation.Target, &operation.Risk, &operation.Decision, &operation.Reason, &operation.Status, &operation.DurationMS, &operation.AffectedRows, &operation.StatementHash, &operation.Error); err != nil {
		return operation, fmt.Errorf("scan audit operation: %w", err)
	}
	operation.Timestamp, _ = time.Parse(time.RFC3339Nano, timestamp)
	return operation, nil
}
func operationValues(operation Operation) []any {
	return []any{operation.ID, operation.Timestamp.UTC().Format(time.RFC3339Nano), operation.Client, operation.Tool, operation.Environment, operation.ResourceType, operation.Resource, operation.Action, operation.Target, operation.Risk, operation.Decision, operation.Reason, operation.Status, operation.DurationMS, operation.AffectedRows, operation.StatementHash, operation.Error}
}
func ensureOperationColumns(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(operations)`)
	if err != nil {
		return fmt.Errorf("inspect audit schema: %w", err)
	}
	columns := map[string]bool{}
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, kind string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &kind, &notNull, &defaultValue, &primaryKey); err != nil {
			rows.Close()
			return err
		}
		columns[name] = true
	}
	rows.Close()
	for name, definition := range map[string]string{"tool": "TEXT NOT NULL DEFAULT ''", "resource_type": "TEXT NOT NULL DEFAULT ''", "target": "TEXT NOT NULL DEFAULT ''", "policy_reason": "TEXT NOT NULL DEFAULT ''", "affected_rows": "INTEGER NOT NULL DEFAULT 0", "statement_hash": "TEXT NOT NULL DEFAULT ''", "error": "TEXT NOT NULL DEFAULT ''"} {
		if !columns[name] {
			if _, err := db.Exec(`ALTER TABLE operations ADD COLUMN ` + name + ` ` + definition); err != nil {
				return fmt.Errorf("migrate audit schema: %w", err)
			}
		}
	}
	return nil
}
