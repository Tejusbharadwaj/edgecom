//go:generate godoc -html . > ../../docs/internal/models/index.html

// Package models defines the core data structures used throughout the EdgeCom service.
// It provides types for both external API communication and internal data representation.
package models

import "time"

// APIResponse represents the response structure from the EdgeCom Energy API.
// It contains an array of time series data points in the API's native format.
//
// The structure matches the JSON response:
//
//	{
//	  "result": [
//	    {
//	      "time": 1637760000,  // Unix timestamp
//	      "value": 42.5        // Measurement value
//	    },
//	    ...
//	  ]
//	}
type APIResponse struct {
	Result []struct {
		// Time is a Unix timestamp representing the measurement time
		Time int64 `json:"time"`
		// Value is the measurement value
		Value float64 `json:"value"`
	} `json:"result"`
}

// TimeSeriesData represents a single time series data point in the internal format.
// This structure is used throughout the application for consistent data handling.
// Parameters:
//   - unixTime: Unix timestamp from the API
//   - value: Measurement value from the API
//
// Returns:
//   - TimeSeriesData with properly converted time and value
type TimeSeriesData struct {
	// Time is the timestamp of the measurement
	Time time.Time `json:"time"`
	// Value is the measurement value
	Value float64 `json:"value"`
}
