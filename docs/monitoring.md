# Templar Monitoring System

The Templar CLI includes a comprehensive monitoring system designed for production use, providing observability, health checks, metrics collection, and structured logging.

## Overview

The monitoring system consists of four main components:

- **Structured Logging**: Advanced logging with context, sanitization, and multiple outputs
- **Metrics Collection**: Counter, gauge, and histogram metrics with automatic collection
- **Health Monitoring**: Component health checks with HTTP endpoints
- **Integration Layer**: Seamless integration with Templar's core functionality

## Quick Start

### Basic Setup

```go
package main

import (
    "context"
    "log"
    
    "github.com/conneroisu/templar/internal/monitoring"
)

func main() {
    // Setup monitoring with default configuration
    monitor, err := monitoring.SetupMonitoring(monitoring.MonitoringConfig{
        EnableHTTPMiddleware: true,
        EnableHealthChecks:   true,
        EnableMetrics:        true,
        LogLevel:             "info",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer monitor.Stop()
    
    // Start monitoring
    if err := monitor.Start(); err != nil {
        log.Fatal(err)
    }
    
    // Your application code here
}
```

### Templar-Specific Setup

```go
// Use Templar-specific monitoring with enhanced integrations
monitor, err := monitoring.SetupTemplarMonitoring("./config/monitoring.yml")
if err != nil {
    log.Fatal(err)
}
defer monitor.GracefulShutdown(context.Background())
```

## Configuration

### YAML Configuration

Create a `.templar-monitoring.yml` file:

```yaml
# Basic monitoring configuration
metrics_enabled: true
metrics_output_path: "./logs/metrics.json"
metrics_prefix: "templar"
metrics_interval: 30s

health_enabled: true
health_check_interval: 30s
health_check_timeout: 10s

# HTTP server for monitoring endpoints
http_enabled: true
http_addr: "localhost"
http_port: 8081

# Logging configuration
log_level: "info"
log_format: "json"
log_output_path: "./logs/templar.log"
structured_logging: true

# Alerting (optional)
alerting_enabled: false
alert_thresholds:
  error_rate: 0.1           # 10%
  response_time: 5s
  memory_usage: 1073741824  # 1GB
  goroutine_count: 1000
  unhealthy_components: 1
alert_cooldown: 5m

# Templar-specific settings
component_paths:
  - "./components"
  - "./views"
  - "./layouts"
cache_directory: ".templar/cache"
build_command: "templ generate"
watch_patterns:
  - "**/*.templ"
  - "**/*.go"
preview_port: 8080
websocket_enabled: true
```

### Environment Variables

Override configuration with environment variables:

```bash
export TEMPLAR_METRICS_ENABLED=true
export TEMPLAR_LOG_LEVEL=debug
export TEMPLAR_HTTP_PORT=9090
export TEMPLAR_METRICS_OUTPUT_PATH=/var/log/templar/metrics.json
```

## Monitoring Endpoints

When HTTP monitoring is enabled, the following endpoints are available:

### Health Endpoints

- **`GET /health`** - Comprehensive health status with detailed checks
- **`GET /health/live`** - Simple liveness probe (returns 200 OK)
- **`GET /health/ready`** - Readiness probe (checks critical components)

Example health response:
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "uptime": "2h30m15s",
  "checks": {
    "filesystem": {
      "name": "filesystem",
      "status": "healthy",
      "message": "Filesystem is accessible",
      "last_checked": "2024-01-15T10:30:00Z",
      "duration": "2ms",
      "critical": true
    },
    "component_registry": {
      "name": "component_registry",
      "status": "healthy",
      "message": "Component registry is accessible",
      "critical": true
    }
  },
  "summary": {
    "total": 5,
    "healthy": 5,
    "unhealthy": 0,
    "degraded": 0,
    "unknown": 0,
    "critical": 3
  },
  "system_info": {
    "hostname": "templar-dev",
    "platform": "linux/amd64",
    "go_version": "go1.24",
    "pid": 12345,
    "start_time": "2024-01-15T08:00:00Z"
  }
}
```

### Metrics Endpoints

- **`GET /metrics`** - Application metrics in JSON format
- **`GET /info`** - System and application information

## Usage Patterns

### Operation Tracking

Track operations with automatic timing and error handling:

```go
// Basic operation tracking
err := monitoring.TrackOperation(ctx, "scanner", "scan_components", func(ctx context.Context) error {
    // Your operation logic here
    return scanComponents()
})

// Templar-specific operation tracking
err := monitor.TrackScanOperation(ctx, "discover_components", func(ctx context.Context) error {
    return discoverComponents()
})
```

### Metrics Recording

```go
// Component operations
monitor.RecordComponentScanned("button", "Button")
monitor.RecordComponentBuilt("Button", true, 150*time.Millisecond)

// File watching
monitor.RecordFileWatchEvent("created", "./components/new-button.templ")

// WebSocket events
monitor.RecordWebSocketEvent("client_connected", activeConnections)

// Cache operations
monitor.RecordCacheEvent("get", true, "component:Button")
```

### HTTP Middleware

Add monitoring to your HTTP server:

```go
import "net/http"

// Basic middleware
middleware := monitoring.GetMiddleware()
handler := middleware(yourHandler)

// Templar-specific middleware with enhanced tracking
templatorMiddleware := monitor.CreateTemplarMiddleware()
handler := templatorMiddleware(yourHandler)

server := &http.Server{
    Addr:    ":8080",
    Handler: handler,
}
```

### Structured Logging

```go
// Basic logging with metrics
monitoring.LogInfo(ctx, "scanner", "scan_complete", "Component scan completed",
    "components_found", 25,
    "duration", "2.5s")

// Error logging with automatic error metrics
monitoring.LogError(ctx, "build", "compile_component", err, "Failed to compile component",
    "component", "Button",
    "file", "./components/button.templ")

// Component-specific error logging
monitoring.LogComponentError(ctx, "renderer", "render_template", err, map[string]interface{}{
    "template": "button.templ",
    "props":    buttonProps,
})
```

### Batch Operations

Track batch processing with progress reporting:

```go
// Create batch tracker
batchTracker := monitoring.NewBatchTracker(monitor, logger, "build_system", len(components))

for _, component := range components {
    err := batchTracker.TrackItem(ctx, component.Name, func() error {
        return buildComponent(component)
    })
    // Handle individual errors as needed
}

// Complete batch and log summary
batchTracker.Complete(ctx)
```

## Health Checks

### Built-in Health Checks

The system includes several built-in health checks:

- **Filesystem**: Verifies read/write access to working directory
- **Memory**: Monitors memory usage and GC performance  
- **Goroutines**: Detects potential goroutine leaks
- **Component Registry**: Checks accessibility of component directories
- **Templ Binary**: Verifies templ binary availability
- **Cache Directory**: Ensures cache directory is accessible

### Custom Health Checks

Create custom health checks for your components:

```go
// Simple health check
customCheck := monitoring.NewHealthCheckFunc("my_service", false, func(ctx context.Context) monitoring.HealthCheck {
    if serviceHealthy() {
        return monitoring.HealthCheck{
            Name:    "my_service",
            Status:  monitoring.HealthStatusHealthy,
            Message: "Service is running normally",
        }
    }
    return monitoring.HealthCheck{
        Name:    "my_service", 
        Status:  monitoring.HealthStatusUnhealthy,
        Message: "Service is not responding",
    }
})

monitor.RegisterHealthCheck(customCheck)

// Component-specific health checks
componentCheck := monitoring.ComponentHealthChecker("button_renderer", func() error {
    return validateButtonRenderer()
})

buildCheck := monitoring.BuildPipelineHealthChecker(func() error {
    return testBuildPipeline()
})

monitor.RegisterHealthCheck(componentCheck)
monitor.RegisterHealthCheck(buildCheck)
```

## Metrics

### Available Metrics

The system automatically collects the following metrics:

#### Application Metrics
- `templar_components_scanned_total` - Total components scanned by type
- `templar_components_built_total` - Total components built with success/failure status
- `templar_build_duration_seconds` - Component build duration histogram
- `templar_http_requests_total` - HTTP requests by method, path, status
- `templar_websocket_connections_total` - WebSocket connection events
- `templar_file_watcher_events_total` - File system events by type
- `templar_cache_operations_total` - Cache hit/miss statistics
- `templar_errors_total` - Error counts by category and component

#### System Metrics
- `templar_uptime_seconds` - Application uptime
- Memory usage (heap allocation, GC stats)
- Goroutine count
- Process information

#### HTTP Metrics
- `templar_http_request_duration_seconds` - Request duration histogram
- Request/response size distributions
- Status code distributions

### Custom Metrics

Add custom metrics to track application-specific data:

```go
// Counter metric
monitor.metrics.Counter("custom_events_total", map[string]string{
    "event_type": "user_action",
    "component":  "button",
})

// Gauge metric  
monitor.metrics.Gauge("active_sessions", float64(sessionCount), nil)

// Histogram metric
monitor.metrics.Histogram("operation_duration_seconds", duration.Seconds(), map[string]string{
    "operation": "template_render",
})

// Timer utility
timer := monitor.metrics.Timer("database_query", map[string]string{
    "query_type": "component_lookup",
})
defer timer()
```

## Integration Examples

### CLI Command Integration

```go
package cmd

import (
    "github.com/spf13/cobra"
    "github.com/conneroisu/templar/internal/monitoring"
)

var serveCmd = &cobra.Command{
    Use:   "serve",
    Short: "Start the development server",
    RunE: func(cmd *cobra.Command, args []string) error {
        ctx := cmd.Context()
        
        // Setup monitoring
        monitor, err := monitoring.SetupTemplarMonitoring("")
        if err != nil {
            return err
        }
        defer monitor.GracefulShutdown(ctx)
        
        // Track the serve operation
        return monitor.TrackServerOperation(ctx, "start_dev_server", func(ctx context.Context) error {
            return startDevServer(ctx)
        })
    },
}
```

### Component Scanner Integration

```go
func ScanComponents(ctx context.Context, paths []string) error {
    return monitoring.MonitorComponentOperation(ctx, "scanner", "scan", func() error {
        for _, path := range paths {
            components, err := scanPath(path)
            if err != nil {
                monitoring.LogComponentError(ctx, "scanner", "scan_path", err, map[string]interface{}{
                    "path": path,
                })
                continue
            }
            
            for _, component := range components {
                // Record successful scan
                if monitor := monitoring.GetGlobalMonitor(); monitor != nil {
                    if tm, ok := monitor.(*monitoring.TemplarMonitor); ok {
                        tm.RecordComponentScanned(component.Type, component.Name)
                    }
                }
            }
        }
        return nil
    })
}
```

### Build Pipeline Integration

```go
func BuildComponent(ctx context.Context, component *Component) error {
    start := time.Now()
    
    err := monitoring.MonitorComponentOperation(ctx, "build", "compile", func() error {
        return compileComponent(component)
    })
    
    duration := time.Since(start)
    success := err == nil
    
    // Record build metrics
    if monitor := monitoring.GetGlobalMonitor(); monitor != nil {
        if tm, ok := monitor.(*monitoring.TemplarMonitor); ok {
            tm.RecordComponentBuilt(component.Name, success, duration)
        }
    }
    
    return err
}
```

## Production Deployment

### Docker Integration

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o templar ./cmd/templar

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/templar .
COPY --from=builder /app/config/monitoring.yml ./config/

# Expose monitoring port
EXPOSE 8081

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8081/health/live || exit 1

CMD ["./templar", "serve", "--monitoring-config", "./config/monitoring.yml"]
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: templar
spec:
  replicas: 3
  selector:
    matchLabels:
      app: templar
  template:
    metadata:
      labels:
        app: templar
    spec:
      containers:
      - name: templar
        image: templar:latest
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 8081
          name: monitoring
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8081
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8081
          initialDelaySeconds: 5
          periodSeconds: 10
        env:
        - name: TEMPLAR_LOG_LEVEL
          value: "info"
        - name: TEMPLAR_METRICS_ENABLED
          value: "true"
        volumeMounts:
        - name: config
          mountPath: /app/config
      volumes:
      - name: config
        configMap:
          name: templar-config
---
apiVersion: v1
kind: Service
metadata:
  name: templar-monitoring
  labels:
    app: templar
spec:
  ports:
  - port: 8081
    targetPort: 8081
    name: monitoring
  selector:
    app: templar
```

### Prometheus Integration

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'templar'
    static_configs:
      - targets: ['localhost:8081']
    metrics_path: '/metrics'
    scrape_interval: 30s
```

## Troubleshooting

### Common Issues

1. **Monitoring endpoints not accessible**
   - Check `http_enabled: true` in configuration
   - Verify port not in use: `netstat -ln | grep 8081`
   - Check firewall settings

2. **Metrics not being collected**
   - Verify `metrics_enabled: true`
   - Check metrics output path permissions
   - Review logs for metric collection errors

3. **Health checks failing**
   - Review health check logs for specific failures
   - Verify component dependencies (filesystem access, binary availability)
   - Check resource constraints (memory, disk space)

4. **High memory usage**
   - Review metric retention settings
   - Check for metric label cardinality issues
   - Monitor goroutine counts for leaks

### Debug Mode

Enable debug logging for detailed monitoring information:

```yaml
log_level: "debug"
```

Or via environment variable:
```bash
export TEMPLAR_LOG_LEVEL=debug
```

### Performance Tuning

Optimize monitoring performance for high-throughput scenarios:

```yaml
# Reduce monitoring overhead
metrics_interval: 60s
health_check_interval: 60s

# Limit metric retention
max_metrics_age: 24h
max_metric_series: 10000

# Optimize HTTP server
http_timeout: 30s
max_connections: 100
```

This comprehensive monitoring system provides production-ready observability for Templar applications while maintaining minimal performance overhead.