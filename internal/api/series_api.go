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

var (
	ErrAPIRequest = errors.New("error making API request")
	ErrAPIStatus  = errors.New("error status from API")
)

type SeriesFetcher struct {
	apiURL    string
	dbService database.TimeSeriesRepository
	logger    *logrus.Logger
}

func NewSeriesFetcher(apiURL string, dbService database.TimeSeriesRepository, logger *logrus.Logger) *SeriesFetcher {
	return &SeriesFetcher{
		apiURL:    apiURL,
		dbService: dbService,
		logger:    logger,
	}
}

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
