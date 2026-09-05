package database

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestOpenRejectsInvalidConfig(t *testing.T) {
	if _, err := Open(Config{Driver: "oracle", DSN: "x"}); err == nil {
		t.Fatal("expected unsupported driver error")
	}
}

func TestIntegration(t *testing.T) {
	for _, test := range []struct{ name, driver, env string }{{"postgres", "postgres", "OPS_GATEWAY_MCP_PG_DSN"}, {"mysql", "mysql", "OPS_GATEWAY_MCP_MYSQL_DSN"}} {
		t.Run(test.name, func(t *testing.T) {
			dsn := os.Getenv(test.env)
			if dsn == "" {
				t.Skip("integration DSN not set")
			}
			db, err := Open(Config{Name: test.name, Driver: test.driver, DSN: dsn, MaxOpenConns: 1})
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := db.Ping(ctx); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Query(ctx, "SELECT 1 AS ok"); err != nil {
				t.Fatal(err)
			}
		})
	}
}
