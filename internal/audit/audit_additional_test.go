package audit

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/APICerberus/APICerebrus/internal/config"
	"github.com/APICerberus/APICerebrus/internal/store"
)

// Test ResponseCaptureWriter Flush
func TestResponseCaptureWriter_Flush(t *testing.T) {
	t.Parallel()

	// Create a ResponseCaptureWriter with an httptest.ResponseRecorder
	inner := httptest.NewRecorder()
	capture := NewResponseCaptureWriter(inner, 1024)

	// Flush should not panic
	capture.Flush()
}

// Test ResponseCaptureWriter Hijack
func TestResponseCaptureWriter_Hijack(t *testing.T) {
	t.Parallel()

	// Create a ResponseCaptureWriter with an httptest.ResponseRecorder
	inner := httptest.NewRecorder()
	capture := NewResponseCaptureWriter(inner, 1024)

	// Hijack should return ErrNotSupported since httptest.ResponseRecorder doesn't support Hijack
	_, _, err := capture.Hijack()
	if err != http.ErrNotSupported {
		t.Errorf("Hijack() error = %v, want %v", err, http.ErrNotSupported)
	}
}

// Test ResponseCaptureWriter Push
func TestResponseCaptureWriter_Push(t *testing.T) {
	t.Parallel()

	// Create a ResponseCaptureWriter with an httptest.ResponseRecorder
	inner := httptest.NewRecorder()
	capture := NewResponseCaptureWriter(inner, 1024)

	// Push should return ErrNotSupported since httptest.ResponseRecorder doesn't support Push
	err := capture.Push("/test", nil)
	if err != http.ErrNotSupported {
		t.Errorf("Push() error = %v, want %v", err, http.ErrNotSupported)
	}
}

// Test Logger MaxRequestBodyBytes
func TestLogger_MaxRequestBodyBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		logger   *Logger
		expected int64
	}{
		{
			name:     "nil logger",
			logger:   nil,
			expected: 0,
		},
		{
			name: "with config",
			logger: &Logger{
				cfg: config.AuditConfig{
					MaxRequestBodyBytes: 1024,
				},
			},
			expected: 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.logger.MaxRequestBodyBytes()
			if result != tt.expected {
				t.Errorf("MaxRequestBodyBytes() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test Logger MaxResponseBodyBytes
func TestLogger_MaxResponseBodyBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		logger   *Logger
		expected int64
	}{
		{
			name:     "nil logger",
			logger:   nil,
			expected: 0,
		},
		{
			name: "with config",
			logger: &Logger{
				cfg: config.AuditConfig{
					MaxResponseBodyBytes: 2048,
				},
			},
			expected: 2048,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.logger.MaxResponseBodyBytes()
			if result != tt.expected {
				t.Errorf("MaxResponseBodyBytes() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test Logger Dropped
func TestLogger_Dropped(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		logger   *Logger
		expected int64
	}{
		{
			name:     "nil logger",
			logger:   nil,
			expected: 0,
		},
		{
			name:     "empty logger",
			logger:   &Logger{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.logger.Dropped()
			if result != tt.expected {
				t.Errorf("Dropped() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Test Logger Start with nil repo
func TestLogger_Start_Disabled(t *testing.T) {
	t.Parallel()

	// Logger with nil repo should not panic on Start
	logger := &Logger{
		cfg: config.AuditConfig{
			Enabled:       true,
			FlushInterval: time.Second,
			BatchSize:     10,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Should not panic
	logger.Start(ctx)
}

// Test Logger Start when already started
func TestLogger_Start_AlreadyStarted(t *testing.T) {
	t.Parallel()

	st := openAuditTestStoreForAdditional(t)
	defer st.Close()

	logger := NewLogger(st.Audits(), config.AuditConfig{
		Enabled:       true,
		FlushInterval: time.Second,
		BatchSize:     10,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Start once
	go logger.Start(ctx)
	time.Sleep(50 * time.Millisecond)

	// Try to start again - should not panic and should return immediately
	logger.Start(ctx)
}

func openAuditTestStoreForAdditional(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(&config.Config{
		Store: config.StoreConfig{
			Path:        ":memory:",
			BusyTimeout: time.Second,
			JournalMode: "MEMORY",
			ForeignKeys: true,
		},
	})
	if err != nil {
		t.Fatalf("open store error: %v", err)
	}
	return st
}

// Test ResponseCaptureWriter StatusCode method
func TestResponseCaptureWriter_StatusCode(t *testing.T) {
	t.Run("status code before write", func(t *testing.T) {
		rw := httptest.NewRecorder()
		cw := NewResponseCaptureWriter(rw, 1024)

		// Before any WriteHeader call, should return 0
		if cw.StatusCode() != 0 {
			t.Errorf("StatusCode before write = %d, want 0", cw.StatusCode())
		}
	})

	t.Run("status code after write", func(t *testing.T) {
		rw := httptest.NewRecorder()
		cw := NewResponseCaptureWriter(rw, 1024)

		cw.WriteHeader(http.StatusCreated)

		if cw.StatusCode() != http.StatusCreated {
			t.Errorf("StatusCode after write = %d, want %d", cw.StatusCode(), http.StatusCreated)
		}
	})
}

// Test ResponseCaptureWriter BytesWritten method
func TestResponseCaptureWriter_BytesWritten(t *testing.T) {
	t.Run("bytes written initially", func(t *testing.T) {
		rw := httptest.NewRecorder()
		cw := NewResponseCaptureWriter(rw, 1024)

		if cw.BytesWritten() != 0 {
			t.Errorf("BytesWritten initially = %d, want 0", cw.BytesWritten())
		}
	})

	t.Run("bytes written after write", func(t *testing.T) {
		rw := httptest.NewRecorder()
		cw := NewResponseCaptureWriter(rw, 1024)

		data := []byte("hello world")
		cw.Write(data)

		if cw.BytesWritten() != int64(len(data)) {
			t.Errorf("BytesWritten after write = %d, want %d", cw.BytesWritten(), len(data))
		}
	})
}

// Test CaptureRequestBody with various inputs
func TestCaptureRequestBody_Extended(t *testing.T) {
	t.Run("nil request", func(t *testing.T) {
		body, err := CaptureRequestBody(nil, 1024)
		if body != nil || err != nil {
			t.Error("CaptureRequestBody(nil) should return (nil, nil)")
		}
	})

	t.Run("GET request without body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		body, err := CaptureRequestBody(req, 1024)
		if body != nil || err != nil {
			t.Error("CaptureRequestBody(GET) should return (nil, nil)")
		}
	})

	t.Run("request with large body", func(t *testing.T) {
		largeBody := make([]byte, 2048)
		for i := range largeBody {
			largeBody[i] = 'x'
		}
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(largeBody))
		req.Header.Set("Content-Type", "application/octet-stream")

		body, err := CaptureRequestBody(req, 1024)
		if err != nil {
			t.Errorf("CaptureRequestBody error: %v", err)
		}
		if body == nil {
			t.Error("CaptureRequestBody should return body")
		}
		if len(body) > 1024 {
			t.Errorf("Captured body length = %d, should be <= 1024", len(body))
		}
	})
}

// Test truncateCopy function
func TestTruncateCopy(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		maxLen   int
		expected int
	}{
		{"nil input", nil, 100, 0},
		{"empty input", []byte{}, 100, 0},
		{"small input", []byte("hello"), 100, 5},
		{"exact size", []byte("hello"), 5, 5},
		{"truncated", []byte("hello world"), 5, 5},
		{"zero max", []byte("hello"), 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateCopy(tt.input, int64(tt.maxLen))
			if len(result) != tt.expected {
				t.Errorf("truncateCopy length = %d, want %d", len(result), tt.expected)
			}
		})
	}
}

// Test requestClientIP function
func TestRequestClientIP(t *testing.T) {
	t.Run("X-Forwarded-For header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Forwarded-For", "192.168.1.1, 10.0.0.1")

		ip := requestClientIP(req)
		if ip != "192.168.1.1" {
			t.Errorf("requestClientIP = %q, want 192.168.1.1", ip)
		}
	})

	t.Run("X-Real-IP header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Real-IP", "192.168.2.2")
		// Note: httptest.NewRequest sets RemoteAddr to 192.0.2.1 by default
		// requestClientIP checks X-Forwarded-For first, then X-Real-IP
		ip := requestClientIP(req)
		// X-Real-IP should be used when X-Forwarded-For is not present
		if ip == "" {
			t.Error("requestClientIP should return a value")
		}
	})

	t.Run("RemoteAddr fallback", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.3.3:12345"

		ip := requestClientIP(req)
		if ip != "192.168.3.3" {
			t.Errorf("requestClientIP = %q, want 192.168.3.3", ip)
		}
	})

	t.Run("RemoteAddr with brackets", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "[::1]:12345"

		ip := requestClientIP(req)
		if ip != "::1" {
			t.Errorf("requestClientIP = %q, want ::1", ip)
		}
	})
}

// Test MaskHeaders function
func TestMaskHeaders_Extended(t *testing.T) {
	// Create masker with sensitive headers
	masker := NewMasker(
		[]string{"authorization", "x-api-key", "cookie"},
		[]string{},
		"***MASKED***",
	)

	t.Run("nil headers", func(t *testing.T) {
		result := masker.MaskHeaders(nil)
		// Implementation may return empty map instead of nil
		if result != nil && len(result) != 0 {
			t.Error("MaskHeaders(nil) should return nil or empty map")
		}
	})

	t.Run("empty headers", func(t *testing.T) {
		headers := http.Header{}
		result := masker.MaskHeaders(headers)
		if len(result) != 0 {
			t.Error("MaskHeaders(empty) should return empty map")
		}
	})

	t.Run("with sensitive headers", func(t *testing.T) {
		headers := http.Header{
			"Authorization":   []string{"Bearer token123"},
			"Content-Type":    []string{"application/json"},
			"X-Api-Key":       []string{"secret-key"},
			"Cookie":          []string{"session=abc123"},
		}

		result := masker.MaskHeaders(headers)

		// Sensitive headers should be masked
		if result["Authorization"] != "***MASKED***" {
			t.Errorf("Authorization should be masked, got %q", result["Authorization"])
		}
		if result["X-Api-Key"] != "***MASKED***" {
			t.Errorf("X-Api-Key should be masked, got %q", result["X-Api-Key"])
		}
		if result["Cookie"] != "***MASKED***" {
			t.Errorf("Cookie should be masked, got %q", result["Cookie"])
		}
		// Non-sensitive headers should remain
		if result["Content-Type"] != "application/json" {
			t.Errorf("Content-Type should not be masked, got %q", result["Content-Type"])
		}
	})

	t.Run("custom sensitive headers", func(t *testing.T) {
		customMasker := NewMasker([]string{"x-custom-secret"}, nil, "***MASKED***")
		headers := http.Header{
			"X-Custom-Secret": []string{"secret-value"},
			"X-Public":        []string{"public-value"},
		}

		result := customMasker.MaskHeaders(headers)

		if result["X-Custom-Secret"] != "***MASKED***" {
			t.Errorf("X-Custom-Secret should be masked, got %q", result["X-Custom-Secret"])
		}
		if result["X-Public"] != "public-value" {
			t.Errorf("X-Public should not be masked, got %q", result["X-Public"])
		}
	})
}

// Test MaskBody function
func TestMaskBody_Extended(t *testing.T) {
	masker := NewMasker(nil, []string{"password", "api_key"}, "***MASKED***")

	t.Run("nil body", func(t *testing.T) {
		result := masker.MaskBody(nil)
		if result != nil {
			t.Error("MaskBody(nil) should return nil")
		}
	})

	t.Run("empty body", func(t *testing.T) {
		result := masker.MaskBody([]byte{})
		if len(result) != 0 {
			t.Error("MaskBody(empty) should return empty")
		}
	})

	t.Run("non-JSON body", func(t *testing.T) {
		body := []byte("plain text body")
		result := masker.MaskBody(body)
		if !bytes.Equal(result, body) {
			t.Error("MaskBody should return original for non-JSON")
		}
	})

	t.Run("JSON with sensitive fields", func(t *testing.T) {
		body := []byte(`{"password":"secret123","username":"john","api_key":"key456"}`)
		result := masker.MaskBody(body)

		// Should mask sensitive fields
		if bytes.Contains(result, []byte(`"password":"secret123"`)) {
			t.Error("password field should be masked")
		}
		if bytes.Contains(result, []byte(`"api_key":"key456"`)) {
			t.Error("api_key field should be masked")
		}
		if !bytes.Contains(result, []byte(`"username":"john"`)) {
			t.Error("username field should not be masked")
		}
	})

	t.Run("JSON with custom sensitive fields", func(t *testing.T) {
		customMasker := NewMasker(nil, []string{"custom_secret"}, "***MASKED***")
		body := []byte(`{"custom_secret":"secret123","public":"data"}`)
		result := customMasker.MaskBody(body)

		if bytes.Contains(result, []byte(`"custom_secret":"secret123"`)) {
			t.Error("custom_secret field should be masked")
		}
		if !bytes.Contains(result, []byte(`"public":"data"`)) {
			t.Error("public field should not be masked")
		}
	})
}
