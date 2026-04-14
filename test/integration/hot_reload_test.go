package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/APICerberus/APICerebrus/internal/config"
)

// configYAML builds a valid config YAML with the given overrides.
// Uses forward slashes for paths to avoid YAML backslash-escape issues on Windows.
func configYAML(dbPath, httpAddr, apiKey string) []byte {
	db := strings.ReplaceAll(dbPath, `\`, `/`)
	return []byte(`gateway:
  http_addr: "` + httpAddr + `"
admin:
  addr: ":19876"
  api_key: "` + apiKey + `"
  token_secret: "test-admin-token-secret-at-least-32-chars-long"
store:
  path: "` + db + `"
`)
}

func TestConfigWatchDetectsFileChange(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test-config.yaml")
	dbPath := filepath.Join(tmpDir, "test.db")

	if err := os.WriteFile(cfgPath, configYAML(dbPath, ":18080", "test-admin-key-at-least-32-characters-long"), 0644); err != nil {
		t.Fatalf("write initial config: %v", err)
	}

	// Ensure file mtime is in the past so the change is detectable
	past := time.Now().Add(-5 * time.Second)
	if err := os.Chtimes(cfgPath, past, past); err != nil {
		t.Fatalf("set mtime: %v", err)
	}

	onChange := make(chan struct{}, 1)

	stop, err := config.Watch(cfgPath, func(cfg *config.Config, loadErr error) {
		if loadErr != nil {
			t.Logf("onChange received error: %v", loadErr)
			return
		}
		if cfg != nil && cfg.Admin.APIKey == "updated-key-at-least-32-characters-long" {
			select {
			case onChange <- struct{}{}:
			default:
			}
		}
	})
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}
	defer stop()

	// Allow watcher to start
	time.Sleep(500 * time.Millisecond)

	// Modify the config file
	if err := os.WriteFile(cfgPath, configYAML(dbPath, ":18081", "updated-key-at-least-32-characters-long"), 0644); err != nil {
		t.Fatalf("write updated config: %v", err)
	}

	select {
	case <-onChange:
		// Success: file change was detected and callback fired with new config
	case <-time.After(6 * time.Second):
		t.Fatal("timed out waiting for config file change detection")
	}
}

func TestConfigWatchReturnsErrorForMissingFile(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "nonexistent", "config.yaml")
	_, err := config.Watch(missingPath, nil)
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestConfigWatchStopDoesNotPanic(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	dbPath := filepath.Join(tmpDir, "test.db")

	if err := os.WriteFile(cfgPath, configYAML(dbPath, ":18080", "test-admin-key-at-least-32-characters-long"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	stop, err := config.Watch(cfgPath, func(_ *config.Config, _ error) {})
	if err != nil {
		t.Fatalf("Watch failed: %v", err)
	}

	// Calling stop twice should not panic
	stop()
	stop()
}
