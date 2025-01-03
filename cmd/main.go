// Command edgecom provides a gRPC service for time series data management.
//
// The service supports:
//   - Historical data bootstrapping (up to 2 years)
//   - Time series data aggregation (MIN, MAX, AVG, SUM)
//   - Configurable time windows (1m, 5m, 1h, 1d)
//   - TimescaleDB integration
//   - Prometheus metrics
//   - Rate limiting and caching
//
// Usage:
//
//	edgecom [flags]
//
// The flags are:
//
//	-port int
//	      The gRPC server port (default 8080)
//	-cache-size int
//	      Size of the LRU cache (default 1000)
//	-rate-limit float
//	      Rate limit in requests per second (default 5.0)
//	-rate-limit-burst int
//	      Maximum burst size for rate limiting (default 10)
//	-conn-string string
//	      Database connection string
//
// Configuration:
//
// The service uses config.yaml for additional configuration:
//
//	server:
//	  port: 8080
//	  url: "https://api.example.com/timeseries"
//
//	database:
//	  host: "localhost"
//	  port: 5432
//	  name: "timeseries"
//	  user: "postgres"
//	  password: "secret"
//	  sslmode: "disable"
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

	"github.com/sirupsen/logrus"
	"github.com/tejusbharadwaj/edgecom/internal/api"
	"github.com/tejusbharadwaj/edgecom/internal/config"
	"github.com/tejusbharadwaj/edgecom/internal/database"
	server "github.com/tejusbharadwaj/edgecom/internal/grpc"
	"github.com/tejusbharadwaj/edgecom/internal/scheduler"
	"google.golang.org/grpc"
)

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

	srv, err := server.SetupServer(repo, serverConfig)
	if err != nil {
		logger.Fatalf("Failed to setup server: %v", err)
	}

	// Start listening
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", appConfig.Server.Port))
	if err != nil {
		logger.Fatalf("Failed to listen: %v", err)
	}

	// Start background services
	errChan := make(chan error, 3)
	doneChan := make(chan bool, 1)

	// Bootstrap historical data in a goroutine
	go func() {
		if err := seriesFetcher.BootstrapHistoricalData(ctx); err != nil {
			errChan <- fmt.Errorf("bootstrap error: %w", err)
			return
		}
		doneChan <- true
	}()

	// Start scheduler in a goroutine
	go func() {
		logger.Info("Starting scheduler...")
		if err := scheduler.Start(); err != nil {
			errChan <- fmt.Errorf("scheduler error: %w", err)
		}
	}()

	// Start gRPC server in a goroutine
	go func() {
		logger.WithFields(logrus.Fields{
			"port": appConfig.Server.Port,
		}).Info("Starting gRPC server")

		if err := srv.Serve(lis); err != nil {
			errChan <- fmt.Errorf("server error: %w", err)
			cancel() // Cancel context to trigger shutdown
		}
	}()

	// Handle shutdown gracefully
	go handleShutdown(ctx, srv, scheduler, logger, repo)

	// Wait for bootstrap to complete first
	select {
	case <-doneChan:
		logger.Info("Bootstrap completed, continuing to run scheduler and server")
	case err := <-errChan:
		logger.Fatalf("Service error during bootstrap: %v", err)
	}

	// Keep the main goroutine alive and monitoring for all services
	for {
		select {
		case err := <-errChan:
			logger.WithError(err).Error("Service error occurred")
			// Optionally, you could add logic here to determine if the error is fatal
			// For now, we'll continue running unless it's a context cancellation
		case <-ctx.Done():
			logger.Info("Context cancelled, shutting down")
			return
		}
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
func handleShutdown(ctx context.Context, srv *grpc.Server, scheduler *scheduler.Scheduler, logger *logrus.Logger, repo database.TimeSeriesRepository) {
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

	logger.Println("Stopping scheduler...")
	scheduler.Stop()
	logger.Println("Scheduler stopped")

	repo.Close()
}

// Create a Postgres repository
func createPostgresRepository(connectionString string) (database.TimeSeriesRepository, error) {
	repo, err := database.NewPostgresRepo(connectionString)
	if err != nil {
		return nil, err
	}
	return repo, nil
}
