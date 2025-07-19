package server

import (
	"net/http"
	"testing"

	"github.com/conneroisu/templar/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCheckOriginValidation tests the checkOrigin function with various inputs
func TestCheckOriginValidation(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}

	server, err := New(cfg)
	require.NoError(t, err)

	tests := []struct {
		name     string
		origin   string
		expected bool
		desc     string
	}{
		{
			name:     "valid localhost 8080",
			origin:   "http://localhost:8080",
			expected: true,
			desc:     "Should accept localhost:8080",
		},
		{
			name:     "valid 127.0.0.1 8080",
			origin:   "http://127.0.0.1:8080",
			expected: true,
			desc:     "Should accept 127.0.0.1:8080",
		},
		{
			name:     "valid localhost 3000",
			origin:   "http://localhost:3000",
			expected: true,
			desc:     "Should accept localhost:3000 (dev server)",
		},
		{
			name:     "valid 127.0.0.1 3000",
			origin:   "http://127.0.0.1:3000",
			expected: true,
			desc:     "Should accept 127.0.0.1:3000 (dev server)",
		},
		{
			name:     "https localhost",
			origin:   "https://localhost:8080",
			expected: true,
			desc:     "Should accept HTTPS origins",
		},
		{
			name:     "external domain",
			origin:   "http://evil.com",
			expected: false,
			desc:     "Should reject external domains",
		},
		{
			name:     "empty origin",
			origin:   "",
			expected: false,
			desc:     "Should reject empty origin",
		},
		{
			name:     "malformed origin",
			origin:   "not-a-url",
			expected: false,
			desc:     "Should reject malformed URLs",
		},
		{
			name:     "javascript protocol",
			origin:   "javascript://localhost:8080",
			expected: false,
			desc:     "Should reject non-HTTP protocols",
		},
		{
			name:     "wrong port",
			origin:   "http://localhost:9999",
			expected: false,
			desc:     "Should reject wrong port numbers",
		},
		{
			name:     "subdomain attack",
			origin:   "http://localhost.evil.com:8080",
			expected: false,
			desc:     "Should reject subdomain attacks",
		},
		{
			name:     "port manipulation",
			origin:   "http://localhost:8080.evil.com",
			expected: false,
			desc:     "Should reject port manipulation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/ws", nil)
			require.NoError(t, err)

			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			result := server.checkOrigin(req)
			assert.Equal(t, tt.expected, result, tt.desc)
		})
	}
}
