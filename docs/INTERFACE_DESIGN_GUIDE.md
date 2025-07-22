# Templar Interface Design Guide

This document establishes standardized conventions for interface design across the Templar codebase to ensure consistency, maintainability, and testability.

## Interface Naming Conventions

### 1. Standard Naming Patterns

**Primary Pattern: Action + Subject + er/or**
```go
// ✅ Good Examples
type ComponentScanner interface    // Scans components
type FileWatcher interface        // Watches files  
type ErrorCollector interface     // Collects errors
type HashProvider interface       // Provides hashes
type ResultProcessor interface    // Processes results
```

**Service Pattern: Domain + Manager/Service**
```go
// ✅ Good Examples
type ComponentRegistry interface  // Manages component registry
type ConfigManager interface     // Manages configuration
type HealthChecker interface     // Checks health
```

**Avoid Generic Names:**
```go
// ❌ Bad Examples
type Handler interface           // Too generic
type Manager interface          // What does it manage?
type Service interface          // What service?
type Context interface          // Use more specific names

// ✅ Better Alternatives
type RequestHandler interface
type ComponentManager interface
type BuildService interface
type ScanContext interface
```

### 2. Interface Size Guidelines

**Single Responsibility Principle**
- Interfaces should have a single, well-defined responsibility
- Maximum 7±2 methods per interface (cognitive load limit)
- Large interfaces should be split using interface composition

```go
// ❌ Bad: Violates Interface Segregation Principle
type MegaService interface {
    // Component operations
    ScanComponent(path string) error
    RegisterComponent(component *Component) error
    
    // File operations  
    WatchFiles(dir string) error
    ReadFile(path string) ([]byte, error)
    
    // Build operations
    CompileTemplate(template string) error
    CacheResult(key string, value interface{}) error
    
    // Server operations
    StartServer(port int) error
    HandleRequest(req *Request) *Response
    
    // Config operations
    LoadConfig(path string) error
    SaveConfig(config *Config) error
    
    // Metrics operations
    RecordMetric(name string, value float64)
    GetMetrics() MetricsSnapshot
}

// ✅ Good: Split into focused interfaces
type ComponentScanner interface {
    ScanComponent(path string) error
    RegisterComponent(component *Component) error
}

type FileWatcher interface {
    WatchFiles(dir string) error
    ReadFile(path string) ([]byte, error)
}

type TemplCompiler interface {
    CompileTemplate(template string) error
}

type BuildCache interface {
    CacheResult(key string, value interface{}) error
    GetCached(key string) (interface{}, bool)
}

type WebServer interface {
    StartServer(port int) error
    HandleRequest(req *Request) *Response
}

type ConfigManager interface {
    LoadConfig(path string) error
    SaveConfig(config *Config) error
}

type MetricsCollector interface {
    RecordMetric(name string, value float64)
    GetMetrics() MetricsSnapshot
}
```

### 3. Interface Composition Patterns

**Use Interface Embedding for Extension**
```go
// ✅ Base interface
type Plugin interface {
    Name() string
    Version() string
    Initialize(ctx context.Context, config PluginConfig) error
    Shutdown(ctx context.Context) error
}

// ✅ Extended interfaces through composition
type ComponentPlugin interface {
    Plugin  // Embedded interface
    HandleComponent(ctx context.Context, component *types.ComponentInfo) (*types.ComponentInfo, error)
}

type BuildPlugin interface {
    Plugin  // Embedded interface
    PreBuild(ctx context.Context, components []*types.ComponentInfo) error
    PostBuild(ctx context.Context, results []BuildResult) error
}

type ServerPlugin interface {
    Plugin  // Embedded interface
    RegisterRoutes(router Router) error
    HandleRequest(ctx context.Context, req *Request) (*Response, error)
}
```

## Interface Placement Strategy

### 1. Central Interfaces Package

**Core application interfaces belong in `internal/interfaces/`:**
```go
// internal/interfaces/core.go
type ComponentRegistry interface { ... }
type FileWatcher interface { ... }
type BuildPipeline interface { ... }
type ComponentScanner interface { ... }

// internal/interfaces/plugins.go  
type Plugin interface { ... }
type ComponentPlugin interface { ... }
type BuildPlugin interface { ... }

// internal/interfaces/services.go
type ConfigManager interface { ... }
type MetricsCollector interface { ... }
type HealthChecker interface { ... }
```

### 2. Domain-Specific Interfaces

**Domain interfaces stay in their respective packages when:**
- Used only within that domain
- Tightly coupled to domain types
- Internal implementation details

```go
// internal/server/interfaces.go
type RequestHandler interface { ... }    // Server-specific
type Middleware interface { ... }        // Server-specific
type SessionManager interface { ... }    // Server-specific

// internal/monitoring/interfaces.go  
type AlertChannel interface { ... }      // Monitoring-specific
type HealthProbe interface { ... }       // Monitoring-specific
```

### 3. Cross-Package Interfaces

**Use central package for interfaces that:**
- Are used by multiple packages
- Define architectural boundaries
- Represent core abstractions
- Enable dependency inversion

## Method Design Conventions

### 1. Method Naming

**Use Clear, Action-Oriented Names**
```go
// ✅ Good Examples
type ComponentRegistry interface {
    Register(component *types.ComponentInfo)           // Clear action
    Get(name string) (*types.ComponentInfo, bool)      // Standard getter
    GetAll() []*types.ComponentInfo                     // Bulk operation
    Remove(name string) bool                            // Clear action
    Count() int                                         // Query operation
    Watch() <-chan types.ComponentEvent                 // Observable pattern
}
```

**Avoid Ambiguous Names**
```go
// ❌ Bad Examples
type BadInterface interface {
    Process(data interface{}) interface{}     // What processing? What data?
    Handle(input string) error               // Handle how?
    Do(params map[string]interface{}) error  // Do what?
}

// ✅ Better Alternatives  
type ComponentProcessor interface {
    ProcessComponent(component *types.ComponentInfo) (*types.ComponentInfo, error)
    HandleComponentEvent(event ComponentEvent) error
    ExecuteBuildPipeline(params BuildParams) (*BuildResult, error)
}
```

### 2. Parameter and Return Type Guidelines

**Use Specific Types Instead of interface{}**
```go
// ❌ Bad: Generic types
type BadCache interface {
    Get(key string) interface{}
    Set(key string, value interface{})
    GetStats() interface{}
}

// ✅ Good: Specific types
type BuildCache interface {
    Get(key string) (*BuildResult, bool)
    Set(key string, result *BuildResult)
    GetStats() CacheStats
}

// ✅ Good: Type-safe generics (Go 1.18+)
type Cache[T any] interface {
    Get(key string) (T, bool)
    Set(key string, value T)
    Delete(key string) bool
}
```

**Use Context for Cancellation and Timeouts**
```go
// ✅ Context-aware operations
type ComponentScanner interface {
    ScanDirectory(ctx context.Context, dir string) error
    ScanFile(ctx context.Context, path string) error
    ScanWithWorkers(ctx context.Context, paths []string, workers int) error
}
```

**Use Functional Options for Complex Configuration**
```go
// ✅ Functional options pattern
type BuildOptions struct {
    Workers    int
    Timeout    time.Duration
    Cache      bool
    Verbose    bool
}

type BuildOption func(*BuildOptions)

func WithWorkers(count int) BuildOption {
    return func(opts *BuildOptions) { opts.Workers = count }
}

func WithTimeout(timeout time.Duration) BuildOption {
    return func(opts *BuildOptions) { opts.Timeout = timeout }
}

type BuildPipeline interface {
    Build(ctx context.Context, components []*types.ComponentInfo, opts ...BuildOption) error
}
```

### 3. Error Handling Patterns

**Consistent Error Return Patterns**
```go
// ✅ Standard patterns
type FileProcessor interface {
    // Single operation - return error
    ProcessFile(path string) error
    
    // Operation with result - return value and error  
    ReadFile(path string) ([]byte, error)
    
    // Query operation - return value and boolean
    GetFile(path string) (*FileInfo, bool)
    
    // Bulk operation - return results and error
    ProcessFiles(paths []string) ([]ProcessResult, error)
}
```

## Observable and Lifecycle Patterns

### 1. Observable Interfaces

**Use Channels for Event Streams**
```go
// ✅ Observable pattern
type ComponentRegistry interface {
    Register(component *types.ComponentInfo)
    Watch() <-chan ComponentEvent              // Receive-only channel
    WatchWithFilter(filter ComponentFilter) <-chan ComponentEvent
}

type FileWatcher interface {
    Watch(path string) error  
    Events() <-chan FileEvent                  // Receive-only channel
    Errors() <-chan error                      // Separate error channel
}
```

### 2. Lifecycle Management

**Standard Lifecycle Interfaces**
```go
// ✅ Lifecycle management
type Startable interface {
    Start(ctx context.Context) error
}

type Stoppable interface {
    Stop() error
}

type Graceful interface {
    Startable
    Stoppable
    Shutdown(ctx context.Context) error        // Graceful shutdown
}

// ✅ Resource management
type Closable interface {
    Close() error
}

type HealthCheckable interface {
    HealthCheck(ctx context.Context) error
}
```

## Dependency Injection Patterns

### 1. Constructor Injection

**Accept Interfaces, Return Concrete Types**
```go
// ✅ Constructor accepts interfaces
func NewBuildPipeline(
    registry interfaces.ComponentRegistry,
    scanner interfaces.ComponentScanner,
    cache interfaces.BuildCache,
    metrics interfaces.MetricsCollector,
) *BuildPipeline {
    return &BuildPipeline{
        registry: registry,
        scanner:  scanner, 
        cache:    cache,
        metrics:  metrics,
    }
}

// ✅ Factory functions return interfaces
func NewComponentRegistry() interfaces.ComponentRegistry {
    return &componentRegistry{
        components: make(map[string]*types.ComponentInfo),
        events:     make(chan ComponentEvent, 100),
    }
}
```

### 2. Interface Validation

**Runtime Interface Compliance Checking**
```go
// ✅ Compile-time interface validation
var (
    _ interfaces.ComponentRegistry = (*componentRegistry)(nil)
    _ interfaces.ComponentScanner  = (*ComponentScanner)(nil)
    _ interfaces.BuildPipeline     = (*BuildPipeline)(nil)
    _ interfaces.FileWatcher       = (*FileWatcher)(nil)
)

// ✅ Runtime validation in tests
func TestInterfaceCompliance(t *testing.T) {
    registry := NewComponentRegistry()
    scanner := NewComponentScanner(registry)
    pipeline := NewBuildPipeline(registry, scanner, nil, nil)
    
    // Validate interface compliance
    assert.Implements(t, (*interfaces.ComponentRegistry)(nil), registry)
    assert.Implements(t, (*interfaces.ComponentScanner)(nil), scanner)
    assert.Implements(t, (*interfaces.BuildPipeline)(nil), pipeline)
}
```

## Documentation Standards

### 1. Interface Documentation

**Comprehensive Interface Documentation**
```go
// ComponentRegistry manages the registry of discovered templ components.
//
// The registry provides thread-safe storage and retrieval of component metadata,
// supports event-driven notifications for component changes, and enables
// dependency analysis between components.
//
// Usage:
//   registry := NewComponentRegistry()
//   registry.Register(component)
//   
//   // Watch for changes
//   for event := range registry.Watch() {
//       log.Printf("Component %s was %s", event.Component.Name, event.Type)
//   }
//
// Thread Safety:
//   All methods are thread-safe and can be called concurrently.
//
// Performance:
//   - Register: O(1) average case
//   - Get: O(1) lookup time  
//   - GetAll: O(n) where n is number of components
//   - Event delivery: Non-blocking with 100-event buffer
type ComponentRegistry interface {
    // Register adds or updates a component in the registry.
    // If a component with the same name exists, it will be replaced.
    // Triggers a ComponentAdded or ComponentUpdated event.
    Register(component *types.ComponentInfo)
    
    // Get retrieves a component by name.
    // Returns the component and true if found, nil and false otherwise.
    Get(name string) (*types.ComponentInfo, bool)
    
    // GetAll returns all registered components.
    // The returned slice is a copy and safe to modify.
    GetAll() []*types.ComponentInfo
    
    // Remove removes a component by name.
    // Returns true if the component was found and removed.
    // Triggers a ComponentRemoved event if successful.
    Remove(name string) bool
    
    // Count returns the total number of registered components.
    Count() int
    
    // Watch returns a channel that receives component events.
    // The channel is buffered with 100 events. If the buffer fills,
    // older events may be dropped.
    // 
    // The returned channel will be closed when the registry is shut down.
    Watch() <-chan ComponentEvent
    
    // DetectCircularDependencies analyzes component dependencies and
    // returns any circular dependency chains found.
    // Returns empty slice if no cycles detected.
    DetectCircularDependencies() [][]string
}
```

### 2. Method Documentation

**Document Behavior, Edge Cases, and Performance**
```go
type FileWatcher interface {
    // Watch starts monitoring the specified path for file system changes.
    // 
    // The path can be a file or directory. For directories, monitoring
    // is recursive and includes all subdirectories.
    //
    // Parameters:
    //   path: File system path to monitor (must exist)
    //
    // Returns:
    //   error: Non-nil if path doesn't exist or monitoring fails
    //
    // Behavior:
    //   - Duplicate calls to Watch with same path are ignored
    //   - Events are delivered through Events() channel
    //   - Errors are delivered through Errors() channel  
    //   - Monitoring continues until Stop() is called
    //
    // Performance:
    //   - Uses efficient OS-specific file system APIs (inotify/kqueue)
    //   - Memory usage scales with number of watched files
    //   - Event delivery is non-blocking with 1000-event buffer
    //
    // Thread Safety:
    //   Safe to call concurrently with other methods.
    //
    // Example:
    //   watcher := NewFileWatcher()
    //   if err := watcher.Watch("./components"); err != nil {
    //       log.Fatal(err)
    //   }
    //   
    //   go func() {
    //       for event := range watcher.Events() {
    //           log.Printf("File %s was %s", event.Path, event.Type)
    //       }
    //   }()
    Watch(path string) error
}
```

## Anti-Patterns to Avoid

### 1. Service Locator Pattern

```go
// ❌ Bad: Service locator anti-pattern
type ServiceLocator interface {
    GetService(name string) interface{}
    RegisterService(name string, service interface{})
}

// ❌ Bad: Global service access
var GlobalServices ServiceLocator

func SomeFunction() {
    db := GlobalServices.GetService("database").(Database)
    cache := GlobalServices.GetService("cache").(Cache)
    // ... use services
}

// ✅ Good: Dependency injection
type UserService struct {
    db    Database
    cache Cache
}

func NewUserService(db Database, cache Cache) *UserService {
    return &UserService{db: db, cache: cache}
}

func (s *UserService) GetUser(id string) (*User, error) {
    // Use injected dependencies
    return s.db.GetUser(id)
}
```

### 2. Fat Interfaces

```go
// ❌ Bad: Fat interface violating ISP
type UserManager interface {
    // User CRUD
    CreateUser(user *User) error
    GetUser(id string) (*User, error) 
    UpdateUser(user *User) error
    DeleteUser(id string) error
    
    // Authentication  
    Login(username, password string) (*Session, error)
    Logout(sessionID string) error
    ValidateSession(sessionID string) bool
    
    // Authorization
    CheckPermission(userID, resource, action string) bool
    GrantRole(userID, role string) error
    RevokeRole(userID, role string) error
    
    // Notifications
    SendEmail(userID, subject, body string) error
    SendSMS(userID, message string) error
    
    // Audit logging
    LogUserAction(userID, action string) error
    GetUserAuditLog(userID string) ([]AuditEntry, error)
}

// ✅ Good: Split into focused interfaces
type UserRepository interface {
    CreateUser(user *User) error
    GetUser(id string) (*User, error)
    UpdateUser(user *User) error  
    DeleteUser(id string) error
}

type AuthenticationService interface {
    Login(username, password string) (*Session, error)
    Logout(sessionID string) error
    ValidateSession(sessionID string) bool
}

type AuthorizationService interface {
    CheckPermission(userID, resource, action string) bool
    GrantRole(userID, role string) error
    RevokeRole(userID, role string) error
}

type NotificationService interface {
    SendEmail(userID, subject, body string) error
    SendSMS(userID, message string) error
}

type AuditLogger interface {
    LogUserAction(userID, action string) error
    GetUserAuditLog(userID string) ([]AuditEntry, error)
}
```

### 3. Leaky Abstractions

```go
// ❌ Bad: Leaky abstraction exposing implementation details
type DatabaseConnection interface {
    GetRawConnection() *sql.DB        // Leaks SQL implementation
    GetConnectionPool() *ConnectionPool // Leaks pooling implementation
    Query(sql string, args ...interface{}) (*sql.Rows, error) // Leaks SQL
}

// ✅ Good: Clean abstraction hiding implementation
type UserRepository interface {
    FindByID(ctx context.Context, id string) (*User, error)
    FindByEmail(ctx context.Context, email string) (*User, error)
    Save(ctx context.Context, user *User) error
    Delete(ctx context.Context, id string) error
    List(ctx context.Context, filter UserFilter) ([]*User, error)
}
```

## Interface Evolution Guidelines

### 1. Backward Compatibility

**Add Methods to New Interfaces**
```go
// ✅ Original interface
type ComponentScanner interface {
    ScanFile(path string) error
    ScanDirectory(dir string) error
}

// ✅ Extended interface (new interface)
type ComponentScannerV2 interface {
    ComponentScanner  // Embed original
    ScanWithContext(ctx context.Context, path string) error
    ScanParallel(paths []string, workers int) error
}

// ✅ Use type assertion for optional features
func useScanner(scanner ComponentScanner) {
    // Use base functionality
    scanner.ScanFile("component.templ")
    
    // Use extended functionality if available
    if v2Scanner, ok := scanner.(ComponentScannerV2); ok {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        v2Scanner.ScanWithContext(ctx, "component.templ")
    }
}
```

### 2. Deprecation Strategy

**Gradual Interface Migration**
```go
// ✅ Mark deprecated methods
type LegacyInterface interface {
    // Deprecated: Use ProcessWithContext instead
    Process(data string) error
    
    // ProcessWithContext processes data with context for cancellation
    ProcessWithContext(ctx context.Context, data string) error
}

// ✅ Create new interface without deprecated methods
type ModernInterface interface {
    ProcessWithContext(ctx context.Context, data string) error
}
```

This interface design guide provides the foundation for consistent, maintainable, and testable interface design across the Templar codebase. Following these conventions will improve code quality, reduce coupling, and enhance developer productivity.