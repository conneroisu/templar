package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// TestRunInitValidation tests that the runInit function properly validates arguments.
func TestRunInitValidation(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorText   string
	}{
		{
			name:        "path traversal attempt",
			args:        []string{"../../../invalid"},
			expectError: true,
			errorText:   "path traversal",
		},
		{
			name:        "semicolon injection attempt",
			args:        []string{"project; rm -rf /"},
			expectError: true,
			errorText:   "dangerous character",
		},
		{
			name:        "pipe injection attempt",
			args:        []string{"project | cat /etc/passwd"},
			expectError: true,
			errorText:   "dangerous character",
		},
		{
			name:        "backtick injection attempt",
			args:        []string{"project`whoami`"},
			expectError: true,
			errorText:   "dangerous character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the runInit function directly
			cmd := &cobra.Command{}
			err := runInit(cmd, tt.args)

			if tt.expectError {
				assert.Error(t, err, "Expected error for test case: %s", tt.name)
				if tt.errorText != "" && err != nil {
					t.Logf("Full error message: %s", err.Error())
					// Check if the error message contains the expected text (case insensitive)
					assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(tt.errorText),
						"Error should contain '%s' for test: %s. Got: %s", tt.errorText, tt.name, err.Error())
				}
			} else if err != nil {
				// For valid cases, we might get other errors (like missing directories)
				// but they shouldn't be validation errors
				assert.NotContains(t, err.Error(), "dangerous character")
				assert.NotContains(t, err.Error(), "path traversal")
			}
		})
	}
}
