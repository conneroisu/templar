package di

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/config"
)

// TestCircularDependencyDetection tests that circular dependencies are detected.
func TestCircularDependencyDetection(t *testing.T) {
	cfg := &config.Config{}
	container := NewServiceContainer(cfg)

	// Register services with circular dependencies
	container.Register("serviceA", func(resolver DependencyResolver) (interface{}, error) {
		// ServiceA depends on ServiceB
		serviceB, err := resolver.Get("serviceB")
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{"name": "serviceA", "dependency": serviceB}, nil
	}).AsSingleton()

	container.Register("serviceB", func(resolver DependencyResolver) (interface{}, error) {
		// ServiceB depends on ServiceA (circular dependency)
		serviceA, err := resolver.Get("serviceA")
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{"name": "serviceB", "dependency": serviceA}, nil
	}).AsSingleton()

	// Attempting to resolve should detect circular dependency
	_, err := container.Get("serviceA")
	if err == nil {
		t.Error("Expected circular dependency error, but got nil")
	}

	if err != nil && !containsString(err.Error(), "circular dependency") {
		t.Errorf("Expected circular dependency error, got: %v", err)
	}
}

// TestConcurrentSingletonCreation tests that singletons are created safely under concurrent access.
func TestConcurrentSingletonCreation(t *testing.T) {
	cfg := &config.Config{}
	container := NewServiceContainer(cfg)

	// Track creation count to ensure singleton behavior
	var creationCount int32
	var mu sync.Mutex

	container.RegisterSingleton(
		"expensiveService",
		func(resolver DependencyResolver) (interface{}, error) {
			mu.Lock()
			creationCount++
			currentCount := creationCount
			mu.Unlock()

			// Simulate expensive creation
			time.Sleep(time.Millisecond * 10)

			return map[string]interface{}{
				"id":      currentCount,
				"created": time.Now(),
			}, nil
		},
	)

	// Launch multiple goroutines trying to get the same singleton
	const numGoroutines = 50
	var wg sync.WaitGroup
	results := make([]interface{}, numGoroutines)
	errors := make([]error, numGoroutines)

	for i := range numGoroutines {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			service, err := container.Get("expensiveService")
			results[index] = service
			errors[index] = err
		}(i)
	}

	wg.Wait()

	// Verify all goroutines got the same instance by checking they're the same pointer
	var firstServiceMap map[string]interface{}
	for i, result := range results {
		if errors[i] != nil {
			t.Errorf("Goroutine %d got error: %v", i, errors[i])
		}

		if result == nil {
			t.Errorf("Goroutine %d got nil result", i)

			continue
		}

		resultMap, ok := result.(map[string]interface{})
		if !ok {
			t.Errorf("Goroutine %d got unexpected type: %T", i, result)

			continue
		}

		if firstServiceMap == nil {
			firstServiceMap = resultMap
		} else {
			// Compare that they have the same ID (indicating singleton behavior)
			firstID := firstServiceMap["id"]
			currentID := resultMap["id"]
			if firstID != currentID {
				t.Errorf("Goroutine %d got different instance (ID %v vs %v)", i, currentID, firstID)
			}
		}
	}

	// Verify service was created only once
	mu.Lock()
	if creationCount != 1 {
		t.Errorf("Expected service to be created once, but was created %d times", creationCount)
	}
	mu.Unlock()
}

// TestDeadlockPrevention tests that dependency resolution doesn't deadlock.
func TestDeadlockPrevention(t *testing.T) {
	cfg := &config.Config{}
	container := NewServiceContainer(cfg)

	// Register a service that depends on another service
	container.RegisterSingleton("serviceA", func(resolver DependencyResolver) (interface{}, error) {
		// This should not deadlock when serviceA is being created
		serviceB, err := resolver.Get("serviceB")
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"name":       "serviceA",
			"dependency": serviceB,
		}, nil
	})

	container.RegisterSingleton("serviceB", func(resolver DependencyResolver) (interface{}, error) {
		// Simple service that doesn't depend on anything
		return map[string]interface{}{
			"name": "serviceB",
		}, nil
	})

	// This should complete without deadlocking
	done := make(chan bool, 1)
	go func() {
		_, err := container.Get("serviceA")
		if err != nil {
			t.Errorf("Failed to resolve serviceA: %v", err)
		}
		done <- true
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		// Success - no deadlock
	case <-time.After(time.Second * 5):
		t.Error("Test timed out - likely deadlock occurred")
	}
}

// TestConcurrentDifferentServices tests concurrent access to different services.
func TestConcurrentDifferentServices(t *testing.T) {
	cfg := &config.Config{}
	container := NewServiceContainer(cfg)

	// Register multiple services
	for i := range 10 {
		serviceName := fmt.Sprintf("service%d", i)
		serviceValue := i
		container.RegisterSingleton(
			serviceName,
			func(resolver DependencyResolver) (interface{}, error) {
				// Simulate some work
				time.Sleep(time.Millisecond * 5)

				return map[string]interface{}{
					"id":   serviceValue,
					"name": serviceName,
				}, nil
			},
		)
	}

	// Concurrently access different services
	const numGoroutines = 20
	var wg sync.WaitGroup
	errors := make([]error, numGoroutines)

	for i := range numGoroutines {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			serviceName := fmt.Sprintf("service%d", index%10)
			_, err := container.Get(serviceName)
			errors[index] = err
		}(i)
	}

	wg.Wait()

	// Check for errors
	for i, err := range errors {
		if err != nil {
			t.Errorf("Goroutine %d got error: %v", i, err)
		}
	}
}

// TestDependencyChain tests a longer chain of dependencies.
func TestDependencyChain(t *testing.T) {
	cfg := &config.Config{}
	container := NewServiceContainer(cfg)

	// Create a chain: A -> B -> C -> D
	container.RegisterSingleton("serviceD", func(resolver DependencyResolver) (interface{}, error) {
		return "serviceD", nil
	})

	container.RegisterSingleton("serviceC", func(resolver DependencyResolver) (interface{}, error) {
		dep, err := resolver.Get("serviceD")
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{"name": "serviceC", "dep": dep}, nil
	})

	container.RegisterSingleton("serviceB", func(resolver DependencyResolver) (interface{}, error) {
		dep, err := resolver.Get("serviceC")
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{"name": "serviceB", "dep": dep}, nil
	})

	container.RegisterSingleton("serviceA", func(resolver DependencyResolver) (interface{}, error) {
		dep, err := resolver.Get("serviceB")
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{"name": "serviceA", "dep": dep}, nil
	})

	// This should resolve without deadlock
	service, err := container.Get("serviceA")
	if err != nil {
		t.Errorf("Failed to resolve dependency chain: %v", err)
	}

	if service == nil {
		t.Error("Expected service, got nil")
	}
}

// TestTransientServiceCreation tests that transient services are created each time.
func TestTransientServiceCreation(t *testing.T) {
	cfg := &config.Config{}
	container := NewServiceContainer(cfg)

	var creationCount int32
	var mu sync.Mutex

	container.Register("transientService", func(resolver DependencyResolver) (interface{}, error) {
		mu.Lock()
		creationCount++
		currentCount := creationCount
		mu.Unlock()

		return map[string]interface{}{
			"id":      currentCount,
			"created": time.Now(),
		}, nil
	}) // Not calling AsSingleton(), so it's transient

	// Get the service multiple times
	const numCalls = 5
	instances := make([]interface{}, numCalls)

	for i := range numCalls {
		service, err := container.Get("transientService")
		if err != nil {
			t.Errorf("Call %d failed: %v", i, err)
		}
		instances[i] = service
	}

	// Verify each call created a new instance
	mu.Lock()
	if creationCount != numCalls {
		t.Errorf("Expected %d creations, got %d", numCalls, creationCount)
	}
	mu.Unlock()

	// Verify instances are different by comparing their IDs
	for i := range numCalls - 1 {
		if instances[i] == nil || instances[i+1] == nil {
			t.Errorf("Instance %d or %d is nil", i, i+1)

			continue
		}

		map1, ok1 := instances[i].(map[string]interface{})
		map2, ok2 := instances[i+1].(map[string]interface{})
		if !ok1 || !ok2 {
			t.Errorf("Instances %d or %d are not maps", i, i+1)

			continue
		}

		id1 := map1["id"]
		id2 := map2["id"]
		if id1 == id2 {
			t.Errorf("Instances %d and %d have the same ID %v (should be different)", i, i+1, id1)
		}
	}
}

// Helper function to check if a string contains a substring.
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
