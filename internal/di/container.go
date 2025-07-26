package di

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"sync"
	"time"

	"github.com/conneroisu/templar/internal/build"
	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/monitoring"
	"github.com/conneroisu/templar/internal/registry"
	"github.com/conneroisu/templar/internal/scanner"
	"github.com/conneroisu/templar/internal/server"
	"github.com/conneroisu/templar/internal/watcher"
)

// dependencyResolver is a wrapper around ServiceContainer that prevents deadlocks
type dependencyResolver struct {
	container *ServiceContainer
	resolving map[string]bool
}

// Get retrieves a service using the safe resolver
func (dr *dependencyResolver) Get(name string) (interface{}, error) {
	return dr.container.getWithResolver(name, dr.resolving)
}

// GetByType retrieves a service by type using the safe resolver
func (dr *dependencyResolver) GetByType(serviceType reflect.Type) (interface{}, error) {
	dr.container.mu.RLock()
	var serviceName string
	found := false

	for _, definition := range dr.container.services {
		if definition.Type == serviceType {
			serviceName = definition.Name
			found = true
			break
		}
	}
	dr.container.mu.RUnlock()

	if found {
		return dr.Get(serviceName)
	}

	return nil, fmt.Errorf("no service found for type %s", serviceType.String())
}

// GetByTag retrieves all services with a specific tag using the safe resolver
func (dr *dependencyResolver) GetByTag(tag string) ([]interface{}, error) {
	dr.container.mu.RLock()
	var serviceNames []string

	for _, definition := range dr.container.services {
		for _, defTag := range definition.Tags {
			if defTag == tag {
				serviceNames = append(serviceNames, definition.Name)
				break
			}
		}
	}
	dr.container.mu.RUnlock()

	var services []interface{}
	for _, serviceName := range serviceNames {
		service, err := dr.Get(serviceName)
		if err != nil {
			return nil, err
		}
		services = append(services, service)
	}

	return services, nil
}

// MustGet retrieves a service and panics if not found
func (dr *dependencyResolver) MustGet(name string) interface{} {
	instance, err := dr.Get(name)
	if err != nil {
		panic(fmt.Sprintf("failed to get service '%s': %v", name, err))
	}
	return instance
}

// ServiceContainer manages dependency injection for the application
type ServiceContainer struct {
	services    map[string]ServiceDefinition
	instances   map[string]interface{}
	singletons  map[string]interface{}
	factories   map[string]FactoryFunc
	creating    map[string]*sync.WaitGroup // Track services being created
	mu          sync.RWMutex
	config      *config.Config
	initialized bool
}

// ServiceDefinition defines how a service should be created and managed
type ServiceDefinition struct {
	Name         string
	Type         reflect.Type
	Factory      FactoryFunc
	Singleton    bool
	Dependencies []string
	Tags         []string
}

// FactoryFunc creates a service instance using the dependency resolver
type FactoryFunc func(resolver DependencyResolver) (interface{}, error)

// TODO: Update ServiceContainer to fully implement interfaces.ServiceContainer interface
// var _ interfaces.ServiceContainer = (*ServiceContainer)(nil)

// DependencyResolver provides safe dependency resolution that prevents circular dependencies
type DependencyResolver interface {
	Get(name string) (interface{}, error)
	GetByType(serviceType reflect.Type) (interface{}, error)
	GetByTag(tag string) ([]interface{}, error)
	MustGet(name string) interface{}
}

// ServiceBuilder helps build service definitions
type ServiceBuilder struct {
	definition ServiceDefinition
	container  *ServiceContainer
}

// NewServiceContainer creates a new dependency injection container
func NewServiceContainer(cfg *config.Config) *ServiceContainer {
	return &ServiceContainer{
		services:   make(map[string]ServiceDefinition),
		instances:  make(map[string]interface{}),
		singletons: make(map[string]interface{}),
		factories:  make(map[string]FactoryFunc),
		creating:   make(map[string]*sync.WaitGroup),
		config:     cfg,
	}
}

// Register registers a service with the container
func (c *ServiceContainer) Register(name string, factory FactoryFunc) *ServiceBuilder {
	c.mu.Lock()
	defer c.mu.Unlock()

	builder := &ServiceBuilder{
		definition: ServiceDefinition{
			Name:         name,
			Factory:      factory,
			Singleton:    false,
			Dependencies: make([]string, 0),
			Tags:         make([]string, 0),
		},
		container: c,
	}

	c.services[name] = builder.definition
	c.factories[name] = factory

	return builder
}

// RegisterSingleton registers a singleton service
func (c *ServiceContainer) RegisterSingleton(name string, factory FactoryFunc) *ServiceBuilder {
	builder := c.Register(name, factory)
	builder.definition.Singleton = true
	c.services[name] = builder.definition
	return builder
}

// RegisterInstance registers an existing instance as a singleton
func (c *ServiceContainer) RegisterInstance(name string, instance interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.singletons[name] = instance
	c.services[name] = ServiceDefinition{
		Name:      name,
		Type:      reflect.TypeOf(instance),
		Singleton: true,
	}
}

// Get retrieves a service from the container
func (c *ServiceContainer) Get(name string) (interface{}, error) {
	return c.getWithResolver(name, make(map[string]bool))
}

// getWithResolver retrieves a service with circular dependency detection
func (c *ServiceContainer) getWithResolver(
	name string,
	resolving map[string]bool,
) (interface{}, error) {
	// Check for circular dependencies
	if resolving[name] {
		return nil, fmt.Errorf("circular dependency detected for service '%s'", name)
	}

	// Check if service is registered
	c.mu.RLock()
	definition, exists := c.services[name]
	factory := c.factories[name]
	c.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("service '%s' not registered", name)
	}

	// For singletons, use creation coordination to avoid race conditions
	if definition.Singleton {
		// First check - read lock
		c.mu.RLock()
		if instance, exists := c.singletons[name]; exists {
			c.mu.RUnlock()
			return instance, nil
		}

		// Check if another goroutine is creating this singleton
		if wg, creating := c.creating[name]; creating {
			c.mu.RUnlock()
			// Wait for the other goroutine to finish creating
			wg.Wait()
			// Now get the created instance
			c.mu.RLock()
			instance := c.singletons[name]
			c.mu.RUnlock()
			return instance, nil
		}
		c.mu.RUnlock()

		// Second check with write lock - establish creation reservation
		c.mu.Lock()
		if instance, exists := c.singletons[name]; exists {
			c.mu.Unlock()
			return instance, nil
		}

		// Check again if another goroutine is creating this singleton
		if wg, creating := c.creating[name]; creating {
			c.mu.Unlock()
			wg.Wait()
			c.mu.RLock()
			instance := c.singletons[name]
			c.mu.RUnlock()
			return instance, nil
		}

		// Reserve creation - we will create this singleton
		wg := &sync.WaitGroup{}
		wg.Add(1)
		c.creating[name] = wg

		// Mark as being resolved to prevent circular dependencies
		resolving[name] = true
		c.mu.Unlock()

		// Create the singleton instance without holding any locks
		instance, err := c.createInstanceSafely(factory, resolving)

		// Remove from resolving map after factory completes
		delete(resolving, name)

		// Store the created instance and notify waiters
		c.mu.Lock()
		if err != nil {
			// Creation failed - clean up and return error
			delete(c.creating, name)
			c.mu.Unlock()
			wg.Done()
			return nil, fmt.Errorf("failed to create singleton service '%s': %w", name, err)
		}

		c.singletons[name] = instance
		delete(c.creating, name)
		c.mu.Unlock()
		wg.Done()

		return instance, nil
	}

	// For transient services, just create a new instance
	resolving[name] = true
	instance, err := c.createInstanceSafely(factory, resolving)
	delete(resolving, name)

	if err != nil {
		return nil, fmt.Errorf("failed to create service '%s': %w", name, err)
	}

	return instance, nil
}

// createInstanceSafely creates an instance with dependency resolution
func (c *ServiceContainer) createInstanceSafely(
	factory FactoryFunc,
	resolving map[string]bool,
) (interface{}, error) {
	if factory == nil {
		return nil, fmt.Errorf("factory is nil")
	}

	// Create a resolver container that can handle circular dependencies
	resolver := &dependencyResolver{
		container: c,
		resolving: resolving,
	}

	return factory(resolver)
}

// MustGet retrieves a service and panics if not found
func (c *ServiceContainer) MustGet(name string) interface{} {
	instance, err := c.Get(name)
	if err != nil {
		panic(fmt.Sprintf("failed to get service '%s': %v", name, err))
	}
	return instance
}

// GetRequired retrieves a service and panics if not found (interface compliance)
func (c *ServiceContainer) GetRequired(name string) interface{} {
	return c.MustGet(name)
}

// Has checks if a service is registered
func (c *ServiceContainer) Has(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, exists := c.services[name]
	return exists
}

// GetByType retrieves a service by its type
func (c *ServiceContainer) GetByType(serviceType reflect.Type) (interface{}, error) {
	c.mu.RLock()
	var serviceName string
	found := false

	for _, definition := range c.services {
		if definition.Type == serviceType {
			serviceName = definition.Name
			found = true
			break
		}
	}
	c.mu.RUnlock()

	if found {
		return c.Get(serviceName)
	}

	return nil, fmt.Errorf("no service found for type %s", serviceType.String())
}

// GetByTag retrieves all services with a specific tag
func (c *ServiceContainer) GetByTag(tag string) ([]interface{}, error) {
	c.mu.RLock()
	var serviceNames []string

	for _, definition := range c.services {
		for _, defTag := range definition.Tags {
			if defTag == tag {
				serviceNames = append(serviceNames, definition.Name)
				break
			}
		}
	}
	c.mu.RUnlock()

	var services []interface{}
	for _, serviceName := range serviceNames {
		service, err := c.Get(serviceName)
		if err != nil {
			return nil, err
		}
		services = append(services, service)
	}

	return services, nil
}

// Initialize sets up all core services with their dependencies
func (c *ServiceContainer) Initialize() error {
	if c.initialized {
		return nil
	}

	// Register core services
	if err := c.registerCoreServices(); err != nil {
		return fmt.Errorf("failed to register core services: %w", err)
	}

	c.initialized = true
	return nil
}

// registerCoreServices registers all the core application services
func (c *ServiceContainer) registerCoreServices() error {
	// Register ComponentRegistry
	c.RegisterSingleton("registry", func(resolver DependencyResolver) (interface{}, error) {
		return registry.NewComponentRegistry(), nil
	}).AsSingleton().WithTag("core")

	// Register ComponentScanner
	c.RegisterSingleton("scanner", func(resolver DependencyResolver) (interface{}, error) {
		reg, err := resolver.Get("registry")
		if err != nil {
			return nil, err
		}
		return scanner.NewComponentScanner(reg.(*registry.ComponentRegistry)), nil
	}).DependsOn("registry").WithTag("core")

	// Register BuildPipeline (using RefactoredBuildPipeline for interface compliance)
	c.RegisterSingleton("buildPipeline", func(resolver DependencyResolver) (interface{}, error) {
		reg, err := resolver.Get("registry")
		if err != nil {
			return nil, err
		}
		workers := 4 // Default worker count
		return build.NewRefactoredBuildPipeline(workers, reg.(*registry.ComponentRegistry)), nil
	}).DependsOn("registry").WithTag("core")

	// Register FileWatcher
	c.RegisterSingleton("watcher", func(resolver DependencyResolver) (interface{}, error) {
		return watcher.NewFileWatcher(300 * time.Millisecond)
	}).WithTag("core")

	// Register PreviewServer with dependency injection
	c.RegisterSingleton("server", func(resolver DependencyResolver) (interface{}, error) {
		// Get required dependencies
		reg, err := resolver.Get("registry")
		if err != nil {
			return nil, err
		}

		watcherService, err := resolver.Get("watcher")
		if err != nil {
			return nil, err
		}

		scannerService, err := resolver.Get("scanner")
		if err != nil {
			return nil, err
		}

		buildPipelineService, err := resolver.Get("buildPipeline")
		if err != nil {
			return nil, err
		}

		// Initialize monitoring if enabled
		var monitor *monitoring.TemplarMonitor
		if c.config.Monitoring.Enabled {
			templatorMonitor, err := monitoring.SetupTemplarMonitoring("")
			if err != nil {
				log.Printf("Warning: Failed to initialize monitoring: %v", err)
			} else {
				monitor = templatorMonitor
			}
		}

		// Use concrete types directly - they now implement interfaces natively
		return server.NewWithDependencies(
			c.config,
			reg.(*registry.ComponentRegistry),
			watcherService.(*watcher.FileWatcher),
			scannerService.(*scanner.ComponentScanner),
			buildPipelineService.(*build.RefactoredBuildPipeline),
			monitor,
		), nil
	}).DependsOn("registry", "watcher", "scanner", "buildPipeline").WithTag("core")

	return nil
}

// Shutdown gracefully shuts down all services
func (c *ServiceContainer) Shutdown(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var errors []error

	// Shutdown services in reverse dependency order
	shutdownOrder := []string{"server", "buildPipeline", "watcher", "scanner", "registry"}

	for _, serviceName := range shutdownOrder {
		if instance, exists := c.singletons[serviceName]; exists {
			if shutdownable, ok := instance.(interface{ Shutdown(context.Context) error }); ok {
				if err := shutdownable.Shutdown(ctx); err != nil {
					errors = append(
						errors,
						fmt.Errorf("failed to shutdown %s: %w", serviceName, err),
					)
				}
			}
		}
	}

	// Clear all instances
	c.singletons = make(map[string]interface{})
	c.instances = make(map[string]interface{})

	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors: %v", errors)
	}

	return nil
}

// ServiceBuilder methods for fluent interface

// AsSingleton marks the service as a singleton
func (sb *ServiceBuilder) AsSingleton() *ServiceBuilder {
	sb.definition.Singleton = true
	sb.updateContainer()
	return sb
}

// DependsOn adds dependencies to the service
func (sb *ServiceBuilder) DependsOn(dependencies ...string) *ServiceBuilder {
	sb.definition.Dependencies = append(sb.definition.Dependencies, dependencies...)
	sb.updateContainer()
	return sb
}

// WithTag adds tags to the service
func (sb *ServiceBuilder) WithTag(tags ...string) *ServiceBuilder {
	sb.definition.Tags = append(sb.definition.Tags, tags...)
	sb.updateContainer()
	return sb
}

// WithType sets the service type
func (sb *ServiceBuilder) WithType(serviceType reflect.Type) *ServiceBuilder {
	sb.definition.Type = serviceType
	sb.updateContainer()
	return sb
}

// updateContainer updates the service definition in the container
func (sb *ServiceBuilder) updateContainer() {
	sb.container.mu.Lock()
	sb.container.services[sb.definition.Name] = sb.definition
	sb.container.mu.Unlock()
}

// Convenience methods for typed service retrieval

// GetRegistry retrieves the component registry
func (c *ServiceContainer) GetRegistry() (*registry.ComponentRegistry, error) {
	service, err := c.Get("registry")
	if err != nil {
		return nil, err
	}
	return service.(*registry.ComponentRegistry), nil
}

// GetScanner retrieves the component scanner
func (c *ServiceContainer) GetScanner() (*scanner.ComponentScanner, error) {
	service, err := c.Get("scanner")
	if err != nil {
		return nil, err
	}
	return service.(*scanner.ComponentScanner), nil
}

// GetBuildPipeline retrieves the build pipeline
func (c *ServiceContainer) GetBuildPipeline() (*build.RefactoredBuildPipeline, error) {
	service, err := c.Get("buildPipeline")
	if err != nil {
		return nil, err
	}
	return service.(*build.RefactoredBuildPipeline), nil
}

// GetFileWatcher retrieves the file watcher
func (c *ServiceContainer) GetFileWatcher() (*watcher.FileWatcher, error) {
	service, err := c.Get("watcher")
	if err != nil {
		return nil, err
	}
	return service.(*watcher.FileWatcher), nil
}

// GetServer retrieves the preview server
func (c *ServiceContainer) GetServer() (*server.PreviewServer, error) {
	service, err := c.Get("server")
	if err != nil {
		return nil, err
	}
	return service.(*server.PreviewServer), nil
}

// ListServices returns a list of all registered service names
func (c *ServiceContainer) ListServices() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	services := make([]string, 0, len(c.services))
	for name := range c.services {
		services = append(services, name)
	}
	return services
}

// GetServiceDefinition returns the definition for a service
func (c *ServiceContainer) GetServiceDefinition(name string) (ServiceDefinition, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	definition, exists := c.services[name]
	return definition, exists
}
