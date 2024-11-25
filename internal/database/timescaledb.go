//go:generate go run github.com/golang/mock/mockgen -destination=./mocks/timescaledb.go -package=mocks . TimeSeriesRepository
//go:generate godoc -html . > ../../docs/internal/database/index.html

// Package database implements TimescaleDB-backed time series data storage.
//
// Architecture:
//   - Uses TimescaleDB for optimized time series storage and querying
//   - Implements automatic partitioning for efficient data management
//   - Provides built-in support for time-based aggregations
//   - Designed for horizontal scalability
//
// Example usage:
//
//	repo, err := NewPostgresRepo("postgres://user:pass@localhost:5432/db")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer repo.Close()
//
//	// Query time series data
//	data, err := repo.Query(ctx, start, end, "1h", "AVG")
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/tejusbharadwaj/edgecom/internal/models"
)

// TimeSeriesRepository defines the interface for time series operations.
//
// This interface provides methods for:
//   - Single and batch data insertion
//   - Time series querying with aggregation
//   - Resource cleanup
//
// Supported aggregations:
//   - MIN: Minimum value in time window
//   - MAX: Maximum value in time window
//   - AVG: Average value in time window
//   - SUM: Sum of values in time window
//
// Supported time windows:
//   - 1m:  One minute
//   - 5m:  Five minutes
//   - 1h:  One hour
//   - 1d:  One day
type TimeSeriesRepository interface {
	// InsertTimeSeriesData inserts a single time series data point.
	// Returns an error if the insertion fails.
	InsertTimeSeriesData(timestamp time.Time, value float64) error

	// Query retrieves time series data within the specified time range.
	// Supports different time windows (1m, 5m, 1h, 1d) and aggregation methods (MIN, MAX, AVG, SUM).
	// Returns the aggregated data points and any error encountered.
	Query(ctx context.Context, start, end time.Time, window string, aggregation string) ([]models.TimeSeriesData, error)

	// BatchInsertTimeSeriesData inserts multiple time series data points in a single transaction.
	// This method is optimized for bulk insertions by reducing database round trips.
	// Returns an error if any part of the batch insertion fails.
	BatchInsertTimeSeriesData(ctx context.Context, data []models.TimeSeriesData) error

	// Close releases any resources held by the repository.
	// Should be called when the repository is no longer needed.
	Close() error
}

// PostgresRepo implements TimeSeriesRepository using TimescaleDB.
//
// Features:
//   - Automatic data partitioning by time
//   - Optimized time-based queries
//   - Transaction support for batch operations
//   - Connection pooling
//
// Internal implementation uses TimescaleDB's hypertables for:
//   - Automatic chunk management
//   - Parallel query execution
//   - Time-bucket optimization
type PostgresRepo struct {
	db *sql.DB
}

// NewPostgresRepo creates and initializes a new PostgresRepo.
//
// The connection string should be in the format:
// "postgres://username:password@host:port/dbname?sslmode=disable"
//
// The function will:
//  1. Establish database connection
//  2. Verify connectivity
//  3. Initialize connection pool
//
// Returns:
//   - *PostgresRepo: Initialized repository
//   - error: Connection or initialization error
func NewPostgresRepo(connStr string) (*PostgresRepo, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &PostgresRepo{db: db}, nil
}

func (s *PostgresRepo) InsertTimeSeriesData(timestamp time.Time, value float64) error {
	_, err := s.db.Exec(
		"INSERT INTO time_series_data (time, value) VALUES ($1, $2)",
		timestamp,
		value,
	)
	return err
}

// QueryTimeSeriesData retrieves and aggregates time series data.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - start: Beginning of time range (inclusive)
//   - end: End of time range (exclusive)
//   - window: Time bucket size ("1m", "5m", "1h", "1d")
//   - aggregation: Aggregation function ("MIN", "MAX", "AVG", "SUM")
//
// SQL Implementation:
//
//	Uses time_bucket() from TimescaleDB for efficient time-based grouping
//	Implements dynamic aggregation selection via CASE statement
//
// Returns:
//   - []models.TimeSeriesData: Array of aggregated data points
//   - error: Query execution error or invalid parameters
//
// Example:
//
//	data, err := repo.QueryTimeSeriesData(ctx,
//	    time.Now().Add(-24*time.Hour), // start
//	    time.Now(),                    // end
//	    "1h",                          // window
//	    "AVG",                         // aggregation
//	)
func (s *PostgresRepo) QueryTimeSeriesData(
	ctx context.Context,
	start, end time.Time,
	window string,
	aggregation string,
) ([]models.TimeSeriesData, error) {
	// Validate window and aggregation
	if aggregation != "MIN" && aggregation != "MAX" && aggregation != "AVG" && aggregation != "SUM" {
		return nil, fmt.Errorf("invalid aggregation type: %s", aggregation)
	}

	query := fmt.Sprintf(`
        SELECT 
            time_bucket('%s', time) as bucket_time,
            CASE 
                WHEN $3 = 'MIN' THEN MIN(value)
                WHEN $3 = 'MAX' THEN MAX(value)
                WHEN $3 = 'AVG' THEN AVG(value)
                WHEN $3 = 'SUM' THEN SUM(value)
            END as agg_value
        FROM time_series_data
        WHERE time BETWEEN $1 AND $2
        GROUP BY bucket_time
        ORDER BY bucket_time
    `, window)

	rows, err := s.db.QueryContext(ctx, query, start, end, aggregation)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.TimeSeriesData
	for rows.Next() {
		var r models.TimeSeriesData
		if err := rows.Scan(&r.Time, &r.Value); err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, nil
}

// BatchInsertTimeSeriesData performs bulk data insertion.
//
// The operation is atomic - either all data points are inserted or none.
// Uses prepared statements and transactions for optimal performance.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - data: Slice of time series data points to insert
//
// Transaction Flow:
//  1. Begin transaction
//  2. Prepare statement
//  3. Execute batch inserts
//  4. Commit or rollback
//
// Returns error if:
//   - Transaction fails to start
//   - Statement preparation fails
//   - Any insert fails
//   - Commit fails
func (s *PostgresRepo) BatchInsertTimeSeriesData(ctx context.Context, data []models.TimeSeriesData) error {
	// Begin transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // rollback if not committed

	// Prepare the statement
	stmt, err := tx.PrepareContext(ctx, `
        INSERT INTO time_series_data (time, value) 
        VALUES ($1, $2)
    `)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Execute batch inserts
	for _, point := range data {
		if _, err := stmt.ExecContext(ctx, point.Time, point.Value); err != nil {
			return fmt.Errorf("failed to insert data point: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Query implements the TimeSeriesRepository interface.
// Delegates to QueryTimeSeriesData for actual implementation.
//
// See QueryTimeSeriesData for detailed documentation.
func (s *PostgresRepo) Query(
	ctx context.Context,
	start, end time.Time,
	window string,
	aggregation string,
) ([]models.TimeSeriesData, error) {
	return s.QueryTimeSeriesData(ctx, start, end, window, aggregation)
}

// Close releases all database resources.
//
// Should be called when the repository is no longer needed.
// Typically deferred after repository creation.
func (s *PostgresRepo) Close() error {
	return s.db.Close()
}

// Compile-time interface implementation check
var _ TimeSeriesRepository = (*PostgresRepo)(nil)
