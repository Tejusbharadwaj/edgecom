package database

//go:generate go run github.com/golang/mock/mockgen -destination=./mocks/timescaledb.go -package=mocks . TimeSeriesRepository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/tejusbharadwaj/edgecom/internal/models"
)

// Using TimescaleDB plugin for Postgres is a better option for time series data base.
//
//	It provides advanced features and optimizations for time-series workloads while retaining full compatibility with standard PostgreSQ
//
// Provides automatic partitionin
// Provides Built-in support for time-based aggregations
// It is designed to be scalable
type PostgresRepo struct {
	db *sql.DB
}

// TimeSeriesRepository defines the interface for time series operations
type TimeSeriesRepository interface {
	InsertTimeSeriesData(timestamp time.Time, value float64) error
	Query(
		ctx context.Context,
		start, end time.Time,
		window string,
		aggregation string,
	) ([]models.TimeSeriesData, error)
	QueryTimeSeriesData(ctx context.Context, start, end time.Time, window string, aggregation string) ([]models.TimeSeriesData, error)
	Close() error
	BatchInsertTimeSeriesData(ctx context.Context, data []models.TimeSeriesData) error
}

// NewPostgresRepo creates a new POstgresRepo
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

func (s *PostgresRepo) Close() error {
	return s.db.Close()
}

// BatchInsertTimeSeriesData inserts a batch of time series data points thus improving performance
// by reducing the number of round trips to the database and  without holding it all in memory.
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

func (s *PostgresRepo) Query(
	ctx context.Context,
	start, end time.Time,
	window string,
	aggregation string,
) ([]models.TimeSeriesData, error) {
	return s.QueryTimeSeriesData(ctx, start, end, window, aggregation)
}

// Ensure PostgresRepo implements TimeSeriesRepository
var _ TimeSeriesRepository = (*PostgresRepo)(nil)
