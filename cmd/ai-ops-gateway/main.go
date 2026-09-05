package main

import (
	"context"
	"embed"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gcc798/ai-ops-gateway/internal/audit"
	"github.com/gcc798/ai-ops-gateway/internal/auth"
	"github.com/gcc798/ai-ops-gateway/internal/database"
	kube "github.com/gcc798/ai-ops-gateway/internal/kubernetes"
	linux "github.com/gcc798/ai-ops-gateway/internal/linux"
	"github.com/gcc798/ai-ops-gateway/internal/observability"
	"github.com/gcc798/ai-ops-gateway/internal/resources"
	"github.com/gcc798/ai-ops-gateway/internal/server"
	"github.com/gcc798/ai-ops-gateway/internal/storage"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
)

//go:embed all:web/dist
var webFS embed.FS

func main() {
	root := &cobra.Command{Use: "ai-ops-gateway", Short: "AI Agent operations security gateway", Version: "0.1.0", SilenceUsage: true}
	root.AddCommand(&cobra.Command{Use: "serve", Short: "Start the MCP, REST and Web server", RunE: func(*cobra.Command, []string) error { return serve() }})
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func serve() error {
	paths := resources.ResolvePaths()
	if err := paths.Ensure(); err != nil {
		return err
	}
	logger, logs := observability.NewLogger(paths.Logs)
	defer logs.Close()
	return run(logger, paths)
}

func run(logger *slog.Logger, paths resources.Paths) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	db, err := storage.Open(ctx, filepath.Join(paths.Data, "ai-ops-gateway.db"))
	cancel()
	if err != nil {
		return err
	}
	defer db.Close()
	resourceStore := resources.New(db)
	store := audit.New(db)
	authStore := auth.New(db)
	if err := resourceStore.Observe(context.Background()); err != nil {
		return err
	}
	resourceStore.SetObserver(func(kind, name, action string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return store.Record(ctx, audit.Operation{ID: uuid.NewString(), Timestamp: time.Now(), Tool: "resource_" + action, ResourceType: kind, Resource: name, Action: action, Status: "succeeded", Decision: "allow"})
	})
	observeCtx, stopObserve := context.WithCancel(context.Background())
	observeDone := make(chan struct{})
	defer func() { stopObserve(); <-observeDone }()
	go func() {
		defer close(observeDone)
		// ponytail: 轮询可能遗漏短暂修改；需要精确历史时应统一资源写入入口。
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-observeCtx.Done():
				return
			case <-ticker.C:
				if err := resourceStore.Observe(observeCtx); err != nil && observeCtx.Err() == nil {
					logger.Error("resource audit failed")
				}
			}
		}
	}()
	for _, item := range []struct{ name, env, scope string }{{"mcp", "AI_OPS_GATEWAY_MCP_TOKEN", auth.ScopeMCP}, {"rest", "AI_OPS_GATEWAY_REST_TOKEN", auth.ScopeREST}, {"admin", "AI_OPS_GATEWAY_ADMIN_TOKEN", auth.ScopeRESTMCP}} {
		token, configured := os.LookupEnv(item.env)
		if !configured {
			continue
		}
		if err := authStore.Upsert(context.Background(), item.name, token, item.scope); err != nil {
			return err
		}
	}

	databases := database.NewManager()
	defer databases.Close()
	databases.SetResolver(func(name string) (*database.Adapter, error) {
		item, err := resourceStore.Database(context.Background(), name)
		if err != nil {
			return nil, err
		}
		return database.Open(database.Config{Name: item.Name, Driver: item.Driver, DSN: item.DSN, MaxOpenConns: 4})
	})

	kubernetes := kube.NewManager()
	kubernetes.SetResolver(func(name string) (*kube.Client, error) {
		item, err := resourceStore.Kubernetes(context.Background(), name)
		if err != nil {
			return nil, err
		}
		return kube.FromKubeconfig(resources.ExpandPath(item.Kubeconfig), item.Context, item.Token)
	})
	linuxHosts := linux.NewManager()
	defer linuxHosts.Close()
	linuxHosts.SetResolver(func(name string) (*linux.Client, error) {
		item, err := resourceStore.Linux(context.Background(), name)
		if err != nil {
			return nil, err
		}
		return linux.Open(linux.Config{
			Name: item.Name, Address: item.Address, User: item.User,
			Password: item.Password, PrivateKey: item.PrivateKey,
			KnownHostsPath: resources.ExpandPath(item.KnownHostsPath),
		})
	})

	e := server.New(paths, logger, webFS, store, authStore, databases, kubernetes, linuxHosts, resourceStore)
	metrics := &http.Server{Addr: "127.0.0.1:9464", Handler: promhttp.Handler(), ReadHeaderTimeout: 5 * time.Second}
	if metrics.Addr != "" {
		go func() {
			if err := metrics.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Error("metrics server stopped", "error", err)
			}
		}()
		defer metrics.Close()
	}
	logger.Info("gateway listening", "addr", "127.0.0.1:9095")
	return e.Start("127.0.0.1:9095")
}
