package config

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
	_ "modernc.org/sqlite"
)

type Paths struct{ Conf, Logs, Data string }

func ResolvePaths() Paths {
	base, _ := os.UserHomeDir()
	root := filepath.Join(base, ".ai-ops-gateway")
	value := func(key, fallback string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return fallback
	}
	return Paths{Conf: value("AI_OPS_GATEWAY_CONF", filepath.Join(root, "conf")), Logs: value("AI_OPS_GATEWAY_LOGS", filepath.Join(root, "logs")), Data: value("AI_OPS_GATEWAY_DATA", filepath.Join(root, "data"))}
}

func (p Paths) Ensure() error {
	for _, dir := range []string{p.Conf, p.Logs, p.Data} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}
	return nil
}

type GatewayConfig struct {
	ListenAddress  string `yaml:"listen_address"`
	MetricsAddress string `yaml:"metrics_address"`
}
type DatabaseConfig struct {
	Name        string `yaml:"name"`
	Environment string `yaml:"environment"`
	Driver      string `yaml:"driver"`
	DSNEnv      string `yaml:"dsn_env"`
}
type LinuxConfig struct {
	Name           string `yaml:"name"`
	Environment    string `yaml:"environment"`
	Address        string `yaml:"address"`
	User           string `yaml:"user"`
	PasswordEnv    string `yaml:"password_env"`
	PrivateKeyPath string `yaml:"private_key_path"`
	KnownHostsPath string `yaml:"known_hosts_path"`
}
type KubernetesConfig struct {
	Name        string `yaml:"name"`
	Environment string `yaml:"environment"`
	Kubeconfig  string `yaml:"kubeconfig"`
	Context     string `yaml:"context"`
}
type Snapshot struct {
	Gateway    GatewayConfig      `yaml:"gateway"`
	Databases  []DatabaseConfig   `yaml:"-"`
	LinuxHosts []LinuxConfig      `yaml:"-"`
	Kubernetes []KubernetesConfig `yaml:"-"`
}
type Manager struct {
	mu      sync.RWMutex
	dir     string
	dbPath  string
	current Snapshot
}

func NewManager(paths Paths) *Manager {
	return &Manager{dir: paths.Conf, dbPath: filepath.Join(paths.Data, "ai-ops-gateway.db")}
}
func (m *Manager) Load() (Snapshot, error) {
	snapshot := Snapshot{Gateway: GatewayConfig{ListenAddress: "127.0.0.1:9095", MetricsAddress: "127.0.0.1:9464"}}
	if err := readOptional(filepath.Join(m.dir, "gateway.yaml"), &snapshot); err != nil {
		return Snapshot{}, err
	}
	if err := m.loadResources(&snapshot); err != nil {
		return Snapshot{}, err
	}
	if err := validate(snapshot); err != nil {
		return Snapshot{}, err
	}
	m.mu.Lock()
	m.current = snapshot
	m.mu.Unlock()
	return snapshot, nil
}
func (m *Manager) Current() Snapshot         { m.mu.RLock(); defer m.mu.RUnlock(); return m.current }
func (m *Manager) Reload() (Snapshot, error) { return m.Load() }

func ExpandPath(path string) string {
	if path == "~" || len(path) > 2 && path[:2] == "~/" {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func readOptional(path string, out any) error {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(b))
	decoder.KnownFields(true)
	if err := decoder.Decode(out); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

func (m *Manager) loadResources(snapshot *Snapshot) error {
	db, err := sql.Open("sqlite", m.dbPath)
	if err != nil {
		return fmt.Errorf("open gateway db: %w", err)
	}
	defer db.Close()
	schema := `
CREATE TABLE IF NOT EXISTS database_resources(name TEXT PRIMARY KEY, environment TEXT NOT NULL DEFAULT '', driver TEXT NOT NULL, dsn_env TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS linux_resources(name TEXT PRIMARY KEY, environment TEXT NOT NULL DEFAULT '', address TEXT NOT NULL, user TEXT NOT NULL, password_env TEXT NOT NULL DEFAULT '', private_key_path TEXT NOT NULL DEFAULT '', known_hosts_path TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS kubernetes_resources(name TEXT PRIMARY KEY, environment TEXT NOT NULL DEFAULT '', kubeconfig TEXT NOT NULL, context TEXT NOT NULL DEFAULT '');`
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("init resource schema: %w", err)
	}
	if err := scanRows(db, `SELECT name,environment,driver,dsn_env FROM database_resources ORDER BY name`, func(rows *sql.Rows) error {
		var item DatabaseConfig
		if err := rows.Scan(&item.Name, &item.Environment, &item.Driver, &item.DSNEnv); err != nil {
			return err
		}
		snapshot.Databases = append(snapshot.Databases, item)
		return nil
	}); err != nil {
		return fmt.Errorf("load database resources: %w", err)
	}
	if err := scanRows(db, `SELECT name,environment,address,user,password_env,private_key_path,known_hosts_path FROM linux_resources ORDER BY name`, func(rows *sql.Rows) error {
		var item LinuxConfig
		if err := rows.Scan(&item.Name, &item.Environment, &item.Address, &item.User, &item.PasswordEnv, &item.PrivateKeyPath, &item.KnownHostsPath); err != nil {
			return err
		}
		snapshot.LinuxHosts = append(snapshot.LinuxHosts, item)
		return nil
	}); err != nil {
		return fmt.Errorf("load linux resources: %w", err)
	}
	if err := scanRows(db, `SELECT name,environment,kubeconfig,context FROM kubernetes_resources ORDER BY name`, func(rows *sql.Rows) error {
		var item KubernetesConfig
		if err := rows.Scan(&item.Name, &item.Environment, &item.Kubeconfig, &item.Context); err != nil {
			return err
		}
		snapshot.Kubernetes = append(snapshot.Kubernetes, item)
		return nil
	}); err != nil {
		return fmt.Errorf("load kubernetes resources: %w", err)
	}
	return nil
}

func scanRows(db *sql.DB, query string, scan func(*sql.Rows) error) error {
	rows, err := db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := scan(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

func validate(snapshot Snapshot) error {
	if snapshot.Gateway.ListenAddress == "" {
		return errors.New("config: gateway.listen_address is required")
	}
	seen := map[string]bool{}
	claim := func(kind, name string) error {
		if name == "" {
			return fmt.Errorf("config: %s name is required", kind)
		}
		if seen[name] {
			return fmt.Errorf("config: duplicate resource name %q", name)
		}
		seen[name] = true
		return nil
	}
	for _, database := range snapshot.Databases {
		if err := claim("database", database.Name); err != nil {
			return err
		}
		if database.DSNEnv == "" || database.Driver != "postgres" && database.Driver != "mysql" {
			return fmt.Errorf("config: database %q requires postgres/mysql driver and dsn_env", database.Name)
		}
	}
	for _, host := range snapshot.LinuxHosts {
		if err := claim("linux host", host.Name); err != nil {
			return err
		}
		if host.Address == "" || host.User == "" || host.KnownHostsPath == "" || host.PasswordEnv == "" && host.PrivateKeyPath == "" {
			return fmt.Errorf("config: linux host %q requires address, user, known_hosts_path and authentication", host.Name)
		}
	}
	for _, cluster := range snapshot.Kubernetes {
		if err := claim("kubernetes cluster", cluster.Name); err != nil {
			return err
		}
		if cluster.Kubeconfig == "" {
			return fmt.Errorf("config: kubernetes cluster %q requires kubeconfig", cluster.Name)
		}
	}
	return nil
}
