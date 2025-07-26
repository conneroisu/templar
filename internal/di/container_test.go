package di

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/conneroisu/templar/internal/config"
)

// Test interfaces and implementations
type TestService interface {
	GetName() string
}

type TestImplementation struct {
	name string
}

func (t *TestImplementation) GetName() string {
	return t.name
}

type TestDependentService struct {
	dependency TestService
}

func (t *TestDependentService) GetDependency() TestService {
	return t.dependency
}

func TestServiceContainer_BasicRegistration(t *testing.T) {
	container := NewServiceContainer(&config.Config{})

	// Test service registration
	container.Register("test", func(resolver DependencyResolver) (interface{}, error) {
		return &TestImplementation{name: "test-service"}, nil
	})

	// Test service retrieval
	service, err := container.Get("test")
	if err != nil {
		t.Fatalf("Failed to get service: %v", err)
	}

	testService, ok := service.(*TestImplementation)
	if !ok {
		t.Fatal("Service is not of expected type")
	}

	if testService.GetName() != "test-service" {
		t.Errorf("Expected name 'test-service', got '%s'", testService.GetName())
	}
}

func TestServiceContainer_SingletonBehavior(t *testing.T) {
	container := NewServiceContainer(&config.Config{})

	// Register singleton service
	container.RegisterSingleton(
		"singleton",
		func(resolver DependencyResolver) (interface{}, error) {
			return &TestImplementation{name: "singleton-service"}, nil
		},
	)

	// Get service twice
	service1, err := container.Get("singleton")
	if err != nil {
		t.Fatalf("Failed to get service first time: %v", err)
	}

	service2, err := container.Get("singleton")
	if err != nil {
		t.Fatalf("Failed to get service second time: %v", err)
	}

	// Should be the same instance
	if service1 != service2 {
		t.Error("Singleton service should return the same instance")
	}
}

func TestServiceContainer_NonSingletonBehavior(t *testing.T) {
	container := NewServiceContainer(&config.Config{})

	// Register non-singleton service
	container.Register("transient", func(resolver DependencyResolver) (interface{}, error) {
		return &TestImplementation{name: "transient-service"}, nil
	})

	// Get service twice
	service1, err := container.Get("transient")
	if err != nil {
		t.Fatalf("Failed to get service first time: %v", err)
	}

	service2, err := container.Get("transient")
	if err != nil {
		t.Fatalf("Failed to get service second time: %v", err)
	}

	// Should be different instances
	if service1 == service2 {
		t.Error("Transient service should return different instances")
	}
}

func TestServiceContainer_DependencyInjection(t *testing.T) {
	container := NewServiceContainer(&config.Config{})

	// Register dependency
	container.RegisterSingleton(
		"dependency",
		func(resolver DependencyResolver) (interface{}, error) {
			return &TestImplementation{name: "dependency-service"}, nil
		},
	)

	// Register service with dependency
	container.Register("dependent", func(resolver DependencyResolver) (interface{}, error) {
		dep, err := resolver.Get("dependency")
		if err != nil {
			return nil, err
		}
		return &TestDependentService{dependency: dep.(TestService)}, nil
	}).DependsOn("dependency")

	// Get dependent service
	service, err := container.Get("dependent")
	if err != nil {
		t.Fatalf("Failed to get dependent service: %v", err)
	}

	dependentService, ok := service.(*TestDependentService)
	if !ok {
		t.Fatal("Service is not of expected type")
	}

	if dependentService.GetDependency().GetName() != "dependency-service" {
		t.Error("Dependency injection failed")
	}
}

func TestServiceContainer_RegisterInstance(t *testing.T) {
	container := NewServiceContainer(&config.Config{})

	// Create and register instance
	instance := &TestImplementation{name: "instance-service"}
	container.RegisterInstance("instance", instance)

	// Get service
	service, err := container.Get("instance")
	if err != nil {
		t.Fatalf("Failed to get instance: %v", err)
	}

	// Should be the same instance
	if service != instance {
		t.Error("RegisterInstance should return the exact same instance")
	}
}

func TestServiceContainer_HasService(t *testing.T) {
	container := NewServiceContainer(&config.Config{})

	// Initially should not have service
	if container.Has("test") {
		t.Error("Container should not have unregistered service")
	}

	// Register service
	container.Register("test", func(resolver DependencyResolver) (interface{}, error) {
		return &TestImplementation{}, nil
	})

	// Should now have service
	if !container.Has("test") {
		t.Error("Container should have registered service")
	}
}

func TestServiceContainer_MustGet(t *testing.T) {
	container := NewServiceContainer(&config.Config{})

	// Register service
	container.Register("test", func(resolver DependencyResolver) (interface{}, error) {
		return &TestImplementation{name: "test"}, nil
	})

	// MustGet should work
	service := container.MustGet("test")
	if service == nil {
		t.Error("MustGet should return service")
	}

	// MustGet with non-existent service should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustGet should panic for non-existent service")
		}
	}()
	container.MustGet("non-existent")
}

func TestServiceContainer_GetByType(t *testing.T) {
	container := NewServiceContainer(&config.Config{})

	// Register service
	container.Register("test", func(resolver DependencyResolver) (interface{}, error) {
		return &TestImplementation{name: "test"}, nil
	}).WithType(reflect.TypeOf((*TestImplementation)(nil)))

	// Get by type
	service, err := container.GetByType(reflect.TypeOf((*TestImplementation)(nil)))
	if err != nil {
		t.Fatalf("Failed to get service by type: %v", err)
	}

	testService, ok := service.(*TestImplementation)
	if !ok {
		t.Error("Service is not of expected type")
	}

	if testService.GetName() != "test" {
		t.Error("Got wrong service instance")
	}
}

func TestServiceContainer_GetByTag(t *testing.T) {
	container := NewServiceContainer(&config.Config{})

	// Register services with tags
	container.Register("service1", func(resolver DependencyResolver) (interface{}, error) {
		return &TestImplementation{name: "service1"}, nil
	}).WithTag("test", "group1")

	container.Register("service2", func(resolver DependencyResolver) (interface{}, error) {
		return &TestImplementation{name: "service2"}, nil
	}).WithTag("test", "group2")

	container.Register("service3", func(resolver DependencyResolver) (interface{}, error) {
		return &TestImplementation{name: "service3"}, nil
	}).WithTag("other")

	// Get services by tag
	services, err := container.GetByTag("test")
	if err != nil {
		t.Fatalf("Failed to get services by tag: %v", err)
	}

	if len(services) != 2 {
		t.Errorf("Expected 2 services with 'test' tag, got %d", len(services))
	}

	// Verify services
	names := make(map[string]bool)
	for _, service := range services {
		impl := service.(*TestImplementation)
		names[impl.GetName()] = true
	}

	if !names["service1"] || !names["service2"] {
		t.Error("Did not get expected services by tag")
	}
}

func TestServiceContainer_Initialize(t *testing.T) {
	cfg := &config.Config{
		Build: config.BuildConfig{
			Command: "templ",
		},
	}
	container := NewServiceContainer(cfg)

	// Initialize should register core services
	err := container.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize container: %v", err)
	}

	// Check that core services are registered
	coreServices := []string{"registry", "scanner", "buildPipeline", "watcher", "server"}
	for _, serviceName := range coreServices {
		if !container.Has(serviceName) {
			t.Errorf("Core service '%s' not registered", serviceName)
		}
	}

	// Test getting core services
	registry, err := container.GetRegistry()
	if err != nil {
		t.Errorf("Failed to get registry: %v", err)
	}
	if registry == nil {
		t.Error("Registry should not be nil")
	}

	scanner, err := container.GetScanner()
	if err != nil {
		t.Errorf("Failed to get scanner: %v", err)
	}
	if scanner == nil {
		t.Error("Scanner should not be nil")
	}

	buildPipeline, err := container.GetBuildPipeline()
	if err != nil {
		t.Errorf("Failed to get build pipeline: %v", err)
	}
	if buildPipeline == nil {
		t.Error("Build pipeline should not be nil")
	}
}

func TestServiceContainer_ConcurrentAccess(t *testing.T) {
	container := NewServiceContainer(&config.Config{})

	// Register singleton service
	container.RegisterSingleton(
		"concurrent",
		func(resolver DependencyResolver) (interface{}, error) {
			time.Sleep(10 * time.Millisecond) // Simulate slow creation
			return &TestImplementation{name: "concurrent-service"}, nil
		},
	)

	// Access service concurrently
	const numGoroutines = 10
	results := make(chan interface{}, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			service, err := container.Get("concurrent")
			if err != nil {
				errors <- err
				return
			}
			results <- service
		}()
	}

	// Collect results
	var services []interface{}
	for i := 0; i < numGoroutines; i++ {
		select {
		case service := <-results:
			services = append(services, service)
		case err := <-errors:
			t.Fatalf("Concurrent access failed: %v", err)
		case <-time.After(time.Second):
			t.Fatal("Concurrent access timed out")
		}
	}

	// All should be the same instance (singleton)
	first := services[0]
	for i, service := range services {
		if service != first {
			t.Errorf("Service %d is different instance (singleton violation)", i)
		}
	}
}

func TestServiceContainer_Shutdown(t *testing.T) {
	container := NewServiceContainer(&config.Config{})

	// Initialize with core services
	err := container.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err = container.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	// Services should still be registered but instances cleared
	if !container.Has("registry") {
		t.Error("Service registration should persist after shutdown")
	}
}

func TestServiceContainer_ListServices(t *testing.T) {
	container := NewServiceContainer(&config.Config{})

	// Register some services
	container.Register("service1", func(resolver DependencyResolver) (interface{}, error) {
		return &TestImplementation{}, nil
	})
	container.Register("service2", func(resolver DependencyResolver) (interface{}, error) {
		return &TestImplementation{}, nil
	})

	services := container.ListServices()
	if len(services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(services))
	}

	// Check service names
	serviceMap := make(map[string]bool)
	for _, name := range services {
		serviceMap[name] = true
	}

	if !serviceMap["service1"] || !serviceMap["service2"] {
		t.Error("Missing expected services in list")
	}
}

func TestServiceContainer_GetServiceDefinition(t *testing.T) {
	container := NewServiceContainer(&config.Config{})

	// Register service with metadata
	container.Register("test", func(resolver DependencyResolver) (interface{}, error) {
		return &TestImplementation{}, nil
	}).WithTag("test-tag").DependsOn("dependency")

	definition, exists := container.GetServiceDefinition("test")
	if !exists {
		t.Fatal("Service definition should exist")
	}

	if definition.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", definition.Name)
	}

	if len(definition.Tags) != 1 || definition.Tags[0] != "test-tag" {
		t.Error("Tag not preserved in definition")
	}

	if len(definition.Dependencies) != 1 || definition.Dependencies[0] != "dependency" {
		t.Error("Dependency not preserved in definition")
	}
}

// Benchmarks

func BenchmarkServiceContainer_GetSingleton(b *testing.B) {
	container := NewServiceContainer(&config.Config{})
	container.RegisterSingleton("test", func(resolver DependencyResolver) (interface{}, error) {
		return &TestImplementation{name: "test"}, nil
	})

	// Prime the singleton
	_, _ = container.Get("test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = container.Get("test")
	}
}

func BenchmarkServiceContainer_GetTransient(b *testing.B) {
	container := NewServiceContainer(&config.Config{})
	container.Register("test", func(resolver DependencyResolver) (interface{}, error) {
		return &TestImplementation{name: "test"}, nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = container.Get("test")
	}
}

func BenchmarkServiceContainer_ConcurrentSingleton(b *testing.B) {
	container := NewServiceContainer(&config.Config{})
	container.RegisterSingleton("test", func(resolver DependencyResolver) (interface{}, error) {
		return &TestImplementation{name: "test"}, nil
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = container.Get("test")
		}
	})
}
