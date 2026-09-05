package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestScopes(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "auth.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Upsert(context.Background(), "mcp", "secret-mcp", ScopeMCP); err != nil {
		t.Fatal(err)
	}
	if s.Authorize(context.Background(), "secret-mcp", ScopeREST) || !s.Authorize(context.Background(), "secret-mcp", ScopeMCP) {
		t.Fatal("mcp scope mismatch")
	}
	var stored string
	if err := s.db.QueryRow(`SELECT token_hash FROM api_tokens WHERE name='mcp'`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored == "secret-mcp" || stored == "" {
		t.Fatal("token must be stored as a hash")
	}
	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer secret-mcp")
	rr := httptest.NewRecorder()
	s.HTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if Identity(r.Context()) != "mcp" {
			t.Fatal("missing authenticated identity")
		}
		w.WriteHeader(http.StatusNoContent)
	}), ScopeMCP).ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status %d", rr.Code)
	}
	if err := s.Upsert(context.Background(), "mcp", "", ScopeMCP); err != nil {
		t.Fatal(err)
	}
	if s.Authorize(context.Background(), "secret-mcp", ScopeMCP) {
		t.Fatal("empty configured token must revoke the previous token")
	}
}
