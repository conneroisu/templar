package components_test

import (
	"context"
	"strings"
	"testing"

	components "github.com/conneroisu/templar/test_components"
)

func TestTestButton(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		variant  string
		size     string
		disabled bool
		want     []string
	}{
		{
			name:     "primary button",
			text:     "Click me",
			variant:  "primary",
			size:     "medium",
			disabled: false,
			want:     []string{"btn", "btn-primary", "btn-medium", "Click me"},
		},
		{
			name:     "disabled button",
			text:     "Disabled",
			variant:  "primary",
			size:     "medium",
			disabled: true,
			want:     []string{"btn-disabled", "disabled"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf strings.Builder
			err := components.TestButton(tt.text, tt.variant, tt.size, tt.disabled, "").Render(context.Background(), &buf)
			if err != nil {
				t.Fatalf("failed to render component: %v", err)
			}

			htmlStr := buf.String()
			for _, want := range tt.want {
				if !strings.Contains(htmlStr, want) {
					t.Errorf("expected HTML to contain %q, got: %s", want, htmlStr)
				}
			}
		})
	}
}