package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/tejusbharadwaj/edgecom/internal/database"
)

type APIResponse struct {
	Result []struct {
		Time  int64   `json:"time"`
		Value float64 `json:"value"`
	} `json:"result"`
}

type SeriesFetcher struct {
	apiURL    string
	dbService database.TimeSeriesRepository
}

var (
	ErrGRPCRequest = errors.New("error making gRPC request")
	ErrGRPCStatus  = errors.New("error status from gRPC service")
)

func NewSeriesFetcher(apiURL string, dbService database.TimeSeriesRepository) *SeriesFetcher {
	return &SeriesFetcher{
		apiURL:    apiURL,
		dbService: dbService,
	}
}

func (f *SeriesFetcher) FetchData(ctx context.Context, start, end time.Time) error {
	url := fmt.Sprintf("%s?start=%s&end=%s",
		f.apiURL,
		start.Format(time.RFC3339),
		end.Format(time.RFC3339))

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrGRPCRequest, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrGRPCRequest, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: got %d", ErrGRPCStatus, resp.StatusCode)
	}

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	if len(apiResp.Result) == 0 {
		return nil
	}

	dataPoints := make([]database.TimeSeriesData, len(apiResp.Result))
	for i, data := range apiResp.Result {
		dataPoints[i] = database.TimeSeriesData{
			Time:  time.Unix(data.Time, 0),
			Value: data.Value,
		}
	}

	if err := f.dbService.BatchInsertTimeSeriesData(ctx, dataPoints); err != nil {
		return fmt.Errorf("failed to insert data points: %v", err)
	}

	return nil
}

func (f *SeriesFetcher) BootstrapHistoricalData(ctx context.Context) error {
	endTime := time.Now()
	startTime := endTime.AddDate(-2, 0, 0)

	return f.FetchData(ctx, startTime, endTime)
}
