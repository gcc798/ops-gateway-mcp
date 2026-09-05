package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Config struct {
	Name, Driver, DSN string
	MaxOpenConns      int
	ConnMaxLifetime   time.Duration
}
type Adapter struct {
	name, driver string
	db           *sql.DB
	lastUsed     atomic.Int64
}

const (
	maxQueryRows  = 1000
	maxQueryBytes = 16 << 20
)

type Column struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
}

type Manager struct {
	mu       sync.RWMutex
	adapters map[string]*Adapter
	resolver func(string) (*Adapter, error)
}

const idleTimeout = 10 * time.Minute

func NewManager() *Manager { return &Manager{adapters: make(map[string]*Adapter)} }
func (m *Manager) SetResolver(resolve func(string) (*Adapter, error)) {
	m.mu.Lock()
	m.resolver = resolve
	m.mu.Unlock()
}
func (m *Manager) Add(a *Adapter) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.adapters[a.name]; ok {
		return fmt.Errorf("database %q already exists", a.name)
	}
	m.adapters[a.name] = a
	a.lastUsed.Store(time.Now().UnixNano())
	return nil
}
func (m *Manager) Get(name string) (*Adapter, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if a, ok := m.adapters[name]; ok {
		if time.Since(time.Unix(0, a.lastUsed.Load())) > idleTimeout {
			_ = a.Close()
			delete(m.adapters, name)
		} else {
			a.lastUsed.Store(time.Now().UnixNano())
			return a, true
		}
	}
	if m.resolver != nil {
		if a, err := m.resolver(name); err == nil {
			m.adapters[name] = a
			a.lastUsed.Store(time.Now().UnixNano())
			return a, true
		}
	}
	return nil, false
}
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, a := range m.adapters {
		if err := a.Close(); err != nil {
			return fmt.Errorf("close %s: %w", name, err)
		}
	}
	return nil
}
func (m *Manager) Remove(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c := m.adapters[name]; c != nil {
		_ = c.Close()
		delete(m.adapters, name)
	}
}

func Open(c Config) (*Adapter, error) {
	if c.Name == "" || c.DSN == "" {
		return nil, fmt.Errorf("database config requires name and dsn")
	}
	if c.Driver != "postgres" && c.Driver != "mysql" {
		return nil, fmt.Errorf("unsupported database driver %q", c.Driver)
	}
	driver := c.Driver
	if driver == "postgres" {
		driver = "pgx"
	}
	db, err := sql.Open(driver, c.DSN)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", c.Name, err)
	}
	if c.MaxOpenConns > 0 {
		db.SetMaxOpenConns(c.MaxOpenConns)
	}
	if c.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(c.ConnMaxLifetime)
	}
	return &Adapter{name: c.Name, driver: c.Driver, db: db}, nil
}
func (a *Adapter) Close() error   { return a.db.Close() }
func (a *Adapter) Driver() string { return a.driver }
func (a *Adapter) Ping(ctx context.Context) error {
	if err := a.db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping %s: %w", a.name, err)
	}
	return nil
}
func (a *Adapter) ListTables(ctx context.Context) ([]string, error) {
	query := `SELECT table_name FROM information_schema.tables WHERE table_schema = current_schema() ORDER BY table_name`
	if a.driver == "mysql" {
		query = `SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE() ORDER BY table_name`
	}
	rows, err := a.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list tables: %w", err)
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, fmt.Errorf("scan table: %w", err)
		}
		tables = append(tables, table)
	}
	return tables, rows.Err()
}
func (a *Adapter) DescribeTable(ctx context.Context, table string) ([]Column, error) {
	query := `SELECT column_name,data_type,is_nullable FROM information_schema.columns WHERE table_schema=current_schema() AND table_name=$1 ORDER BY ordinal_position`
	if a.driver == "mysql" {
		query = `SELECT column_name,data_type,is_nullable FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name=? ORDER BY ordinal_position`
	}
	rows, err := a.db.QueryContext(ctx, query, table)
	if err != nil {
		return nil, fmt.Errorf("describe table: %w", err)
	}
	defer rows.Close()
	out := []Column{}
	for rows.Next() {
		var c Column
		var nullable string
		if err := rows.Scan(&c.Name, &c.Type, &nullable); err != nil {
			return nil, fmt.Errorf("scan column: %w", err)
		}
		c.Nullable = nullable == "YES"
		out = append(out, c)
	}
	return out, rows.Err()
}
func (a *Adapter) Query(ctx context.Context, statement string) ([]map[string]any, error) {
	if strings.TrimSpace(statement) == "" {
		return nil, fmt.Errorf("statement is required")
	}
	rows, err := a.db.QueryContext(ctx, statement)
	if err != nil {
		return nil, fmt.Errorf("query %s: %w", a.name, err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("read columns: %w", err)
	}
	result := make([]map[string]any, 0)
	resultBytes := 0
	for rows.Next() {
		if len(result) >= maxQueryRows {
			return nil, fmt.Errorf("query result exceeds %d rows", maxQueryRows)
		}
		values := make([]any, len(columns))
		refs := make([]any, len(columns))
		for i := range values {
			refs[i] = &values[i]
		}
		if err := rows.Scan(refs...); err != nil {
			return nil, fmt.Errorf("scan query row: %w", err)
		}
		row := make(map[string]any, len(columns))
		for i, col := range columns {
			row[col] = values[i]
		}
		rowBytes := approximateRowBytes(row)
		if resultBytes+rowBytes > maxQueryBytes {
			return nil, fmt.Errorf("query result exceeds %d bytes", maxQueryBytes)
		}
		resultBytes += rowBytes
		result = append(result, row)
	}
	return result, rows.Err()
}
func approximateRowBytes(row map[string]any) int {
	total := 0
	for key, value := range row {
		total += len(key) + len(fmt.Sprint(value))
	}
	return total
}
func (a *Adapter) Exec(ctx context.Context, statement string) (int64, error) {
	result, err := a.db.ExecContext(ctx, statement)
	if err != nil {
		return 0, fmt.Errorf("execute %s: %w", a.name, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("affected rows: %w", err)
	}
	return rows, nil
}
