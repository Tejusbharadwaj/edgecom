-- Create extension if it doesn't exist
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Create the time series table and configure TimescaleDB
CREATE TABLE IF NOT EXISTS time_series_data (
    time TIMESTAMPTZ NOT NULL,
    value DOUBLE PRECISION NOT NULL
);

-- Create hypertable
SELECT create_hypertable('time_series_data', 'time', 
    chunk_time_interval => INTERVAL '1 day', 
    if_not_exists => TRUE
);

-- Enable compression
ALTER TABLE time_series_data SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = '',
    timescaledb.compress_orderby = 'time'
);

-- Add compression policy
SELECT add_compression_policy('time_series_data', 
    INTERVAL '7 days',
    if_not_exists => TRUE
);

-- Add single index for time-based queries
CREATE INDEX IF NOT EXISTS idx_time_series_data_time ON time_series_data (time DESC);

-- Bootstrap with 2 years of sample data
INSERT INTO time_series_data (time, value)
SELECT
    generate_series(
        NOW() - INTERVAL '2 years',
        NOW(),
        INTERVAL '1 hour'
    ) AS time,
    random() * 100 AS value; -- Replace with your actual data generation logic 