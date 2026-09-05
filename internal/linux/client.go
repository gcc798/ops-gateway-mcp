package linux

import (
	"context"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type Config struct {
	Name, Address, User, Password, PrivateKey, KnownHostsPath string
	Timeout                                                   time.Duration
}
type Client struct {
	name     string
	client   *ssh.Client
	timeout  time.Duration
	lastUsed time.Time
}
type Manager struct {
	mu       sync.Mutex
	hosts    map[string]*Client
	resolver func(string) (*Client, error)
}

func NewManager() *Manager { return &Manager{hosts: make(map[string]*Client)} }
func (m *Manager) SetResolver(resolve func(string) (*Client, error)) {
	m.mu.Lock()
	m.resolver = resolve
	m.mu.Unlock()
}
func (m *Manager) Get(name string) (*Client, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.hosts[name]; ok {
		if time.Since(c.lastUsed) > 10*time.Minute {
			_ = c.Close()
			delete(m.hosts, name)
		} else {
			c.lastUsed = time.Now()
			return c, true
		}
	}
	if m.resolver != nil {
		if c, err := m.resolver(name); err == nil {
			c.lastUsed = time.Now()
			m.hosts[name] = c
			return c, true
		}
	}
	return nil, false
}
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, client := range m.hosts {
		if err := client.Close(); err != nil {
			return fmt.Errorf("close %s: %w", name, err)
		}
	}
	return nil
}
func (m *Manager) Remove(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c := m.hosts[name]; c != nil {
		_ = c.Close()
		delete(m.hosts, name)
	}
}
func Open(c Config) (*Client, error) {
	if c.Name == "" || c.Address == "" || c.User == "" {
		return nil, fmt.Errorf("linux config requires name, address and user")
	}
	var auth []ssh.AuthMethod
	if c.Password != "" {
		auth = append(auth, ssh.Password(c.Password))
	}
	if c.PrivateKey != "" {
		signer, e := ssh.ParsePrivateKey([]byte(c.PrivateKey))
		if e != nil {
			return nil, fmt.Errorf("parse private key: %w", e)
		}
		auth = append(auth, ssh.PublicKeys(signer))
	}
	if len(auth) == 0 {
		return nil, fmt.Errorf("linux config requires authentication")
	}
	if c.KnownHostsPath == "" {
		return nil, fmt.Errorf("linux config requires known_hosts")
	}
	hostKeyCallback, e := knownhosts.New(c.KnownHostsPath)
	if e != nil {
		return nil, fmt.Errorf("load known_hosts: %w", e)
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	cfg := &ssh.ClientConfig{User: c.User, Auth: auth, HostKeyCallback: hostKeyCallback, Timeout: timeout}
	conn, e := net.DialTimeout("tcp", c.Address, timeout)
	if e != nil {
		return nil, fmt.Errorf("connect %s: %w", c.Name, e)
	}
	_ = conn.SetDeadline(time.Now().Add(timeout))
	cc, ch, reqs, e := ssh.NewClientConn(conn, c.Address, cfg)
	if e != nil {
		conn.Close()
		return nil, fmt.Errorf("ssh handshake %s: %w", c.Name, e)
	}
	_ = conn.SetDeadline(time.Time{})
	return &Client{name: c.Name, client: ssh.NewClient(cc, ch, reqs), timeout: timeout, lastUsed: time.Now()}, nil
}
func (c *Client) Close() error { return c.client.Close() }
func (c *Client) run(ctx context.Context, command string, limit int64) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	if command == "" {
		return "", fmt.Errorf("command required")
	}
	sess, e := c.client.NewSession()
	if e != nil {
		return "", fmt.Errorf("new ssh session: %w", e)
	}
	defer sess.Close()
	out, e := sess.StdoutPipe()
	if e != nil {
		return "", e
	}
	sess.Stderr = io.Discard
	if e = sess.Start(command); e != nil {
		return "", fmt.Errorf("start command: %w", e)
	}
	type result struct {
		data []byte
		err  error
	}
	done := make(chan result, 1)
	go func() {
		data, err := io.ReadAll(io.LimitReader(out, limit+1))
		if int64(len(data)) > limit {
			_ = sess.Close()
			err = fmt.Errorf("command output exceeds %d bytes", limit)
		} else if err == nil {
			err = sess.Wait()
		}
		done <- result{data, err}
	}()
	select {
	case <-ctx.Done():
		_ = sess.Close()
		return "", ctx.Err()
	case result := <-done:
		if result.err != nil {
			return "", fmt.Errorf("command failed: %w", result.err)
		}
		return string(result.data), nil
	}
}
func (c *Client) SystemInfo(ctx context.Context) (string, error) {
	return c.run(ctx, "uname -a", 1<<20)
}
func (c *Client) DiskUsage(ctx context.Context) (string, error) { return c.run(ctx, "df -h", 1<<20) }
func (c *Client) MemoryUsage(ctx context.Context) (string, error) {
	return c.run(ctx, "free -h", 1<<20)
}
func (c *Client) Processes(ctx context.Context) (string, error) { return c.run(ctx, "ps aux", 2<<20) }
func (c *Client) ServiceStatus(ctx context.Context, service string) (string, error) {
	if !ValidServiceName(service) {
		return "", fmt.Errorf("invalid service name")
	}
	return c.run(ctx, "systemctl status --no-pager -- "+quote(service), 1<<20)
}
func (c *Client) ServiceLogs(ctx context.Context, service string, lines int) (string, error) {
	if !ValidServiceName(service) {
		return "", fmt.Errorf("invalid service name")
	}
	if lines <= 0 || lines > 5000 {
		lines = 200
	}
	return c.run(ctx, fmt.Sprintf("journalctl --no-pager -n %d -u %s", lines, quote(service)), 4<<20)
}
func (c *Client) RestartService(ctx context.Context, service string) error {
	if !ValidServiceName(service) {
		return fmt.Errorf("invalid service name")
	}
	_, err := c.run(ctx, "sudo -n systemctl restart -- "+quote(service), 1<<20)
	return err
}
func (c *Client) ListDirectory(ctx context.Context, path string) (string, error) {
	if !safePath(path) {
		return "", fmt.Errorf("safe absolute path required")
	}
	return c.run(ctx, "ls -la -- "+quote(path), 2<<20)
}
func (c *Client) ReadFile(ctx context.Context, path string) (string, error) {
	if !safePath(path) {
		return "", fmt.Errorf("safe absolute path required")
	}
	return c.run(ctx, "cat -- "+quote(path), 4<<20)
}
func safePath(path string) bool {
	if !filepath.IsAbs(path) {
		return false
	}
	for _, p := range strings.Split(filepath.ToSlash(path), "/") {
		if p == ".." {
			return false
		}
	}
	return true
}
func ValidServiceName(service string) bool {
	if service == "" {
		return false
	}
	for _, r := range service {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("_.@-", r) {
			continue
		}
		return false
	}
	return true
}
func quote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'" }
