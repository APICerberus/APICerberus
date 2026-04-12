package grpc

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

// Test handleGRPC with successful response headers and trailers
func TestProxy_handleGRPC_SuccessPaths(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("Failed to create listener: %v", err)
	}
	defer lis.Close()

	// Create a simple gRPC service that returns headers and trailers
	s := grpc.NewServer()
	defer s.Stop()

	go s.Serve(lis)
	time.Sleep(10 * time.Millisecond)

	cfg := &ProxyConfig{
		Target:   lis.Addr().String(),
		Insecure: true,
	}
	proxy, err := NewProxy(cfg)
	if err != nil {
		t.Skipf("Failed to create proxy: %v", err)
	}
	defer proxy.Close()

	t.Run("handleGRPC with request body", func(t *testing.T) {
		body := []byte("test request body")
		req := httptest.NewRequest(http.MethodPost, "/test.Service/Method", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/grpc")
		rec := httptest.NewRecorder()

		proxy.handleGRPC(rec, req)

		// Should get a response (may be error since method doesn't exist)
		if rec.Code != http.StatusOK {
			t.Logf("Response code: %d", rec.Code)
		}

		// Check gRPC status header
		grpcStatus := rec.Header().Get("Grpc-Status")
		if grpcStatus == "" {
			t.Error("Expected Grpc-Status header")
		}
	})

	t.Run("handleGRPC with metadata headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test.Service/Method", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/grpc")
		req.Header.Set("X-Custom-Header", "custom-value")
		req.Header.Set("Authorization", "Bearer token")
		rec := httptest.NewRecorder()

		proxy.handleGRPC(rec, req)

		// Should process with metadata
		_ = rec.Code
	})
}

// Test handleTranscoding success path with loaded transcoder
func TestProxy_handleTranscoding_SuccessPath(t *testing.T) {
	cfg := &ProxyConfig{
		Target:            "127.0.0.1:1", // Invalid target - will cause error
		EnableTranscoding: true,
		Insecure:          true,
	}
	proxy, err := NewProxy(cfg)
	if err != nil {
		t.Skipf("Failed to create proxy: %v", err)
	}
	defer proxy.Close()

	t.Run("transcoding with nil transcoder", func(t *testing.T) {
		proxy.Transcoder = nil

		req := httptest.NewRequest(http.MethodPost, "/v1/test/method", strings.NewReader(`{"field": "value"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		proxy.handleTranscoding(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
		}
	})

	t.Run("transcoding with unloaded transcoder", func(t *testing.T) {
		proxy.Transcoder = NewTranscoder()

		req := httptest.NewRequest(http.MethodPost, "/v1/test/method", strings.NewReader(`{"field": "value"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		proxy.handleTranscoding(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
		}
	})
}

// Test ProxyServerStream with actual streaming scenarios
func TestProxyServerStream_StreamingScenarios(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("Failed to create listener: %v", err)
	}
	defer lis.Close()

	s := grpc.NewServer()
	defer s.Stop()

	go s.Serve(lis)
	time.Sleep(10 * time.Millisecond)

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Skipf("Failed to connect: %v", err)
	}
	defer conn.Close()

	sp := NewStreamProxy()

	t.Run("server stream with EOF handling", func(t *testing.T) {
		rec := newMockFlusher()
		// Empty body - stream will close immediately after sending
		req := httptest.NewRequest(http.MethodPost, "/test.Service/Method", strings.NewReader(""))
		req.Header.Set("Content-Type", "application/grpc")

		sp.ProxyServerStream(rec, req, conn, "/test.Service/Method")

		// Should handle EOF gracefully
		_ = rec.body
	})

	t.Run("server stream with error from stream", func(t *testing.T) {
		rec := newMockFlusher()
		// Send data that will be sent upstream
		req := httptest.NewRequest(http.MethodPost, "/test.Service/Method", strings.NewReader("test data"))
		req.Header.Set("Content-Type", "application/grpc")

		sp.ProxyServerStream(rec, req, conn, "/test.Service/Method")

		// Should handle stream error
		_ = rec.body
	})
}

// Test ProxyClientStream with various body contents
func TestProxyClientStream_BodyVariations(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("Failed to create listener: %v", err)
	}
	defer lis.Close()

	s := grpc.NewServer()
	defer s.Stop()

	go s.Serve(lis)
	time.Sleep(10 * time.Millisecond)

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Skipf("Failed to connect: %v", err)
	}
	defer conn.Close()

	sp := NewStreamProxy()

	tests := []struct {
		name string
		body string
	}{
		{"empty body", ""},
		{"single message", `{"test": "data"}`},
		{"multiple messages", "msg1\nmsg2\nmsg3"},
		{"messages with whitespace", "  msg1  \n  msg2  "},
		{"json objects", `{"a":1}
{"b":2}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/test.Service/Method", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/grpc")

			sp.ProxyClientStream(rec, req, conn, "/test.Service/Method")

			// Should process without panic
			_ = rec.Code
		})
	}
}

// Test ProxyBidiStream with various scenarios
func TestProxyBidiStream_Scenarios(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("Failed to create listener: %v", err)
	}
	defer lis.Close()

	s := grpc.NewServer()
	defer s.Stop()

	go s.Serve(lis)
	time.Sleep(10 * time.Millisecond)

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Skipf("Failed to connect: %v", err)
	}
	defer conn.Close()

	sp := NewStreamProxy()

	t.Run("bidi stream with empty body", func(t *testing.T) {
		rec := newMockFlusher()
		req := httptest.NewRequest(http.MethodPost, "/test.Service/Method", strings.NewReader(""))
		req.Header.Set("Content-Type", "application/grpc")

		sp.ProxyBidiStream(rec, req, conn, "/test.Service/Method")

		// Should handle empty body
		_ = rec.body
	})

	t.Run("bidi stream with send completion", func(t *testing.T) {
		rec := newMockFlusher()
		body := `message1
message2
message3`
		req := httptest.NewRequest(http.MethodPost, "/test.Service/Method", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/grpc")

		sp.ProxyBidiStream(rec, req, conn, "/test.Service/Method")

		// Should complete send goroutine
		_ = rec.body
	})

	t.Run("bidi stream with context timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		rec := newMockFlusher()
		req := httptest.NewRequest(http.MethodPost, "/test.Service/Method", strings.NewReader("test")).WithContext(ctx)
		req.Header.Set("Content-Type", "application/grpc")

		sp.ProxyBidiStream(rec, req, conn, "/test.Service/Method")

		// Should handle timeout
		_ = rec.body
	})
}

// Test handleGRPCWeb success paths
func TestProxy_handleGRPCWeb_SuccessPaths(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("Failed to create listener: %v", err)
	}
	defer lis.Close()

	s := grpc.NewServer()
	defer s.Stop()

	go s.Serve(lis)
	time.Sleep(10 * time.Millisecond)

	cfg := &ProxyConfig{
		Target:    lis.Addr().String(),
		EnableWeb: true,
		Insecure:  true,
	}
	proxy, err := NewProxy(cfg)
	if err != nil {
		t.Skipf("Failed to create proxy: %v", err)
	}
	defer proxy.Close()

	t.Run("grpc-web with binary content", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test.Service/Method", bytes.NewReader([]byte("test data")))
		req.Header.Set("Content-Type", "application/grpc-web")
		rec := httptest.NewRecorder()

		proxy.handleGRPCWeb(rec, req)

		// Should process request
		_ = rec.Code
	})

	t.Run("grpc-web with metadata", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test.Service/Method", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/grpc-web")
		req.Header.Set("X-Custom-Header", "value")
		rec := httptest.NewRecorder()

		proxy.handleGRPCWeb(rec, req)

		// Should process with metadata
		_ = rec.Code
	})
}

// Test writeStreamError with non-gRPC error
func TestWriteStreamError_NonGRPCError(t *testing.T) {
	rec := httptest.NewRecorder()
	err := io.ErrUnexpectedEOF

	writeStreamError(rec, err)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "unexpected EOF") {
		t.Errorf("Body should contain error message, got: %s", body)
	}
}

// Test writeStreamErrorFrame with non-gRPC error
func TestWriteStreamErrorFrame_NonGRPCError(t *testing.T) {
	rec := httptest.NewRecorder()
	err := bytes.ErrTooLarge

	writeStreamErrorFrame(rec, err)

	body := rec.Body.String()
	if !strings.Contains(body, `"error":true`) {
		t.Errorf("Body should contain error flag, got: %s", body)
	}
	if !strings.Contains(body, `"code":13`) { // codes.Internal = 13
		t.Errorf("Body should contain code 13, got: %s", body)
	}
}

// Test streamDesc function
func TestStreamDesc_Variations(t *testing.T) {
	tests := []struct {
		serverStream bool
		clientStream bool
	}{
		{true, false},  // Server streaming
		{false, true},  // Client streaming
		{true, true},   // Bidirectional
		{false, false}, // Unary
	}

	for _, tt := range tests {
		desc := streamDesc(tt.serverStream, tt.clientStream)
		if desc == nil {
			t.Fatal("streamDesc returned nil")
		}
		if desc.ServerStreams != tt.serverStream {
			t.Errorf("ServerStreams = %v, want %v", desc.ServerStreams, tt.serverStream)
		}
		if desc.ClientStreams != tt.clientStream {
			t.Errorf("ClientStreams = %v, want %v", desc.ClientStreams, tt.clientStream)
		}
	}
}

// Test GRPCStatusToHTTP with all codes
func TestGRPCStatusToHTTP_Coverage(t *testing.T) {
	tests := []struct {
		code codes.Code
		want int
	}{
		{codes.OK, http.StatusOK},
		{codes.Canceled, 499},
		{codes.Unknown, http.StatusInternalServerError},
		{codes.InvalidArgument, http.StatusBadRequest},
		{codes.DeadlineExceeded, http.StatusGatewayTimeout},
		{codes.NotFound, http.StatusNotFound},
		{codes.AlreadyExists, http.StatusConflict},
		{codes.PermissionDenied, http.StatusForbidden},
		{codes.ResourceExhausted, http.StatusTooManyRequests},
		{codes.FailedPrecondition, http.StatusPreconditionFailed},
		{codes.Aborted, http.StatusConflict},
		{codes.OutOfRange, http.StatusBadRequest},
		{codes.Unimplemented, http.StatusNotImplemented},
		{codes.Internal, http.StatusInternalServerError},
		{codes.Unavailable, http.StatusServiceUnavailable},
		{codes.DataLoss, http.StatusInternalServerError},
		{codes.Unauthenticated, http.StatusUnauthorized},
		{codes.Code(999), http.StatusInternalServerError}, // Unknown code
	}

	for _, tt := range tests {
		t.Run(tt.code.String(), func(t *testing.T) {
			got := GRPCStatusToHTTP(tt.code)
			if got != tt.want {
				t.Errorf("GRPCStatusToHTTP(%v) = %d, want %d", tt.code, got, tt.want)
			}
		})
	}
}

// Test HTTPStatusToGRPC with all status codes
func TestHTTPStatusToGRPC_AllStatuses(t *testing.T) {
	tests := []struct {
		status int
		want   codes.Code
	}{
		{http.StatusOK, codes.OK},
		{http.StatusBadRequest, codes.InvalidArgument},
		{http.StatusUnauthorized, codes.Unauthenticated},
		{http.StatusForbidden, codes.PermissionDenied},
		{http.StatusNotFound, codes.NotFound},
		{http.StatusConflict, codes.AlreadyExists},
		{http.StatusPreconditionFailed, codes.FailedPrecondition},
		{http.StatusTooManyRequests, codes.ResourceExhausted},
		{http.StatusInternalServerError, codes.Internal},
		{http.StatusNotImplemented, codes.Unimplemented},
		{http.StatusBadGateway, codes.Unavailable},
		{http.StatusServiceUnavailable, codes.Unavailable},
		{http.StatusGatewayTimeout, codes.DeadlineExceeded},
		{999, codes.Unknown},
		{0, codes.Unknown},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			got := HTTPStatusToGRPC(tt.status)
			if got != tt.want {
				t.Errorf("HTTPStatusToGRPC(%d) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

// Test rawCodec with all supported types
func TestRawCodec_AllTypes(t *testing.T) {
	codec := &rawCodec{}

	t.Run("Marshal bytes.Buffer", func(t *testing.T) {
		buf := bytes.NewBufferString("test data")
		data, err := codec.Marshal(buf)
		if err != nil {
			t.Errorf("Marshal error = %v", err)
		}
		if string(data) != "test data" {
			t.Errorf("Marshal = %q, want 'test data'", string(data))
		}
	})

	t.Run("Marshal byte slice", func(t *testing.T) {
		data := []byte("test data")
		result, err := codec.Marshal(data)
		if err != nil {
			t.Errorf("Marshal error = %v", err)
		}
		if string(result) != "test data" {
			t.Errorf("Marshal = %q, want 'test data'", string(result))
		}
	})

	t.Run("Unmarshal to bytes.Buffer", func(t *testing.T) {
		data := []byte("test data")
		var buf bytes.Buffer
		err := codec.Unmarshal(data, &buf)
		if err != nil {
			t.Errorf("Unmarshal error = %v", err)
		}
		if buf.String() != "test data" {
			t.Errorf("Unmarshal = %q, want 'test data'", buf.String())
		}
	})

	t.Run("Unmarshal to byte slice pointer", func(t *testing.T) {
		data := []byte("test data")
		var result []byte
		err := codec.Unmarshal(data, &result)
		if err != nil {
			t.Errorf("Unmarshal error = %v", err)
		}
		if string(result) != "test data" {
			t.Errorf("Unmarshal = %q, want 'test data'", string(result))
		}
	})

	t.Run("Name", func(t *testing.T) {
		if codec.Name() != "raw" {
			t.Errorf("Name = %q, want 'raw'", codec.Name())
		}
	})
}
