package components

import (
	"context"
	"strings"
	"testing"
)

func TestButton(t *testing.T) {
	tests := []struct {
		name     string
		props    ButtonProps
		contains []string
	}{
		{
			name: "basic button",
			props: ButtonProps{
				Text:    "Click me",
				Variant: "primary",
			},
			contains: []string{
				"Click me",
				"btn-primary",
				"<button",
			},
		},
		{
			name: "disabled button",
			props: ButtonProps{
				Text:     "Disabled",
				Disabled: true,
			},
			contains: []string{
				"disabled",
				"btn-disabled",
			},
		},
		{
			name: "button with size",
			props: ButtonProps{
				Text:    "Large Button",
				Size:    "large",
				Variant: "secondary",
			},
			contains: []string{
				"Large Button",
				"btn-large",
				"btn-secondary",
			},
		},
		{
			name: "button with onclick",
			props: ButtonProps{
				Text:    "Interactive",
				OnClick: "alert('clicked')",
			},
			contains: []string{
				"Interactive",
				"onclick",
				"alert('clicked')",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf strings.Builder
			err := Button(tt.props).Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("Failed to render: %v", err)
			}

			html := buf.String()
			for _, want := range tt.contains {
				if !strings.Contains(html, want) {
					t.Errorf("Expected HTML to contain %q, got: %s", want, html)
				}
			}
		})
	}
}

func TestGetButtonClasses(t *testing.T) {
	tests := []struct {
		name     string
		props    ButtonProps
		expected string
	}{
		{
			name:     "basic button",
			props:    ButtonProps{},
			expected: "btn",
		},
		{
			name: "button with variant",
			props: ButtonProps{
				Variant: "primary",
			},
			expected: "btn btn-primary",
		},
		{
			name: "button with size",
			props: ButtonProps{
				Size: "large",
			},
			expected: "btn btn-large",
		},
		{
			name: "disabled button",
			props: ButtonProps{
				Disabled: true,
			},
			expected: "btn btn-disabled",
		},
		{
			name: "button with all options",
			props: ButtonProps{
				Variant:  "primary",
				Size:     "small",
				Disabled: true,
			},
			expected: "btn btn-primary btn-small btn-disabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getButtonClasses(tt.props)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}
