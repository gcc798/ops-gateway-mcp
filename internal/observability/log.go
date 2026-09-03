package observability

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type DailyWriter struct {
	mu   sync.Mutex
	dir  string
	date string
	file *os.File
}

func NewLogger(dir string) (*slog.Logger, *DailyWriter) {
	w := &DailyWriter{dir: dir}
	return slog.New(slog.NewJSONHandler(io.MultiWriter(os.Stdout, w), nil)), w
}

func (w *DailyWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	today := time.Now().Format(time.DateOnly)
	if w.date != today {
		if w.file != nil {
			_ = w.file.Close()
		}
		file, err := os.OpenFile(filepath.Join(w.dir, "ai-ops-gateway-"+today+".log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			return 0, fmt.Errorf("open daily log: %w", err)
		}
		w.file, w.date = file, today
	}
	return w.file.Write(p)
}

func (w *DailyWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}
