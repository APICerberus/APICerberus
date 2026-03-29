package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/APICerberus/APICerebrus/internal/config"
)

// Setup creates a configured slog logger and returns a cleanup function.
func Setup(cfg config.LoggingConfig) (*slog.Logger, func() error, error) {
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, nil, err
	}

	out, closer, err := buildOutput(cfg)
	if err != nil {
		return nil, nil, err
	}

	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	switch strings.ToLower(strings.TrimSpace(cfg.Format)) {
	case "", "json":
		handler = slog.NewJSONHandler(out, opts)
	case "text":
		handler = slog.NewTextHandler(out, opts)
	default:
		if closer != nil {
			_ = closer.Close()
		}
		return nil, nil, fmt.Errorf("unsupported logging format: %s", cfg.Format)
	}

	cleanup := func() error {
		if closer == nil {
			return nil
		}
		return closer.Close()
	}
	return slog.New(handler), cleanup, nil
}

// WithRequest enriches the logger with request-scoped metadata.
func WithRequest(logger *slog.Logger, correlationID, route, method string) *slog.Logger {
	if logger == nil {
		logger = slog.Default()
	}
	return logger.With(
		slog.String("correlation_id", correlationID),
		slog.String("route", route),
		slog.String("method", method),
	)
}

func parseLevel(level string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unsupported log level: %s", level)
	}
}

func buildOutput(cfg config.LoggingConfig) (io.Writer, io.Closer, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Output)) {
	case "", "stdout":
		return os.Stdout, nil, nil
	case "stderr":
		return os.Stderr, nil, nil
	case "file":
		if strings.TrimSpace(cfg.File) == "" {
			return nil, nil, fmt.Errorf("logging.file is required when logging.output=file")
		}
		w, err := newRotatingFileWriter(
			cfg.File,
			int64(cfg.Rotation.MaxSizeMB)*1024*1024,
			cfg.Rotation.MaxBackups,
			cfg.Rotation.Compress,
		)
		if err != nil {
			return nil, nil, err
		}
		return w, w, nil
	default:
		return nil, nil, fmt.Errorf("unsupported logging output: %s", cfg.Output)
	}
}
