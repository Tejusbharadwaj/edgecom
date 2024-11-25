package server

import (
	"context"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

// HealthChecker implements the gRPC health checking protocol
type HealthChecker struct {
	grpc_health_v1.UnimplementedHealthServer
	mu     sync.RWMutex
	status map[string]grpc_health_v1.HealthCheckResponse_ServingStatus
}

func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		status: make(map[string]grpc_health_v1.HealthCheckResponse_ServingStatus),
	}
}

func (h *HealthChecker) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if status, ok := h.status[req.Service]; ok {
		return &grpc_health_v1.HealthCheckResponse{
			Status: status,
		}, nil
	}

	// If no status is registered for this service, return NOT_SERVING
	return nil, status.Error(codes.NotFound, "unknown service")
}

func (h *HealthChecker) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	// Implement watching logic if needed
	return status.Error(codes.Unimplemented, "watching is not supported")
}

// SetServingStatus sets the serving status of a service
func (h *HealthChecker) SetServingStatus(service string, status grpc_health_v1.HealthCheckResponse_ServingStatus) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.status[service] = status
}
