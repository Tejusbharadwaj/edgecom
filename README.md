# EdgeCom Energy Time Series Service

[![Go Reference](https://pkg.go.dev/badge/github.com/tejusbharadwaj/edgecom.svg)](https://pkg.go.dev/github.com/tejusbharadwaj/edgecom)
[![Go Report Card](https://goreportcard.com/badge/github.com/tejusbharadwaj/edgecom)](https://goreportcard.com/report/github.com/tejusbharadwaj/edgecom)
[![Documentation Status](https://godoc.org/github.com/tejusbharadwaj/edgecom?status.svg)](http://godoc.org/github.com/tejusbharadwaj/edgecom)

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

- Go 1.22 or later
- Docker and Docker Compose
- TimescaleDB
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

### Project Structure

```
.
├── cmd/                # Application entry point
├── internal/
│   ├── api/           # API client for EdgeCom Energy
│   ├── database/      # Database interactions
│   ├── grpc/          # gRPC service implementation
│   └── scheduler/     # Background job scheduler
├── proto/             # Protocol buffer definitions
├── migrations/        # Database migrations
├── integration-tests/ # Integration tests
├── config.yaml       # Configuration file
└── docker-compose.yml
```

### Running Tests

```bash
# Run unit tests
go test ./...

# Run integration tests
docker compose --profile test up --build
```

### Building Locally

```bash
go build -o edgecom ./cmd/main.go
```

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

