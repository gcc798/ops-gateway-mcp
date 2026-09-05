package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gcc798/ops-gateway-mcp/internal/storage"
	"github.com/jmoiron/sqlx"
)

const (
	ScopeMCP     = "mcp"
	ScopeREST    = "rest"
	ScopeRESTMCP = "rest+mcp"
)

type Store struct{ db *sqlx.DB }

func Open(path string) (*Store, error) {
	db, err := storage.Open(context.Background(), path)
	if err != nil {
		return nil, err
	}
	return New(db), nil
}
func New(db *sqlx.DB) *Store  { return &Store{db: db} }
func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Upsert(ctx context.Context, name, token, scope string) error {
	if token == "" {
		_, err := s.db.ExecContext(ctx, `UPDATE api_tokens SET revoked_at=? WHERE name=?`, time.Now().UTC().Format(time.RFC3339Nano), name)
		return err
	}
	if scope != ScopeMCP && scope != ScopeREST && scope != ScopeRESTMCP {
		return fmt.Errorf("invalid token scope %q", scope)
	}
	hash := hashToken(token)
	_, err := s.db.ExecContext(ctx, `
  INSERT INTO api_tokens(token_hash,name,scope,created_at) VALUES(?,?,?,?)
  ON CONFLICT(name) DO UPDATE SET
   token_hash=excluded.token_hash,scope=excluded.scope,revoked_at=NULL
 `, hash, name, scope, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) Authorize(ctx context.Context, token, requiredScope string) bool {
	_, ok := s.Authenticate(ctx, token, requiredScope)
	return ok
}

type identityKey struct{}

func WithIdentity(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, identityKey{}, name)
}
func Identity(ctx context.Context) string {
	name, _ := ctx.Value(identityKey{}).(string)
	return name
}
func (s *Store) Authenticate(ctx context.Context, token, requiredScope string) (string, bool) {
	if token == "" {
		return "", false
	}
	var identity struct {
		Scope string `db:"scope"`
		Name  string `db:"name"`
	}
	err := s.db.GetContext(ctx, &identity, `
 SELECT scope,name FROM api_tokens
 WHERE token_hash=? AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > ?)
 `, hashToken(token), time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil || !(identity.Scope == requiredScope || identity.Scope == ScopeRESTMCP) {
		return "", false
	}
	_, _ = s.db.ExecContext(ctx, `UPDATE api_tokens SET last_used_at=? WHERE token_hash=?`, time.Now().UTC().Format(time.RFC3339Nano), hashToken(token))
	return identity.Name, true
}

func (s *Store) HTTP(next http.Handler, scope string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name, ok := s.Authenticate(r.Context(), Bearer(r.Header.Get("Authorization")), scope)
		if !ok {
			w.Header().Set("WWW-Authenticate", `Bearer realm="ops-gateway-mcp"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(WithIdentity(r.Context(), name)))
	})
}

func Bearer(header string) string {
	parts := strings.Fields(header)
	if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
		return parts[1]
	}
	return ""
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
