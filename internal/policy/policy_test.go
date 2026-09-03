package policy

import "testing"

func TestEvaluateSQL(t *testing.T) {
	tests := []struct {
		name, sql string
		decision  Decision
	}{
		{"select", "SELECT * FROM users", DecisionAllow},
		{"update where", "UPDATE users\nSET active=1 WHERE id=1", DecisionConfirm},
		{"update without where", "UPDATE users SET active=1", DecisionDeny},
		{"delete where", "DELETE FROM users WHERE id=1", DecisionConfirm},
		{"delete without where", "DELETE FROM users", DecisionDeny},
		{"drop", "DROP TABLE users", DecisionDeny},
		{"truncate", "TRUNCATE users", DecisionDeny},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EvaluateSQL(tt.sql).Decision; got != tt.decision {
				t.Fatalf("got %s, want %s", got, tt.decision)
			}
		})
	}
}

func TestEvaluateSQLByDialect(t *testing.T) {
	for _, tt := range []struct {
		dialect, sql string
		want         Decision
	}{
		{"postgres", "WITH x AS (DELETE FROM users WHERE id=1 RETURNING *) SELECT * FROM x", DecisionDeny},
		{"postgres", "SELECT 1; DELETE FROM users WHERE id=1", DecisionDeny},
		{"mysql", "SELECT 1; DELETE FROM users WHERE id=1", DecisionDeny},
		{"mysql", "SELECT * FROM users", DecisionAllow},
		{"mysql", "UPDATE users SET active=1 WHERE id=1", DecisionConfirm},
		{"mysql", "DELETE FROM users", DecisionDeny},
	} {
		if got := EvaluateSQLDialect(tt.dialect, tt.sql).Decision; got != tt.want {
			t.Errorf("%s %q: got %s want %s", tt.dialect, tt.sql, got, tt.want)
		}
	}
}

func TestEvaluateAction(t *testing.T) {
	if got := EvaluateAction("linux", "restart_service").Decision; got != DecisionConfirm {
		t.Fatalf("restart: got %s", got)
	}
	if got := EvaluateAction("linux", "exec").Decision; got != DecisionDeny {
		t.Fatalf("unknown action: got %s", got)
	}
}

func TestProductionPolicy(t *testing.T) {
	if got := EvaluateSQLForEnvironment("prod", "postgres", "INSERT INTO t VALUES (1)").Decision; got != DecisionDeny {
		t.Fatalf("prod insert: got %s", got)
	}
	if got := EvaluateSQLForEnvironment("prod", "postgres", "UPDATE t SET v=1 WHERE id=1").Decision; got != DecisionConfirm {
		t.Fatalf("prod scoped update: got %s", got)
	}
}
