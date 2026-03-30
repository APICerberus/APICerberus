package grpc

import (
	"context"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	grpcstatus "google.golang.org/grpc/status"
)

// HealthServer implements the gRPC health checking protocol.
type HealthServer struct {
	healthpb.UnimplementedHealthServer
	checker HealthChecker
}

// HealthChecker is the interface for health status checking.
type HealthChecker interface {
	Check(service string) (Status, error)
	Watch(service string) <-chan Status
}

// Status represents the health status of a service.
type Status int

const (
	// StatusUnknown indicates the status is unknown.
	StatusUnknown Status = iota
	// StatusServing indicates the service is healthy.
	StatusServing
	// StatusNotServing indicates the service is not healthy.
	StatusNotServing
)

// String returns the string representation of the status.
func (s Status) String() string {
	switch s {
	case StatusServing:
		return "SERVING"
	case StatusNotServing:
		return "NOT_SERVING"
	default:
		return "UNKNOWN"
	}
}

// ToGRPC converts the status to gRPC health status.
func (s Status) ToGRPC() healthpb.HealthCheckResponse_ServingStatus {
	switch s {
	case StatusServing:
		return healthpb.HealthCheckResponse_SERVING
	case StatusNotServing:
		return healthpb.HealthCheckResponse_NOT_SERVING
	default:
		return healthpb.HealthCheckResponse_UNKNOWN
	}
}

// NewHealthServer creates a new health server.
func NewHealthServer(checker HealthChecker) *HealthServer {
	return &HealthServer{
		checker: checker,
	}
}

// Check implements the Check method of the health protocol.
func (s *HealthServer) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	service := req.Service
	if service == "" {
		// Overall health check
		service = ""
	}

	status, err := s.checker.Check(service)
	if err != nil {
		return nil, grpcstatus.Errorf(codes.Internal, "health check failed: %v", err)
	}

	return &healthpb.HealthCheckResponse{
		Status: status.ToGRPC(),
	}, nil
}

// Watch implements the Watch method of the health protocol.
func (s *HealthServer) Watch(req *healthpb.HealthCheckRequest, stream healthpb.Health_WatchServer) error {
	service := req.Service

	// Get initial health status
	healthStatus, err := s.checker.Check(service)
	if err != nil {
		return grpcstatus.Errorf(codes.Internal, "health check failed: %v", err)
	}

	// Send initial status
	if err := stream.Send(&healthpb.HealthCheckResponse{
		Status: healthStatus.ToGRPC(),
	}); err != nil {
		return err
	}

	// Watch for changes
	watchCh := s.checker.Watch(service)
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case newStatus := <-watchCh:
			if err := stream.Send(&healthpb.HealthCheckResponse{
				Status: newStatus.ToGRPC(),
			}); err != nil {
				return err
			}
		}
	}
}

// RegisterHealthServer registers the health server with a gRPC server.
func RegisterHealthServer(s *grpc.Server, checker HealthChecker) {
	healthpb.RegisterHealthServer(s, NewHealthServer(checker))
}

// SimpleHealthChecker is a simple in-memory health checker.
type SimpleHealthChecker struct {
	mu       sync.RWMutex
	statuses map[string]Status
	watchers map[string][]chan<- Status
}

// NewSimpleHealthChecker creates a new simple health checker.
func NewSimpleHealthChecker() *SimpleHealthChecker {
	return &SimpleHealthChecker{
		statuses: make(map[string]Status),
		watchers: make(map[string][]chan<- Status),
	}
}

// Check returns the health status of a service.
func (c *SimpleHealthChecker) Check(service string) (Status, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if service == "" {
		// Overall health - check all services
		for _, status := range c.statuses {
			if status != StatusServing {
				return StatusNotServing, nil
			}
		}
		return StatusServing, nil
	}

	status, ok := c.statuses[service]
	if !ok {
		return StatusUnknown, nil
	}
	return status, nil
}

// Watch returns a channel that receives status updates for a service.
func (c *SimpleHealthChecker) Watch(service string) <-chan Status {
	ch := make(chan Status, 1)
	c.mu.Lock()
	c.watchers[service] = append(c.watchers[service], ch)
	c.mu.Unlock()
	return ch
}

// SetStatus sets the health status of a service.
func (c *SimpleHealthChecker) SetStatus(service string, status Status) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.statuses[service] = status

	// Notify watchers
	for _, ch := range c.watchers[service] {
		select {
		case ch <- status:
		default:
		}
	}
}
