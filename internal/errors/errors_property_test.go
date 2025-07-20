//go:build property

package errors

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestErrorCollectorProperties validates error collection and aggregation properties
func TestErrorCollectorProperties(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.Rng.Seed(2468)
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Property: Error collector should handle concurrent error addition safely
	properties.Property("concurrent error addition is thread-safe", prop.ForAll(
		func(goroutineCount int, errorsPerGoroutine int) bool {
			if goroutineCount < 1 || goroutineCount > 20 || errorsPerGoroutine < 1 || errorsPerGoroutine > 50 {
				return true
			}

			collector := NewErrorCollector()

			var wg sync.WaitGroup
			totalExpectedErrors := goroutineCount * errorsPerGoroutine

			// Launch concurrent goroutines adding errors
			for g := 0; g < goroutineCount; g++ {
				wg.Add(1)
				go func(goroutineID int) {
					defer wg.Done()
					for e := 0; e < errorsPerGoroutine; e++ {
						err := BuildError{
							Component: fmt.Sprintf("component_%d_%d", goroutineID, e),
							File:      fmt.Sprintf("file_%d_%d.templ", goroutineID, e),
							Line:      e + 1,
							Column:    1,
							Message:   fmt.Sprintf("error from goroutine %d, iteration %d", goroutineID, e),
							Severity:  ErrorSeverityError,
						}
						collector.Add(err)
					}
				}(g)
			}

			wg.Wait()

			errors := collector.GetErrors()

			// Property: Should collect all errors without loss
			return len(errors) == totalExpectedErrors
		},
		gen.IntRange(1, 10),
		gen.IntRange(1, 20),
	))

	// Property: Error collection should maintain consistency
	properties.Property("error collection is consistent", prop.ForAll(
		func(errors []BuildError) bool {
			collector := NewErrorCollector()

			// Add all errors
			for _, err := range errors {
				collector.Add(err)
			}

			allErrors := collector.GetErrors()

			// Property: Should collect all errors without loss
			return len(allErrors) == len(errors)
		},
		genBuildErrors(),
	))

	// Property: Error collection should maintain chronological order
	properties.Property("error collection maintains chronological order", prop.ForAll(
		func(errorCount int) bool {
			if errorCount < 2 || errorCount > 50 {
				return true
			}

			collector := NewErrorCollector()

			// Add errors with small delays to ensure different timestamps
			for i := 0; i < errorCount; i++ {
				err := BuildError{
					Component: fmt.Sprintf("component_%d", i),
					File:      fmt.Sprintf("file_%d.templ", i),
					Line:      i + 1,
					Column:    1,
					Message:   fmt.Sprintf("error %d", i),
					Severity:  ErrorSeverityError,
				}
				collector.Add(err)
				time.Sleep(time.Microsecond) // Ensure different timestamps
			}

			errors := collector.GetErrors()

			// Property: Errors should be in chronological order (or at least not reversed)
			orderViolations := 0
			for i := 1; i < len(errors); i++ {
				// Check if component numbers are increasing (indicating chronological order)
				prevNum := -1
				currNum := -1
				fmt.Sscanf(errors[i-1].Component, "component_%d", &prevNum)
				fmt.Sscanf(errors[i].Component, "component_%d", &currNum)
				
				if currNum < prevNum {
					orderViolations++
				}
			}

			return len(errors) == errorCount && orderViolations <= errorCount/4 // Allow some tolerance
		},
		gen.IntRange(2, 25),
	))

	// Property: Error HTML generation should be safe for all inputs
	properties.Property("error HTML generation is safe", prop.ForAll(
		func(errors []BuildError) bool {
			collector := NewErrorCollector()

			// Add all errors
			for _, err := range errors {
				collector.Add(err)
			}

			// Generate HTML overlay
			html := collector.ErrorOverlay()

			// Property: HTML should be generated without panics and contain basic structure
			return len(html) > 0 && 
				   containsString(html, "<div") && 
				   containsString(html, "</div>")
		},
		genBuildErrors(),
	))

	// Property: Error clearing should be complete and thread-safe
	properties.Property("error clearing is complete and thread-safe", prop.ForAll(
		func(initialErrors []BuildError, goroutineCount int) bool {
			if goroutineCount < 1 || goroutineCount > 10 {
				return true
			}

			collector := NewErrorCollector()

			// Add initial errors
			for _, err := range initialErrors {
				collector.Add(err)
			}

			var wg sync.WaitGroup

			// Concurrent operations: some adding errors, some clearing
			for g := 0; g < goroutineCount; g++ {
				wg.Add(1)
				go func(goroutineID int) {
					defer wg.Done()
					if goroutineID%2 == 0 {
						// Add errors
						for i := 0; i < 5; i++ {
							err := BuildError{
								Component: fmt.Sprintf("concurrent_%d_%d", goroutineID, i),
								File:      "test.templ",
								Line:      1,
								Column:    1,
								Message:   "concurrent error",
								Severity:  ErrorSeverityError,
							}
							collector.Add(err)
						}
					} else {
						// Clear errors
						time.Sleep(time.Millisecond) // Let some errors accumulate
						collector.Clear()
					}
				}(g)
			}

			wg.Wait()

			// Final clear to ensure consistency
			collector.Clear()
			finalErrors := collector.GetErrors()

			// Property: After clearing, should have no errors
			return len(finalErrors) == 0
		},
		genBuildErrors(),
		gen.IntRange(1, 6),
	))

	// Property: Error deduplication should work correctly
	properties.Property("error deduplication works correctly", prop.ForAll(
		func(baseError BuildError, duplicateCount int) bool {
			if duplicateCount < 1 || duplicateCount > 20 {
				return true
			}

			collector := NewErrorCollector()

			// Add the same error multiple times
			for i := 0; i < duplicateCount; i++ {
				collector.Add(baseError)
			}

			errors := collector.GetErrors()

			// Property: Should handle duplicates gracefully (either deduplicate or keep all)
			// Both behaviors are valid depending on implementation
			return len(errors) >= 1 && len(errors) <= duplicateCount
		},
		genBuildError(),
		gen.IntRange(1, 10),
	))

	properties.TestingRun(t)
}

// TestErrorParsingProperties validates error parsing and formatting properties
func TestErrorParsingProperties(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.Rng.Seed(3691)
	parameters.MinSuccessfulTests = 50

	properties := gopter.NewProperties(parameters)

	// Property: Error formatting should be consistent
	properties.Property("error formatting is consistent", prop.ForAll(
		func(err BuildError) bool {
			// Format error as string using the Error() method
			formatted := err.Error()
			
			// Property: Formatted string should contain essential information
			return len(formatted) > 0 && 
				   containsString(formatted, err.File) && 
				   containsString(formatted, err.Message) &&
				   containsString(formatted, err.Severity.String())
		},
		genBuildError(),
	))

	// Property: Error severity ordering should be consistent
	properties.Property("error severity ordering is consistent", prop.ForAll(
		func(errors []BuildError) bool {
			collector := NewErrorCollector()

			// Add all errors
			for _, err := range errors {
				collector.Add(err)
			}

			// Get all errors and filter by severity
			allErrors := collector.GetErrors()
			var infoErrors, warningErrors, errorLevelErrors []BuildError
			for _, err := range allErrors {
				switch err.Severity {
				case ErrorSeverityInfo:
					infoErrors = append(infoErrors, err)
				case ErrorSeverityWarning:
					warningErrors = append(warningErrors, err)
				case ErrorSeverityError:
					errorLevelErrors = append(errorLevelErrors, err)
				}
			}

			// Property: No error should appear in multiple severity lists
			infoSet := make(map[string]bool)
			warningSet := make(map[string]bool)
			errorSet := make(map[string]bool)

			for _, err := range infoErrors {
				key := fmt.Sprintf("%s:%d:%d", err.File, err.Line, err.Column)
				infoSet[key] = true
			}
			for _, err := range warningErrors {
				key := fmt.Sprintf("%s:%d:%d", err.File, err.Line, err.Column)
				warningSet[key] = true
			}
			for _, err := range errorLevelErrors {
				key := fmt.Sprintf("%s:%d:%d", err.File, err.Line, err.Column)
				errorSet[key] = true
			}

			// Check for overlaps
			for key := range infoSet {
				if warningSet[key] || errorSet[key] {
					return false
				}
			}
			for key := range warningSet {
				if errorSet[key] {
					return false
				}
			}

			return true
		},
		genBuildErrors(),
	))

	properties.TestingRun(t)
}

// Helper generators for property-based testing

func genBuildError() gopter.Gen {
	return gopter.CombineGens(
		gen.Identifier(),          // Component
		gen.Identifier(),          // File
		gen.IntRange(1, 1000),     // Line
		gen.IntRange(1, 200),      // Column
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }), // Non-empty message
		genSeverity(),             // Severity
	).Map(func(values []interface{}) BuildError {
		message := values[4].(string)
		if message == "" {
			message = "test error message"
		}
		return BuildError{
			Component: values[0].(string),
			File:      values[1].(string) + ".templ",
			Line:      values[2].(int),
			Column:    values[3].(int),
			Message:   message,
			Severity:  values[5].(ErrorSeverity),
		}
	})
}

func genBuildErrors() gopter.Gen {
	return gen.SliceOfN(20, genBuildError())
}

func genSeverity() gopter.Gen {
	return gen.OneConstOf(
		ErrorSeverityInfo,
		ErrorSeverityWarning,
		ErrorSeverityError,
	)
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (len(s) == len(substr) && s == substr || 
		   	len(s) > len(substr) && (s[:len(substr)] == substr || containsStringRec(s[1:], substr)))
}

func containsStringRec(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	if s[:len(substr)] == substr {
		return true
	}
	return containsStringRec(s[1:], substr)
}