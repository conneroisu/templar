package performance

import (
	"context"
	"strings"
	"templar/examples/basic/components"
	"testing"
)

func BenchmarkButtonRender(b *testing.B) {
	props := components.ButtonProps{
		Text:    "Benchmark",
		Variant: "primary",
		Size:    "large",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf strings.Builder
		_ = components.Button(props).Render(context.Background(), &buf)
	}
}

func BenchmarkConcurrentRender(b *testing.B) {
	props := components.ButtonProps{Text: "Test", Variant: "primary"}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var buf strings.Builder
			_ = components.Button(props).Render(context.Background(), &buf)
		}
	})
}

func BenchmarkCardRender(b *testing.B) {
	props := components.CardProps{
		Title:    "Benchmark Card",
		Subtitle: "Testing performance",
		Shadow:   true,
		Padding:  "large",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf strings.Builder
		_ = components.Card(props).Render(context.Background(), &buf)
	}
}

func BenchmarkFormFieldRender(b *testing.B) {
	props := components.FormFieldProps{
		Name:        "email",
		Type:        "email",
		Label:       "Email Address",
		Placeholder: "Enter your email",
		Required:    true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf strings.Builder
		_ = components.FormField(props).Render(context.Background(), &buf)
	}
}

func BenchmarkMultipleComponents(b *testing.B) {
	buttonProps := components.ButtonProps{Text: "Submit", Variant: "primary"}
	cardProps := components.CardProps{Title: "Form Card", Shadow: true}
	formProps := components.FormFieldProps{
		Name:     "username",
		Type:     "text",
		Label:    "Username",
		Required: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf strings.Builder
		_ = components.Button(buttonProps).Render(context.Background(), &buf)
		buf.Reset()
		_ = components.Card(cardProps).Render(context.Background(), &buf)
		buf.Reset()
		_ = components.FormField(formProps).Render(context.Background(), &buf)
	}
}

func BenchmarkStringBuilderReuse(b *testing.B) {
	props := components.ButtonProps{Text: "Reuse Test", Variant: "secondary"}
	var buf strings.Builder

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = components.Button(props).Render(context.Background(), &buf)
	}
}

func BenchmarkMemoryUsage(b *testing.B) {
	props := components.CardProps{
		Title:    "Memory Test",
		Subtitle: "Testing memory allocation patterns",
		ImageUrl: "https://example.com/image.jpg",
		Shadow:   true,
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var buf strings.Builder
		_ = components.Card(props).Render(context.Background(), &buf)
	}
}
