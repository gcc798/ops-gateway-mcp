package resources

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gcc798/ai-ops-gateway/internal/pagination"
	"github.com/gcc798/ai-ops-gateway/internal/storage"
	"github.com/jmoiron/sqlx"
)

type Paths struct{ Logs, Data string }

func ResolvePaths() Paths {
	base, _ := os.UserHomeDir()
	root := filepath.Join(base, ".ai-ops-gateway")
	value := func(key, fallback string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return fallback
	}
	return Paths{Logs: value("AI_OPS_GATEWAY_LOGS", filepath.Join(root, "logs")), Data: value("AI_OPS_GATEWAY_DATA", filepath.Join(root, "data"))}
}
func (p Paths) Ensure() error {
	for _, path := range []string{p.Data, p.Logs} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			return err
		}
	}
	return nil
}
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// 已认证的内部开发人员可以明文读取资源配置。
type Resource struct {
	Name           string `json:"name" db:"name"`
	Environment    string `json:"environment" db:"environment"`
	Driver         string `json:"driver,omitempty" db:"driver"`
	DSN            string `json:"dsn,omitempty" db:"dsn"`
	Address        string `json:"address,omitempty" db:"address"`
	User           string `json:"user,omitempty" db:"user"`
	Password       string `json:"password,omitempty" db:"password"`
	PrivateKey     string `json:"private_key,omitempty" db:"private_key"`
	KnownHostsPath string `json:"known_hosts_path,omitempty" db:"known_hosts_path"`
	Kubeconfig     string `json:"kubeconfig,omitempty" db:"kubeconfig"`
	Context        string `json:"context,omitempty" db:"context"`
	Token          string `json:"token,omitempty" db:"token"`
}

func (r Resource) Revision() string {
	b, _ := json.Marshal(r)
	return fmt.Sprintf("%x", sha256.Sum256(b))
}

type Filter struct {
	pagination.Query
	Name        string `json:"name,omitempty" db:"name"`
	Environment string `json:"environment,omitempty" db:"environment"`
	Driver      string `json:"driver,omitempty" db:"driver"`
	Address     string `json:"address,omitempty" db:"address"`
	User        string `json:"user,omitempty" db:"user"`
	Context     string `json:"context,omitempty" db:"context"`
}
type Store struct {
	db       *sqlx.DB
	mu       sync.Mutex
	state    map[string]string
	observer func(kind, name, action string) error
}

var columns = map[string]string{
	"database":   "name,environment,driver,dsn",
	"linux":      "name,environment,address,user,password,private_key,known_hosts_path",
	"kubernetes": "name,environment,kubeconfig,context,token",
}

func table(kind string) (string, error) {
	if _, ok := columns[kind]; !ok {
		return "", fmt.Errorf("invalid resource type")
	}
	return kind + "_resources", nil
}
func Open(path string) (*Store, error) {
	db, err := storage.Open(context.Background(), path)
	if err != nil {
		return nil, err
	}
	return New(db), nil
}
func New(db *sqlx.DB) *Store  { return &Store{db: db, state: make(map[string]string)} }
func (s *Store) Close() error { return s.db.Close() }
func (s *Store) SetObserver(observer func(kind, name, action string) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.observer = observer
}

func (s *Store) Get(ctx context.Context, kind, name string) (Resource, error) {
	tbl, err := table(kind)
	if err != nil {
		return Resource{}, err
	}
	var r Resource
	err = s.db.GetContext(ctx, &r, `SELECT `+columns[kind]+` FROM `+tbl+` WHERE name=?`, name)

	return r, err
}
func (s *Store) Database(ctx context.Context, name string) (Resource, error) {
	return s.Get(ctx, "database", name)
}
func (s *Store) Linux(ctx context.Context, name string) (Resource, error) {
	return s.Get(ctx, "linux", name)
}
func (s *Store) Kubernetes(ctx context.Context, name string) (Resource, error) {
	return s.Get(ctx, "kubernetes", name)
}
func (s *Store) List(ctx context.Context, kind string, f Filter) (pagination.Result[Resource], error) {
	result := pagination.Result[Resource]{Items: []Resource{}}
	if err := f.Validate(); err != nil {
		return result, err
	}
	result.Page, result.PageSize = f.Page, f.PageSize
	tbl, err := table(kind)
	if err != nil {
		return result, err
	}
	where, args := " WHERE 1=1", []any{}
	for _, field := range []struct {
		col, value string
		exact      bool
	}{{"name", f.Name, false}, {"environment", f.Environment, true}, {"driver", f.Driver, true}, {"address", f.Address, false}, {"user", f.User, false}, {"context", f.Context, false}} {
		if field.value == "" {
			continue
		}
		if (field.col == "driver" && kind != "database") || ((field.col == "address" || field.col == "user") && kind != "linux") || (field.col == "context" && kind != "kubernetes") {
			return result, fmt.Errorf("filter %s not supported for %s", field.col, kind)
		}
		if field.exact {
			where += " AND " + field.col + "=?"
		} else {
			where += " AND instr(lower(" + field.col + "),lower(?))>0"
		}
		args = append(args, field.value)
	}
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM `+tbl+where, args...).Scan(&result.Total); err != nil {
		return result, err
	}
	err = tx.SelectContext(ctx, &result.Items, `SELECT `+columns[kind]+` FROM `+tbl+where+` ORDER BY name LIMIT ? OFFSET ?`, append(args, f.PageSize, f.Offset())...)
	if err != nil {
		return result, err
	}
	return result, tx.Commit()
}

// 写入审计前先关闭查询结果，避免多个 SQLite 连接之间相互等待锁。
func (s *Store) Observe(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := map[string]string{}
	for kind, cols := range columns {
		tbl, _ := table(kind)
		rows, err := s.db.QueryxContext(ctx, `SELECT `+cols+` FROM `+tbl)
		if err != nil {
			return err
		}
		for rows.Next() {
			var r Resource
			if err := rows.StructScan(&r); err != nil {
				rows.Close()
				return err
			}

			next[kind+":"+r.Name] = r.Revision()
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
	}
	if s.observer != nil {
		for key, rev := range next {
			action := ""
			if old, ok := s.state[key]; !ok {
				action = "added"
			} else if old != rev {
				action = "updated"
			}
			if action != "" {
				kind, name, _ := strings.Cut(key, ":")
				if err := s.observer(kind, name, action); err != nil {
					return err
				}
			}
		}
		for key := range s.state {
			if _, ok := next[key]; !ok {
				kind, name, _ := strings.Cut(key, ":")
				if err := s.observer(kind, name, "removed"); err != nil {
					return err
				}
			}
		}
	}
	s.state = next
	return nil
}
