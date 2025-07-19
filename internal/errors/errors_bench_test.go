package errors

import (
	"fmt"
	"testing"
	"time"
)

func BenchmarkErrorCollector_Add(b *testing.B) {
	collector := NewErrorCollector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := BuildError{
			Component: fmt.Sprintf("Component%d", i),
			File:      fmt.Sprintf("file%d.go", i),
			Line:      i,
			Column:    i % 80,
			Message:   fmt.Sprintf("Error message %d", i),
			Severity:  ErrorSeverityError,
		}
		collector.Add(err)
	}
}

func BenchmarkErrorCollector_GetErrors(b *testing.B) {
	collector := NewErrorCollector()

	// Pre-populate with errors
	for i := 0; i < 1000; i++ {
		err := BuildError{
			Component: fmt.Sprintf("Component%d", i),
			File:      fmt.Sprintf("file%d.go", i),
			Line:      i,
			Column:    i % 80,
			Message:   fmt.Sprintf("Error message %d", i),
			Severity:  ErrorSeverityError,
		}
		collector.Add(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.GetErrors()
	}
}

func BenchmarkErrorCollector_GetErrorsByFile(b *testing.B) {
	collector := NewErrorCollector()

	// Pre-populate with errors across multiple files
	for i := 0; i < 1000; i++ {
		err := BuildError{
			Component: fmt.Sprintf("Component%d", i),
			File:      fmt.Sprintf("file%d.go", i%10), // 10 different files
			Line:      i,
			Column:    i % 80,
			Message:   fmt.Sprintf("Error message %d", i),
			Severity:  ErrorSeverityError,
		}
		collector.Add(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.GetErrorsByFile(fmt.Sprintf("file%d.go", i%10))
	}
}

func BenchmarkErrorCollector_GetErrorsByComponent(b *testing.B) {
	collector := NewErrorCollector()

	// Pre-populate with errors across multiple components
	for i := 0; i < 1000; i++ {
		err := BuildError{
			Component: fmt.Sprintf("Component%d", i%20), // 20 different components
			File:      fmt.Sprintf("file%d.go", i),
			Line:      i,
			Column:    i % 80,
			Message:   fmt.Sprintf("Error message %d", i),
			Severity:  ErrorSeverityError,
		}
		collector.Add(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.GetErrorsByComponent(fmt.Sprintf("Component%d", i%20))
	}
}

func BenchmarkErrorCollector_ErrorOverlay(b *testing.B) {
	collector := NewErrorCollector()

	// Pre-populate with errors
	for i := 0; i < 10; i++ {
		err := BuildError{
			Component: fmt.Sprintf("Component%d", i),
			File:      fmt.Sprintf("file%d.go", i),
			Line:      i + 1,
			Column:    (i % 80) + 1,
			Message:   fmt.Sprintf("Error message %d with some details", i),
			Severity:  ErrorSeverityError,
			Timestamp: time.Now(),
		}
		collector.Add(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.ErrorOverlay()
	}
}

func BenchmarkErrorCollector_Clear(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector := NewErrorCollector()

		// Add some errors
		for j := 0; j < 100; j++ {
			err := BuildError{
				Component: fmt.Sprintf("Component%d", j),
				File:      fmt.Sprintf("file%d.go", j),
				Message:   fmt.Sprintf("Error message %d", j),
				Severity:  ErrorSeverityError,
			}
			collector.Add(err)
		}

		collector.Clear()
	}
}

func BenchmarkErrorCollector_Memory(b *testing.B) {
	b.ReportAllocs()

	collector := NewErrorCollector()

	for i := 0; i < b.N; i++ {
		err := BuildError{
			Component: fmt.Sprintf("Component%d", i),
			File:      fmt.Sprintf("file%d.go", i),
			Line:      i,
			Column:    i % 80,
			Message:   fmt.Sprintf("Error message %d", i),
			Severity:  ErrorSeverityError,
			Timestamp: time.Now(),
		}
		collector.Add(err)
	}
}

func BenchmarkParseTemplError(b *testing.B) {
	errorOutput := []byte(`compilation failed: syntax error at line 10, column 5
unexpected token 'func' at line 15, column 1
missing '}' at line 20, column 10`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseTemplError(errorOutput, "TestComponent")
	}
}

func BenchmarkBuildError_Error(b *testing.B) {
	err := BuildError{
		Component: "TestComponent",
		File:      "test.go",
		Line:      42,
		Column:    15,
		Message:   "unexpected token",
		Severity:  ErrorSeverityError,
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = err.Error()
	}
}

func BenchmarkErrorSeverity_String(b *testing.B) {
	severities := []ErrorSeverity{
		ErrorSeverityInfo,
		ErrorSeverityWarning,
		ErrorSeverityError,
		ErrorSeverityFatal,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		severity := severities[i%len(severities)]
		_ = severity.String()
	}
}

func BenchmarkErrorCollector_Concurrent(b *testing.B) {
	collector := NewErrorCollector()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			err := BuildError{
				Component: fmt.Sprintf("Component%d", i),
				File:      fmt.Sprintf("file%d.go", i),
				Line:      i,
				Column:    i % 80,
				Message:   fmt.Sprintf("Error message %d", i),
				Severity:  ErrorSeverityError,
			}
			collector.Add(err)

			// Occasionally read errors too
			if i%10 == 0 {
				collector.GetErrors()
			}
			i++
		}
	})
}
