package errors

import (
	"fmt"
	"testing"
	"time"
)

func BenchmarkErrorCollector_Add(b *testing.B) {
	collector := NewErrorCollector()

	b.ResetTimer()
	for i := range b.N {
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
	for i := range 1000 {
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
	for range b.N {
		collector.GetErrors()
	}
}

func BenchmarkErrorCollector_GetErrorsByFile(b *testing.B) {
	collector := NewErrorCollector()

	// Pre-populate with errors across multiple files
	for i := range 1000 {
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
	for i := range b.N {
		collector.GetErrorsByFile(fmt.Sprintf("file%d.go", i%10))
	}
}

func BenchmarkErrorCollector_GetErrorsByComponent(b *testing.B) {
	collector := NewErrorCollector()

	// Pre-populate with errors across multiple components
	for i := range 1000 {
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
	for i := range b.N {
		collector.GetErrorsByComponent(fmt.Sprintf("Component%d", i%20))
	}
}

func BenchmarkErrorCollector_ErrorOverlay(b *testing.B) {
	collector := NewErrorCollector()

	// Pre-populate with errors
	for i := range 10 {
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
	for range b.N {
		collector.ErrorOverlay()
	}
}

func BenchmarkErrorCollector_Clear(b *testing.B) {
	b.ResetTimer()
	for range b.N {
		collector := NewErrorCollector()

		// Add some errors
		for j := range 100 {
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

	for i := range b.N {
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
	for range b.N {
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
	for range b.N {
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
	for i := range b.N {
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
