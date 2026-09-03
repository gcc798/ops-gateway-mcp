package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
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
	client kubernetes.Interface
	config *rest.Config
}

type Manager struct{ clusters map[string]*Client }

func NewManager() *Manager                         { return &Manager{clusters: make(map[string]*Client)} }
func (m *Manager) Add(name string, c *Client)      { m.clusters[name] = c }
func (m *Manager) Get(name string) (*Client, bool) { c, ok := m.clusters[name]; return c, ok }
func (m *Manager) Names() []string {
	out := make([]string, 0, len(m.clusters))
	for n := range m.clusters {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func FromKubeconfig(path, contextName string) (*Client, error) {
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
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create kubernetes client: %w", err)
	}
	return &Client{client: client, config: config}, nil
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
	patch, _ := json.Marshal(map[string]any{"spec": map[string]any{"template": map[string]any{"metadata": map[string]any{"annotations": map[string]string{"ai-ops-gateway/restartedAt": time.Now().UTC().Format(time.RFC3339Nano)}}}}})
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
