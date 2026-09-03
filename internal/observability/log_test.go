package observability

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDailyWriter(t *testing.T) {
	dir := t.TempDir()
	w := &DailyWriter{dir: dir}
	if _, err := w.Write([]byte("ok\n")); err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	b, err := os.ReadFile(filepath.Join(dir, "ai-ops-gateway-"+time.Now().Format(time.DateOnly)+".log"))
	if err != nil || string(b) != "ok\n" {
		t.Fatalf("content=%q err=%v", b, err)
	}
}
