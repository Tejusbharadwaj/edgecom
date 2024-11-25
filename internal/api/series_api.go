// Package api provides functionality for interacting with the EdgeCom Energy API.
//
// The package implements:
//   - Robust HTTP client with timeouts and context support
//   - Automatic data conversion and storage
//   - Historical data bootstrapping
//   - Structured logging
//   - Error handling with custom error types
//
// Example:
//
//	fetcher := api.NewSeriesFetcher(
//	    "https://api.example.com/timeseries",
//	    dbService,
//	    logger,
//	)
//
//	if err := fetcher.FetchData(ctx, start, end); err != nil {
//	    log.Printf("Failed to fetch data: %v", err)
//	    return err
//	}
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tejusbharadwaj/edgecom/internal/database"
	"github.com/tejusbharadwaj/edgecom/internal/models"
)

// Error types for API-related errors
var (
	// ErrAPIRequest is returned when there's an error making an API request
	ErrAPIRequest = errors.New("error making API request")
	// ErrAPIStatus is returned when the API returns a non-200 status code
	ErrAPIStatus = errors.New("error status from API")
)

// SeriesFetcher is a struct that fetches data from the EdgeCom Energy API and stores it in a database.
type SeriesFetcher struct {
	apiURL    string
	dbService database.TimeSeriesRepository
	logger    *logrus.Logger
}

// NewSeriesFetcher creates a new SeriesFetcher instance.
// Parameters:
//   - apiURL: The base URL for the EdgeCom API
//   - dbService: Repository for storing time series data
//   - logger: Structured logger for operation tracking
//
// Returns:
//   - A configured SeriesFetcher instance ready for use
func NewSeriesFetcher(apiURL string, dbService database.TimeSeriesRepository, logger *logrus.Logger) *SeriesFetcher {
	return &SeriesFetcher{
		apiURL:    apiURL,
		dbService: dbService,
		logger:    logger,
	}
}

// FetchData fetches data from the EdgeCom Energy API for a given time range and stores it in the database.
// The method:
//  1. Constructs the API request with proper formatting
//  2. Executes the request with timeout
//  3. Processes the response
//  4. Stores the data in the database
func (f *SeriesFetcher) FetchData(ctx context.Context, start, end time.Time) error {
	url := fmt.Sprintf("%s?start=%s&end=%s",
		f.apiURL,
		start.Format("2006-01-02T15:04:05"),
		end.Format("2006-01-02T15:04:05"))

	f.logger.WithFields(logrus.Fields{
		"url":   url,
		"start": start,
		"end":   end,
	}).Debug("Fetching data from API")

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAPIRequest, err)
	}

	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", "EdgeCom-Client/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAPIRequest, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		f.logger.WithFields(logrus.Fields{
			"status": resp.StatusCode,
			"body":   string(body),
		}).Error("API request failed")
		return fmt.Errorf("%w: got %d", ErrAPIStatus, resp.StatusCode)
	}

	var apiResp models.APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	if len(apiResp.Result) == 0 {
		f.logger.Debug("No data points received from API")
		return nil
	}

	dataPoints := make([]models.TimeSeriesData, len(apiResp.Result))
	for i, data := range apiResp.Result {
		dataPoints[i] = models.TimeSeriesData{
			Time:  time.Unix(data.Time, 0),
			Value: data.Value,
		}
	}

	if err := f.dbService.BatchInsertTimeSeriesData(ctx, dataPoints); err != nil {
		return fmt.Errorf("failed to insert data points: %v", err)
	}

	f.logger.WithField("count", len(dataPoints)).Debug("Successfully inserted data points")
	return nil
}

// BootstrapHistoricalData initializes the database with historical data.
// It attempts to fetch the last 2 years of data, with a fallback to
// the last 24 hours if the full historical fetch fails.
//
// The method implements a graceful degradation strategy:
//  1. Attempts to fetch 2 years of historical data
//  2. On failure, falls back to last 24 hours
//  3. Logs all operations and failures
func (f *SeriesFetcher) BootstrapHistoricalData(ctx context.Context) error {
	endTime := time.Now()
	startTime := endTime.AddDate(-2, 0, 0)

	f.logger.WithFields(logrus.Fields{
		"startTime": startTime,
		"endTime":   endTime,
	}).Info("Starting historical data bootstrap")

	if err := f.FetchData(ctx, startTime, endTime); err != nil {
		f.logger.WithError(err).Error("Failed to fetch historical data")

		// If historical data fetch fails, try to get at least the last 24 hours
		recentStart := endTime.Add(-24 * time.Hour)
		f.logger.Info("Attempting to fetch last 24 hours of data")

		if err := f.FetchData(ctx, recentStart, endTime); err != nil {
			return fmt.Errorf("failed to fetch recent data: %v", err)
		}
	}

	f.logger.Info("Historical data bootstrap completed")
	return nil
}
