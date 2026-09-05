package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

type Client struct {
	client   kubernetes.Interface
	config   *rest.Config
	lastUsed time.Time
}

type Manager struct {
	mu       sync.Mutex
	clusters map[string]*Client
	resolver func(string) (*Client, error)
}

func NewManager() *Manager { return &Manager{clusters: make(map[string]*Client)} }
func (m *Manager) SetResolver(resolve func(string) (*Client, error)) {
	m.mu.Lock()
	m.resolver = resolve
	m.mu.Unlock()
}
func (m *Manager) Get(name string) (*Client, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.clusters[name]; ok {
		if time.Since(c.lastUsed) > 10*time.Minute {
			c.Close()
			delete(m.clusters, name)
		} else {
			c.lastUsed = time.Now()
			return c, true
		}
	}
	if m.resolver != nil {
		if c, err := m.resolver(name); err == nil {
			c.lastUsed = time.Now()
			m.clusters[name] = c
			return c, true
		}
	}
	return nil, false
}
func (c *Client) Close() {
	if c.config == nil {
		return
	}
	if transport, ok := c.config.Transport.(interface{ CloseIdleConnections() }); ok {
		transport.CloseIdleConnections()
	}
}
func (m *Manager) Remove(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c := m.clusters[name]; c != nil {
		c.Close()
		delete(m.clusters, name)
	}
}

func FromKubeconfig(path, contextName string, tokens ...string) (*Client, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("read kubeconfig: %w", err)
	}
	overrides := &clientcmd.ConfigOverrides{}
	if contextName != "" {
		overrides.CurrentContext = contextName
	}
	loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(&clientcmd.ClientConfigLoadingRules{ExplicitPath: path}, overrides)
	config, err := loader.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig: %w", err)
	}
	config.Timeout = 10 * time.Second
	if len(tokens) > 0 && tokens[0] != "" {
		config.BearerToken = tokens[0]
		config.BearerTokenFile = ""
		config.Username, config.Password = "", ""
		config.CertFile, config.KeyFile = "", ""
		config.CertData, config.KeyData = nil, nil
		config.ExecProvider = nil
		config.AuthProvider = nil
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create kubernetes client: %w", err)
	}
	return &Client{client: client, config: config, lastUsed: time.Now()}, nil
}
func (c *Client) Ping(ctx context.Context) error {
	if _, err := c.client.Discovery().RESTClient().Get().AbsPath("/version").DoRaw(ctx); err != nil {
		return fmt.Errorf("kubernetes ping: %w", err)
	}
	return nil
}
func (c *Client) ListPods(ctx context.Context, namespace string) ([]string, error) {
	pods, err := c.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list pods: %w", err)
	}
	out := make([]string, 0, len(pods.Items))
	for _, p := range pods.Items {
		out = append(out, p.Name)
	}
	return out, nil
}
func (c *Client) Pod(ctx context.Context, namespace, name string) (corev1.Pod, error) {
	p, err := c.client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return corev1.Pod{}, fmt.Errorf("get pod: %w", err)
	}
	return *p, nil
}
func (c *Client) ListDeployments(ctx context.Context, namespace string) ([]string, error) {
	ds, err := c.client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list deployments: %w", err)
	}
	out := make([]string, 0, len(ds.Items))
	for _, d := range ds.Items {
		out = append(out, d.Name)
	}
	return out, nil
}
func (c *Client) Deployment(ctx context.Context, namespace, name string) (appsv1.Deployment, error) {
	deployment, err := c.client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return appsv1.Deployment{}, fmt.Errorf("get deployment: %w", err)
	}
	return *deployment, nil
}
func (c *Client) RolloutRestart(ctx context.Context, namespace, name string) error {
	patch, _ := json.Marshal(map[string]any{"spec": map[string]any{"template": map[string]any{"metadata": map[string]any{"annotations": map[string]string{"ops-gateway-mcp/restartedAt": time.Now().UTC().Format(time.RFC3339Nano)}}}}})
	if _, err := c.client.AppsV1().Deployments(namespace).Patch(ctx, name, types.StrategicMergePatchType, patch, metav1.PatchOptions{}); err != nil {
		return fmt.Errorf("restart deployment: %w", err)
	}
	return nil
}
func (c *Client) ListServices(ctx context.Context, namespace string) ([]string, error) {
	ss, err := c.client.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list services: %w", err)
	}
	out := make([]string, 0, len(ss.Items))
	for _, s := range ss.Items {
		out = append(out, s.Name)
	}
	return out, nil
}
func (c *Client) Logs(ctx context.Context, namespace, pod, container string, tail int64) (string, error) {
	if tail <= 0 || tail > 10000 {
		tail = 500
	}
	req := c.client.CoreV1().Pods(namespace).GetLogs(pod, &corev1.PodLogOptions{Container: container, TailLines: &tail})
	stream, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("pod logs: %w", err)
	}
	defer stream.Close()
	b, err := io.ReadAll(io.LimitReader(stream, 2<<20))
	if err != nil {
		return "", fmt.Errorf("read pod logs: %w", err)
	}
	return string(b), nil
}
func (c *Client) DownloadFile(ctx context.Context, namespace, pod, container, path string, maxBytes int64) ([]byte, error) {
	if namespace == "" || pod == "" || path == "" {
		return nil, fmt.Errorf("namespace, pod and path are required")
	}
	if maxBytes <= 0 || maxBytes > 16<<20 {
		maxBytes = 16 << 20
	}
	command := []string{"cat", "--", path}
	req := c.client.CoreV1().RESTClient().Post().Resource("pods").Namespace(namespace).Name(pod).SubResource("exec").VersionedParams(&corev1.PodExecOptions{Container: container, Command: command, Stdout: true, Stderr: true}, schemeParameterCodec())
	exec, err := remotecommand.NewSPDYExecutor(c.config, "POST", req.URL())
	if err != nil {
		return nil, fmt.Errorf("create pod exec: %w", err)
	}
	out := limitedBuffer{limit: maxBytes}
	var errOut limitedBuffer
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{Stdout: &out, Stderr: &errOut})
	if err != nil {
		return nil, fmt.Errorf("download file: %w", err)
	}
	if errOut.Len() > 0 {
		return nil, fmt.Errorf("pod read failed")
	}
	return out.Bytes(), nil
}

type limitedBuffer struct {
	data  []byte
	limit int64
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.limit == 0 {
		b.limit = 16 << 20
	}
	if int64(len(b.data)+len(p)) > b.limit {
		return 0, fmt.Errorf("output exceeds limit")
	}
	b.data = append(b.data, p...)
	return len(p), nil
}
func (b *limitedBuffer) Bytes() []byte { return b.data }
func (b *limitedBuffer) Len() int      { return len(b.data) }

var _ io.Writer = (*limitedBuffer)(nil)

func schemeParameterCodec() runtime.ParameterCodec { return scheme.ParameterCodec }
