package admin

import (
	"fmt"
	"testing"
)

// TestForecastRequestValidation tests forecast request validation
func TestForecastRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		req     ForecastRequest
		wantErr bool
	}{
		{
			name:    "empty metric",
			req:     ForecastRequest{RouteID: "test", Horizon: 24},
			wantErr: true,
		},
		{
			name:    "zero horizon",
			req:     ForecastRequest{Metric: "requests", RouteID: "test", Horizon: 0},
			wantErr: true,
		},
		{
			name:    "negative horizon",
			req:     ForecastRequest{Metric: "requests", RouteID: "test", Horizon: -1},
			wantErr: true,
		},
		{
			name:    "too large horizon",
			req:     ForecastRequest{Metric: "requests", RouteID: "test", Horizon: 1000},
			wantErr: true,
		},
		{
			name:    "valid request",
			req:     ForecastRequest{Metric: "requests", RouteID: "test", Horizon: 24},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateForecastRequest(&tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateForecastRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// validateForecastRequest validates a forecast request
func validateForecastRequest(req *ForecastRequest) error {
	if req.Metric == "" {
		return fmt.Errorf("metric is required")
	}
	if req.Horizon <= 0 {
		return fmt.Errorf("horizon must be positive")
	}
	if req.Horizon > 720 {
		return fmt.Errorf("horizon cannot exceed 720 hours (30 days)")
	}
	return nil
}
