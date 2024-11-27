// Package server implements the gRPC service for time series data querying.
//
// The server provides:
//   - Time series data querying with various aggregations
//   - Request validation and error handling
//   - Middleware support for:
//   - Request rate limiting
//   - Response caching
//   - Metrics collection
//   - Logging
//   - Context management
//   - Prometheus metrics integration
//   - gRPC reflection for debugging
//
// Example Usage:
//
//	config := DefaultServerConfig()
//	repo := database.NewTimeScaleDB(...)
//
//	server, err := SetupServer(repo, config)
//	if err != nil {
//	    log.Fatalf("Failed to setup server: %v", err)
//	}
//
//	lis, err := net.Listen("tcp", ":50051")
//	if err != nil {
//	    log.Fatalf("Failed to listen: %v", err)
//	}
//
//	if err := server.Serve(lis); err != nil {
//	    log.Fatalf("Failed to serve: %v", err)
//	}
//
//go:generate mockgen -source=../../proto/timeseries_grpc.pb.go -destination=mocks/mock_timeseries.go -package=mocks TimeSeriesServiceServer
package server

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/tejusbharadwaj/edgecom/internal/database"
	middleware "github.com/tejusbharadwaj/edgecom/internal/grpc/middlewares"
	pb "github.com/tejusbharadwaj/edgecom/proto"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// ServerConfig holds configuration options for the gRPC server.
// It controls caching, rate limiting, and other server behaviors.
type ServerConfig struct {
	CacheSize      int     // Size of the LRU cache
	RateLimit      float64 // Requests per second
	RateLimitBurst int     // Maximum burst size for rate limiting
}

// DefaultServerConfig returns a ServerConfig with sensible defaults
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		CacheSize:      1000,
		RateLimit:      5.0, // 5 requests per second
		RateLimitBurst: 10,  // Burst of 10 requests
	}
}

// TimeSeriesService implements the gRPC service for querying time series data.
// It handles request validation, data retrieval, and response formatting.
type TimeSeriesService struct {
	pb.UnimplementedTimeSeriesServiceServer
	repository database.TimeSeriesRepository
	validator  *RequestValidator
}

// NewTimeSeriesService creates a new service instance
func NewTimeSeriesService(repo database.TimeSeriesRepository) *TimeSeriesService {
	return &TimeSeriesService{
		repository: repo,
		validator:  NewRequestValidator(),
	}
}

// QueryTimeSeries retrieves time series data based on the provided request parameters.
// It supports various time windows and aggregation methods.
func (s *TimeSeriesService) QueryTimeSeries(
	ctx context.Context,
	req *pb.TimeSeriesRequest,
) (*pb.TimeSeriesResponse, error) {
	// Convert protobuf timestamps
	start := req.Start.AsTime()
	end := req.End.AsTime()

	// Validate request
	if err := s.validator.Validate(
		start, end, req.Window, req.Aggregation,
	); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%s", err.Error())
	}

	// Query data
	dataPoints, err := s.repository.Query(
		ctx, start, end, req.Window, req.Aggregation,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query failed: %v", err)
	}

	// Convert to protobuf response
	var pbResults []*pb.TimeSeriesDataPoint
	for _, dp := range dataPoints {
		pbResults = append(pbResults, &pb.TimeSeriesDataPoint{
			Time:  timestamppb.New(dp.Time),
			Value: dp.Value,
		})
	}

	return &pb.TimeSeriesResponse{
		Data: pbResults,
	}, nil
}

// gRPC Server Configuration without the middleware (for development and debug only)
func ConfigureGRPCServer(
	repo database.TimeSeriesRepository,
	opts ...grpc.ServerOption,
) *grpc.Server {
	// Create gRPC server with optional configurations
	srv := grpc.NewServer(opts...)

	// Register service
	timeSeriesService := NewTimeSeriesService(repo)
	pb.RegisterTimeSeriesServiceServer(srv, timeSeriesService)

	return srv
}

// SetupServer initializes and configures the gRPC server with all middleware
func SetupServer(repo database.TimeSeriesRepository, config ServerConfig) (*grpc.Server, error) {
	// Use the default registry
	return SetupServerWithRegistry(repo, logrus.StandardLogger(), prometheus.DefaultRegisterer)
}

// SetupServerWithRegistry initializes the server with a custom registry
func SetupServerWithRegistry(repo database.TimeSeriesRepository, logger *logrus.Logger, reg prometheus.Registerer) (*grpc.Server, error) {
	// Initialize middleware components
	cache, err := middleware.NewCache(1000)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %v", err)
	}

	rateLimiter := middleware.NewRateLimiter(5.0, 10)

	// Initialize metrics
	requests := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_requests_total",
			Help: "Total number of gRPC requests handled",
		},
		[]string{"method"},
	)

	latency := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)

	// Register metrics
	if err := reg.Register(requests); err != nil {
		return nil, fmt.Errorf("failed to register requests metric: %v", err)
	}
	if err := reg.Register(latency); err != nil {
		return nil, fmt.Errorf("failed to register latency metric: %v", err)
	}

	// Create server with chained interceptors
	server := grpc.NewServer(
		grpc.UnaryInterceptor(
			chainUnaryInterceptors(
				middleware.ContextMiddleware,
				rateLimiter.InterceptorFunc(),
				middleware.LoggingInterceptor,
				middleware.NewMetricsInterceptor(requests, latency),
				cache.InterceptorFunc(),
			),
		),
	)

	// Register the time series service
	timeSeriesService := NewTimeSeriesService(repo)
	pb.RegisterTimeSeriesServiceServer(server, timeSeriesService)

	// Register health service
	healthChecker := NewHealthChecker()
	grpc_health_v1.RegisterHealthServer(server, healthChecker)

	// Set initial status
	healthChecker.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthChecker.SetServingStatus("timeseries.TimeSeriesService", grpc_health_v1.HealthCheckResponse_SERVING)

	// Enable reflection for debugging
	reflection.Register(server)

	return server, nil
}

// chainUnaryInterceptors creates a single interceptor from multiple interceptors
func chainUnaryInterceptors(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		chain := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			interceptor := interceptors[i]
			chainedInterceptor := chain
			chain = func(currentCtx context.Context, currentReq interface{}) (interface{}, error) {
				return interceptor(currentCtx, currentReq, info, chainedInterceptor)
			}
		}
		return chain(ctx, req)
	}
}
