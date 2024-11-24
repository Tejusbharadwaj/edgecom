package server

import (
	"fmt"
	"time"
)

const maxTimeRange = 2 * 365 * 24 * time.Hour

type RequestValidator struct {
	validWindows      map[string]bool
	validAggregations map[string]bool
}

func NewRequestValidator() *RequestValidator {
	return &RequestValidator{
		validWindows: map[string]bool{
			"1m": true,
			"5m": true,
			"1h": true,
			"1d": true,
		},
		validAggregations: map[string]bool{
			"MIN": true,
			"MAX": true,
			"AVG": true,
			"SUM": true,
		},
	}
}

// Validate checks if the request parameters are valid
func (v *RequestValidator) Validate(start, end time.Time, window, aggregation string) error {
	// Validate timestamps are present
	if start.IsZero() || end.IsZero() || start.Equal(time.Unix(0, 0)) || end.Equal(time.Unix(0, 0)) {
		return fmt.Errorf("missing timestamp")
	}

	// Validate time range
	if start.After(end) {
		return fmt.Errorf("start time must be before end time")
	}

	// Validate maximum time range
	if end.Sub(start) > maxTimeRange {
		return fmt.Errorf("time range exceeds maximum allowed")
	}

	// Validate window
	if window == "" {
		return fmt.Errorf("invalid window: ")
	}
	if !v.validWindows[window] {
		return fmt.Errorf("invalid window: %s", window)
	}

	// Validate aggregation
	if aggregation == "" {
		return fmt.Errorf("invalid aggregation")
	}
	if !v.validAggregations[aggregation] {
		return fmt.Errorf("invalid aggregation: %s", aggregation)
	}

	return nil
}
