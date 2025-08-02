package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockOriginValidator provides a mock origin validator for testing.
type MockOriginValidator struct {
	mock.Mock
}

func (m *MockOriginValidator) ValidateOrigin(origin string) bool {
	args := m.Called(origin)

	return args.Bool(0)
}

// MockTemplarMonitor provides a mock monitor for testing.
type MockTemplarMonitor struct {
	mock.Mock
}

func (m *MockTemplarMonitor) CreateTemplarMiddleware() func(http.Handler) http.Handler {
	args := m.Called()

	return args.Get(0).(func(http.Handler) http.Handler)
}

func (m *MockTemplarMonitor) RecordWebSocketEvent(event string, count int) {
	m.Called(event, count)
}

// TestMiddlewareChain tests the core MiddlewareChain functionality.
func TestMiddlewareChain(t *testing.T) {
	fixtures := testutils.NewTestFixtures()

	t.Run("NewMiddlewareChain", func(t *testing.T) {
		t.Run("creates chain with valid dependencies", func(t *testing.T) {
			mockValidator := &MockOriginValidator{}
			config := fixtures.BasicConfig()

			deps := MiddlewareDependencies{
				Config:          config,
				OriginValidator: mockValidator,
			}

			chain := NewMiddlewareChain(deps)

			assert.NotNil(t, chain)
			assert.Equal(t, config, chain.config)
			assert.Equal(t, mockValidator, chain.originValidator)
			assert.NotNil(t, chain.middlewares)
			assert.Greater(t, len(chain.middlewares), 0, "Should have default middlewares")
		})

		t.Run("panics with nil config", func(t *testing.T) {
			mockValidator := &MockOriginValidator{}

			deps := MiddlewareDependencies{
				Config:          nil,
				OriginValidator: mockValidator,
			}

			assert.Panics(t, func() {
				NewMiddlewareChain(deps)
			}, "Should panic with nil config")
		})

		t.Run("panics with nil origin validator", func(t *testing.T) {
			config := fixtures.BasicConfig()

			deps := MiddlewareDependencies{
				Config:          config,
				OriginValidator: nil,
			}

			assert.Panics(t, func() {
				NewMiddlewareChain(deps)
			}, "Should panic with nil origin validator")
		})

		t.Run("panics with empty environment", func(t *testing.T) {
			mockValidator := &MockOriginValidator{}
			config := fixtures.BasicConfig()
			config.Server.Environment = ""

			deps := MiddlewareDependencies{
				Config:          config,
				OriginValidator: mockValidator,
			}

			assert.Panics(t, func() {
				NewMiddlewareChain(deps)
			}, "Should panic with empty environment")
		})

		t.Run("includes rate limiter when provided", func(t *testing.T) {
			mockValidator := &MockOriginValidator{}
			config := fixtures.BasicConfig()
			rateLimiter := &RateLimiter{ /* initialize as needed */ }

			deps := MiddlewareDependencies{
				Config:          config,
				OriginValidator: mockValidator,
				RateLimiter:     rateLimiter,
			}

			chain := NewMiddlewareChain(deps)

			assert.Equal(t, rateLimiter, chain.rateLimiter)
		})

		t.Run("includes monitor when provided", func(t *testing.T) {
			mockValidator := &MockOriginValidator{}
			mockMonitor := &MockTemplarMonitor{}
			config := fixtures.BasicConfig()

			// Setup mock expectations
			mockMonitor.On("CreateTemplarMiddleware").Return(func(next http.Handler) http.Handler {
				return next
			})

			deps := MiddlewareDependencies{
				Config:          config,
				OriginValidator: mockValidator,
				Monitor:         mockMonitor,
			}

			chain := NewMiddlewareChain(deps)

			assert.Equal(t, mockMonitor, chain.monitor)
			mockMonitor.AssertExpectations(t)
		})
	})

	t.Run("AddMiddleware", func(t *testing.T) {
		t.Run("adds middleware to chain", func(t *testing.T) {
			chain := createTestChain(t, fixtures.BasicConfig())
			initialCount := len(chain.middlewares)

			testMiddleware := func(next http.Handler) http.Handler {
				return next
			}

			chain.AddMiddleware(testMiddleware)

			assert.Equal(t, initialCount+1, len(chain.middlewares))
		})

		t.Run("maintains middleware order", func(t *testing.T) {
			chain := createTestChain(t, fixtures.BasicConfig())
			chain.middlewares = nil // Clear default middlewares for this test

			middleware1 := createTestMiddleware("middleware1")
			middleware2 := createTestMiddleware("middleware2")
			middleware3 := createTestMiddleware("middleware3")

			chain.AddMiddleware(middleware1)
			chain.AddMiddleware(middleware2)
			chain.AddMiddleware(middleware3)

			assert.Equal(t, 3, len(chain.middlewares))
		})
	})

	t.Run("AddMiddlewareAt", func(t *testing.T) {
		t.Run("inserts middleware at specific position", func(t *testing.T) {
			chain := createTestChain(t, fixtures.BasicConfig())
			chain.middlewares = nil // Clear for predictable testing

			middleware1 := createTestMiddleware("middleware1")
			middleware2 := createTestMiddleware("middleware2")
			middleware3 := createTestMiddleware("middleware3")

			chain.AddMiddleware(middleware1)
			chain.AddMiddleware(middleware3)
			chain.AddMiddlewareAt(1, middleware2)

			assert.Equal(t, 3, len(chain.middlewares))
		})

		t.Run("appends when index is out of bounds", func(t *testing.T) {
			chain := createTestChain(t, fixtures.BasicConfig())
			initialCount := len(chain.middlewares)

			testMiddleware := createTestMiddleware("test")

			chain.AddMiddlewareAt(999, testMiddleware)

			assert.Equal(t, initialCount+1, len(chain.middlewares))
		})

		t.Run("appends when index is negative", func(t *testing.T) {
			chain := createTestChain(t, fixtures.BasicConfig())
			initialCount := len(chain.middlewares)

			testMiddleware := createTestMiddleware("test")

			chain.AddMiddlewareAt(-1, testMiddleware)

			assert.Equal(t, initialCount+1, len(chain.middlewares))
		})
	})

	t.Run("Apply", func(t *testing.T) {
		t.Run("applies middlewares in correct order", func(t *testing.T) {
			chain := createTestChain(t, fixtures.BasicConfig())
			chain.middlewares = nil // Clear default middlewares

			var executionOrder []string

			// Create middlewares that record execution order
			middleware1 := func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					executionOrder = append(executionOrder, "middleware1-before")
					next.ServeHTTP(w, r)
					executionOrder = append(executionOrder, "middleware1-after")
				})
			}

			middleware2 := func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					executionOrder = append(executionOrder, "middleware2-before")
					next.ServeHTTP(w, r)
					executionOrder = append(executionOrder, "middleware2-after")
				})
			}

			baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				executionOrder = append(executionOrder, "handler")
			})

			chain.AddMiddleware(middleware1)
			chain.AddMiddleware(middleware2)

			wrappedHandler := chain.Apply(baseHandler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(w, req)

			// Verify execution order (middleware2 should be outermost)
			expected := []string{
				"middleware2-before",
				"middleware1-before",
				"handler",
				"middleware1-after",
				"middleware2-after",
			}
			assert.Equal(t, expected, executionOrder)
		})

		t.Run("panics with nil handler", func(t *testing.T) {
			chain := createTestChain(t, fixtures.BasicConfig())

			assert.Panics(t, func() {
				chain.Apply(nil)
			}, "Should panic with nil handler")
		})

		t.Run("panics with nil middleware in chain", func(t *testing.T) {
			chain := createTestChain(t, fixtures.BasicConfig())
			chain.middlewares = []Middleware{nil}

			baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

			assert.Panics(t, func() {
				chain.Apply(baseHandler)
			}, "Should panic with nil middleware")
		})

		t.Run("handles empty middleware chain", func(t *testing.T) {
			chain := createTestChain(t, fixtures.BasicConfig())
			chain.middlewares = []Middleware{}

			baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			wrappedHandler := chain.Apply(baseHandler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	})

	t.Run("CORS Middleware", func(t *testing.T) {
		t.Run("allows valid origins", func(t *testing.T) {
			mockValidator := &MockOriginValidator{}
			mockValidator.On("ValidateOrigin", "https://example.com").Return(true)

			config := fixtures.BasicConfig()
			config.Server.Environment = "production"

			chain := createTestChainWithValidator(t, config, mockValidator)

			baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			wrappedHandler := chain.Apply(baseHandler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Origin", "https://example.com")
			w := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(w, req)

			assert.Equal(t, "https://example.com", w.Header().Get("Access-Control-Allow-Origin"))
			mockValidator.AssertExpectations(t)
		})

		t.Run("blocks invalid origins in production", func(t *testing.T) {
			mockValidator := &MockOriginValidator{}
			mockValidator.On("ValidateOrigin", "https://malicious.com").Return(false)

			config := fixtures.BasicConfig()
			config.Server.Environment = "production"

			chain := createTestChainWithValidator(t, config, mockValidator)

			baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			wrappedHandler := chain.Apply(baseHandler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Origin", "https://malicious.com")
			w := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(w, req)

			assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
			mockValidator.AssertExpectations(t)
		})

		t.Run("allows wildcard in development", func(t *testing.T) {
			mockValidator := &MockOriginValidator{}
			mockValidator.On("ValidateOrigin", "https://localhost:3000").Return(false)

			config := fixtures.BasicConfig()
			config.Server.Environment = "development"

			chain := createTestChainWithValidator(t, config, mockValidator)

			baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			wrappedHandler := chain.Apply(baseHandler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Origin", "https://localhost:3000")
			w := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(w, req)

			assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
			mockValidator.AssertExpectations(t)
		})

		t.Run("handles preflight requests", func(t *testing.T) {
			mockValidator := &MockOriginValidator{}
			config := fixtures.BasicConfig()

			chain := createTestChainWithValidator(t, config, mockValidator)

			baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Error("Base handler should not be called for OPTIONS request")
			})

			wrappedHandler := chain.Apply(baseHandler)

			req := httptest.NewRequest(http.MethodOptions, "/test", nil)
			w := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "GET, POST, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
			assert.Equal(t, "Content-Type", w.Header().Get("Access-Control-Allow-Headers"))
		})
	})

	t.Run("Logging Middleware", func(t *testing.T) {
		t.Run("logs requests", func(t *testing.T) {
			chain := createTestChain(t, fixtures.BasicConfig())

			baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(10 * time.Millisecond) // Simulate some processing time
				w.WriteHeader(http.StatusOK)
			})

			wrappedHandler := chain.Apply(baseHandler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})

		t.Run("records monitoring events when monitor is available", func(t *testing.T) {
			mockValidator := &MockOriginValidator{}
			mockMonitor := &MockTemplarMonitor{}
			config := fixtures.BasicConfig()

			// Setup mock expectations
			mockMonitor.On("CreateTemplarMiddleware").Return(func(next http.Handler) http.Handler {
				return next
			})
			mockMonitor.On("RecordWebSocketEvent", "http_request", 1).Return()

			deps := MiddlewareDependencies{
				Config:          config,
				OriginValidator: mockValidator,
				Monitor:         mockMonitor,
			}

			chain := NewMiddlewareChain(deps)

			baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			wrappedHandler := chain.Apply(baseHandler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(w, req)

			mockMonitor.AssertExpectations(t)
		})
	})
}

// TestMiddlewareChainConcurrency tests concurrent access to the middleware chain.
func TestMiddlewareChainConcurrency(t *testing.T) {
	t.Run("Apply is safe for concurrent access", func(t *testing.T) {
		chain := createTestChain(t, testutils.NewTestFixtures().BasicConfig())

		baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		// Test concurrent Apply calls
		const numGoroutines = 10
		done := make(chan bool, numGoroutines)

		for range numGoroutines {
			go func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("Panic in concurrent Apply: %v", r)
					}
					done <- true
				}()

				wrappedHandler := chain.Apply(baseHandler)

				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				w := httptest.NewRecorder()

				wrappedHandler.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
			}()
		}

		// Wait for all goroutines to complete
		for range numGoroutines {
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("Timeout waiting for concurrent Apply operations")
			}
		}
	})
}

// Helper functions for testing

func createTestChain(t *testing.T, config *config.Config) *MiddlewareChain {
	mockValidator := &MockOriginValidator{}

	return createTestChainWithValidator(t, config, mockValidator)
}

func createTestChainWithValidator(t *testing.T, config *config.Config, validator OriginValidator) *MiddlewareChain {
	deps := MiddlewareDependencies{
		Config:          config,
		OriginValidator: validator,
	}

	return NewMiddlewareChain(deps)
}

func createTestMiddleware(name string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware", name)
			next.ServeHTTP(w, r)
		})
	}
}

// TestMiddlewareChainEdgeCases tests edge cases and error conditions.
func TestMiddlewareChainEdgeCases(t *testing.T) {
	t.Run("chain with middleware that returns nil handler", func(t *testing.T) {
		chain := createTestChain(t, testutils.NewTestFixtures().BasicConfig())
		chain.middlewares = nil // Clear defaults

		badMiddleware := func(next http.Handler) http.Handler {
			return nil // This should cause a panic
		}

		chain.AddMiddleware(badMiddleware)

		baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

		assert.Panics(t, func() {
			chain.Apply(baseHandler)
		}, "Should panic when middleware returns nil handler")
	})

	t.Run("chain with nil middlewares slice", func(t *testing.T) {
		chain := createTestChain(t, testutils.NewTestFixtures().BasicConfig())
		chain.middlewares = nil

		baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

		assert.Panics(t, func() {
			chain.Apply(baseHandler)
		}, "Should panic when middlewares slice is nil")
	})
}

// BenchmarkMiddlewareChain benchmarks middleware chain performance.
func BenchmarkMiddlewareChain(b *testing.B) {
	fixtures := testutils.NewTestFixtures()
	chain := createTestChain(&testing.T{}, fixtures.BasicConfig())

	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := chain.Apply(baseHandler)

	b.ResetTimer()

	for range b.N {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)
	}
}

// BenchmarkMiddlewareChainWithManyMiddlewares benchmarks performance with many middlewares.
func BenchmarkMiddlewareChainWithManyMiddlewares(b *testing.B) {
	fixtures := testutils.NewTestFixtures()
	chain := createTestChain(&testing.T{}, fixtures.BasicConfig())

	// Add many test middlewares
	for range 20 {
		chain.AddMiddleware(createTestMiddleware("test"))
	}

	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := chain.Apply(baseHandler)

	b.ResetTimer()

	for range b.N {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)
	}
}
