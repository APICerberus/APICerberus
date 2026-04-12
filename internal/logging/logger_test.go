package logging

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/APICerberus/APICerebrus/internal/config"
)

func TestSetupInvalidLevel(t *testing.T) {
	t.Parallel()

	_, _, err := Setup(config.LoggingConfig{
		Level:  "invalid",
		Format: "json",
		Output: "stdout",
	})
	if err == nil {
		t.Fatalf("expected invalid level error")
	}
}

func TestWithRequestAddsFields(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	base := slog.New(slog.NewJSONHandler(&out, nil))
	l := WithRequest(base, "req-1", "users", "GET")
	l.Info("test-message")

	body := out.String()
	for _, want := range []string{`"correlation_id":"req-1"`, `"route":"users"`, `"method":"GET"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected log to contain %s, got %s", want, body)
		}
	}
}

func TestRotatingFileWriter(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "gateway.log")

	w, err := newRotatingFileWriter(path, 64, 2, false)
	if err != nil {
		t.Fatalf("newRotatingFileWriter error: %v", err)
	}
	defer w.Close()

	chunk := strings.Repeat("x", 80)
	if _, err := w.Write([]byte(chunk)); err != nil {
		t.Fatalf("write 1 failed: %v", err)
	}
	if _, err := w.Write([]byte(chunk)); err != nil {
		t.Fatalf("write 2 failed: %v", err)
	}

	if _, err := os.Stat(path + ".1"); err != nil {
		t.Fatalf("expected rotated file %s.1 to exist: %v", path, err)
	}
}

func TestSetupFileOutput(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "app.log")
	logger, cleanup, err := Setup(config.LoggingConfig{
		Level:  "info",
		Format: "json",
		Output: "file",
		File:   filePath,
		Rotation: config.LogRotationConfig{
			MaxSizeMB:  1,
			MaxBackups: 2,
			Compress:   false,
		},
	})
	if err != nil {
		t.Fatalf("Setup error: %v", err)
	}
	defer func() { _ = cleanup() }()

	logger.Info("hello", "k", "v")
	if err := cleanup(); err != nil {
		t.Fatalf("cleanup error: %v", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read log file error: %v", err)
	}
	if !strings.Contains(string(data), `"msg":"hello"`) {
		t.Fatalf("expected log file to contain message, got: %s", string(data))
	}
}
