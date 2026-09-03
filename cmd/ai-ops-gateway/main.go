package main

import (
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gcc798/ai-ops-gateway/internal/audit"
	"github.com/gcc798/ai-ops-gateway/internal/config"
	"github.com/gcc798/ai-ops-gateway/internal/database"
	kube "github.com/gcc798/ai-ops-gateway/internal/kubernetes"
	linux "github.com/gcc798/ai-ops-gateway/internal/linux"
	"github.com/gcc798/ai-ops-gateway/internal/observability"
	"github.com/gcc798/ai-ops-gateway/internal/server"
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
	paths := config.ResolvePaths()
	if err := paths.Ensure(); err != nil {
		return err
	}
	logger, logs := observability.NewLogger(paths.Logs)
	defer logs.Close()
	return run(logger, paths)
}

func run(logger *slog.Logger, paths config.Paths) error {
	snapshot, err := config.NewManager(paths).Load()
	if err != nil {
		return err
	}
	store, err := audit.Open(filepath.Join(paths.Data, "ai-ops-gateway.db"))
	if err != nil {
		return err
	}
	defer store.Close()

	databases := database.NewManager()
	defer databases.Close()
	for _, item := range snapshot.Databases {
		if err := registerDatabase(databases, item.Name, item.Driver, os.Getenv(item.DSNEnv)); err != nil {
			return err
		}
	}

	kubernetes := kube.NewManager()
	for _, item := range snapshot.Kubernetes {
		client, err := kube.FromKubeconfig(config.ExpandPath(item.Kubeconfig), item.Context)
		if err != nil {
			return fmt.Errorf("configure kubernetes %s: %w", item.Name, err)
		}
		kubernetes.Add(item.Name, client)
	}
	linuxHosts := linux.NewManager()
	defer linuxHosts.Close()
	for _, item := range snapshot.LinuxHosts {
		client, err := linux.Open(linux.Config{Name: item.Name, Address: item.Address, User: item.User, Password: os.Getenv(item.PasswordEnv), PrivateKeyPath: config.ExpandPath(item.PrivateKeyPath), KnownHostsPath: config.ExpandPath(item.KnownHostsPath)})
		if err != nil {
			return fmt.Errorf("configure linux %s: %w", item.Name, err)
		}
		linuxHosts.Add(item.Name, client)
	}

	e := server.New(paths, logger, webFS, store, databases, kubernetes, linuxHosts)
	metrics := &http.Server{Addr: snapshot.Gateway.MetricsAddress, Handler: promhttp.Handler(), ReadHeaderTimeout: 5 * time.Second}
	if snapshot.Gateway.MetricsAddress != "" {
		go func() {
			if err := metrics.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Error("metrics server stopped", "error", err)
			}
		}()
		defer metrics.Close()
	}
	logger.Info("gateway listening", "addr", snapshot.Gateway.ListenAddress)
	return e.Start(snapshot.Gateway.ListenAddress)
}

func registerDatabase(manager *database.Manager, name, driver, dsn string) error {
	if dsn == "" {
		return fmt.Errorf("database %s: referenced DSN environment variable is empty", name)
	}
	adapter, err := database.Open(database.Config{Name: name, Driver: driver, DSN: dsn, MaxOpenConns: 4})
	if err != nil {
		return fmt.Errorf("configure database %s: %w", name, err)
	}
	if err := manager.Add(adapter); err != nil {
		adapter.Close()
		return err
	}
	return nil
}
