package types

import (
	"testing"
	"time"
)

func TestHealthCheck_GetInitialDelay(t *testing.T) {
	tests := []struct {
		name     string
		check    HealthCheck
		expected time.Duration
	}{
		{
			name:     "zero initial delay",
			check:    HealthCheck{InitialDelaySeconds: 0},
			expected: 0,
		},
		{
			name:     "positive initial delay",
			check:    HealthCheck{InitialDelaySeconds: 10},
			expected: 10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				if got := tt.check.GetInitialDelay(); got != tt.expected {
					t.Errorf("GetInitialDelay() = %v, want %v", got, tt.expected)
				}
			},
		)
	}
}

func TestHealthCheck_GetPeriod(t *testing.T) {
	tests := []struct {
		name     string
		check    HealthCheck
		expected time.Duration
	}{
		{
			name:     "zero period defaults to 10s",
			check:    HealthCheck{PeriodSeconds: 0},
			expected: 10 * time.Second,
		},
		{
			name:     "custom period",
			check:    HealthCheck{PeriodSeconds: 5},
			expected: 5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				if got := tt.check.GetPeriod(); got != tt.expected {
					t.Errorf("GetPeriod() = %v, want %v", got, tt.expected)
				}
			},
		)
	}
}

func TestHealthCheck_GetTimeout(t *testing.T) {
	tests := []struct {
		name     string
		check    HealthCheck
		expected time.Duration
	}{
		{
			name:     "zero timeout defaults to 1s",
			check:    HealthCheck{TimeoutSeconds: 0},
			expected: 1 * time.Second,
		},
		{
			name:     "custom timeout",
			check:    HealthCheck{TimeoutSeconds: 3},
			expected: 3 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				if got := tt.check.GetTimeout(); got != tt.expected {
					t.Errorf("GetTimeout() = %v, want %v", got, tt.expected)
				}
			},
		)
	}
}

func TestHealthCheck_GetSuccessThreshold(t *testing.T) {
	tests := []struct {
		name     string
		check    HealthCheck
		expected int
	}{
		{
			name:     "zero threshold defaults to 1",
			check:    HealthCheck{SuccessThreshold: 0},
			expected: 1,
		},
		{
			name:     "custom threshold",
			check:    HealthCheck{SuccessThreshold: 3},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				if got := tt.check.GetSuccessThreshold(); got != tt.expected {
					t.Errorf("GetSuccessThreshold() = %v, want %v", got, tt.expected)
				}
			},
		)
	}
}

func TestHealthCheck_GetFailureThreshold(t *testing.T) {
	tests := []struct {
		name     string
		check    HealthCheck
		expected int
	}{
		{
			name:     "zero threshold defaults to 3",
			check:    HealthCheck{FailureThreshold: 0},
			expected: 3,
		},
		{
			name:     "custom threshold",
			check:    HealthCheck{FailureThreshold: 5},
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				if got := tt.check.GetFailureThreshold(); got != tt.expected {
					t.Errorf("GetFailureThreshold() = %v, want %v", got, tt.expected)
				}
			},
		)
	}
}
