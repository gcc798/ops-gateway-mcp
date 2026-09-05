package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gcc798/ai-ops-gateway/internal/audit"
	"github.com/gcc798/ai-ops-gateway/internal/auth"
	"github.com/gcc798/ai-ops-gateway/internal/database"
	kube "github.com/gcc798/ai-ops-gateway/internal/kubernetes"
	linux "github.com/gcc798/ai-ops-gateway/internal/linux"
	"github.com/gcc798/ai-ops-gateway/internal/policy"
	"github.com/gcc798/ai-ops-gateway/internal/resources"
	"github.com/google/uuid"
)

type SQLResult struct {
	OperationID string        `json:"operation_id,omitempty"`
	Policy      policy.Result `json:"policy"`
	Status      string        `json:"status"`
}
type ActionResult struct {
	OperationID string        `json:"operation_id"`
	Policy      policy.Result `json:"policy"`
	Status      string        `json:"status"`
}
type Service struct {
	Resources  *resources.Store
	resourceMu sync.Mutex
	revisions  map[string]string
	Audit      *audit.Store
	Databases  *database.Manager
	Kubernetes *kube.Manager
	Linux      *linux.Manager
}

func (s *Service) LinuxClient(name string) (*linux.Client, error) {
	if s.Linux == nil {
		return nil, errors.New("linux manager unavailable")
	}
	if err := s.syncResource("linux", name); err != nil {
		return nil, err
	}
	c, ok := s.Linux.Get(name)
	if !ok {
		return nil, fmt.Errorf("host %q not found", name)
	}
	return c, nil
}

func (s *Service) KubernetesClient(name string) (*kube.Client, error) {
	if s.Kubernetes == nil {
		return nil, errors.New("kubernetes manager unavailable")
	}
	if err := s.syncResource("kubernetes", name); err != nil {
		return nil, err
	}
	c, ok := s.Kubernetes.Get(name)
	if !ok {
		return nil, fmt.Errorf("cluster %q not found", name)
	}
	return c, nil
}

func (s *Service) Database(name string) (*database.Adapter, error) {
	if s.Databases == nil {
		return nil, errors.New("database manager unavailable")
	}
	if err := s.syncResource("database", name); err != nil {
		return nil, err
	}
	db, ok := s.Databases.Get(name)
	if !ok {
		return nil, fmt.Errorf("database %q not found", name)
	}
	return db, nil
}

func (s *Service) EvaluateSQL(ctx context.Context, client, environment, resource, statement string) (SQLResult, error) {
	if strings.TrimSpace(statement) == "" {
		return SQLResult{}, errors.New("statement is required")
	}
	db, err := s.Database(resource)
	if err != nil {
		return SQLResult{}, err
	}
	metadata, err := s.operationResource(ctx, "database", resource)
	if err != nil {
		return SQLResult{}, err
	}
	environment = metadata.Environment
	client = auth.Identity(ctx)
	result := policy.EvaluateSQLForEnvironment(environment, db.Driver(), statement)
	out := SQLResult{OperationID: uuid.NewString(), Policy: result, Status: string(result.Decision)}
	operation := audit.Operation{
		ID: out.OperationID, RequestID: audit.RequestID(ctx),
		Client: client, Tool: "db_prepare_execute",
		Environment: environment, ResourceType: "database", Resource: resource,
		Action: "sql", Status: "evaluated", Timestamp: time.Now(),
		Risk: string(result.Risk), Decision: string(result.Decision), Reason: result.Reason,
		StatementHash: statementHash(statement), ResourceRevision: metadata.Revision(),
	}
	if result.Decision == policy.DecisionConfirm {
		if s.Audit != nil {
			operation.Status = "pending"
			err := s.Audit.Prepare(ctx, operation, audit.Pending{
				OperationID: out.OperationID, Resource: resource, Dialect: db.Driver(),
				Statement: statement, StatementHash: operation.StatementHash,
				ExpiresAt: time.Now().Add(10 * time.Minute),
			})
			if err != nil {
				return SQLResult{}, err
			}
		}
	} else if s.Audit != nil {
		if err := s.Audit.Record(ctx, operation); err != nil {
			return SQLResult{}, err
		}
	}
	return out, nil
}

func (s *Service) QuerySQL(ctx context.Context, databaseName, statement string) (rows []map[string]any, err error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	started := time.Now()
	tool := audit.Tool(ctx)
	if tool == "" {
		tool = "db_query"
	}
	op := audit.Operation{
		ID: uuid.NewString(), Timestamp: started, RequestID: audit.RequestID(ctx),
		Client: auth.Identity(ctx), Tool: tool,
		ResourceType: "database", Resource: databaseName,
		Action: "query", StatementHash: statementHash(statement),
	}
	defer func() {
		op.Status = "succeeded"
		if err != nil {
			op.Status = "failed"
			op.Error = "database query failed"
		}
		op.DurationMS = time.Since(started).Milliseconds()
		if s.Audit != nil {
			finishCtx, finishCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer finishCancel()
			if auditErr := s.Audit.Record(finishCtx, op); auditErr != nil {
				rows = nil
				err = errors.New("audit recording failed")
			}
		}
	}()
	db, err := s.Database(databaseName)
	if err != nil {
		return nil, err
	}
	metadata, err := s.operationResource(ctx, "database", databaseName)
	if err != nil {
		return nil, err
	}
	op.Environment = metadata.Environment
	decision := policy.EvaluateSQLForEnvironment(metadata.Environment, db.Driver(), statement)
	op.Decision, op.Risk, op.Reason = string(decision.Decision), string(decision.Risk), decision.Reason
	if decision.Decision != policy.DecisionAllow {
		return nil, errors.New("only one AST-verified read-only SQL statement is allowed")
	}
	rows, err = db.Query(ctx, statement)
	if err != nil {
		return nil, errors.New("database query failed")
	}
	return rows, nil
}
func statementHash(statement string) string {
	sum := sha256.Sum256([]byte(statement))
	return hex.EncodeToString(sum[:])
}
func (s *Service) ConfirmSQL(ctx context.Context, operationID string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if operationID == "" {
		return errors.New("operation_id is required")
	}
	if s.Audit == nil {
		return errors.New("audit store unavailable")
	}
	op, err := s.Audit.Get(ctx, operationID)
	if err != nil {
		return err
	}
	if op.Status != "pending" {
		return fmt.Errorf("operation %s is not pending", operationID)
	}
	pending, err := s.Audit.Pending(ctx, operationID)
	if err != nil {
		return err
	}
	if time.Now().After(pending.ExpiresAt) {
		_ = s.Audit.Complete(ctx, operationID, "expired", 0)
		return errors.New("operation expired")
	}
	if statementHash(pending.Statement) != pending.StatementHash {
		return errors.New("statement integrity check failed")
	}
	metadata, err := s.operationResource(ctx, "database", pending.Resource)
	if err != nil {
		return err
	}
	if op.ResourceRevision == "" || op.ResourceRevision != metadata.Revision() {
		return errors.New("resource changed; prepare a new operation")
	}
	result := policy.EvaluateSQLForEnvironment(metadata.Environment, pending.Dialect, pending.Statement)
	if result.Decision != policy.DecisionConfirm {
		return errors.New("operation is no longer confirmable")
	}
	db, err := s.Database(pending.Resource)
	if err != nil {
		return err
	}
	if db.Driver() != pending.Dialect {
		return errors.New("resource dialect changed")
	}
	if err := s.Audit.ClaimBy(ctx, operationID, auth.Identity(ctx)); err != nil {
		return err
	}
	started := time.Now()
	rows, err := db.Exec(ctx, pending.Statement)
	duration := time.Since(started).Milliseconds()
	finishCtx, finishCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer finishCancel()
	if err != nil {
		return errors.Join(errors.New("database execution failed"), s.Audit.CompleteResult(finishCtx, operationID, "failed", duration, 0, "database execution failed"))
	}
	return s.Audit.CompleteResult(finishCtx, operationID, "succeeded", duration, rows, "")
}

func (s *Service) PrepareAction(ctx context.Context, client, environment, resourceType, resource, action, target string) (ActionResult, error) {
	metadata, err := s.operationResource(ctx, resourceType, resource)
	if err != nil {
		return ActionResult{}, err
	}
	environment = metadata.Environment
	client = auth.Identity(ctx)
	result := policy.EvaluateActionForEnvironment(environment, resourceType, action)
	out := ActionResult{Policy: result, Status: string(result.Decision)}
	if result.Decision != policy.DecisionConfirm {
		return out, nil
	}
	switch resourceType {
	case "linux":
		if _, err := s.LinuxClient(resource); err != nil {
			return ActionResult{}, err
		}
		if !linux.ValidServiceName(target) {
			return ActionResult{}, errors.New("invalid service name")
		}
	case "kubernetes":
		cluster, err := s.KubernetesClient(resource)
		if err != nil {
			return ActionResult{}, err
		}
		namespace, deployment, ok := strings.Cut(target, "/")
		if !ok || namespace == "" || deployment == "" {
			return ActionResult{}, errors.New("target must be namespace/deployment")
		}
		if _, err := cluster.Deployment(ctx, namespace, deployment); err != nil {
			return ActionResult{}, err
		}
	default:
		return out, nil
	}
	if s.Audit == nil {
		return ActionResult{}, errors.New("audit store unavailable")
	}
	out.OperationID = uuid.NewString()
	err = s.Audit.PrepareAction(ctx, audit.Operation{
		ID: out.OperationID, RequestID: audit.RequestID(ctx),
		Client: client, Tool: resourceType + "_" + action,
		Environment: environment, ResourceType: resourceType, Resource: resource,
		Action: action, Target: target, Status: "pending", Timestamp: time.Now(),
		Risk: string(result.Risk), Decision: string(result.Decision), Reason: result.Reason,
		ResourceRevision: metadata.Revision(),
	}, audit.PendingAction{
		OperationID: out.OperationID, ResourceType: resourceType, Resource: resource,
		Action: action, Target: target, ExpiresAt: time.Now().Add(10 * time.Minute),
	})
	return out, err
}

func (s *Service) ConfirmAction(ctx context.Context, operationID string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if operationID == "" || s.Audit == nil {
		return errors.New("valid operation_id and audit store are required")
	}
	pending, err := s.Audit.PendingAction(ctx, operationID)
	if err != nil {
		return err
	}
	if time.Now().After(pending.ExpiresAt) {
		_ = s.Audit.Complete(ctx, operationID, "expired", 0)
		return errors.New("operation expired")
	}
	op, err := s.Audit.Get(ctx, operationID)
	if err != nil {
		return err
	}
	metadata, err := s.operationResource(ctx, pending.ResourceType, pending.Resource)
	if err != nil {
		return err
	}
	if op.ResourceRevision == "" || op.ResourceRevision != metadata.Revision() {
		return errors.New("resource changed; prepare a new operation")
	}
	if policy.EvaluateActionForEnvironment(metadata.Environment, pending.ResourceType, pending.Action).Decision != policy.DecisionConfirm {
		return errors.New("operation is no longer confirmable")
	}
	if err := s.Audit.ClaimBy(ctx, operationID, auth.Identity(ctx)); err != nil {
		return err
	}
	started := time.Now()
	switch pending.ResourceType {
	case "linux":
		host, getErr := s.LinuxClient(pending.Resource)
		if getErr != nil {
			err = getErr
		} else {
			err = host.RestartService(ctx, pending.Target)
		}
	case "kubernetes":
		cluster, getErr := s.KubernetesClient(pending.Resource)
		namespace, deployment, ok := strings.Cut(pending.Target, "/")
		if getErr != nil {
			err = getErr
		} else if !ok {
			err = errors.New("invalid frozen target")
		} else {
			err = cluster.RolloutRestart(ctx, namespace, deployment)
		}
	default:
		err = errors.New("unsupported frozen action")
	}
	status := "succeeded"
	if err != nil {
		status = "failed"
	}
	errorMessage := ""
	if err != nil {
		errorMessage = "operation execution failed"
	}
	finishCtx, finishCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer finishCancel()
	if auditErr := s.Audit.CompleteResult(finishCtx, operationID, status, time.Since(started).Milliseconds(), 0, errorMessage); auditErr != nil && err == nil {
		return auditErr
	}
	return err
}

func (s *Service) ConfirmOperation(ctx context.Context, operationID string) error {
	if s.Audit == nil {
		return errors.New("audit store unavailable")
	}
	op, err := s.Audit.Get(ctx, operationID)
	if err != nil {
		return err
	}
	if op.Action == "sql" {
		return s.ConfirmSQL(ctx, operationID)
	}
	return s.ConfirmAction(ctx, operationID)
}
