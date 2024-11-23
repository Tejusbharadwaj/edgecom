package server

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/tejusbharadwaj/edgecom/proto"
)

// MockRepository for testing
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Query(
	ctx context.Context,
	start, end time.Time,
	window, aggregation string,
) ([]DataPoint, error) {
	args := m.Called(ctx, start, end, window, aggregation)
	if result := args.Get(0); result != nil {
		return result.([]DataPoint), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestQueryTimeSeries(t *testing.T) {
	tests := []struct {
		name          string
		request       *pb.TimeSeriesRequest
		mockData      []DataPoint
		mockError     error
		expectedCode  codes.Code
		expectedError string
	}{
		{
			name: "Success case",
			request: &pb.TimeSeriesRequest{
				Start:       timestamppb.New(time.Now()),
				End:         timestamppb.New(time.Now().Add(24 * time.Hour)),
				Window:      "1h",
				Aggregation: "AVG",
			},
			mockData: []DataPoint{
				{Time: time.Now(), Value: 100.0},
				{Time: time.Now().Add(time.Hour), Value: 200.0},
			},
			mockError:    nil,
			expectedCode: codes.OK,
		},
		{
			name: "Invalid window",
			request: &pb.TimeSeriesRequest{
				Start:       timestamppb.New(time.Now()),
				End:         timestamppb.New(time.Now().Add(24 * time.Hour)),
				Window:      "invalid",
				Aggregation: "AVG",
			},
			mockData:      nil,
			mockError:     nil,
			expectedCode:  codes.InvalidArgument,
			expectedError: "invalid window: invalid",
		},
		{
			name: "Invalid aggregation",
			request: &pb.TimeSeriesRequest{
				Start:       timestamppb.New(time.Now()),
				End:         timestamppb.New(time.Now().Add(24 * time.Hour)),
				Window:      "1h",
				Aggregation: "INVALID",
			},
			mockData:      nil,
			mockError:     nil,
			expectedCode:  codes.InvalidArgument,
			expectedError: "invalid aggregation: INVALID",
		},
		{
			name: "Invalid time range",
			request: &pb.TimeSeriesRequest{
				Start:       timestamppb.New(time.Now().Add(24 * time.Hour)),
				End:         timestamppb.New(time.Now()),
				Window:      "1h",
				Aggregation: "AVG",
			},
			mockData:      nil,
			mockError:     nil,
			expectedCode:  codes.InvalidArgument,
			expectedError: "start time must be before end time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := new(MockRepository)
			service := NewTimeSeriesService(mockRepo)

			if tt.mockData != nil || tt.mockError != nil {
				mockRepo.On(
					"Query",
					mock.Anything,
					tt.request.Start.AsTime(),
					tt.request.End.AsTime(),
					tt.request.Window,
					tt.request.Aggregation,
				).Return(tt.mockData, tt.mockError)
			}

			// Execute
			resp, err := service.QueryTimeSeries(context.Background(), tt.request)

			// Assertions
			if tt.expectedCode != codes.OK {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, tt.expectedCode, st.Code())
				assert.Contains(t, st.Message(), tt.expectedError)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Len(t, resp.Data, len(tt.mockData))
				for i, dp := range tt.mockData {
					assert.Equal(t, dp.Value, resp.Data[i].Value)
					assert.Equal(t, timestamppb.New(dp.Time).AsTime().Unix(), resp.Data[i].Time.AsTime().Unix())
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestSetupServer(t *testing.T) {
	// Test server setup with default config
	repo := new(MockRepository)
	config := DefaultServerConfig()

	server, err := SetupServer(repo, config)
	require.NoError(t, err)
	require.NotNil(t, server)

	// Test with invalid cache size
	invalidConfig := ServerConfig{
		CacheSize: -1, // Invalid cache size
	}
	server, err = SetupServer(repo, invalidConfig)
	require.Error(t, err)
	require.Nil(t, server)
}

func TestRequestValidator(t *testing.T) {
	validator := NewRequestValidator()

	tests := []struct {
		name        string
		start       time.Time
		end         time.Time
		window      string
		aggregation string
		wantErr     bool
	}{
		{
			name:        "Valid input",
			start:       time.Now(),
			end:         time.Now().Add(time.Hour),
			window:      "1h",
			aggregation: "AVG",
			wantErr:     false,
		},
		{
			name:        "Invalid window",
			start:       time.Now(),
			end:         time.Now().Add(time.Hour),
			window:      "2h",
			aggregation: "AVG",
			wantErr:     true,
		},
		{
			name:        "Invalid aggregation",
			start:       time.Now(),
			end:         time.Now().Add(time.Hour),
			window:      "1h",
			aggregation: "MEDIAN",
			wantErr:     true,
		},
		{
			name:        "Invalid time range",
			start:       time.Now().Add(time.Hour),
			end:         time.Now(),
			window:      "1h",
			aggregation: "AVG",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.start, tt.end, tt.window, tt.aggregation)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
