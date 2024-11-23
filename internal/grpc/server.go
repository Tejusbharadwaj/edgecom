package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/prometheus/client_golang/prometheus"
	middleware "github.com/tejusbharadwaj/edgecom/internal/grpc/middlewares"
	pb "github.com/tejusbharadwaj/edgecom/proto"
)

// ServerConfig holds configuration options for the gRPC server
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

// DataRepository defines the interface for data access
type DataRepository interface {
	Query(
		ctx context.Context,
		start, end time.Time,
		window string,
		aggregation string,
	) ([]DataPoint, error)
}

// DataPoint represents a generic time series data point
type DataPoint struct {
	Time  time.Time
	Value float64
}

// TimeSeriesService encapsulates business logic
type TimeSeriesService struct {
	pb.UnimplementedTimeSeriesServiceServer
	repository DataRepository
	validator  *RequestValidator
}

// NewTimeSeriesService creates a new service instance
func NewTimeSeriesService(repo DataRepository) *TimeSeriesService {
	return &TimeSeriesService{
		repository: repo,
		validator:  NewRequestValidator(),
	}
}

// RequestValidator handles input validation
type RequestValidator struct {
	validWindows      map[string]bool
	validAggregations map[string]bool
}

func NewRequestValidator() *RequestValidator {
	return &RequestValidator{
		validWindows: map[string]bool{
			"1m": true,
			"5m": true,
			"1h": true,
			"1d": true,
		},
		validAggregations: map[string]bool{
			"MIN": true,
			"MAX": true,
			"AVG": true,
			"SUM": true,
		},
	}
}

// Validate checks if the request parameters are valid
func (v *RequestValidator) Validate(
	start, end time.Time,
	window, aggregation string,
) error {
	// Validate time range
	if start.After(end) {
		return errors.New("start time must be before end time")
	}

	// Validate window
	if !v.validWindows[window] {
		return fmt.Errorf("invalid window: %s", window)
	}

	// Validate aggregation
	if !v.validAggregations[aggregation] {
		return fmt.Errorf("invalid aggregation: %s", aggregation)
	}

	return nil
}

// QueryTimeSeries implements the gRPC service method
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
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
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
	repo DataRepository,
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
func SetupServer(repo DataRepository, config ServerConfig) (*grpc.Server, error) {
	// Initialize the cache
	if err := middleware.InitializeCache(config.CacheSize); err != nil {
		return nil, err
	}

	// Register Prometheus metrics
	prometheus.MustRegister(middleware.Requests)
	prometheus.MustRegister(middleware.Latency)

	// Create server with chained interceptors
	server := grpc.NewServer(
		grpc.UnaryInterceptor(
			chainUnaryInterceptors(
				middleware.ContextMiddleware,       // Add request ID first
				middleware.RateLimitingInterceptor, // Rate limit early
				middleware.LoggingInterceptor,      // Log all requests (with request ID)
				middleware.MetricsInterceptor,      // Collect metrics
				middleware.CachingInterceptor,      // Cache last to avoid caching errors
			),
		),
	)

	// Register the time series service
	timeSeriesService := NewTimeSeriesService(repo)
	pb.RegisterTimeSeriesServiceServer(server, timeSeriesService)

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
