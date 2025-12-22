package models

import (
	"errors"
	"testing"
	"time"
)

func TestResult_Success(t *testing.T) {
	tests := []struct {
		name     string
		result   Result
		expected bool
	}{
		{
			name:     "returns true when error is nil",
			result:   Result{Error: nil},
			expected: true,
		},
		{
			name:     "returns false when error is set",
			result:   Result{Error: errors.New("some error")},
			expected: false,
		},
		{
			name: "returns true with audiobook but no error",
			result: Result{
				Audiobook: &Audiobook{Path: "/test"},
				Error:     nil,
				Duration:  time.Second,
			},
			expected: true,
		},
		{
			name: "returns false with audiobook and error",
			result: Result{
				Audiobook: &Audiobook{Path: "/test"},
				Error:     ErrNoMetadataFound,
				Duration:  time.Second * 2,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.Success()
			if got != tt.expected {
				t.Errorf("Success() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSummary_String(t *testing.T) {
	tests := []struct {
		name     string
		summary  Summary
		expected string
	}{
		{
			name: "formats all zeros",
			summary: Summary{
				Total:     0,
				Processed: 0,
				Skipped:   0,
				Errors:    0,
			},
			expected: "[OK] 0 processed | [-] 0 skipped | [ERR] 0 errors",
		},
		{
			name: "formats with values",
			summary: Summary{
				Total:     10,
				Processed: 7,
				Skipped:   2,
				Errors:    1,
				Duration:  time.Minute * 5,
			},
			expected: "[OK] 7 processed | [-] 2 skipped | [ERR] 1 errors",
		},
		{
			name: "formats single values",
			summary: Summary{
				Total:     1,
				Processed: 1,
				Skipped:   0,
				Errors:    0,
			},
			expected: "[OK] 1 processed | [-] 0 skipped | [ERR] 0 errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.summary.String()
			if got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestErrorVariables(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "ErrNoMetadataFound has correct message",
			err:      ErrNoMetadataFound,
			expected: "no metadata found for query",
		},
		{
			name:     "ErrSourceTimeout has correct message",
			err:      ErrSourceTimeout,
			expected: "metadata source timeout",
		},
		{
			name:     "ErrInvalidPath has correct message",
			err:      ErrInvalidPath,
			expected: "invalid audiobook path",
		},
		{
			name:     "ErrDestinationExists has correct message",
			err:      ErrDestinationExists,
			expected: "destination already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("Error() = %q, want %q", tt.err.Error(), tt.expected)
			}
		})
	}
}

func TestErrorsAreDistinct(t *testing.T) {
	// Verify all error variables are distinct
	allErrors := []error{
		ErrNoMetadataFound,
		ErrSourceTimeout,
		ErrInvalidPath,
		ErrDestinationExists,
	}

	for i, err1 := range allErrors {
		for j, err2 := range allErrors {
			if i != j && errors.Is(err1, err2) {
				t.Errorf("Error %v should not match %v", err1, err2)
			}
		}
	}
}
