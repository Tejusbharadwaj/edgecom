# EdgeCom Energy Time Series Service

[![Go Reference](https://pkg.go.dev/badge/github.com/tejusbharadwaj/edgecom.svg)](https://pkg.go.dev/github.com/tejusbharadwaj/edgecom@v0.1.7)
[![Go Report Card](https://goreportcard.com/badge/github.com/tejusbharadwaj/edgecom)](https://goreportcard.com/report/github.com/tejusbharadwaj/edgecom)
[![Release](https://img.shields.io/github/v/release/tejusbharadwaj/edgecom)](https://github.com/tejusbharadwaj/edgecom/releases)
[![License](https://img.shields.io/github/license/tejusbharadwaj/edgecom)](LICENSE)

A gRPC service for fetching, storing, and querying time series data from EdgeCom Energy's API.

## Features

- Historical data bootstrapping (up to 2 years)
- Time series data aggregation (MIN, MAX, AVG, SUM)
- Configurable time windows (1m, 5m, 1h, 1d)
- gRPC API with reflection support
- TimescaleDB integration for efficient time series storage
- Prometheus metrics integration
- Structured logging with logrus
- Request caching and rate limiting

## Prerequisites

- Go 1.22.0 or later
- Docker and Docker Compose v2
- TimescaleDB 2.x
- Access to EdgeCom Energy API
- grpcurl (for testing)

## Installation

1. Clone the repository:

```bash
git clone https://github.com/tejusbharadwaj/edgecom.git
cd edgecom
```

2. Copy the example config and modify as needed:

```bash
cp config.example.yaml config.yaml
```

3. Run all tests (unit tests and integration tests):

```bash
docker compose --profile test up --build
```

4. Run the application:

```bash
docker compose up --build
```

## Service Ports

The service exposes:
- gRPC server on port 50051 (mapped from container port 8080)
- PostgreSQL/TimescaleDB on port 5432

## Configuration

Configuration is managed through `config.yaml`:

```yaml
server:
  port: 8080  # Container port (mapped to 50051 on host)
  host: "0.0.0.0"
  url: "https://api.edgecomenergy.net/core/asset/{asset-id}/series"

database:
  host: "db"
  port: 5432
  name: "edgecom"
  user: "edgecom"
  password: "edgecom"
  ssl_mode: "disable"
  max_connections: 10
  connection_timeout: 5

logging:
  level: "info"
  format: "json"
```

## API Reference

### gRPC Service Definition

```protobuf
service TimeSeriesService {
    rpc QueryTimeSeries(TimeSeriesRequest) returns (TimeSeriesResponse) {}
}

message TimeSeriesRequest {
    google.protobuf.Timestamp start = 1;
    google.protobuf.Timestamp end = 2;
    string window = 3;       // "1m", "5m", "1h", "1d"
    string aggregation = 4;  // "MIN", "MAX", "AVG", "SUM"
}
```

### Testing the API

Using grpcurl:

```bash
# List available services
grpcurl -plaintext localhost:50051 list

# Query time series data
grpcurl -plaintext -d '{
  "start": "2024-11-23T00:00:00Z",
  "end": "2024-11-24T00:00:00Z",
  "window": "1h",
  "aggregation": "AVG"
}' localhost:50051 edgecom.TimeSeriesService/QueryTimeSeries
```

## Development
## Project Structure

```
.
├── cmd/                 # Application entry point
├── internal/
│   ├── api/             # API client for EdgeCom Energy
│   ├── database/        # Database interactions and repository interface
│   ├── grpc/            # gRPC service implementation
│   │   ├── server.go
│   │   └── middlewares/ # gRPC middleware components
│   └── scheduler/       # Background job scheduler
├── proto/               # Protocol buffer definitions
├── migrations/          # Database migrations
├── integration-tests/   # Integration tests
├── k8s/                 # Kubernetes manifests
└── config.yaml          # Configuration file
```

## Architecture

The service follows a clean architecture pattern:
- Repository pattern for data access
- Clear separation of concerns between packages
- Dependency injection for better testability
- Middleware chain for cross-cutting concerns

Key Components:
1. TimeSeriesRepository interface in database package
2. gRPC service implementation in grpc package
3. Background scheduler for data collection
4. Middleware stack for:
   - Rate limiting
   - Caching
   - Metrics
   - Logging


### Running Tests

```bash
# Run unit tests
go test ./...

# Run integration tests
docker compose --profile test up --build
```

### Building Locally

While you can build the application locally, it's recommended to use Docker Compose as it handles all configurations, dependencies, and environment setup automatically.

#### Option 1: Using Docker Compose (Recommended)
```bash
# This will handle all configurations, database setup, and dependencies
docker compose up --build
```

#### Option 2: Manual Build (Advanced)
```bash
# Only use this if you have specific requirements that prevent using Docker Compose
# You'll need to:
# 1. Set up TimescaleDB manually
# 2. Configure environment variables
# 3. Handle dependencies

go build -o edgecom ./cmd/main.go
```

> **Note**: Docker Compose is the preferred method as it ensures consistent environments and handles all necessary configurations. Only use manual building if you have specific requirements that prevent using Docker Compose.

## Monitoring

The service includes:
- Request rate limiting (5 req/s with burst of 10)
- LRU cache for frequent queries (1000 entries)
- Prometheus metrics for:
  - Request counts
  - Request latencies
  - Cache hit/miss ratios

## Error Handling

The service implements graceful degradation:
- Validates all incoming requests
- Implements retry logic for API requests
- Provides detailed error logging
- Graceful shutdown handling

## Deployment Options

### Option 1: Docker Compose (Primary Method)

Docker Compose is the preferred and officially supported deployment method for this service. It was designed and optimized for Docker Compose deployment, making it the most reliable and straightforward option.

```bash
# Start the service
docker compose up --build

# Stop the service
docker compose down
```

For development and testing:
```bash
# Run all tests (unit tests and integration tests)
docker compose --profile test up --build
```

### Option 2: Kubernetes (Alternative)

> **Note**: While Kubernetes deployment is supported, Docker Compose is the primary and recommended method. Use Kubernetes only if it's specifically required for your infrastructure needs.

#### Prerequisites for Kubernetes Deployment
- Kubernetes cluster (local or cloud)
- kubectl configured with your cluster
- Helm (optional, for database deployment)

#### Kubernetes Deployment Steps

1. Create required Kubernetes resources:
```bash
# Using deployment script
./scripts/deploy.sh

# Or manually apply each manifest
kubectl apply -f k8s/config.yaml
kubectl apply -f k8s/database.yaml
kubectl apply -f k8s/database-service.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
```

2. Verify the deployment:
```bash
kubectl get pods
kubectl get services
```

3. To cleanup Kubernetes resources:
```bash
./scripts/cleanup.sh
```

## Deployment Environments

### Local Development (Recommended)
- Use Docker Compose for the simplest and most reliable setup
- Automatic hot-reloading for development
- Integrated test environment
- Matches the primary deployment method
- Minimal configuration required

### Production
#### Using Docker Compose (Recommended)
- Simple, reliable deployment
- Easy configuration management
- Straightforward scaling
- Built-in service discovery
- Automatic container recovery

#### Using Kubernetes (Alternative)
- Available for specific infrastructure requirements
- Scalable deployment with Kubernetes
- Configurable resource limits
- Rolling updates support
- Health checks and auto-healing
- Load balancing

