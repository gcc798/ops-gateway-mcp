package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gcc798/ai-ops-gateway/internal/audit"
	"github.com/gcc798/ai-ops-gateway/internal/auth"
	"github.com/gcc798/ai-ops-gateway/internal/resources"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type nameInput struct {
	Name string `json:"name" jsonschema:"Logical resource name"`
}
type databaseQueryInput struct {
	Name      string `json:"name" jsonschema:"Logical database name"`
	Statement string `json:"statement" jsonschema:"Read-only SQL statement"`
}
type tableInput struct {
	Name  string `json:"name"`
	Table string `json:"table"`
}
type prepareSQLInput struct {
	Client      string `json:"client"`
	Environment string `json:"environment"`
	Name        string `json:"name"`
	Statement   string `json:"statement"`
}
type kubeNamespaceInput struct {
	Cluster   string `json:"cluster"`
	Namespace string `json:"namespace"`
}
type kubePodInput struct {
	Cluster   string `json:"cluster"`
	Namespace string `json:"namespace"`
	Pod       string `json:"pod"`
	Container string `json:"container,omitempty"`
}
type kubeDownloadInput struct {
	Cluster   string `json:"cluster"`
	Namespace string `json:"namespace"`
	Pod       string `json:"pod"`
	Container string `json:"container,omitempty"`
	Path      string `json:"path" jsonschema:"Any non-empty file path inside the container"`
}
type kubeDeploymentInput struct {
	Client      string `json:"client,omitempty"`
	Environment string `json:"environment,omitempty"`
	Cluster     string `json:"cluster"`
	Namespace   string `json:"namespace"`
	Deployment  string `json:"deployment"`
}
type linuxPathInput struct {
	Name string `json:"name"`
	Path string `json:"path"`
}
type linuxServiceInput struct {
	Client      string `json:"client,omitempty"`
	Environment string `json:"environment,omitempty"`
	Name        string `json:"name"`
	Service     string `json:"service"`
	Lines       int    `json:"lines,omitempty"`
}
type valueOutput struct {
	Value any `json:"value"`
}

func newMCPHandler(d *Dependencies) http.Handler {
	s := mcp.NewServer(&mcp.Implementation{Name: "ai-ops-gateway", Version: "v0.1.0"}, nil)
	s.AddReceivingMiddleware(mcpAuditMiddleware(d))
	addDatabaseTools(s, d)
	addKubernetesTools(s, d)
	addLinuxTools(s, d)
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return s }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true, PropagateRequestCancellation: true})
}

func addDatabaseTools(s *mcp.Server, d *Dependencies) {
	mcp.AddTool(s, &mcp.Tool{Name: "db_list_connections", Description: "List database resources with page, page_size, name, environment and driver filters"}, func(ctx context.Context, _ *mcp.CallToolRequest, in resources.Filter) (*mcp.CallToolResult, valueOutput, error) {
		v, err := d.App.ListResources(ctx, "database", in)
		return nil, valueOutput{v}, err
	})
	mcp.AddTool(s, &mcp.Tool{Name: "db_ping", Description: "Test a logical database connection"}, func(ctx context.Context, _ *mcp.CallToolRequest, in nameInput) (*mcp.CallToolResult, valueOutput, error) {
		db, err := d.App.Database(in.Name)
		if err != nil {
			return nil, valueOutput{}, err
		}
		if err = db.Ping(ctx); err != nil {
			return nil, valueOutput{}, err
		}
		return nil, valueOutput{"ok"}, nil
	})
	mcp.AddTool(s, &mcp.Tool{Name: "db_list_tables", Description: "List tables in a logical database"}, func(ctx context.Context, _ *mcp.CallToolRequest, in nameInput) (*mcp.CallToolResult, valueOutput, error) {
		db, err := d.App.Database(in.Name)
		if err != nil {
			return nil, valueOutput{}, err
		}
		v, err := db.ListTables(ctx)
		return nil, valueOutput{v}, err
	})
	mcp.AddTool(s, &mcp.Tool{Name: "db_describe_table", Description: "Describe columns of a table"}, func(ctx context.Context, _ *mcp.CallToolRequest, in tableInput) (*mcp.CallToolResult, valueOutput, error) {
		db, err := d.App.Database(in.Name)
		if err != nil {
			return nil, valueOutput{}, err
		}
		v, err := db.DescribeTable(ctx, in.Table)
		return nil, valueOutput{v}, err
	})
	mcp.AddTool(s, &mcp.Tool{Name: "db_query", Description: "Execute SQL only when the Gateway policy classifies it as read-only"}, func(ctx context.Context, _ *mcp.CallToolRequest, in databaseQueryInput) (*mcp.CallToolResult, valueOutput, error) {
		v, err := d.App.QuerySQL(ctx, in.Name, in.Statement)
		return nil, valueOutput{v}, err
	})
	mcp.AddTool(s, &mcp.Tool{Name: "db_explain", Description: "Execute an AST-verified read-only EXPLAIN statement"}, func(ctx context.Context, _ *mcp.CallToolRequest, in databaseQueryInput) (*mcp.CallToolResult, valueOutput, error) {
		v, err := d.App.QuerySQL(ctx, in.Name, in.Statement)
		return nil, valueOutput{v}, err
	})
	mcp.AddTool(s, &mcp.Tool{Name: "db_prepare_execute", Description: "Evaluate a write statement and create an immutable pending operation when confirmation is required"}, func(ctx context.Context, _ *mcp.CallToolRequest, in prepareSQLInput) (*mcp.CallToolResult, valueOutput, error) {
		v, err := d.App.EvaluateSQL(ctx, in.Client, in.Environment, in.Name, in.Statement)
		return nil, valueOutput{v}, err
	})
}

func addKubernetesTools(s *mcp.Server, d *Dependencies) {
	mcp.AddTool(s, &mcp.Tool{Name: "k8s_list_clusters", Description: "List Kubernetes resources with page, page_size, name, environment and context filters"}, func(ctx context.Context, _ *mcp.CallToolRequest, in resources.Filter) (*mcp.CallToolResult, valueOutput, error) {
		v, err := d.App.ListResources(ctx, "kubernetes", in)
		return nil, valueOutput{v}, err
	})
	mcp.AddTool(s, &mcp.Tool{Name: "k8s_get_pods", Description: "List pods in a namespace"}, func(ctx context.Context, _ *mcp.CallToolRequest, in kubeNamespaceInput) (*mcp.CallToolResult, valueOutput, error) {
		c, err := d.App.KubernetesClient(in.Cluster)
		if err != nil {
			return nil, valueOutput{}, err
		}
		v, err := c.ListPods(ctx, in.Namespace)
		return nil, valueOutput{v}, err
	})
	mcp.AddTool(s, &mcp.Tool{Name: "k8s_get_pod", Description: "Get one pod"}, func(ctx context.Context, _ *mcp.CallToolRequest, in kubePodInput) (*mcp.CallToolResult, valueOutput, error) {
		c, err := d.App.KubernetesClient(in.Cluster)
		if err != nil {
			return nil, valueOutput{}, err
		}
		pod, err := c.Pod(ctx, in.Namespace, in.Pod)
		return nil, valueOutput{map[string]any{"name": pod.Name, "phase": pod.Status.Phase, "node": pod.Spec.NodeName, "containers": pod.Spec.Containers}}, err
	})
	mcp.AddTool(s, &mcp.Tool{Name: "k8s_get_deployments", Description: "List deployments in a namespace"}, func(ctx context.Context, _ *mcp.CallToolRequest, in kubeNamespaceInput) (*mcp.CallToolResult, valueOutput, error) {
		c, err := d.App.KubernetesClient(in.Cluster)
		if err != nil {
			return nil, valueOutput{}, err
		}
		v, err := c.ListDeployments(ctx, in.Namespace)
		return nil, valueOutput{v}, err
	})
	mcp.AddTool(s, &mcp.Tool{Name: "k8s_get_deployment", Description: "Get one deployment"}, func(ctx context.Context, _ *mcp.CallToolRequest, in kubeDeploymentInput) (*mcp.CallToolResult, valueOutput, error) {
		c, err := d.App.KubernetesClient(in.Cluster)
		if err != nil {
			return nil, valueOutput{}, err
		}
		deployment, err := c.Deployment(ctx, in.Namespace, in.Deployment)
		return nil, valueOutput{map[string]any{"name": deployment.Name, "replicas": deployment.Status.Replicas, "ready_replicas": deployment.Status.ReadyReplicas, "available_replicas": deployment.Status.AvailableReplicas}}, err
	})
	mcp.AddTool(s, &mcp.Tool{Name: "k8s_rollout_restart", Description: "Prepare a policy-controlled deployment restart; does not execute before confirmation"}, func(ctx context.Context, _ *mcp.CallToolRequest, in kubeDeploymentInput) (*mcp.CallToolResult, valueOutput, error) {
		result, err := d.App.PrepareAction(ctx, in.Client, in.Environment, "kubernetes", in.Cluster, "rollout_restart", in.Namespace+"/"+in.Deployment)
		return nil, valueOutput{result}, err
	})
	mcp.AddTool(s, &mcp.Tool{Name: "k8s_get_services", Description: "List services in a namespace"}, func(ctx context.Context, _ *mcp.CallToolRequest, in kubeNamespaceInput) (*mcp.CallToolResult, valueOutput, error) {
		c, err := d.App.KubernetesClient(in.Cluster)
		if err != nil {
			return nil, valueOutput{}, err
		}
		v, err := c.ListServices(ctx, in.Namespace)
		return nil, valueOutput{v}, err
	})
	mcp.AddTool(s, &mcp.Tool{Name: "k8s_logs", Description: "Read recent pod logs"}, func(ctx context.Context, _ *mcp.CallToolRequest, in kubePodInput) (*mcp.CallToolResult, valueOutput, error) {
		c, err := d.App.KubernetesClient(in.Cluster)
		if err != nil {
			return nil, valueOutput{}, err
		}
		v, err := c.Logs(ctx, in.Namespace, in.Pod, in.Container, 500)
		return nil, valueOutput{v}, err
	})
	mcp.AddTool(s, &mcp.Tool{Name: "k8s_download_file", Description: "Read a file from a container without modifying it"}, func(ctx context.Context, _ *mcp.CallToolRequest, in kubeDownloadInput) (*mcp.CallToolResult, any, error) {
		c, err := d.App.KubernetesClient(in.Cluster)
		if err != nil {
			return nil, nil, err
		}
		v, err := c.DownloadFile(ctx, in.Namespace, in.Pod, in.Container, in.Path, 0)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.EmbeddedResource{Resource: &mcp.ResourceContents{URI: fmt.Sprintf("container://%s/%s/%s%s", in.Cluster, in.Namespace, in.Pod, in.Path), MIMEType: "application/octet-stream", Blob: v}}}}, nil, nil
	})
}

func mcpAuditMiddleware(d *Dependencies) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, request mcp.Request) (mcp.Result, error) {
			started := time.Now()
			if p, ok := request.GetParams().(*mcp.CallToolParamsRaw); ok && method == "tools/call" {
				ctx = audit.WithTool(ctx, p.Name)
			}
			result, err := next(ctx, method, request)
			if method != "tools/call" || d.Audits == nil {
				return result, err
			}
			params, ok := request.GetParams().(*mcp.CallToolParamsRaw)
			if !ok {
				return result, err
			}
			var arguments map[string]any
			_ = json.Unmarshal(params.Arguments, &arguments)
			value := func(key string) string { value, _ := arguments[key].(string); return value }
			resource := value("name")
			if resource == "" {
				resource = value("cluster")
			}
			status := "succeeded"
			if err != nil {
				status = "failed"
			} else if toolResult, ok := result.(*mcp.CallToolResult); ok && toolResult.GetError() != nil {
				status = "failed"
			}
			if status == "succeeded" && (params.Name == "db_prepare_execute" || params.Name == "linux_restart_service" || params.Name == "k8s_rollout_restart") {
				return result, err
			}
			if params.Name == "db_query" || params.Name == "db_explain" {
				return result, err
			}
			resourceType := "gateway"
			for _, prefix := range []string{"db", "linux", "k8s"} {
				if len(params.Name) > len(prefix) && params.Name[:len(prefix)] == prefix {
					resourceType = map[string]string{"db": "database", "linux": "linux", "k8s": "kubernetes"}[prefix]
				}
			}
			statementHash := ""
			if statement := value("statement"); statement != "" {
				sum := sha256.Sum256([]byte(statement))
				statementHash = hex.EncodeToString(sum[:])
			}
			environment := ""
			if resource != "" && d.App.Resources != nil {
				if r, e := d.App.Resource(ctx, resourceType, resource); e == nil {
					environment = r.Environment
				}
			}
			decision := "allow"
			if status == "failed" {
				decision = ""
			}
			auditCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cancel()
			if auditErr := d.Audits.Record(auditCtx, audit.Operation{ID: uuid.NewString(), Client: auth.Identity(ctx), RequestID: audit.RequestID(ctx), Tool: params.Name, Environment: environment, ResourceType: resourceType, Resource: resource, Action: params.Name, Status: status, Timestamp: started, Risk: "low", Decision: decision, DurationMS: time.Since(started).Milliseconds(), StatementHash: statementHash}); auditErr != nil {
				return nil, errors.New("audit recording failed")
			}
			if err != nil {
				return nil, errors.New("tool operation failed")
			}
			if status == "failed" {
				return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "tool operation failed"}}}, nil
			}
			return result, err
		}
	}
}

func addLinuxTools(s *mcp.Server, d *Dependencies) {
	mcp.AddTool(s, &mcp.Tool{Name: "linux_list_hosts", Description: "List Linux resources with page, page_size, name, environment, address and user filters"}, func(ctx context.Context, _ *mcp.CallToolRequest, in resources.Filter) (*mcp.CallToolResult, valueOutput, error) {
		v, err := d.App.ListResources(ctx, "linux", in)
		return nil, valueOutput{v}, err
	})
	add := func(name, description string, run func(context.Context, nameInput) (string, error)) {
		mcp.AddTool(s, &mcp.Tool{Name: name, Description: description}, func(ctx context.Context, _ *mcp.CallToolRequest, in nameInput) (*mcp.CallToolResult, valueOutput, error) {
			v, err := run(ctx, in)
			return nil, valueOutput{v}, err
		})
	}
	add("linux_system_info", "Read operating system and kernel information", func(ctx context.Context, in nameInput) (string, error) {
		h, e := d.App.LinuxClient(in.Name)
		if e != nil {
			return "", e
		}
		return h.SystemInfo(ctx)
	})
	add("linux_disk_usage", "Read filesystem usage", func(ctx context.Context, in nameInput) (string, error) {
		h, e := d.App.LinuxClient(in.Name)
		if e != nil {
			return "", e
		}
		return h.DiskUsage(ctx)
	})
	add("linux_memory_usage", "Read memory usage", func(ctx context.Context, in nameInput) (string, error) {
		h, e := d.App.LinuxClient(in.Name)
		if e != nil {
			return "", e
		}
		return h.MemoryUsage(ctx)
	})
	add("linux_processes", "Read process list", func(ctx context.Context, in nameInput) (string, error) {
		h, e := d.App.LinuxClient(in.Name)
		if e != nil {
			return "", e
		}
		return h.Processes(ctx)
	})
	mcp.AddTool(s, &mcp.Tool{Name: "linux_service_status", Description: "Read systemd service status"}, func(ctx context.Context, _ *mcp.CallToolRequest, in linuxServiceInput) (*mcp.CallToolResult, valueOutput, error) {
		host, err := d.App.LinuxClient(in.Name)
		if err != nil {
			return nil, valueOutput{}, err
		}
		value, err := host.ServiceStatus(ctx, in.Service)
		return nil, valueOutput{value}, err
	})
	mcp.AddTool(s, &mcp.Tool{Name: "linux_service_logs", Description: "Read recent systemd journal entries"}, func(ctx context.Context, _ *mcp.CallToolRequest, in linuxServiceInput) (*mcp.CallToolResult, valueOutput, error) {
		host, err := d.App.LinuxClient(in.Name)
		if err != nil {
			return nil, valueOutput{}, err
		}
		value, err := host.ServiceLogs(ctx, in.Service, in.Lines)
		return nil, valueOutput{value}, err
	})
	mcp.AddTool(s, &mcp.Tool{Name: "linux_restart_service", Description: "Prepare a policy-controlled systemd restart; does not execute before confirmation"}, func(ctx context.Context, _ *mcp.CallToolRequest, in linuxServiceInput) (*mcp.CallToolResult, valueOutput, error) {
		result, err := d.App.PrepareAction(ctx, in.Client, in.Environment, "linux", in.Name, "restart_service", in.Service)
		return nil, valueOutput{result}, err
	})
	mcp.AddTool(s, &mcp.Tool{Name: "linux_list_directory", Description: "List one absolute directory path"}, func(ctx context.Context, _ *mcp.CallToolRequest, in linuxPathInput) (*mcp.CallToolResult, valueOutput, error) {
		h, err := d.App.LinuxClient(in.Name)
		if err != nil {
			return nil, valueOutput{}, err
		}
		v, err := h.ListDirectory(ctx, in.Path)
		return nil, valueOutput{v}, err
	})
	mcp.AddTool(s, &mcp.Tool{Name: "linux_read_file", Description: "Read one absolute file path without modifying it"}, func(ctx context.Context, _ *mcp.CallToolRequest, in linuxPathInput) (*mcp.CallToolResult, valueOutput, error) {
		h, err := d.App.LinuxClient(in.Name)
		if err != nil {
			return nil, valueOutput{}, err
		}
		v, err := h.ReadFile(ctx, in.Path)
		return nil, valueOutput{v}, err
	})
}
