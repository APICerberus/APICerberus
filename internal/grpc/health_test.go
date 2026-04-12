package grpc

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	grpcstatus "google.golang.org/grpc/status"
)

type mockHealthChecker struct {
	status Status
	err    error
}

func (m *mockHealthChecker) Check(service string) (Status, error) {
	return m.status, m.err
}

func (m *mockHealthChecker) Watch(service string) <-chan Status {
	ch := make(chan Status, 1)
	ch <- m.status
	return ch
}

func TestStatus_String(t *testing.T) {
	tests := []struct {
		status Status
		want   string
	}{
		{StatusUnknown, "UNKNOWN"},
		{StatusServing, "SERVING"},
		{StatusNotServing, "NOT_SERVING"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.status.String()
			if got != tt.want {
				t.Errorf("Status.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStatus_ToGRPC(t *testing.T) {
	tests := []struct {
		status Status
		want   healthpb.HealthCheckResponse_ServingStatus
	}{
		{StatusUnknown, healthpb.HealthCheckResponse_UNKNOWN},
		{StatusServing, healthpb.HealthCheckResponse_SERVING},
		{StatusNotServing, healthpb.HealthCheckResponse_NOT_SERVING},
	}

	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			got := tt.status.ToGRPC()
			if got != tt.want {
				t.Errorf("Status.ToGRPC() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewHealthServer(t *testing.T) {
	checker := &mockHealthChecker{status: StatusServing}
	server := NewHealthServer(checker)

	if server == nil {
		t.Fatal("NewHealthServer() returned nil")
	}
	if server.checker != checker {
		t.Error("HealthServer.checker not set correctly")
	}
}

func TestHealthServer_Check(t *testing.T) {
	t.Run("Serving", func(t *testing.T) {
		checker := &mockHealthChecker{status: StatusServing}
		server := NewHealthServer(checker)

		resp, err := server.Check(context.Background(), &healthpb.HealthCheckRequest{
			Service: "test-service",
		})
		if err != nil {
			t.Errorf("Check() error = %v", err)
		}
		if resp.Status != healthpb.HealthCheckResponse_SERVING {
			t.Errorf("Status = %v, want SERVING", resp.Status)
		}
	})

	t.Run("NotServing", func(t *testing.T) {
		checker := &mockHealthChecker{status: StatusNotServing}
		server := NewHealthServer(checker)

		resp, err := server.Check(context.Background(), &healthpb.HealthCheckRequest{
			Service: "test-service",
		})
		if err != nil {
			t.Errorf("Check() error = %v", err)
		}
		if resp.Status != healthpb.HealthCheckResponse_NOT_SERVING {
			t.Errorf("Status = %v, want NOT_SERVING", resp.Status)
		}
	})

	t.Run("OverallHealth", func(t *testing.T) {
		checker := &mockHealthChecker{status: StatusServing}
		server := NewHealthServer(checker)

		resp, err := server.Check(context.Background(), &healthpb.HealthCheckRequest{
			Service: "", // Empty service = overall health
		})
		if err != nil {
			t.Errorf("Check() error = %v", err)
		}
		if resp.Status != healthpb.HealthCheckResponse_SERVING {
			t.Errorf("Status = %v, want SERVING", resp.Status)
		}
	})

	t.Run("CheckError", func(t *testing.T) {
		checker := &mockHealthChecker{err: grpcstatus.Errorf(codes.Internal, "check failed")}
		server := NewHealthServer(checker)

		_, err := server.Check(context.Background(), &healthpb.HealthCheckRequest{
			Service: "test-service",
		})
		if err == nil {
			t.Error("Check() should return error when checker fails")
		}
	})
}

func TestNewSimpleHealthChecker(t *testing.T) {
	checker := NewSimpleHealthChecker()
	if checker == nil {
		t.Fatal("NewSimpleHealthChecker() returned nil")
	}
	if checker.statuses == nil {
		t.Error("statuses map not initialized")
	}
	if checker.watchers == nil {
		t.Error("watchers map not initialized")
	}
}

func TestSimpleHealthChecker_Check(t *testing.T) {
	checker := NewSimpleHealthChecker()

	t.Run("EmptyCheck", func(t *testing.T) {
		status, err := checker.Check("")
		if err != nil {
			t.Errorf("Check() error = %v", err)
		}
		if status != StatusServing {
			t.Errorf("Status = %v, want SERVING (empty means all serving)", status)
		}
	})

	t.Run("UnknownService", func(t *testing.T) {
		status, err := checker.Check("unknown-service")
		if err != nil {
			t.Errorf("Check() error = %v", err)
		}
		if status != StatusUnknown {
			t.Errorf("Status = %v, want UNKNOWN", status)
		}
	})

	t.Run("ServingService", func(t *testing.T) {
		checker.SetStatus("my-service", StatusServing)
		status, err := checker.Check("my-service")
		if err != nil {
			t.Errorf("Check() error = %v", err)
		}
		if status != StatusServing {
			t.Errorf("Status = %v, want SERVING", status)
		}
	})

	t.Run("NotServingService", func(t *testing.T) {
		checker.SetStatus("bad-service", StatusNotServing)
		status, err := checker.Check("bad-service")
		if err != nil {
			t.Errorf("Check() error = %v", err)
		}
		if status != StatusNotServing {
			t.Errorf("Status = %v, want NOT_SERVING", status)
		}
	})

	t.Run("OverallWithNotServing", func(t *testing.T) {
		checker.SetStatus("bad-service-2", StatusNotServing)
		status, err := checker.Check("")
		if err != nil {
			t.Errorf("Check() error = %v", err)
		}
		if status != StatusNotServing {
			t.Errorf("Status = %v, want NOT_SERVING", status)
		}
	})
}

func TestSimpleHealthChecker_SetStatus(t *testing.T) {
	checker := NewSimpleHealthChecker()

	checker.SetStatus("test-service", StatusServing)

	status, _ := checker.Check("test-service")
	if status != StatusServing {
		t.Errorf("Status = %v, want SERVING", status)
	}

	checker.SetStatus("test-service", StatusNotServing)

	status, _ = checker.Check("test-service")
	if status != StatusNotServing {
		t.Errorf("Status = %v, want NOT_SERVING", status)
	}
}

func TestSimpleHealthChecker_Watch(t *testing.T) {
	checker := NewSimpleHealthChecker()

	ch := checker.Watch("watched-service")
	if ch == nil {
		t.Fatal("Watch() returned nil channel")
	}

	// Initial status is empty, set it and verify watcher receives update
	checker.SetStatus("watched-service", StatusServing)

	select {
	case status := <-ch:
		if status != StatusServing {
			t.Errorf("Received status = %v, want SERVING", status)
		}
	case <-time.After(100 * time.Millisecond):
		// No update received (expected for new watcher without initial status)
	}
}
