package kubernetes

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestDownloadFileRequiresPath(t *testing.T) {
	c := &Client{}
	if _, err := c.DownloadFile(context.Background(), "default", "pod", "", "", 0); err == nil {
		t.Fatal("expected empty path error")
	}
}

func TestIntegration(t *testing.T) {
	path := os.Getenv("KUBECONFIG")
	if path == "" {
		t.Skip("KUBECONFIG not set")
	}
	client, err := FromKubeconfig(path, "")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := client.ListPods(ctx, "kube-system"); err != nil {
		t.Fatal(err)
	}
}
