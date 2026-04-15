package logging

import (
	"log/slog"
	"testing"

	"github.com/APICerberus/APICerebrus/internal/config"
)

func TestParseLevel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  slog.Level
		err   bool
	}{
		{"", slog.LevelInfo, false},
		{"info", slog.LevelInfo, false},
		{"debug", slog.LevelDebug, false},
		{"warn", slog.LevelWarn, false},
		{"warning", slog.LevelWarn, false},
		{"error", slog.LevelError, false},
		{"INFO", slog.LevelInfo, false},
		{"  Debug  ", slog.LevelDebug, false},
		{"invalid", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got, err := parseLevel(tt.input)
			if tt.err {
				if err == nil {
					t.Error("expected error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if got != tt.want {
					t.Errorf("parseLevel(%q) = %v, want %v", tt.input, got, tt.want)
				}
			}
		})
	}
}

func TestBuildOutput_Stdout(t *testing.T) {
	t.Parallel()
	w, closer, err := buildOutput(config.LoggingConfig{Output: "stdout"})
	if err != nil {
		t.Fatalf("buildOutput: %v", err)
	}
	if w == nil {
		t.Error("expected non-nil writer")
	}
	if closer != nil {
		t.Error("stdout should have nil closer")
	}
}

func TestBuildOutput_Empty(t *testing.T) {
	t.Parallel()
	w, _, err := buildOutput(config.LoggingConfig{Output: ""})
	if err != nil {
		t.Fatalf("buildOutput: %v", err)
	}
	if w == nil {
		t.Error("expected non-nil writer")
	}
}

func TestBuildOutput_Stderr(t *testing.T) {
	t.Parallel()
	w, _, err := buildOutput(config.LoggingConfig{Output: "stderr"})
	if err != nil {
		t.Fatalf("buildOutput: %v", err)
	}
	if w == nil {
		t.Error("expected non-nil writer")
	}
}

func TestBuildOutput_Unsupported(t *testing.T) {
	t.Parallel()
	_, _, err := buildOutput(config.LoggingConfig{Output: "network"})
	if err == nil {
		t.Error("expected error for unsupported output")
	}
}

func TestBuildOutput_FileWithoutPath(t *testing.T) {
	t.Parallel()
	_, _, err := buildOutput(config.LoggingConfig{Output: "file", File: ""})
	if err == nil {
		t.Error("expected error for file output without path")
	}
}
