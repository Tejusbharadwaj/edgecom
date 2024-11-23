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

	pb "github.com/tejusbharadwaj/edgecom/proto"
)

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

// gRPC Server Configuration
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
