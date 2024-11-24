package models

import "time"

// APIResponse represents the response from the external API
type APIResponse struct {
	Result []struct {
		Time  int64   `json:"time"`
		Value float64 `json:"value"`
	} `json:"result"`
}

// TimeSeriesData represents a single time series data point
type TimeSeriesData struct {
	Time  time.Time `json:"time"`
	Value float64   `json:"value"`
}
