// Package edgecom implements a time series data service for EdgeCom Energy.
//
// # Architecture
//
// The service is structured into several key packages:
//   - api: External API client for data fetching
//   - database: TimescaleDB integration for time series storage
//   - grpc: gRPC service implementation
//   - models: Shared data structures
//   - scheduler: Background data fetching and processing
//
// Key Features
//
//   - Historical Data:
//     The service can bootstrap up to 2 years of historical data and
//     maintains it through periodic updates.
//
//   - Time Series Operations:
//     Supports various aggregations (MIN, MAX, AVG, SUM) over
//     different time windows (1m, 5m, 1h, 1d).
//
//   - Performance:
//     Uses TimescaleDB for efficient time series storage and
//     implements caching for frequently accessed data.
//
// Example Usage
//
//	client := grpc.NewTimeSeriesServiceClient(conn)
//	resp, err := client.QueryTimeSeries(ctx, &pb.TimeSeriesRequest{
//	    StartTime: startProto,
//	    EndTime:   endProto,
//	    Window:    "1h",
//	    Aggregation: "AVG",
//	})
//
// For more information about specific packages, see their respective
// documentation.
package edgecom
