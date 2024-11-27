package server_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/tejusbharadwaj/edgecom/internal/database/mocks"
	server "github.com/tejusbharadwaj/edgecom/internal/grpc"
	"github.com/tejusbharadwaj/edgecom/internal/models"
	pb "github.com/tejusbharadwaj/edgecom/proto"
)

func TestQueryTimeSeries(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockTimeSeriesRepository(ctrl)

	svc := server.NewTimeSeriesService(mockRepo)

	tests := []struct {
		name          string
		request       *pb.TimeSeriesRequest
		setupMock     func()
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
			setupMock: func() {
				mockRepo.EXPECT().
					Query(gomock.Any(), gomock.Any(), gomock.Any(), "1h", "AVG").
					Return([]models.TimeSeriesData{
						{Time: time.Now(), Value: 100.0},
						{Time: time.Now().Add(time.Hour), Value: 200.0},
					}, nil)
			},
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
			setupMock:     func() {},
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
			setupMock:     func() {},
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
			setupMock:     func() {},
			expectedCode:  codes.InvalidArgument,
			expectedError: "start time must be before end time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			resp, err := svc.QueryTimeSeries(context.Background(), tt.request)

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
				assert.NotEmpty(t, resp.Data)
			}
		})
	}
}

func TestSetupServer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockTimeSeriesRepository(ctrl)

	config := server.ServerConfig{
		CacheSize:      1000,
		RateLimit:      5.0,
		RateLimitBurst: 10,
	}

	srv, err := server.SetupServer(mockRepo, config)
	require.NoError(t, err)
	require.NotNil(t, srv)

	// Test with invalid config
	invalidConfig := server.ServerConfig{
		CacheSize: -1,
	}
	srv, err = server.SetupServer(mockRepo, invalidConfig)
	require.Error(t, err)
	require.Nil(t, srv)
}

func TestValidateRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockTimeSeriesRepository(ctrl)
	svc := server.NewTimeSeriesService(mockRepo)

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
			req := &pb.TimeSeriesRequest{
				Start:       timestamppb.New(tt.start),
				End:         timestamppb.New(tt.end),
				Window:      tt.window,
				Aggregation: tt.aggregation,
			}

			// Set up mock expectations BEFORE calling the method
			if !tt.wantErr {
				mockRepo.EXPECT().
					Query(
						gomock.Any(),
						gomock.Any(), // Use matchers for time values
						gomock.Any(),
						tt.window,
						tt.aggregation,
					).
					Return([]models.TimeSeriesData{
						{Time: tt.start, Value: 100.0},
					}, nil)
			}

			// Call the method after setting up expectations
			resp, err := svc.QueryTimeSeries(context.Background(), req)

			if tt.wantErr {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, codes.InvalidArgument, st.Code())
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, resp)
				assert.NotEmpty(t, resp.Data)
			}
		})
	}
}
