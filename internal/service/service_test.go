package service

import (
	"context"
	"testing"

	"github.com/gcc798/ai-ops-gateway/internal/database"
	"github.com/gcc798/ai-ops-gateway/internal/policy"
)

func TestEvaluateSQLDecisions(t *testing.T) {
	m := database.NewManager()
	db, err := database.Open(database.Config{Name: "db", Driver: "postgres", DSN: "postgres://unused"})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = m.Add(db); err != nil {
		t.Fatal(err)
	}
	s := &Service{Databases: m}
	for _, tt := range []struct {
		sql  string
		want policy.Decision
	}{{"SELECT 1", policy.DecisionAllow}, {"UPDATE t SET x=1", policy.DecisionDeny}, {"UPDATE t SET x=1 WHERE id=1", policy.DecisionConfirm}} {
		got, err := s.EvaluateSQL(context.Background(), "test", "dev", "db", tt.sql)
		if err != nil {
			t.Fatal(err)
		}
		if got.Policy.Decision != tt.want {
			t.Fatalf("%s: got %s, want %s", tt.sql, got.Policy.Decision, tt.want)
		}
	}
}
