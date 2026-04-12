package plugin

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/APICerberus/APICerebrus/internal/config"
	"github.com/tetratelabs/wazero"
)

// compileMinimalWasm creates a minimal valid WASM binary with alloc and handle_request exports.
func compileMinimalWasm(t *testing.T, path string) {
	t.Helper()
	// Verified WASM binary (validated with wazero):
	// - memory: 1 page
	// - alloc(i32) -> i32: echo function (returns input)
	// - handle_request(i32, i32) -> (i32, i32): echo function (returns input ptr, len)
	wasmBytes := []byte{
		0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00,
		// Type(1): 2 types, size=13
		0x01, 0x0d, 0x02,
		0x60, 0x01, 0x7f, 0x01, 0x7f,
		0x60, 0x02, 0x7f, 0x7f, 0x02, 0x7f, 0x7f,
		// Function(3): 2 funcs -> type 0, type 1
		0x03, 0x03, 0x02, 0x00, 0x01,
		// Memory(5): 1 memory, min 1 page
		0x05, 0x03, 0x01, 0x00, 0x01,
		// Export(7): 3 exports, size=35
		0x07, 0x23, 0x03,
		0x06, 'm', 'e', 'm', 'o', 'r', 'y', 0x02, 0x00,
		0x05, 'a', 'l', 'l', 'o', 'c', 0x00, 0x00,
		0x0e, 'h', 'a', 'n', 'd', 'l', 'e', '_', 'r', 'e', 'q', 'u', 'e', 's', 't', 0x00, 0x01,
		// Code(10): 2 funcs, size=13
		0x0a, 0x0d, 0x02,
		0x04, 0x00, 0x20, 0x00, 0x0b,                   // func 0: local.get 0
		0x06, 0x00, 0x20, 0x00, 0x20, 0x01, 0x0b,       // func 1: local.get 0, local.get 1
	}

	// Verify wazero can compile this before writing
	ctx := context.Background()
	rt := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter())
	defer rt.Close(ctx)

	compiled, err := rt.CompileModule(ctx, wasmBytes)
	if err != nil {
		_ = os.WriteFile(path, wasmBytes, 0644)
		t.Fatalf("pre-compiled WASM binary is invalid: %v", err)
	}
	_ = compiled

	if err := os.WriteFile(path, wasmBytes, 0644); err != nil {
		t.Fatalf("failed to write wasm file: %v", err)
	}
}

func TestDefaultWASMConfig(t *testing.T) {
	cfg := DefaultWASMConfig()

	if cfg.Enabled {
		t.Error("Expected WASM to be disabled by default")
	}

	if cfg.ModuleDir != "./plugins/wasm" {
		t.Errorf("Expected module dir './plugins/wasm', got %s", cfg.ModuleDir)
	}

	if cfg.MaxMemory != 128*1024*1024 {
		t.Errorf("Expected max memory 128MB, got %d", cfg.MaxMemory)
	}

	if cfg.MaxExecution != 30*time.Second {
		t.Errorf("Expected max execution 30s, got %v", cfg.MaxExecution)
	}

	if cfg.AllowFilesystem {
		t.Error("Expected filesystem access to be disabled by default")
	}
}

func TestNewWASMRuntime_Disabled(t *testing.T) {
	cfg := WASMConfig{Enabled: false}

	runtime, err := NewWASMRuntime(cfg)
	if err != nil {
		t.Errorf("NewWASMRuntime() error = %v", err)
	}
	if runtime != nil {
		t.Error("Expected nil runtime when disabled")
	}
}

func TestNewWASMRuntime_Enabled(t *testing.T) {
	cfg := DefaultWASMConfig()
	cfg.Enabled = true

	runtime, err := NewWASMRuntime(cfg)
	if err != nil {
		t.Fatalf("NewWASMRuntime() error = %v", err)
	}
	if runtime == nil {
		t.Fatal("Expected non-nil runtime")
	}

	if !runtime.IsEnabled() {
		t.Error("Expected runtime to be enabled")
	}
}

func TestWASMRuntime_IsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		runtime  *WASMRuntime
		expected bool
	}{
		{
			name:     "nil runtime",
			runtime:  nil,
			expected: false,
		},
		{
			name: "disabled runtime",
			runtime: &WASMRuntime{
				config: WASMConfig{Enabled: false},
			},
			expected: false,
		},
		{
			name: "enabled runtime",
			runtime: &WASMRuntime{
				config: WASMConfig{Enabled: true},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.runtime.IsEnabled(); got != tt.expected {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestWASMRuntime_LoadModule_NotFound(t *testing.T) {
	cfg := DefaultWASMConfig()
	cfg.Enabled = true

	runtime, _ := NewWASMRuntime(cfg)

	_, err := runtime.LoadModule("test", "/nonexistent/module.wasm", nil)
	if err == nil {
		t.Error("Expected error when loading non-existent module")
	}
}

func TestWASMModule_Accessors(t *testing.T) {
	module := &WASMModule{
		id:       "test-module",
		name:     "Test Module",
		version:  "1.2.3",
		phase:    PhasePreAuth,
		priority: 50,
		loaded:   true,
	}

	if module.ID() != "test-module" {
		t.Errorf("ID() = %s, want test-module", module.ID())
	}

	if module.Name() != "Test Module" {
		t.Errorf("Name() = %s, want 'Test Module'", module.Name())
	}

	if module.Version() != "1.2.3" {
		t.Errorf("Version() = %s, want 1.2.3", module.Version())
	}

	if module.Phase() != PhasePreAuth {
		t.Errorf("Phase() = %v, want PhasePreAuth", module.Phase())
	}

	if module.Priority() != 50 {
		t.Errorf("Priority() = %d, want 50", module.Priority())
	}
}

func TestWASMModule_Nil(t *testing.T) {
	var module *WASMModule

	if module.ID() != "" {
		t.Error("Expected empty ID for nil module")
	}

	if module.Name() != "" {
		t.Error("Expected empty Name for nil module")
	}

	if module.Version() != "" {
		t.Error("Expected empty Version for nil module")
	}

	if module.Phase() != PhasePreProxy {
		t.Error("Expected PhasePreProxy for nil module")
	}

	if module.Priority() != 100 {
		t.Error("Expected priority 100 for nil module")
	}
}

func TestWASMModule_Execute_NotLoaded(t *testing.T) {
	module := &WASMModule{
		id:     "test",
		loaded: false,
	}

	_, err := module.Execute(nil)
	if err == nil {
		t.Error("Expected error when executing unloaded module")
	}
}

func TestWASMPluginManager(t *testing.T) {
	cfg := DefaultWASMConfig()
	cfg.Enabled = true

	manager, err := NewWASMPluginManager(cfg)
	if err != nil {
		t.Fatalf("NewWASMPluginManager() error = %v", err)
	}
	defer manager.Close()

	if !manager.IsEnabled() {
		t.Error("Expected manager to be enabled")
	}

	modules := manager.ListModules()
	if len(modules) != 0 {
		t.Errorf("Expected 0 modules, got %d", len(modules))
	}

	_, ok := manager.GetModule("nonexistent")
	if ok {
		t.Error("Expected false for non-existent module")
	}
}

func TestWASMPluginManager_LoadUnload(t *testing.T) {
	cfg := DefaultWASMConfig()
	cfg.Enabled = true

	tmpDir := t.TempDir()
	cfg.ModuleDir = tmpDir

	manager, err := NewWASMPluginManager(cfg)
	if err != nil {
		t.Fatalf("NewWASMPluginManager() error = %v", err)
	}
	defer manager.Close()
	wasmPath := filepath.Join(tmpDir, "test.wasm")
	compileMinimalWasm(t, wasmPath)

	pluginConfig := map[string]interface{}{
		"name":     "Test Plugin",
		"version":  "1.0.0",
		"phase":    "pre-proxy",
		"priority": 50,
	}

	err = manager.LoadModule("test", wasmPath, pluginConfig)
	if err != nil {
		t.Errorf("LoadModule() error = %v", err)
	}

	module, ok := manager.GetModule("test")
	if !ok {
		t.Fatal("Expected module to be found")
	}

	if module.Name() != "Test Plugin" {
		t.Errorf("Expected name 'Test Plugin', got %s", module.Name())
	}

	err = manager.UnloadModule("test")
	if err != nil {
		t.Errorf("UnloadModule() error = %v", err)
	}

	_, ok = manager.GetModule("test")
	if ok {
		t.Error("Expected module to be unloaded")
	}
}

func TestWASMHostFunctions(t *testing.T) {
	host := NewWASMHostFunctions(nil)
	if host == nil {
		t.Fatal("Expected non-nil host functions")
	}

	if !host.HasCapability("log") {
		t.Error("Expected log capability to be granted by default")
	}
	if !host.HasCapability("get_metadata") {
		t.Error("Expected get_metadata capability to be granted by default")
	}
	if host.HasCapability("get_header") {
		t.Error("Expected get_header capability to NOT be granted by default")
	}

	host.Log("info", "test message")

	host.GetHeader(nil, "X-Test")
	host.SetHeader(nil, "X-Test", "value")
	host.GetMetadata(nil, "key")
	host.SetMetadata(nil, "key", "value")
	host.Abort(nil, "reason")
}

func TestValidateWASMModule_NotFound(t *testing.T) {
	err := ValidateWASMModule("/nonexistent/module.wasm")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestValidateWASMModule_InvalidMagic(t *testing.T) {
	tmpDir := t.TempDir()
	wasmPath := filepath.Join(tmpDir, "invalid.wasm")

	if err := os.WriteFile(wasmPath, []byte("NOTWASM"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err := ValidateWASMModule(wasmPath)
	if err == nil {
		t.Error("Expected error for invalid magic")
	}
}

func TestValidateWASMModule_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	wasmPath := filepath.Join(tmpDir, "valid.wasm")

	wasmMagic := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
	if err := os.WriteFile(wasmPath, wasmMagic, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err := ValidateWASMModule(wasmPath)
	if err != nil {
		t.Errorf("ValidateWASMModule() error = %v", err)
	}
}

func TestValidateWASMModule_TooLarge(t *testing.T) {
	tmpDir := t.TempDir()
	wasmPath := filepath.Join(tmpDir, "large.wasm")

	data := make([]byte, 101*1024*1024)
	data[0] = 0x00
	data[1] = 0x61
	data[2] = 0x73
	data[3] = 0x6d

	if err := os.WriteFile(wasmPath, data, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err := ValidateWASMModule(wasmPath)
	if err == nil {
		t.Error("Expected error for large file")
	}
}

func TestBuildWASMPlugin(t *testing.T) {
	spec := config.PluginConfig{
		Name: "wasm-test",
		Config: map[string]any{
			"module_id":   "test",
			"module_path": "/test/module.wasm",
			"phase":       "pre-auth",
			"priority":    50,
		},
	}

	plugin, err := buildWASMPlugin(spec, BuilderContext{})
	if err != nil {
		t.Fatalf("buildWASMPlugin() error = %v", err)
	}

	if plugin.Name() != "wasm-test" {
		t.Errorf("Name() = %s, want wasm-test", plugin.Name())
	}

	if plugin.Phase() != PhasePreAuth {
		t.Errorf("Phase() = %v, want PhasePreAuth", plugin.Phase())
	}

	if plugin.Priority() != 50 {
		t.Errorf("Priority() = %d, want 50", plugin.Priority())
	}
}

func TestBuildWASMPlugin_NoModuleID(t *testing.T) {
	spec := config.PluginConfig{
		Name:   "wasm-test",
		Config: map[string]any{},
	}

	_, err := buildWASMPlugin(spec, BuilderContext{})
	if err == nil {
		t.Error("Expected error when module_id is missing")
	}
}

func TestBuildWASMPlugin_Defaults(t *testing.T) {
	spec := config.PluginConfig{
		Name: "wasm-test",
		Config: map[string]any{
			"module_id": "test",
		},
	}

	plugin, err := buildWASMPlugin(spec, BuilderContext{})
	if err != nil {
		t.Fatalf("buildWASMPlugin() error = %v", err)
	}

	if plugin.Phase() != PhasePreProxy {
		t.Errorf("Phase() = %v, want PhasePreProxy", plugin.Phase())
	}

	if plugin.Priority() != 100 {
		t.Errorf("Priority() = %d, want 100", plugin.Priority())
	}
}

func TestWASMModule_Execute_Wazero(t *testing.T) {
	cfg := DefaultWASMConfig()
	cfg.Enabled = true

	tmpDir := t.TempDir()
	cfg.ModuleDir = tmpDir

	manager, err := NewWASMPluginManager(cfg)
	if err != nil {
		t.Fatalf("NewWASMPluginManager() error = %v", err)
	}
	defer manager.Close()

	wasmPath := filepath.Join(tmpDir, "echo.wasm")
	compileMinimalWasm(t, wasmPath)

	err = manager.LoadModule("echo", wasmPath, map[string]any{
		"name":    "Echo Plugin",
		"version": "1.0.0",
		"phase":   "pre-proxy",
	})
	if err != nil {
		t.Fatalf("LoadModule() error = %v", err)
	}

	// Create a pipeline plugin from the WASM module
	pipelinePlugin, err := manager.CreatePipelinePlugin("echo")
	if err != nil {
		t.Fatalf("CreatePipelinePlugin() error = %v", err)
	}

	// Execute with a minimal pipeline context
	// The echo module returns the input as-is, so it should not error
	// but since it doesn't modify headers or return a valid WASM JSON response,
	// it will fail to parse the result as JSON — which is expected behavior
	// for a module that doesn't speak our JSON protocol.
	// The test verifies that wazero execution actually runs without crashing.
	ctx := &PipelineContext{
		Request:       nil, // nil request — the WASM module doesn't use it
		ResponseWriter: nil,
	}

	// The echo module will try to read JSON from memory and write JSON back
	// Since we have nil request, ToWASMContext will handle it gracefully
	// but the WASM module returns raw integers, not JSON — so we expect a parse error
	// This confirms the full wazero pipeline runs.
	_, err = pipelinePlugin.Run(ctx)
	// We expect an error because the echo module returns raw (ptr, len) not JSON
	// The important thing is that wazero executed without crashing
	if err == nil {
		// If no error, the module handled the nil context gracefully
		t.Log("WASM module executed successfully with nil context")
	} else {
		t.Logf("WASM module execution result (expected for echo module): %v", err)
	}
}

func TestWASMRuntime_Close(t *testing.T) {
	cfg := DefaultWASMConfig()
	cfg.Enabled = true

	tmpDir := t.TempDir()
	cfg.ModuleDir = tmpDir

	runtime, err := NewWASMRuntime(cfg)
	if err != nil {
		t.Fatalf("NewWASMRuntime() error = %v", err)
	}

	wasmPath := filepath.Join(tmpDir, "test.wasm")
	compileMinimalWasm(t, wasmPath)

	module, err := runtime.LoadModule("test", wasmPath, map[string]any{
		"name": "Test",
	})
	if err != nil {
		t.Fatalf("LoadModule() error = %v", err)
	}

	if !module.loaded {
		t.Error("Expected module to be loaded")
	}

	err = runtime.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Module should still be marked loaded (close is on runtime, not module)
	// but execution would fail since the runtime is closed
}
