package policy

import (
	"fmt"
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v6"
	"google.golang.org/protobuf/reflect/protoreflect"
	"vitess.io/vitess/go/vt/sqlparser"
)

type Decision string

const (
	DecisionAllow   Decision = "allow"
	DecisionConfirm Decision = "confirm"
	DecisionDeny    Decision = "deny"
)

type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

type Result struct {
	Decision Decision  `json:"decision"`
	Risk     RiskLevel `json:"risk"`
	Reason   string    `json:"reason"`
	RuleID   string    `json:"rule_id"`
}

func EvaluateSQL(sql string) Result { return EvaluateSQLDialect("postgres", sql) }
func EvaluateSQLForEnvironment(environment, dialect, sql string) Result {
	result := EvaluateSQLDialect(dialect, sql)
	if strings.EqualFold(environment, "prod") && result.Decision == DecisionConfirm && result.RuleID == "sql.mutation" {
		return deny("schema and insert mutations are prohibited in prod", "sql.prod_mutation")
	}
	return result
}
func EvaluateAction(resourceType, action string) Result {
	if resourceType == "linux" && action == "restart_service" || resourceType == "kubernetes" && action == "rollout_restart" {
		return confirm("restart requires confirmation", RiskHigh, resourceType+".restart")
	}
	return deny("action is not allowlisted", "action.default_deny")
}
func EvaluateActionForEnvironment(environment, resourceType, action string) Result {
	result := EvaluateAction(resourceType, action)
	if strings.EqualFold(environment, "prod") && result.Decision == DecisionConfirm {
		result.Risk = RiskHigh
		result.Reason = "production restart requires explicit confirmation"
		result.RuleID = result.RuleID + ".prod"
	}
	return result
}
func EvaluateSQLDialect(dialect, sql string) Result {
	if strings.TrimSpace(sql) == "" {
		return deny("empty statement", "sql.invalid")
	}
	switch dialect {
	case "postgres":
		return evaluatePostgres(sql)
	case "mysql":
		return evaluateMySQL(sql)
	default:
		return deny("unsupported SQL dialect", "sql.dialect")
	}
}

func evaluatePostgres(sql string) Result {
	tree, err := pg_query.Parse(sql)
	if err != nil {
		return deny("invalid PostgreSQL statement", "sql.invalid")
	}
	if len(tree.Stmts) != 1 {
		return deny("exactly one statement is required", "sql.multiple")
	}
	n := tree.Stmts[0].Stmt
	kind := nodeKind(n.ProtoReflect())
	switch kind {
	case "SelectStmt":
		if containsMutation(n.ProtoReflect()) {
			return deny("data-modifying CTE is prohibited", "sql.cte_write")
		}
		return allow("read-only statement", "sql.read")
	case "VariableShowStmt":
		return allow("read-only statement", "sql.read")
	case "ExplainStmt":
		if containsMutation(n.GetExplainStmt().GetQuery().ProtoReflect()) {
			return deny("EXPLAIN of a mutation is prohibited", "sql.explain_write")
		}
		return allow("read-only explain", "sql.explain")
	case "UpdateStmt":
		if n.GetUpdateStmt().GetWhereClause() == nil {
			return deny("UPDATE without WHERE is prohibited", "sql.no_where")
		}
		return confirm("write requires confirmation", RiskMedium, "sql.write")
	case "DeleteStmt":
		if n.GetDeleteStmt().GetWhereClause() == nil {
			return deny("DELETE without WHERE is prohibited", "sql.no_where")
		}
		return confirm("write requires confirmation", RiskMedium, "sql.write")
	case "InsertStmt":
		return confirm("data mutation requires confirmation", RiskHigh, "sql.mutation")
	case "DropStmt", "TruncateStmt":
		return deny("destructive statement is prohibited", "sql.destructive")
	default:
		if strings.HasPrefix(kind, "Create") || strings.HasPrefix(kind, "Alter") {
			return confirm("schema mutation requires confirmation", RiskHigh, "sql.mutation")
		}
		return deny(fmt.Sprintf("PostgreSQL AST node %s is not allowlisted", kind), "sql.default_deny")
	}
}

func evaluateMySQL(sql string) Result {
	p := sqlparser.NewTestParser()
	stmts, err := p.ParseMultipleIgnoreEmpty(sql)
	if err != nil {
		return deny("invalid MySQL statement", "sql.invalid")
	}
	if len(stmts) != 1 {
		return deny("exactly one statement is required", "sql.multiple")
	}
	switch n := stmts[0].(type) {
	case *sqlparser.Select:
		if n.Into != nil {
			return deny("SELECT INTO is prohibited", "sql.write")
		}
		return allow("read-only statement", "sql.read")
	case *sqlparser.Union, *sqlparser.Show, *sqlparser.ExplainStmt, *sqlparser.ExplainTab:
		return allow("read-only statement", "sql.read")
	case *sqlparser.Update:
		if n.Where == nil {
			return deny("UPDATE without WHERE is prohibited", "sql.no_where")
		}
		return confirm("write requires confirmation", RiskMedium, "sql.write")
	case *sqlparser.Delete:
		if n.Where == nil {
			return deny("DELETE without WHERE is prohibited", "sql.no_where")
		}
		return confirm("write requires confirmation", RiskMedium, "sql.write")
	case *sqlparser.Insert:
		return confirm("data mutation requires confirmation", RiskHigh, "sql.mutation")
	case *sqlparser.DropTable, *sqlparser.DropDatabase, *sqlparser.TruncateTable:
		return deny("destructive statement is prohibited", "sql.destructive")
	case *sqlparser.CreateTable, *sqlparser.CreateDatabase, *sqlparser.AlterTable, *sqlparser.AlterDatabase:
		return confirm("schema mutation requires confirmation", RiskHigh, "sql.mutation")
	default:
		return deny(fmt.Sprintf("MySQL AST node %T is not allowlisted", n), "sql.default_deny")
	}
}

func nodeKind(m protoreflect.Message) string {
	kind := "Unknown"
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if fd.Kind() == protoreflect.MessageKind && m.Has(fd) {
			kind = string(v.Message().Descriptor().Name())
			return false
		}
		return true
	})
	return kind
}
func containsMutation(m protoreflect.Message) bool {
	bad := false
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if fd.IsList() && fd.Kind() == protoreflect.MessageKind {
			l := v.List()
			for i := 0; i < l.Len(); i++ {
				if mutationMessage(l.Get(i).Message()) || containsMutation(l.Get(i).Message()) {
					bad = true
					return false
				}
			}
		} else if fd.Kind() == protoreflect.MessageKind && m.Has(fd) {
			child := v.Message()
			if mutationMessage(child) || containsMutation(child) {
				bad = true
				return false
			}
		}
		return true
	})
	return bad
}
func mutationMessage(m protoreflect.Message) bool {
	switch m.Descriptor().Name() {
	case "InsertStmt", "UpdateStmt", "DeleteStmt", "MergeStmt", "CopyStmt", "CallStmt", "CreateTableAsStmt":
		return true
	}
	return false
}
func allow(reason, rule string) Result { return Result{DecisionAllow, RiskLow, reason, rule} }
func confirm(reason string, risk RiskLevel, rule string) Result {
	return Result{DecisionConfirm, risk, reason, rule}
}
func deny(reason, rule string) Result { return Result{DecisionDeny, RiskHigh, reason, rule} }
