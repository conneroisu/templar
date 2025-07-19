# Templar API Documentation

Welcome to the comprehensive API documentation for Templar, a rapid prototyping CLI tool for Go templ components with browser preview functionality, hot reload capability, and streamlined development workflows.

## Table of Contents

- [Quick Start](#quick-start)
- [REST API Endpoints](#rest-api-endpoints)
- [WebSocket API](#websocket-api)
- [CLI Commands](#cli-commands)
- [Configuration API](#configuration-api)
- [Component Registry API](#component-registry-api)
- [Build Pipeline API](#build-pipeline-api)
- [Preview System API](#preview-system-api)
- [Performance Monitoring API](#performance-monitoring-api)
- [Security API](#security-api)
- [Error Handling](#error-handling)
- [Rate Limiting](#rate-limiting)
- [Examples](#examples)

## Quick Start

```bash
# Start the development server
templar serve --port 8080

# Initialize a new project
templar init --template blog

# Preview a specific component
templar preview Button --props '{"text":"Hello World"}'

# List all components
templar list --format json
```

## REST API Endpoints

### Health and Status

#### `GET /health`
Returns the server health status.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0",
  "build_info": {
    "commit": "abc123",
    "build_time": "2024-01-15T08:00:00Z"
  },
  "checks": {
    "server": {"status": "healthy", "message": "HTTP server operational"},
    "registry": {"status": "healthy", "components": 42},
    "watcher": {"status": "healthy", "message": "File watcher operational"},
    "build": {"status": "healthy", "message": "Build pipeline operational"}
  }
}
```

### Component Management

#### `GET /components`
Lists all discovered components.

**Query Parameters:**
- `format` (string): Response format (`json`, `table`). Default: `json`
- `filter` (string): Filter components by name pattern
- `include_props` (boolean): Include component properties. Default: `false`

**Response:**
```json
{
  "components": [
    {
      "name": "Button",
      "package": "components",
      "file_path": "./components/button.templ",
      "parameters": [
        {
          "name": "text",
          "type": "string",
          "optional": false,
          "default": null
        },
        {
          "name": "variant",
          "type": "string",
          "optional": true,
          "default": "primary"
        }
      ],
      "imports": ["context"],
      "last_modified": "2024-01-15T10:25:00Z",
      "hash": "abc123def456",
      "dependencies": ["Icon"]
    }
  ],
  "total_count": 1,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

#### `GET /component/{name}`
Gets detailed information about a specific component.

**Path Parameters:**
- `name` (string): Component name

**Response:**
```json
{
  "name": "Button",
  "package": "components", 
  "file_path": "./components/button.templ",
  "parameters": [...],
  "imports": [...],
  "dependencies": [...],
  "metadata": {
    "description": "A reusable button component",
    "author": "developer@example.com",
    "version": "1.0.0",
    "tags": ["ui", "interactive"]
  },
  "examples": [
    {
      "name": "Primary Button",
      "props": {"text": "Click me", "variant": "primary"}
    }
  ]
}
```

### Component Preview

#### `GET /preview/{component}`
Renders a component preview.

**Path Parameters:**
- `component` (string): Component name

**Query Parameters:**
- `props` (string): JSON-encoded component properties
- `theme` (string): UI theme (`light`, `dark`). Default: `light`
- `viewport` (string): Viewport size (`mobile`, `tablet`, `desktop`). Default: `desktop`
- `layout` (string): Preview layout template. Default: `default`

**Response:**
```json
{
  "html": "<button class=\"btn btn-primary\">Click me</button>",
  "css": ".btn { padding: 8px 16px; border: none; border-radius: 4px; }",
  "javascript": "// Component-specific JS",
  "metadata": {
    "component_name": "Button",
    "props": {"text": "Click me", "variant": "primary"},
    "theme": "light",
    "viewport_size": {"width": 1200, "height": 800, "scale": 1.0},
    "generated_at": "2024-01-15T10:30:00Z",
    "cache_key": "button_abc123",
    "version": "1.0.0"
  },
  "performance": {
    "render_time": "10ms",
    "template_time": "5ms", 
    "asset_load_time": "2ms",
    "cache_hit": false,
    "memory_used": 1024
  }
}
```

#### `POST /preview/{component}`
Renders a component preview with complex props via POST body.

**Request Body:**
```json
{
  "props": {
    "title": "Complex Component",
    "items": [{"id": 1, "name": "Item 1"}],
    "config": {"theme": "dark", "size": "large"}
  },
  "options": {
    "theme": "dark",
    "viewport": {"width": 768, "height": 1024},
    "mock_data": true,
    "show_debug_info": true
  }
}
```

### Build Management

#### `GET /api/build/status`
Returns the current build pipeline status.

**Response:**
```json
{
  "status": "healthy",
  "total_builds": 156,
  "failed_builds": 3,
  "cache_hits": 89,
  "errors": 0,
  "timestamp": 1705312200
}
```

#### `GET /api/build/metrics`
Returns detailed build pipeline metrics.

**Response:**
```json
{
  "build_metrics": {
    "total_builds": 156,
    "successful_builds": 153,
    "failed_builds": 3,
    "cache_hits": 89,
    "average_duration": "150ms",
    "total_duration": "23.4s"
  },
  "cache_metrics": {
    "entries": 45,
    "size_bytes": 2048576,
    "max_size": 104857600,
    "hit_rate": 0.57
  },
  "timestamp": 1705312200
}
```

#### `GET /api/build/errors`
Returns recent build errors.

**Response:**
```json
{
  "errors": [
    {
      "component": "BrokenComponent",
      "file": "./components/broken.templ",
      "line": 5,
      "column": 12,
      "message": "undefined variable: invalidVar",
      "severity": "error",
      "timestamp": "2024-01-15T10:25:00Z"
    }
  ],
  "count": 1,
  "timestamp": 1705312200
}
```

#### `DELETE /api/build/cache`
Clears the build cache.

**Response:**
```json
{
  "message": "Cache cleared successfully",
  "timestamp": 1705312200
}
```

### Performance Monitoring

#### `GET /api/performance/metrics`
Returns system performance metrics.

**Response:**
```json
{
  "cpu_usage": 45.2,
  "memory_usage_mb": 128,
  "goroutine_count": 15,
  "gc_pause_time": "2ms",
  "build_throughput": 2.5,
  "cache_hit_rate": 0.67,
  "active_connections": 3,
  "last_updated": "2024-01-15T10:30:00Z"
}
```

#### `GET /api/performance/optimization`
Returns performance optimization status and settings.

**Response:**
```json
{
  "optimizations": {
    "cpu_optimization": true,
    "memory_optimization": true,
    "io_optimization": true,
    "cache_optimization": true
  },
  "settings": {
    "max_goroutines": 16,
    "gc_target_percent": 100,
    "io_concurrency_limit": 8,
    "cache_optimization_level": 2
  },
  "current_state": {
    "worker_count": 4,
    "queue_load": 0.23,
    "memory_threshold": 0.8,
    "cpu_threshold": 0.9
  }
}
```

### Security Monitoring

#### `GET /api/security/headers`
Returns current security header configuration.

**Response:**
```json
{
  "csp_policy": "default-src 'self'; script-src 'self' 'unsafe-inline'",
  "hsts_config": {
    "max_age": 31536000,
    "include_subdomains": true,
    "preload": false
  },
  "frame_options": "DENY",
  "content_type_options": "nosniff",
  "xss_protection": "1; mode=block",
  "referrer_policy": "strict-origin-when-cross-origin"
}
```

#### `GET /api/security/rate-limit/status`
Returns rate limiting status and statistics.

**Response:**
```json
{
  "enabled": true,
  "requests_per_minute": 1000,
  "burst_size": 50,
  "active_limiters": 5,
  "blocked_ips": ["192.168.1.100"],
  "recent_blocks": [
    {
      "ip": "192.168.1.100",
      "reason": "rate_limit_exceeded",
      "timestamp": "2024-01-15T10:25:00Z",
      "requests_count": 1500
    }
  ]
}
```

## WebSocket API

### Live Reload Connection

Connect to `/ws` for live reload functionality.

**Connection URL:** `ws://localhost:8080/ws`

**Message Types:**

#### Component Update Event
```json
{
  "type": "component_updated",
  "target": "Button",
  "data": {
    "file_path": "./components/button.templ",
    "hash": "new_hash_123"
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

#### Build Success Event
```json
{
  "type": "build_success", 
  "target": "Button",
  "data": {
    "duration": "150ms",
    "cache_hit": false
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

#### Build Error Event
```json
{
  "type": "build_error",
  "target": "BrokenComponent", 
  "data": {
    "errors": [
      {
        "line": 5,
        "column": 12,
        "message": "syntax error",
        "severity": "error"
      }
    ]
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

#### Full Reload Event
```json
{
  "type": "full_reload",
  "data": {
    "reason": "configuration_changed"
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## CLI Commands

### `templar init`
Initialize a new templar project.

**Usage:**
```bash
templar init [flags]
```

**Flags:**
- `--template, -t` (string): Project template (blog, dashboard, component-library)
- `--minimal, -m` (boolean): Create minimal project structure
- `--name, -n` (string): Project name
- `--path, -p` (string): Project path (default: current directory)

**Examples:**
```bash
# Initialize with blog template
templar init --template blog --name my-blog

# Create minimal project
templar init --minimal

# Initialize in specific directory
templar init --path ./my-project --template dashboard
```

### `templar serve`
Start the development server.

**Usage:**
```bash
templar serve [flags]
```

**Flags:**
- `--port, -p` (int): Server port (default: 8080)
- `--host` (string): Server host (default: localhost)
- `--no-open` (boolean): Don't open browser automatically
- `--hot-reload` (boolean): Enable hot reload (default: true)
- `--config, -c` (string): Configuration file path

**Examples:**
```bash
# Start on default port
templar serve

# Start on custom port
templar serve --port 3000

# Start without opening browser
templar serve --no-open

# Use custom config
templar serve --config ./templar.yml
```

### `templar list`
List all discovered components.

**Usage:**
```bash
templar list [flags]
```

**Flags:**
- `--format, -f` (string): Output format (table, json, yaml) (default: table)
- `--with-props` (boolean): Include component properties
- `--filter` (string): Filter components by name pattern
- `--sort` (string): Sort by field (name, modified, package)

**Examples:**
```bash
# List all components
templar list

# List with JSON output
templar list --format json

# List with properties
templar list --with-props

# Filter by name pattern
templar list --filter "Button*"
```

### `templar preview`
Preview a specific component.

**Usage:**
```bash
templar preview [component] [flags]
```

**Flags:**
- `--props` (string): Component properties as JSON
- `--mock` (string): Mock data file path
- `--theme` (string): UI theme (light, dark) (default: light)
- `--viewport` (string): Viewport size (mobile, tablet, desktop) (default: desktop)
- `--output, -o` (string): Output file path
- `--format` (string): Output format (html, pdf, png)

**Examples:**
```bash
# Preview component with props
templar preview Button --props '{"text":"Hello","variant":"primary"}'

# Preview with mock data
templar preview UserCard --mock ./mocks/user.json

# Preview with dark theme
templar preview Dashboard --theme dark --viewport tablet

# Export to file
templar preview Button --output button.html --format html
```

### `templar build`
Build all components.

**Usage:**
```bash
templar build [flags]
```

**Flags:**
- `--production` (boolean): Production build with optimizations
- `--output, -o` (string): Output directory
- `--clean` (boolean): Clean output directory before build
- `--parallel, -j` (int): Number of parallel workers (default: CPU count)

**Examples:**
```bash
# Development build
templar build

# Production build
templar build --production

# Build to specific directory
templar build --output ./dist

# Clean build
templar build --clean --production
```

### `templar watch`
Watch for file changes and rebuild.

**Usage:**
```bash
templar watch [flags]
```

**Flags:**
- `--paths` ([]string): Paths to watch (default: ./components, ./views)
- `--ignore` ([]string): Patterns to ignore
- `--debounce` (duration): Debounce duration (default: 300ms)
- `--command` (string): Command to run on changes

**Examples:**
```bash
# Watch default paths
templar watch

# Watch specific paths
templar watch --paths ./src,./components

# Watch with custom command
templar watch --command "go generate ./..."
```

## Configuration API

### Configuration File Format

Templar uses YAML configuration files (`.templar.yml`).

```yaml
server:
  port: 8080
  host: "localhost"
  open: true
  environment: "development"
  middleware: ["cors", "logging", "security"]
  allowed_origins: ["http://localhost:3000"]

components:
  scan_paths: ["./components", "./views", "./examples"]
  exclude_patterns: ["*_test.templ", "*.bak"]
  auto_discover: true

build:
  command: "templ generate"
  args: []
  watch: ["**/*.templ"]
  ignore: ["node_modules", ".git", "*.tmp"]
  cache_dir: ".templar/cache"
  parallel_workers: 4

development:
  hot_reload: true
  css_injection: true
  error_overlay: true
  source_maps: true
  debug_mode: false

preview:
  default_theme: "light"
  mock_data: "auto"
  wrapper_template: "layout.templ"
  auto_props: true
  session_timeout: "1h"

performance:
  enable_optimization: true
  max_memory_mb: 512
  gc_target_percent: 100
  io_concurrency_limit: 8

security:
  enable_headers: true
  csp_policy: "default-src 'self'"
  rate_limiting:
    enabled: true
    requests_per_minute: 1000
    burst_size: 50
  blocked_user_agents: []

logging:
  level: "info"
  format: "json"
  output: "stdout"
  file_rotation: true
  max_file_size: "10MB"
  max_files: 5
```

### Environment Variables

All configuration options can be overridden with environment variables using the `TEMPLAR_` prefix:

```bash
export TEMPLAR_SERVER_PORT=3000
export TEMPLAR_DEVELOPMENT_HOT_RELOAD=false
export TEMPLAR_LOGGING_LEVEL=debug
export TEMPLAR_SECURITY_RATE_LIMITING_ENABLED=true
```

## Error Handling

### Error Response Format

All API endpoints return errors in a consistent format:

```json
{
  "error": {
    "type": "validation_error",
    "code": "INVALID_COMPONENT_NAME",
    "message": "Component name 'invalid-name' contains invalid characters",
    "details": {
      "component": "invalid-name",
      "allowed_pattern": "^[A-Za-z][A-Za-z0-9]*$"
    },
    "timestamp": "2024-01-15T10:30:00Z",
    "request_id": "req_123456789"
  }
}
```

### Error Types

- `validation_error`: Invalid input or parameters
- `not_found_error`: Requested resource not found
- `build_error`: Component build failure
- `security_error`: Security policy violation
- `rate_limit_error`: Rate limit exceeded
- `internal_error`: Internal server error

### HTTP Status Codes

- `200 OK`: Successful request
- `201 Created`: Resource created successfully
- `400 Bad Request`: Invalid request parameters
- `401 Unauthorized`: Authentication required
- `403 Forbidden`: Access denied
- `404 Not Found`: Resource not found
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: Server error
- `503 Service Unavailable`: Service temporarily unavailable

## Rate Limiting

### Rate Limit Headers

All responses include rate limiting headers:

```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1705312800
Retry-After: 60
```

### Rate Limit Exceeded Response

When rate limits are exceeded (HTTP 429):

```json
{
  "error": {
    "type": "rate_limit_error",
    "code": "RATE_LIMIT_EXCEEDED", 
    "message": "Rate limit exceeded. Please try again later.",
    "details": {
      "limit": 1000,
      "window": "1 minute",
      "retry_after": 60
    }
  }
}
```

## Examples

### Complete Component Preview Workflow

```bash
# 1. Initialize project
templar init --template component-library --name ui-components

# 2. Start development server
templar serve --port 8080

# 3. List available components
curl "http://localhost:8080/components?format=json"

# 4. Preview a component
curl "http://localhost:8080/preview/Button?props=%7B%22text%22%3A%22Click%20me%22%7D"

# 5. Monitor build status
curl "http://localhost:8080/api/build/status"

# 6. Get performance metrics
curl "http://localhost:8080/api/performance/metrics"
```

### WebSocket Live Reload Integration

```javascript
// Connect to WebSocket for live reload
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  
  switch (data.type) {
    case 'component_updated':
      console.log(`Component ${data.target} updated`);
      // Reload preview or refresh component
      break;
      
    case 'build_error':
      console.error(`Build error in ${data.target}:`, data.data.errors);
      // Show error overlay
      break;
      
    case 'full_reload':
      window.location.reload();
      break;
  }
};
```

### Component Properties Validation

```bash
# Preview with invalid props to see validation
curl -X POST "http://localhost:8080/preview/Button" \
  -H "Content-Type: application/json" \
  -d '{
    "props": {
      "text": 123,  // Should be string
      "invalid_prop": "value"  // Not defined in component
    }
  }'
```

### Performance Monitoring Integration

```bash
# Get current performance metrics
curl "http://localhost:8080/api/performance/metrics" | jq

# Monitor build pipeline performance
curl "http://localhost:8080/api/build/metrics" | jq '.build_metrics'

# Check cache efficiency
curl "http://localhost:8080/api/build/metrics" | jq '.cache_metrics.hit_rate'
```

This API documentation provides comprehensive coverage of all Templar functionality, from basic component preview to advanced performance monitoring and security features.