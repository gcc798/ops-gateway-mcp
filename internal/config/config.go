package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
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
	current Snapshot
}

func NewManager(paths Paths) *Manager { return &Manager{dir: paths.Conf} }
func (m *Manager) Load() (Snapshot, error) {
	snapshot := Snapshot{Gateway: GatewayConfig{ListenAddress: "127.0.0.1:9095", MetricsAddress: "127.0.0.1:9464"}}
	if err := readResourceFiles(filepath.Join(m.dir, "database"), "databases", &snapshot.Databases); err != nil {
		return Snapshot{}, err
	}
	if err := readResourceFiles(filepath.Join(m.dir, "linux"), "hosts", &snapshot.LinuxHosts); err != nil {
		return Snapshot{}, err
	}
	if err := readResourceFiles(filepath.Join(m.dir, "kubernetes"), "clusters", &snapshot.Kubernetes); err != nil {
		return Snapshot{}, err
	}
	if err := readOptional(filepath.Join(m.dir, "gateway.yaml"), &snapshot); err != nil {
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

func readResourceFiles[T any](dir, key string, out *[]T) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return err
	}
	for _, path := range files {
		wrapper := map[string][]T{}
		if err := readOptional(path, &wrapper); err != nil {
			return err
		}
		*out = append(*out, wrapper[key]...)
	}
	return nil
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
