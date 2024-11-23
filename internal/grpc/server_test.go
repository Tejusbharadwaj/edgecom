package server

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	return args.Get(0).([]DataPoint), args.Error(1)
}

func TestQueryTimeSeries(t *testing.T) {
	// Setup
	mockRepo := new(MockRepository)
	service := NewTimeSeriesService(mockRepo)

	// Prepare test data
	now := time.Now()
	mockData := []DataPoint{
		{Time: now, Value: 100.0},
		{Time: now.Add(time.Hour), Value: 200.0},
	}

	// Mock repository expectation
	mockRepo.On(
		"Query",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return(mockData, nil)

	// Create request
	req := &pb.TimeSeriesRequest{
		Start:       timestamppb.New(now),
		End:         timestamppb.New(now.Add(24 * time.Hour)),
		Window:      "1h",
		Aggregation: "AVG",
	}

	// Execute
	resp, err := service.QueryTimeSeries(context.Background(), req)

	// Assertions
	assert.NoError(t, err)
	assert.Len(t, resp.Data, 2)
	assert.Equal(t, 100.0, resp.Data[0].Value)
}
