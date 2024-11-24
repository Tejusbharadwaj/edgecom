package server

import (
	"testing"
	"time"
)

func TestRequestValidator_Validate(t *testing.T) {
	validator := NewRequestValidator()
	now := time.Now()

	tests := []struct {
		name        string
		start       time.Time
		end         time.Time
		window      string
		aggregation string
		wantErr     bool
		errMessage  string
	}{
		{
			name:        "valid request",
			start:       now.Add(-24 * time.Hour),
			end:         now,
			window:      "1h",
			aggregation: "AVG",
			wantErr:     false,
		},
		{
			name:        "missing timestamp",
			start:       time.Time{},
			end:         now,
			window:      "1h",
			aggregation: "AVG",
			wantErr:     true,
			errMessage:  "missing timestamp",
		},
		{
			name:        "invalid time range",
			start:       now,
			end:         now.Add(-24 * time.Hour),
			window:      "1h",
			aggregation: "AVG",
			wantErr:     true,
			errMessage:  "start time must be before end time",
		},
		{
			name:        "exceeds max time range",
			start:       now.Add(-3 * 365 * 24 * time.Hour),
			end:         now,
			window:      "1h",
			aggregation: "AVG",
			wantErr:     true,
			errMessage:  "time range exceeds maximum allowed",
		},
		{
			name:        "invalid window",
			start:       now.Add(-24 * time.Hour),
			end:         now,
			window:      "2h",
			aggregation: "AVG",
			wantErr:     true,
			errMessage:  "invalid window: 2h",
		},
		{
			name:        "empty window",
			start:       now.Add(-24 * time.Hour),
			end:         now,
			window:      "",
			aggregation: "AVG",
			wantErr:     true,
			errMessage:  "invalid window: ",
		},
		{
			name:        "invalid aggregation",
			start:       now.Add(-24 * time.Hour),
			end:         now,
			window:      "1h",
			aggregation: "INVALID",
			wantErr:     true,
			errMessage:  "invalid aggregation: INVALID",
		},
		{
			name:        "empty aggregation",
			start:       now.Add(-24 * time.Hour),
			end:         now,
			window:      "1h",
			aggregation: "",
			wantErr:     true,
			errMessage:  "invalid aggregation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.start, tt.end, tt.window, tt.aggregation)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMessage {
				t.Errorf("Validate() error message = %v, want %v", err.Error(), tt.errMessage)
			}
		})
	}
}
