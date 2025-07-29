# Configuration Reference

This document provides a comprehensive reference for all configuration options available in Templar. Configuration can be specified through YAML files, environment variables, or command-line flags.

## Table of Contents

- [Configuration Loading](#configuration-loading)
- [Server Configuration](#server-configuration)
- [Build Configuration](#build-configuration)
- [Preview Configuration](#preview-configuration)
- [Components Configuration](#components-configuration)
- [Development Configuration](#development-configuration)
- [Production Configuration](#production-configuration)
- [Plugins Configuration](#plugins-configuration)
- [CSS Configuration](#css-configuration)
- [Monitoring Configuration](#monitoring-configuration)
- [Timeout Configuration](#timeout-configuration)
- [Environment Variables](#environment-variables)
- [Configuration Examples](#configuration-examples)

## Configuration Loading

Templar loads configuration from multiple sources in the following precedence order (highest to lowest):

1. **Command-line flags** (`--port`, `--config`, etc.)
2. **Environment variables** (`TEMPLAR_CONFIG_FILE`, `TEMPLAR_SERVER_PORT`, etc.)
3. **Configuration file** (`.templar.yml` or custom path via `TEMPLAR_CONFIG_FILE`)
4. **Built-in defaults**

### Configuration File Locations

Templar searches for configuration files in this order:

1. Path specified by `--config` flag
2. Path specified by `TEMPLAR_CONFIG_FILE` environment variable
3. `.templar.yml` in current directory
4. `.templar.yaml` in current directory
5. `$HOME/.templar.yml`
6. `$HOME/.config/templar.yml`

## Server Configuration

Controls the HTTP server, authentication, and network settings.

```yaml
server:
  port: 8080                    # Server port (default: 8080)
  host: "localhost"             # Server host (default: localhost)
  open: true                    # Auto-open browser (default: true)
  no-open: false                # Disable auto-open (overrides open)
  environment: "development"    # Environment: development, staging, production
  middleware:                   # HTTP middleware stack
    - cors
    - logger
    - compression
    - security
  allowed_origins:              # CORS allowed origins
    - "https://yourdomain.com"
    - "https://www.yourdomain.com"
  
  # Authentication configuration
  auth:
    enabled: false              # Enable authentication (default: false)
    mode: "none"                # Auth mode: "token", "basic", "none"
    token: ""                   # Bearer token for token auth
    username: ""                # Username for basic auth
    password: ""                # Password for basic auth
    allowed_ips: []             # IP allowlist (empty = allow all)
    require_auth: false         # Require auth for non-localhost
    localhost_bypass: true      # Allow localhost without auth
```

### Environment Variables

```bash
TEMPLAR_SERVER_PORT=8080
TEMPLAR_SERVER_HOST=localhost
TEMPLAR_SERVER_OPEN=true
TEMPLAR_SERVER_ENVIRONMENT=development
TEMPLAR_SERVER_AUTH_ENABLED=false
TEMPLAR_SERVER_AUTH_MODE=none
TEMPLAR_SERVER_AUTH_TOKEN=your-secret-token
TEMPLAR_SERVER_AUTH_USERNAME=admin
TEMPLAR_SERVER_AUTH_PASSWORD=secret
```

## Build Configuration

Controls component building, watching, and caching behavior.

```yaml
build:
  command: "templ generate"     # Build command (default: "templ generate")
  watch:                        # File patterns to watch
    - "**/*.templ"
    - "**/*.go"
  ignore:                       # Patterns to ignore
    - "node_modules"
    - ".git"
    - "*_test.go"
    - "vendor/**"
  cache_dir: ".templar/cache"   # Cache directory (default: .templar/cache)
```

### Environment Variables

```bash
TEMPLAR_BUILD_COMMAND="templ generate"
TEMPLAR_BUILD_CACHE_DIR=.templar/cache
```

## Preview Configuration

Controls component preview and mock data behavior.

```yaml
preview:
  mock_data: "auto"             # Mock data mode: "auto", "manual", "none"
  wrapper: "layout.templ"       # Wrapper template (default: layout.templ)
  auto_props: true              # Auto-generate component props (default: true)
```

### Environment Variables

```bash
TEMPLAR_PREVIEW_MOCK_DATA=auto
TEMPLAR_PREVIEW_WRAPPER=layout.templ
TEMPLAR_PREVIEW_AUTO_PROPS=true
```

## Components Configuration

Controls component discovery and scanning behavior.

```yaml
components:
  scan_paths:                   # Directories to scan for components
    - "./components"
    - "./views" 
    - "./examples"
  exclude_patterns:             # Patterns to exclude from scanning
    - "*_test.templ"
    - "*.bak"
    - "*.example.templ"
```

### Environment Variables

```bash
TEMPLAR_COMPONENTS_SCAN_PATHS="./components,./views,./examples"
TEMPLAR_COMPONENTS_EXCLUDE_PATTERNS="*_test.templ,*.bak"
```

## Development Configuration

Controls development-specific features like hot reload and error overlays.

```yaml
development:
  hot_reload: true              # Enable hot reload (default: true)
  css_injection: true           # Enable CSS injection (default: true)
  state_preservation: false     # Preserve component state on reload
  error_overlay: true           # Show error overlay in browser (default: true)
```

### Environment Variables

```bash
TEMPLAR_DEVELOPMENT_HOT_RELOAD=true
TEMPLAR_DEVELOPMENT_CSS_INJECTION=true
TEMPLAR_DEVELOPMENT_STATE_PRESERVATION=false
TEMPLAR_DEVELOPMENT_ERROR_OVERLAY=true
```

## Production Configuration

Comprehensive production build and deployment settings.

```yaml
production:
  # Output configuration
  output_dir: "dist"            # Build output directory
  static_dir: "static"          # Static assets directory
  assets_dir: "assets"          # Processed assets directory
  
  # Build optimization
  minification:
    css: true                   # Minify CSS (default: true)
    javascript: true            # Minify JavaScript (default: true)
    html: true                  # Minify HTML (default: true)
    json: false                 # Minify JSON (default: false)
    remove_comments: true       # Remove comments (default: true)
    strip_debug: true           # Strip debug info (default: true)
  
  # Asset compression
  compression:
    enabled: true               # Enable compression (default: true)
    algorithms:                 # Compression algorithms
      - "gzip"
      - "brotli"
    level: 6                    # Compression level 1-9 (default: 6)
    extensions:                 # File extensions to compress
      - ".html"
      - ".css"
      - ".js"
      - ".json"
      - ".xml"
      - ".svg"
  
  # Asset optimization
  asset_optimization:
    # Image optimization
    images:
      enabled: true             # Enable image optimization
      quality: 85               # Image quality 1-100 (default: 85)
      progressive: true         # Progressive JPEG
      responsive: false         # Generate responsive images
      formats:                  # Output formats
        - "webp"
        - "avif"
    
    # Font optimization
    fonts:
      enabled: false            # Enable font optimization
      subsetting: false         # Font subsetting
      preload: false            # Font preloading
      formats:
        - "woff2"
        - "woff"
    
    # Icon optimization
    icons:
      enabled: false            # Enable icon optimization
      sprite: false             # Create icon sprite
      svg_optimization: true    # Optimize SVG icons
      format: "svg"             # Icon format: svg, sprite, font
    
    critical_css: true          # Extract critical CSS
    tree_shaking: true          # Enable tree shaking
    dead_code_elimination: false # Remove dead code
  
  # Bundling and code splitting
  bundling:
    enabled: true               # Enable bundling
    strategy: "adaptive"        # Bundling strategy: single, multiple, adaptive
    chunk_size_limit: 250000    # Max chunk size in bytes
    externals: []               # External dependencies to exclude
    splitting: true             # Enable automatic code splitting
  
  code_splitting:
    enabled: true               # Enable code splitting
    vendor_split: true          # Split vendor code
    async_chunks: true          # Create async chunks
    common_chunks: true         # Extract common chunks
    manual_chunks: []           # Manual chunk definitions
  
  # Deployment settings
  deployment:
    target: "static"            # Deployment target: static, docker, serverless
    environment: "production"   # Deployment environment
    base_url: ""                # Base URL for deployment
    asset_prefix: ""            # Prefix for asset URLs
    headers:                    # Custom HTTP headers
      X-Content-Type-Options: "nosniff"
      X-Frame-Options: "DENY"
      X-XSS-Protection: "1; mode=block"
    redirects: []               # URL redirect rules
    error_pages: {}             # Custom error pages
  
  # CDN configuration
  cdn:
    enabled: false              # Enable CDN integration
    provider: ""                # CDN provider: cloudflare, aws, etc.
    base_path: ""               # CDN base path
    cache_ttl: 86400            # Cache TTL in seconds
    invalidation: false         # Auto-invalidate on deploy
    headers: {}                 # CDN-specific headers
  
  # Performance settings
  performance:
    budget_limits:              # Performance budgets
      bundle_size: 500000       # Max bundle size (500KB)
      image_size: 1000000       # Max image size (1MB)
      css_size: 100000          # Max CSS size (100KB)
      js_size: 300000           # Max JS size (300KB)
    preconnect: []              # Domains to preconnect
    prefetch: []                # Resources to prefetch
    preload: []                 # Resources to preload
    lazy_loading: true          # Enable lazy loading
    service_worker: false       # Generate service worker
    manifest_file: false        # Generate web manifest
  
  # Security settings
  security:
    # Content Security Policy
    csp:
      enabled: false            # Enable CSP
      directives:
        default-src: "'self'"
        script-src: "'self' 'unsafe-inline'"
        style-src: "'self' 'unsafe-inline'"
        img-src: "'self' data: https:"
      report_uri: ""            # CSP violation reporting URI
    
    hsts: true                  # Enable HSTS
    x_frame_options: "DENY"     # X-Frame-Options header
    x_content_type_options: true # X-Content-Type-Options header
    
    # Security scanning
    scan:
      enabled: true             # Enable security scanning
      dependencies: true        # Scan dependencies
      secrets: false            # Scan for secrets
      static_analysis: false    # Static code analysis
      allowed_risks: []         # Acceptable risk levels
    
    # Secrets management
    secrets:
      detection: true           # Detect secrets in code
      validation: false         # Validate secret formats
      patterns: []              # Custom secret patterns
      exclusions: []            # Files to exclude from scanning
  
  # Validation settings
  validation:
    enabled: true               # Enable build validation
    
    # Accessibility checks
    accessibility:
      enabled: true             # Enable accessibility checks
      level: "AA"               # WCAG level: A, AA, AAA
      rules: []                 # Specific rules to check
      ignore_rules: []          # Rules to ignore
    
    # Performance checks
    performance:
      enabled: true             # Enable performance checks
      bundle_size: 500000       # Max bundle size
      load_time: 3000           # Max load time (ms)
      lighthouse: false         # Run Lighthouse audits
      metrics: {}               # Custom performance metrics
    
    # SEO checks
    seo:
      enabled: false            # Enable SEO checks
      meta_tags: true           # Validate meta tags
      sitemap: false            # Generate/validate sitemap
      robots: false             # Generate robots.txt
      schema: false             # Validate structured data
      open_graph: false         # Validate Open Graph tags
    
    # Link checks
    links:
      enabled: false            # Enable link checking
      internal: true            # Check internal links
      external: false           # Check external links
      images: true              # Check image links
      timeout: 30               # Request timeout (seconds)
      ignore_urls: []           # URLs to ignore
    
    # Standards validation
    standards:
      enabled: false            # Enable standards validation
      html: true                # Validate HTML
      css: true                 # Validate CSS
      javascript: false         # Validate JavaScript
      w3c: false                # Use W3C validators
  
  # Environment-specific overrides
  environments:
    staging:
      variables:
        NODE_ENV: "staging"
      features:
        debug_mode: true
        analytics: false
      deployment:
        environment: "staging"
        base_url: "https://staging.example.com"
    
    production:
      variables:
        NODE_ENV: "production"
      features:
        debug_mode: false
        analytics: true
        hot_reload: false
      deployment:
        environment: "production"
        base_url: "https://example.com"
      monitoring:
        analytics:
          enabled: true
          privacy: true
        performance:
          enabled: true
          vitals: true
```

## Plugins Configuration

Controls plugin discovery, loading, and configuration.

```yaml
plugins:
  enabled:                      # Enabled plugins
    - "tailwind"
    - "hotreload"
  disabled: []                  # Disabled plugins
  discovery_paths:              # Plugin discovery paths
    - "./plugins"
    - "~/.templar/plugins"
  configurations:               # Plugin-specific configurations
    tailwind:
      config_path: "./tailwind.config.js"
      output_path: "./static/css/tailwind.css"
    hotreload:
      debounce_ms: 100
```

### Environment Variables

```bash
TEMPLAR_PLUGINS_ENABLED="tailwind,hotreload"
TEMPLAR_PLUGINS_DISABLED=""
TEMPLAR_PLUGINS_DISCOVERY_PATHS="./plugins,~/.templar/plugins"
```

## CSS Configuration

Controls CSS framework integration and processing.

```yaml
css:
  framework: "tailwind"         # CSS framework: tailwind, bootstrap, bulma
  output_path: "./static/css"   # Output path for generated CSS
  source_paths:                 # Paths to scan for CSS classes
    - "./components"
    - "./templates"
  
  # Optimization settings
  optimization:
    purge: true                 # Purge unused CSS
    minify: true                # Minify CSS output
  
  # Theming settings
  theming:
    extract_variables: false    # Extract CSS variables
    style_guide: false          # Generate style guide
  
  # CSS variables
  variables:
    primary-color: "#3b82f6"
    secondary-color: "#64748b"
  
  # Framework-specific options
  options:
    tailwind:
      content: ["./components/**/*.templ"]
      theme:
        extend:
          colors:
            brand: "#1e40af"
```

## Monitoring Configuration

Controls logging, metrics, and observability features.

```yaml
monitoring:
  enabled: true                 # Enable monitoring (default: true)
  log_level: "info"             # Log level: debug, info, warn, error
  log_format: "json"            # Log format: json, text
  metrics_path: "./logs/metrics.json" # Metrics output path
  http_port: 8081               # Metrics HTTP port
  alerts_enabled: false         # Enable alerting (default: false)
```

### Environment Variables

```bash
TEMPLAR_MONITORING_ENABLED=true
TEMPLAR_MONITORING_LOG_LEVEL=info
TEMPLAR_MONITORING_LOG_FORMAT=json
TEMPLAR_MONITORING_METRICS_PATH=./logs/metrics.json
TEMPLAR_MONITORING_HTTP_PORT=8081
TEMPLAR_MONITORING_ALERTS_ENABLED=false
```

## Timeout Configuration

Controls timeout settings for various operations.

```yaml
timeouts:
  # Build and compilation timeouts
  build: "5m"                   # Build operation timeout
  external: "3m"                # External command timeout
  plugin: "2m"                  # Plugin execution timeout
  render: "30s"                 # Template rendering timeout
  
  # File system operations
  file_io: "30s"                # File I/O timeout
  file_scan: "2m"               # File scanning timeout
  file_watch: "10s"             # File watching timeout
  
  # Network operations
  network: "30s"                # Network operation timeout
  http: "30s"                   # HTTP request timeout
  websocket: "60s"              # WebSocket operation timeout
  health_check: "10s"           # Health check timeout
  
  # Server operations
  startup: "30s"                # Server startup timeout
  shutdown: "30s"               # Server shutdown timeout
  context: "30s"                # Default context timeout
  
  # Development and testing
  development: "10m"            # Development operations timeout
  testing: "5m"                 # Test execution timeout
  
  # Background operations
  background: "15m"             # Background task timeout
  cleanup: "1m"                 # Cleanup operation timeout
```

## Environment Variables

All configuration values can be overridden using environment variables with the `TEMPLAR_` prefix. The variable names follow the nested structure using underscores.

### Naming Convention

```bash
# server.port -> TEMPLAR_SERVER_PORT
# server.auth.enabled -> TEMPLAR_SERVER_AUTH_ENABLED
# components.scan_paths -> TEMPLAR_COMPONENTS_SCAN_PATHS
# production.minification.css -> TEMPLAR_PRODUCTION_MINIFICATION_CSS
```

### Complete Environment Variables List

```bash
# Server configuration
TEMPLAR_SERVER_PORT=8080
TEMPLAR_SERVER_HOST=localhost
TEMPLAR_SERVER_OPEN=true
TEMPLAR_SERVER_ENVIRONMENT=development
TEMPLAR_SERVER_AUTH_ENABLED=false
TEMPLAR_SERVER_AUTH_MODE=none
TEMPLAR_SERVER_AUTH_TOKEN=""
TEMPLAR_SERVER_AUTH_USERNAME=""
TEMPLAR_SERVER_AUTH_PASSWORD=""
TEMPLAR_SERVER_AUTH_REQUIRE_AUTH=false
TEMPLAR_SERVER_AUTH_LOCALHOST_BYPASS=true

# Build configuration
TEMPLAR_BUILD_COMMAND="templ generate"
TEMPLAR_BUILD_CACHE_DIR=".templar/cache"

# Components configuration
TEMPLAR_COMPONENTS_SCAN_PATHS="./components,./views,./examples"
TEMPLAR_COMPONENTS_EXCLUDE_PATTERNS="*_test.templ,*.bak"

# Development configuration
TEMPLAR_DEVELOPMENT_HOT_RELOAD=true
TEMPLAR_DEVELOPMENT_CSS_INJECTION=true
TEMPLAR_DEVELOPMENT_STATE_PRESERVATION=false
TEMPLAR_DEVELOPMENT_ERROR_OVERLAY=true

# Production configuration
TEMPLAR_PRODUCTION_OUTPUT_DIR="dist"
TEMPLAR_PRODUCTION_MINIFICATION_CSS=true
TEMPLAR_PRODUCTION_MINIFICATION_JAVASCRIPT=true
TEMPLAR_PRODUCTION_COMPRESSION_ENABLED=true
TEMPLAR_PRODUCTION_SECURITY_HSTS=true

# Monitoring configuration
TEMPLAR_MONITORING_ENABLED=true
TEMPLAR_MONITORING_LOG_LEVEL=info
TEMPLAR_MONITORING_LOG_FORMAT=json
TEMPLAR_MONITORING_HTTP_PORT=8081

# Timeout configuration
TEMPLAR_TIMEOUTS_BUILD=5m
TEMPLAR_TIMEOUTS_EXTERNAL=3m
TEMPLAR_TIMEOUTS_HTTP=30s
TEMPLAR_TIMEOUTS_WEBSOCKET=60s
```

## Configuration Examples

### Development Configuration

```yaml
# .templar.yml - Development setup
server:
  port: 3000
  host: "localhost"
  open: true
  environment: "development"

build:
  command: "templ generate"
  watch:
    - "**/*.templ"
    - "**/*.go"
  ignore:
    - "node_modules"
    - ".git"

components:
  scan_paths:
    - "./components"
    - "./views"

development:
  hot_reload: true
  css_injection: true
  error_overlay: true

plugins:
  enabled:
    - "tailwind"
    - "hotreload"

monitoring:
  enabled: true
  log_level: "debug"
```

### Production Configuration

```yaml
# .templar.yml - Production setup
server:
  port: 8080
  host: "0.0.0.0"
  open: false
  environment: "production"
  middleware:
    - cors
    - logger
    - compression
    - security
  allowed_origins:
    - "https://yourdomain.com"
  auth:
    enabled: true
    mode: "token"
    token: "${TEMPLAR_AUTH_TOKEN}"
    require_auth: true
    localhost_bypass: false

build:
  command: "templ generate"
  cache_dir: "/app/.templar/cache"

components:
  scan_paths: ["/app/components"]

development:
  hot_reload: false
  css_injection: false
  error_overlay: false

production:
  output_dir: "/app/dist"
  minification:
    css: true
    javascript: true
    html: true
    remove_comments: true
  compression:
    enabled: true
    algorithms: ["gzip", "brotli"]
  asset_optimization:
    critical_css: true
    tree_shaking: true
    images:
      enabled: true
      quality: 85
      formats: ["webp", "avif"]
  security:
    hsts: true
    x_frame_options: "DENY"
    csp:
      enabled: true
      directives:
        default-src: "'self'"
        script-src: "'self'"
        style-src: "'self' 'unsafe-inline'"
  deployment:
    target: "docker"
    environment: "production"

monitoring:
  enabled: true
  log_level: "info"
  log_format: "json"
  http_port: 8081

timeouts:
  build: "5m"
  http: "30s"
  startup: "30s"
  shutdown: "30s"
```

### Staging Configuration

```yaml
# .templar.staging.yml - Staging environment
server:
  port: 8080
  host: "0.0.0.0"
  environment: "staging"
  auth:
    enabled: true
    mode: "basic"
    username: "admin"
    password: "${STAGING_PASSWORD}"

production:
  environments:
    staging:
      variables:
        NODE_ENV: "staging"
      features:
        debug_mode: true
        analytics: false
      deployment:
        environment: "staging"
        base_url: "https://staging.yourdomain.com"
      monitoring:
        analytics:
          enabled: false
        performance:
          enabled: true

monitoring:
  enabled: true
  log_level: "debug"
  alerts_enabled: true
```

### Multi-environment Configuration

```yaml
# .templar.yml - Multi-environment setup
server:
  port: 8080
  host: "${TEMPLAR_HOST:-localhost}"
  environment: "${TEMPLAR_ENV:-development}"

production:
  environments:
    development:
      variables:
        NODE_ENV: "development"
      features:
        debug_mode: true
        hot_reload: true
        analytics: false
      deployment:
        base_url: "http://localhost:8080"
    
    staging:
      variables:
        NODE_ENV: "staging"
      features:
        debug_mode: true
        analytics: false
      deployment:
        base_url: "https://staging.yourdomain.com"
    
    production:
      variables:
        NODE_ENV: "production"
      features:
        debug_mode: false
        analytics: true
      deployment:
        base_url: "https://yourdomain.com"
      monitoring:
        analytics:
          enabled: true
          provider: "google"
          privacy: true

# Usage:
# TEMPLAR_ENV=development templar serve  # Development
# TEMPLAR_ENV=staging templar serve      # Staging
# TEMPLAR_ENV=production templar serve   # Production
```

### Docker Configuration

```yaml
# .templar.docker.yml - Docker deployment
server:
  host: "0.0.0.0"
  port: 8080
  auth:
    enabled: true
    mode: "token"
    token: "${TEMPLAR_AUTH_TOKEN}"

build:
  cache_dir: "/app/.templar/cache"

components:
  scan_paths: ["/app/components"]

production:
  output_dir: "/app/dist"
  static_dir: "/app/static"
  security:
    hsts: true
    csp:
      enabled: true

monitoring:
  enabled: true
  log_format: "json"
  metrics_path: "/app/logs/metrics.json"

timeouts:
  startup: "60s"
  shutdown: "30s"
```

### Kubernetes Configuration

```yaml
# .templar.k8s.yml - Kubernetes deployment
server:
  host: "0.0.0.0"
  port: 8080
  auth:
    enabled: true
    mode: "token"
    token: "${TEMPLAR_AUTH_TOKEN}"

production:
  deployment:
    target: "kubernetes"
    environment: "production"
  security:
    hsts: true
    csp:
      enabled: true
      directives:
        default-src: "'self'"

monitoring:
  enabled: true
  log_format: "json"
  http_port: 8081

# Health check configuration
health:
  check_interval: "30s"
  timeout: "5s"
  readiness_delay: "10s"

# Resource limits (handled by K8s)
resources:
  memory_limit: "512Mi"
  cpu_limit: "500m"
```

---

This configuration reference covers all available options in Templar. For specific deployment scenarios, see the [DEPLOYMENT.md](./DEPLOYMENT.md) guide.