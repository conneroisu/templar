// Package performance provides security tests for the performance detector
// to prevent path traversal vulnerabilities and ensure secure baseline operations.
package performance

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestValidateBaselineDirectory_PathTraversal tests path traversal prevention
func TestValidateBaselineDirectory_PathTraversal(t *testing.T) {
	tests := []struct {
		name        string
		baselineDir string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "valid subdirectory",
			baselineDir: "benchmarks",
			shouldError: false,
		},
		{
			name:        "valid nested subdirectory",
			baselineDir: "benchmarks/performance",
			shouldError: false,
		},
		{
			name:        "parent directory traversal with dots",
			baselineDir: "../malicious",
			shouldError: true,
			errorMsg:    "path traversal detected",
		},
		{
			name:        "nested parent directory traversal",
			baselineDir: "benchmarks/../../../etc",
			shouldError: true,
			errorMsg:    "path traversal detected",
		},
		{
			name:        "absolute path outside cwd",
			baselineDir: "/tmp/malicious",
			shouldError: true,
			errorMsg:    "outside current working directory",
		},
		{
			name:        "root directory access attempt",
			baselineDir: "/",
			shouldError: true,
			errorMsg:    "outside current working directory",
		},
		{
			name:        "system directory access attempt",
			baselineDir: "/etc/passwd",
			shouldError: true,
			errorMsg:    "access to restricted path denied",
		},
		{
			name:        "hidden parent traversal",
			baselineDir: "benchmarks/./../../malicious",
			shouldError: true,
			errorMsg:    "path traversal detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create detector with test baseline directory
			detector := NewPerformanceDetector(tt.baselineDir, DefaultThresholds())

			err := detector.validateBaselineDirectory()

			if tt.shouldError {
				if err == nil {
					t.Errorf(
						"Expected error for baseline directory %s, but got none",
						tt.baselineDir,
					)
					return
				}
				if tt.errorMsg != "" && !containsErrorMessage(err.Error(), tt.errorMsg) {
					t.Errorf(
						"Expected error message to contain '%s', got: %s",
						tt.errorMsg,
						err.Error(),
					)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for baseline directory %s, got: %v", tt.baselineDir, err)
				}
			}
		})
	}
}

// TestSaveBaseline_PathValidation tests path validation in baseline saving
func TestSaveBaseline_PathValidation(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "templar-security-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	detector := NewPerformanceDetector(tempDir, DefaultThresholds())

	tests := []struct {
		name          string
		benchmarkName string
		shouldError   bool
		errorMsg      string
	}{
		{
			name:          "valid benchmark name",
			benchmarkName: "BenchmarkValidTest",
			shouldError:   false,
		},
		{
			name:          "benchmark name with path traversal",
			benchmarkName: "../malicious",
			shouldError:   true,
			errorMsg:      "path traversal detected",
		},
		{
			name:          "benchmark name with dangerous chars - sanitized",
			benchmarkName: "Benchmark;rm -rf /",
			shouldError:   false, // sanitizeFilename removes dangerous chars
		},
		{
			name:          "benchmark name with shell injection - sanitized",
			benchmarkName: "Benchmark$(echo hack)",
			shouldError:   false, // sanitizeFilename removes dangerous chars
		},
		{
			name:          "empty benchmark name",
			benchmarkName: "",
			shouldError:   true,
			errorMsg:      "invalid benchmark name after sanitization",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseline := &PerformanceBaseline{
				BenchmarkName: tt.benchmarkName,
				Samples:       []float64{100.0, 200.0, 150.0},
				Mean:          150.0,
				Median:        150.0,
				StdDev:        50.0,
				Min:           100.0,
				Max:           200.0,
				LastUpdated:   time.Now(),
				SampleCount:   3,
			}

			err := detector.saveBaseline(baseline)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for benchmark name %s, but got none", tt.benchmarkName)
					return
				}
				if tt.errorMsg != "" && !containsErrorMessage(err.Error(), tt.errorMsg) {
					t.Errorf(
						"Expected error message to contain '%s', got: %s",
						tt.errorMsg,
						err.Error(),
					)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for benchmark name %s, got: %v", tt.benchmarkName, err)
				}
			}
		})
	}
}

// TestUpdateBaselines_SecurityValidation tests comprehensive security in baseline updates
func TestUpdateBaselines_SecurityValidation(t *testing.T) {
	tests := []struct {
		name        string
		baselineDir string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "valid relative baseline directory",
			baselineDir: "test_baselines",
			shouldError: false,
		},
		{
			name:        "baseline directory with traversal",
			baselineDir: "../malicious",
			shouldError: true,
			errorMsg:    "path traversal detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewPerformanceDetector(tt.baselineDir, DefaultThresholds())

			results := []BenchmarkResult{
				{
					Name:        "BenchmarkTest",
					Iterations:  1000,
					NsPerOp:     1500.0,
					BytesPerOp:  1024,
					AllocsPerOp: 10,
					Timestamp:   time.Now(),
				},
			}

			err := detector.UpdateBaselines(results)

			if tt.shouldError {
				if err == nil {
					t.Errorf(
						"Expected error for baseline directory %s, but got none",
						tt.baselineDir,
					)
					return
				}
				if tt.errorMsg != "" && !containsErrorMessage(err.Error(), tt.errorMsg) {
					t.Errorf(
						"Expected error message to contain '%s', got: %s",
						tt.errorMsg,
						err.Error(),
					)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for baseline directory %s, got: %v", tt.baselineDir, err)
				} else {
					// Clean up created directory after test
					defer os.RemoveAll(tt.baselineDir)

					// Verify file was created with correct permissions
					expectedFile := filepath.Join(tt.baselineDir, "BenchmarkTest.json")
					info, err := os.Stat(expectedFile)
					if err != nil {
						t.Errorf("Expected baseline file to be created, got error: %v", err)
						return
					}

					// Check file permissions are restrictive (0600)
					if info.Mode().Perm() != 0600 {
						t.Errorf("Expected file permissions 0600, got %o", info.Mode().Perm())
					}
				}
			}
		})
	}
}

// TestFilePermissions_Security tests that created files have secure permissions
func TestFilePermissions_Security(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "templar-permissions-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	detector := NewPerformanceDetector(tempDir, DefaultThresholds())

	baseline := &PerformanceBaseline{
		BenchmarkName: "BenchmarkPermissionTest",
		Samples:       []float64{100.0},
		Mean:          100.0,
		Median:        100.0,
		StdDev:        0.0,
		Min:           100.0,
		Max:           100.0,
		LastUpdated:   time.Now(),
		SampleCount:   1,
	}

	err = detector.saveBaseline(baseline)
	if err != nil {
		t.Fatalf("Failed to save baseline: %v", err)
	}

	// Check directory permissions (should be 0700)
	dirInfo, err := os.Stat(tempDir)
	if err != nil {
		t.Fatalf("Failed to stat directory: %v", err)
	}

	if dirInfo.Mode().Perm() != 0700 {
		t.Errorf("Expected directory permissions 0700, got %o", dirInfo.Mode().Perm())
	}

	// Check file permissions (should be 0600)
	filePath := filepath.Join(tempDir, "BenchmarkPermissionTest.json")
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("Failed to stat baseline file: %v", err)
	}

	if fileInfo.Mode().Perm() != 0600 {
		t.Errorf("Expected file permissions 0600, got %o", fileInfo.Mode().Perm())
	}
}

// TestSymlinkAttack_Prevention tests prevention of symlink-based attacks
func TestSymlinkAttack_Prevention(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "templar-symlink-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a malicious target outside the temp directory
	maliciousTarget := "/tmp/malicious-target"

	// Create symlink pointing outside the baseline directory
	symlinkPath := filepath.Join(tempDir, "malicious-symlink")
	err = os.Symlink(maliciousTarget, symlinkPath)
	if err != nil {
		t.Skipf("Cannot create symlink for test: %v", err)
	}

	detector := NewPerformanceDetector(symlinkPath, DefaultThresholds())

	// Attempt to validate the symlink path
	err = detector.validateBaselineDirectory()
	if err == nil {
		t.Error("Expected validation to fail for symlink pointing outside directory")
	}
}

// Helper function to check if error message contains expected substring
func containsErrorMessage(errorMsg, expectedSubstring string) bool {
	return len(expectedSubstring) == 0 ||
		len(errorMsg) > 0 &&
			(errorMsg == expectedSubstring ||
				len(errorMsg) >= len(expectedSubstring) &&
					errorMsg[:len(expectedSubstring)] == expectedSubstring ||
				containsSubstring(errorMsg, expectedSubstring))
}

// Helper function to check substring containment
func containsSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
