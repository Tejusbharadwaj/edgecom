package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tejusbharadwaj/edgecom/internal/api"
	"github.com/tejusbharadwaj/edgecom/internal/config"
	"github.com/tejusbharadwaj/edgecom/internal/database"
	server "github.com/tejusbharadwaj/edgecom/internal/grpc"
	"github.com/tejusbharadwaj/edgecom/internal/scheduler"
	"google.golang.org/grpc"
)

// Command edgecom provides a gRPC service for time series data management.
//
// The service supports:
//   - Historical data bootstrapping (up to 2 years)
//   - Time series data aggregation (MIN, MAX, AVG, SUM)
//   - Configurable time windows (1m, 5m, 1h, 1d)
//   - TimescaleDB integration
//   - Prometheus metrics
//
// Usage:
//
//	edgecom [flags]
//
// The flags are:
//
//	-config string
//	      path to config file (default "config.yaml")
//	-port int
//	      gRPC server port (default 50051)
func main() {
	// Parse command line flags
	cfg := parseFlags()

	// Load configuration
	appConfig, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Construct connection string from config
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		appConfig.Database.Host,
		appConfig.Database.Port,
		appConfig.Database.User,
		appConfig.Database.Password,
		appConfig.Database.Name,
		appConfig.Database.SSLMode,
	)

	// Initialize structured logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	logger.WithFields(logrus.Fields{
		"port": appConfig.Server.Port,
	}).Info("Starting server")

	// Create repository using the connection string from config.yaml
	repo, err := createPostgresRepository(connStr)
	if err != nil {
		logger.Fatalf("Failed to create repository: %v", err)
	}

	// Create a context that will be canceled on shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize components
	seriesFetcher := api.NewSeriesFetcher(appConfig.Server.URL, repo, logger)
	scheduler := scheduler.NewScheduler(ctx, seriesFetcher, logger)

	// Create and setup gRPC server
	serverConfig := server.ServerConfig{
		CacheSize:      cfg.CacheSize,
		RateLimit:      cfg.RateLimit,
		RateLimitBurst: cfg.RateLimitBurst,
	}

	srv, err := server.SetupServer(&repositoryAdapter{
		repository: repo,
	}, serverConfig)
	if err != nil {
		logger.Fatalf("Failed to setup server: %v", err)
	}

	// Start listening
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", appConfig.Server.Port))
	if err != nil {
		logger.Fatalf("Failed to listen: %v", err)
	}

	// Start background services
	errChan := make(chan error, 1)

	// Bootstrap historical data in a goroutine
	go func() {
		if err := seriesFetcher.BootstrapHistoricalData(ctx); err != nil {
			errChan <- fmt.Errorf("bootstrap error: %w", err)
		}
	}()

	// Start scheduler in a goroutine
	go func() {
		if err := scheduler.Start(); err != nil {
			errChan <- fmt.Errorf("scheduler error: %w", err)
		}
	}()

	// Handle shutdown gracefully
	go handleShutdown(ctx, srv, logger, repo)

	// Start gRPC server
	logger.WithFields(logrus.Fields{
		"port": appConfig.Server.Port,
	}).Info("Starting gRPC server")

	// Monitor for errors from background services
	go func() {
		if err := srv.Serve(lis); err != nil {
			errChan <- fmt.Errorf("server error: %w", err)
		}
	}()

	// Wait for any error
	if err := <-errChan; err != nil {
		logger.Fatalf("Service error: %v", err)
	}
}

type Config struct {
	Port             int
	CacheSize        int
	RateLimit        float64
	RateLimitBurst   int
	ConnectionString string
}

func parseFlags() *Config {
	cfg := &Config{}

	flag.IntVar(&cfg.Port, "port", 8080, "The gRPC server port")
	flag.IntVar(&cfg.CacheSize, "cache-size", 1000, "Size of the LRU cache")
	flag.Float64Var(&cfg.RateLimit, "rate-limit", 5.0, "Rate limit in requests per second")
	flag.IntVar(&cfg.RateLimitBurst, "rate-limit-burst", 10, "Maximum burst size for rate limiting")
	flag.StringVar(&cfg.ConnectionString, "conn-string", "", "Database connection string")

	flag.Parse()

	return cfg
}

// Handle graceful shutdown
func handleShutdown(ctx context.Context, srv *grpc.Server, logger *logrus.Logger, repo database.TimeSeriesRepository) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case <-ctx.Done():
		logger.Println("Context canceled, initiating shutdown")
	case sig := <-sigChan:
		logger.Printf("Received signal %v, initiating shutdown", sig)
	}

	// Perform graceful shutdown
	logger.Println("Gracefully stopping server...")
	srv.GracefulStop()
	logger.Println("Server stopped")

	// Clean up the repository
	repo.Close()
}

// Add this adapter struct and method
type repositoryAdapter struct {
	repository database.TimeSeriesRepository
}

func (ra *repositoryAdapter) Query(
	ctx context.Context,
	start, end time.Time,
	window string,
	aggregation string,
) ([]server.DataPoint, error) {
	data, err := ra.repository.Query(ctx, start, end, window, aggregation)
	if err != nil {
		return nil, err
	}

	// Convert database.TimeSeriesData to server.DataPoint
	dataPoints := make([]server.DataPoint, len(data))
	for i, d := range data {
		dataPoints[i] = server.DataPoint{
			Time:  d.Time,
			Value: d.Value,
		}
	}
	return dataPoints, nil
}

// Create a Postgres repository
func createPostgresRepository(connectionString string) (database.TimeSeriesRepository, error) {
	repo, err := database.NewPostgresRepo(connectionString)
	if err != nil {
		return nil, err
	}
	return repo, nil
}
