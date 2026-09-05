package audit

import (
	"context"
	"fmt"
	"time"

	"github.com/gcc798/ops-gateway-mcp/internal/storage"
	"github.com/jmoiron/sqlx"
)

type Operation struct {
	ID               string    `json:"operation_id" db:"operation_id"`
	Timestamp        time.Time `json:"timestamp" db:"timestamp"`
	Client           string    `json:"client" db:"client"`
	Tool             string    `json:"tool" db:"tool"`
	Environment      string    `json:"environment" db:"environment"`
	ResourceType     string    `json:"resource_type" db:"resource_type"`
	Resource         string    `json:"resource" db:"resource"`
	Action           string    `json:"action" db:"action"`
	Target           string    `json:"target" db:"target"`
	Risk             string    `json:"risk" db:"risk"`
	Decision         string    `json:"decision" db:"policy_decision"`
	Reason           string    `json:"reason" db:"policy_reason"`
	Status           string    `json:"status" db:"status"`
	DurationMS       int64     `json:"duration_ms" db:"duration_ms"`
	AffectedRows     int64     `json:"affected_rows" db:"affected_rows"`
	StatementHash    string    `json:"statement_hash,omitempty" db:"statement_hash"`
	Error            string    `json:"error,omitempty" db:"error"`
	ResourceRevision string    `json:"resource_revision,omitempty" db:"resource_revision"`
	RequestID        string    `json:"request_id,omitempty" db:"request_id"`
	ConfirmedBy      string    `json:"confirmed_by,omitempty" db:"confirmed_by"`
}
type Pending struct {
	OperationID   string    `db:"operation_id"`
	Resource      string    `db:"resource"`
	Dialect       string    `db:"dialect"`
	Statement     string    `db:"statement"`
	StatementHash string    `db:"statement_hash"`
	ExpiresAt     time.Time `db:"expires_at"`
}
type PendingAction struct {
	OperationID  string    `db:"operation_id"`
	ResourceType string    `db:"resource_type"`
	Resource     string    `db:"resource"`
	Action       string    `db:"action"`
	Target       string    `db:"target"`
	ExpiresAt    time.Time `db:"expires_at"`
}
type Store struct{ db *sqlx.DB }

// SQLite 使用 RFC3339Nano 文本保存时间，内部行类型负责与 API 时间类型转换。
type operationRow struct {
	Operation
	Timestamp string `db:"timestamp"`
}
type pendingRow struct {
	Pending
	ExpiresAt string `db:"expires_at"`
}
type pendingActionRow struct {
	PendingAction
	ExpiresAt string `db:"expires_at"`
}

const operationInsert = `
 INSERT INTO operations(
  operation_id,timestamp,client,tool,environment,resource_type,resource,action,target,
  risk,policy_decision,policy_reason,status,duration_ms,affected_rows,
  statement_hash,error,resource_revision,request_id,confirmed_by
 ) VALUES(
  :operation_id,:timestamp,:client,:tool,:environment,:resource_type,:resource,:action,:target,
  :risk,:policy_decision,:policy_reason,:status,:duration_ms,:affected_rows,
  :statement_hash,:error,:resource_revision,:request_id,:confirmed_by
 )
`
const operationSelect = `
 SELECT operation_id,timestamp,client,tool,environment,resource_type,resource,action,target,
  risk,policy_decision,policy_reason,status,duration_ms,affected_rows,
  statement_hash,error,resource_revision,request_id,confirmed_by
 FROM operations
`

func Open(path string) (*Store, error) {
	db, err := storage.Open(context.Background(), path)
	if err != nil {
		return nil, err
	}
	return New(db), nil
}
func New(db *sqlx.DB) *Store { return &Store{db: db} }

func (s *Store) Close() error { return s.db.Close() }
func (s *Store) Record(ctx context.Context, operation Operation) error {
	row := operationRow{Operation: operation, Timestamp: operation.Timestamp.UTC().Format(time.RFC3339Nano)}
	if _, err := s.db.NamedExecContext(ctx, operationInsert, row); err != nil {
		return fmt.Errorf("record audit operation: %w", err)
	}
	return nil
}
func (s *Store) Prepare(ctx context.Context, operation Operation, pending Pending) error {
	return s.prepare(ctx, operation, func(tx *sqlx.Tx) error {
		_, err := tx.NamedExecContext(ctx, `
 INSERT INTO pending_operations(operation_id,resource,dialect,statement,statement_hash,expires_at)
 VALUES(:operation_id,:resource,:dialect,:statement,:statement_hash,:expires_at)
 `, pendingRow{Pending: pending, ExpiresAt: pending.ExpiresAt.UTC().Format(time.RFC3339Nano)})
		return err
	})
}
func (s *Store) PrepareAction(ctx context.Context, operation Operation, pending PendingAction) error {
	return s.prepare(ctx, operation, func(tx *sqlx.Tx) error {
		_, err := tx.NamedExecContext(ctx, `
 INSERT INTO pending_actions(operation_id,resource_type,resource,action,target,expires_at)
 VALUES(:operation_id,:resource_type,:resource,:action,:target,:expires_at)
 `, pendingActionRow{PendingAction: pending, ExpiresAt: pending.ExpiresAt.UTC().Format(time.RFC3339Nano)})
		return err
	})
}
func (s *Store) prepare(ctx context.Context, operation Operation, freeze func(*sqlx.Tx) error) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	row := operationRow{Operation: operation, Timestamp: operation.Timestamp.UTC().Format(time.RFC3339Nano)}
	if _, err = tx.NamedExecContext(ctx, operationInsert, row); err != nil {
		return fmt.Errorf("record pending operation: %w", err)
	}
	if err = freeze(tx); err != nil {
		return fmt.Errorf("freeze pending operation: %w", err)
	}
	return tx.Commit()
}
func (s *Store) Pending(ctx context.Context, id string) (Pending, error) {
	var row pendingRow
	if err := s.db.GetContext(ctx, &row, `
 SELECT operation_id,resource,dialect,statement,statement_hash,expires_at
 FROM pending_operations WHERE operation_id=?`, id); err != nil {
		return Pending{}, fmt.Errorf("get pending operation: %w", err)
	}
	var err error
	row.Pending.ExpiresAt, err = time.Parse(time.RFC3339Nano, row.ExpiresAt)
	return row.Pending, err
}
func (s *Store) PendingAction(ctx context.Context, id string) (PendingAction, error) {
	var row pendingActionRow
	if err := s.db.GetContext(ctx, &row, `
 SELECT operation_id,resource_type,resource,action,target,expires_at
 FROM pending_actions WHERE operation_id=?`, id); err != nil {
		return PendingAction{}, fmt.Errorf("get pending action: %w", err)
	}
	var err error
	row.PendingAction.ExpiresAt, err = time.Parse(time.RFC3339Nano, row.ExpiresAt)
	return row.PendingAction, err
}
func (s *Store) Claim(ctx context.Context, id string) error {
	return s.ClaimBy(ctx, id, "")
}
func (s *Store) ClaimBy(ctx context.Context, id, actor string) error {
	result, err := s.db.ExecContext(ctx, `UPDATE operations SET status='executing',confirmed_by=? WHERE operation_id=? AND status='pending'`, actor, id)
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
	tx, err := s.db.BeginTxx(ctx, nil)
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
	return selectOperations(ctx, s.db, operationSelect+` ORDER BY julianday(timestamp) DESC,operation_id DESC LIMIT ?`, limit)
}
func (s *Store) Get(ctx context.Context, id string) (Operation, error) {
	var row operationRow
	if err := s.db.GetContext(ctx, &row, operationSelect+` WHERE operation_id=?`, id); err != nil {
		return Operation{}, err
	}
	return row.operation()
}
func (r operationRow) operation() (Operation, error) {
	var err error
	r.Operation.Timestamp, err = time.Parse(time.RFC3339Nano, r.Timestamp)
	return r.Operation, err
}
func selectOperations(ctx context.Context, db sqlx.QueryerContext, query string, args ...any) ([]Operation, error) {
	var rows []operationRow
	if err := sqlx.SelectContext(ctx, db, &rows, query, args...); err != nil {
		return nil, err
	}
	operations := make([]Operation, 0, len(rows))
	for _, row := range rows {
		operation, err := row.operation()
		if err != nil {
			return nil, fmt.Errorf("read operation timestamp: %w", err)
		}
		operations = append(operations, operation)
	}
	return operations, nil
}
