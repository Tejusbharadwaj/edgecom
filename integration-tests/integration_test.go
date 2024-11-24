//go:build integration
// +build integration

package integration_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tejusbharadwaj/edgecom/internal/api"
	"github.com/tejusbharadwaj/edgecom/internal/database"
	server "github.com/tejusbharadwaj/edgecom/internal/grpc"
	pb "github.com/tejusbharadwaj/edgecom/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const bufSize = 1024 * 1024

var (
	lis      *bufconn.Listener
	logger   *logrus.Logger
	db       *sql.DB
	registry = prometheus.NewRegistry()
)

// Add this model definition at the top of the file
type TimeSeriesPoint struct {
	Time  int64   `json:"time"`
	Value float64 `json:"value"`
}

type APIResponse struct {
	Result []TimeSeriesPoint `json:"result"`
}

func setupTestDB(t *testing.T) database.TimeSeriesRepository {
	// Get database connection details from environment variables
	dbHost := getEnvOrDefault("DB_HOST", "db")
	dbPort := getEnvOrDefault("DB_PORT", "5432")
	dbUser := getEnvOrDefault("DB_USER", "edgecom")
	dbPass := getEnvOrDefault("DB_PASSWORD", "edgecom")
	dbName := getEnvOrDefault("DB_NAME", "edgecom")

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPass, dbName,
	)

	repo, err := database.NewPostgresRepo(connStr)
	require.NoError(t, err)

	// Clean up any existing test data
	db, err := sql.Open("postgres", connStr)
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec("TRUNCATE TABLE time_series_data")
	require.NoError(t, err)

	return repo
}

// Helper function to get environment variables with defaults
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func setupGRPCServer(t *testing.T, repo database.TimeSeriesRepository) (*grpc.Server, func()) {
	registry = prometheus.NewRegistry()
	lis = bufconn.Listen(bufSize)

	srv, err := server.SetupServerWithRegistry(
		&testRepositoryAdapter{repo},
		logger,
		registry,
	)
	require.NoError(t, err)

	go func() {
		if err := srv.Serve(lis); err != nil {
			logger.Errorf("Error serving: %v", err)
		}
	}()

	cleanup := func() {
		srv.Stop()
		lis.Close()
	}

	return srv, cleanup
}

func setupTestClient(ctx context.Context) (pb.TimeSeriesServiceClient, error) {
	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return pb.NewTimeSeriesServiceClient(conn), nil
}

// Add test configuration
type TestConfig struct {
	DBConfig struct {
		Host     string
		Port     int
		User     string
		Password string
		DBName   string
	}
}

// Move setup code into a helper function
func setupTestEnvironment(t *testing.T) (pb.TimeSeriesServiceClient, database.TimeSeriesRepository, func()) {
	// Initialize logger
	logger = logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Setup test database
	repo := setupTestDB(t)

	// Setup gRPC server
	_, cleanup := setupGRPCServer(t, repo)

	// Setup gRPC client
	ctx := context.Background()
	client, err := setupTestClient(ctx)
	require.NoError(t, err)

	// Return cleanup function that handles all teardown
	return client, repo, func() {
		cleanup()
		repo.Close()
	}
}

// Add this helper function to reset between tests
func resetTestEnvironment() {
	registry = prometheus.NewRegistry()
	prometheus.DefaultRegisterer = registry
}

// Update TestTimeSeriesE2E to use the client and fetcher
func TestTimeSeriesE2E(t *testing.T) {
	resetTestEnvironment()
	client, repo, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Setup API mock server for fetching data
	mockAPI := setupMockAPIServer(t)
	defer mockAPI.Close()

	// Create series fetcher
	fetcher := api.NewSeriesFetcher(mockAPI.URL, repo, logger)

	ctx := context.Background()

	// Use FetchData instead of FetchAndStore
	now := time.Now()
	startTime := now.AddDate(0, -1, 0)

	// Fetch the data
	err := fetcher.FetchData(ctx, startTime, now)
	require.NoError(t, err)

	// Query the data we just fetched
	req := &pb.TimeSeriesRequest{
		Start:       timestamppb.New(startTime),
		End:         timestamppb.New(now),
		Window:      "1h",
		Aggregation: "AVG",
	}

	resp, err := client.QueryTimeSeries(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Greater(t, len(resp.Data), 0)
}

// Consider breaking down the large E2E test into smaller, focused test functions
func TestTimeSeriesBasicQueries(t *testing.T) {
	resetTestEnvironment()
	client, repo, cleanup := setupTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()
	startTime := now.AddDate(0, -1, 0)

	// Setup mock API and fetch data first
	mockAPI := setupMockAPIServer(t)
	defer mockAPI.Close()

	// Create series fetcher and populate data
	fetcher := api.NewSeriesFetcher(mockAPI.URL, repo, logger)
	err := fetcher.FetchData(ctx, startTime, now)
	require.NoError(t, err)

	// Query the data
	req := &pb.TimeSeriesRequest{
		Start:       timestamppb.New(startTime),
		End:         timestamppb.New(now),
		Window:      "1h",
		Aggregation: "AVG",
	}

	resp, err := client.QueryTimeSeries(ctx, req)
	require.NoError(t, err)
	assert.Greater(t, len(resp.Data), 0)
}

func TestTimeSeriesErrorCases(t *testing.T) {
	resetTestEnvironment()
	client, _, cleanup := setupTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()
	startTime := now.AddDate(0, -1, 0)

	testCases := []struct {
		name        string
		window      string
		aggregation string
		wantErr     bool
	}{
		{"invalid window", "invalid", "AVG", true},
		{"invalid aggregation", "1h", "INVALID", true},
		{"valid request", "1h", "AVG", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &pb.TimeSeriesRequest{
				Start:       timestamppb.New(startTime),
				End:         timestamppb.New(now),
				Window:      tc.window,
				Aggregation: tc.aggregation,
			}

			_, err := client.QueryTimeSeries(ctx, req)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

type testRepositoryAdapter struct {
	repo database.TimeSeriesRepository
}

func (ra *testRepositoryAdapter) Query(
	ctx context.Context,
	start, end time.Time,
	window string,
	aggregation string,
) ([]server.DataPoint, error) {
	data, err := ra.repo.Query(ctx, start, end, window, aggregation)
	if err != nil {
		return nil, err
	}

	dataPoints := make([]server.DataPoint, len(data))
	for i, d := range data {
		dataPoints[i] = server.DataPoint{
			Time:  d.Time,
			Value: d.Value,
		}
	}
	return dataPoints, nil
}

func setupMockAPIServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse query parameters
		start := r.URL.Query().Get("start")
		end := r.URL.Query().Get("end")

		require.NotEmpty(t, start)
		require.NotEmpty(t, end)

		// Generate mock time series data
		startTime, err := time.Parse("2006-01-02T15:04:05", start)
		require.NoError(t, err)

		endTime, err := time.Parse("2006-01-02T15:04:05", end)
		require.NoError(t, err)

		mockData := generateMockData(startTime, endTime)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockData)
	}))
}

// Update generateMockData to use local types instead of models
func generateMockData(start, end time.Time) APIResponse {
	var result []TimeSeriesPoint
	current := start

	for current.Before(end) {
		result = append(result, TimeSeriesPoint{
			Time:  current.Unix(),
			Value: rand.Float64() * 100, // Random values between 0 and 100
		})
		current = current.Add(5 * time.Minute)
	}

	return APIResponse{
		Result: result,
	}
}

func TestMiddlewareIntegration(t *testing.T) {
	resetTestEnvironment()
	client, repo, cleanup := setupTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()
	startTime := now.AddDate(0, -1, 0)

	// Setup test data
	mockAPI := setupMockAPIServer(t)
	defer mockAPI.Close()
	fetcher := api.NewSeriesFetcher(mockAPI.URL, repo, logger)
	err := fetcher.FetchData(ctx, startTime, now)
	require.NoError(t, err)

	req := &pb.TimeSeriesRequest{
		Start:       timestamppb.New(startTime),
		End:         timestamppb.New(now),
		Window:      "1h",
		Aggregation: "AVG",
	}

	// Test Cache Hit
	resp1, err := client.QueryTimeSeries(ctx, req)
	require.NoError(t, err)
	resp2, err := client.QueryTimeSeries(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, resp1, resp2, "Cache should return same response")

	// Test Rate Limiting
	for i := 0; i < 10; i++ {
		_, err := client.QueryTimeSeries(ctx, req)
		if err != nil {
			assert.Contains(t, err.Error(), "rate limit exceeded")
			break
		}
	}
}

func TestTimeSeriesEdgeCases(t *testing.T) {
	resetTestEnvironment()
	client, _, cleanup := setupTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	testCases := []struct {
		name    string
		req     *pb.TimeSeriesRequest
		wantErr string
	}{
		{
			name: "start time after end time",
			req: &pb.TimeSeriesRequest{
				Start:       timestamppb.New(now),
				End:         timestamppb.New(now.Add(-time.Hour)),
				Window:      "1h",
				Aggregation: "AVG",
			},
			wantErr: "start time must be before end time",
		},
		{
			name: "time range too large",
			req: &pb.TimeSeriesRequest{
				Start:       timestamppb.New(now.AddDate(-5, 0, 0)),
				End:         timestamppb.New(now),
				Window:      "1h",
				Aggregation: "AVG",
			},
			wantErr: "time range exceeds maximum allowed",
		},
		{
			name:    "empty request",
			req:     &pb.TimeSeriesRequest{},
			wantErr: "missing timestamp",
		},
		{
			name: "missing timestamps",
			req: &pb.TimeSeriesRequest{
				Window:      "1h",
				Aggregation: "AVG",
			},
			wantErr: "missing timestamp",
		},
		{
			name: "invalid aggregation with valid window",
			req: &pb.TimeSeriesRequest{
				Start:       timestamppb.New(now.Add(-time.Hour)),
				End:         timestamppb.New(now),
				Window:      "1h",
				Aggregation: "",
			},
			wantErr: "invalid aggregation",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := client.QueryTimeSeries(ctx, tc.req)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr,
					"Expected error containing %q, got %q", tc.wantErr, err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
