# Templar Usage Examples

This directory contains comprehensive examples demonstrating Templar's features and capabilities.

## Quick Start Examples

### Basic Component Creation

```bash
# Initialize a new Templar project
templar init

# Create a simple component
cat > components/button.templ << 'EOF'
package components

templ Button(text string, variant string) {
    <button class={ "btn", "btn-" + variant }>{ text }</button>
}
EOF

# Generate and serve
templar serve
```

### Development Workflow

```bash
# Watch for changes and auto-rebuild
templar watch &

# List all discovered components
templar list --format json

# Preview specific component with props
templar preview Button --props '{"text":"Click Me","variant":"primary"}'
```

## Project Structure Examples

### Blog Template Structure

```
my-blog/
├── .templar.yml
├── components/
│   ├── layout.templ
│   ├── header.templ
│   ├── footer.templ
│   └── post.templ
├── pages/
│   ├── home.templ
│   └── about.templ
└── assets/
    ├── styles.css
    └── main.js
```

### E-commerce Site Structure

```
shop/
├── .templar.yml
├── components/
│   ├── product-card.templ
│   ├── shopping-cart.templ
│   ├── checkout-form.templ
│   └── navigation.templ
├── layouts/
│   ├── main.templ
│   └── checkout.templ
└── static/
    ├── images/
    └── styles/
```

## Configuration Examples

### Basic Configuration (`.templar.yml`)

```yaml
server:
  port: 8080
  host: "localhost"
  open: true

components:
  scan_paths: 
    - "./components"
    - "./layouts"
    - "./views"
  exclude_patterns:
    - "*_test.templ"
    - "*.backup"

build:
  command: "templ generate"
  watch: ["**/*.templ", "**/*.go"]
  cache_dir: ".templar/cache"

development:
  hot_reload: true
  error_overlay: true
```

### Advanced Configuration with Security

```yaml
server:
  port: 3000
  host: "0.0.0.0"
  open: false
  middleware: ["cors", "logging", "security", "ratelimit"]

security:
  csp:
    default_src: ["'self'"]
    script_src: ["'self'", "'unsafe-inline'"]
    style_src: ["'self'", "'unsafe-inline'"]
  cors:
    allowed_origins: ["http://localhost:3000"]
    allowed_methods: ["GET", "POST"]

rate_limit:
  requests_per_minute: 100
  burst: 10

components:
  scan_paths: ["./src/components", "./src/layouts"]
  auto_discovery: true
  mock_data: "auto"

preview:
  wrapper: "layouts/preview.templ"
  auto_props: true
  sandbox: true
```

## Component Examples

### Button Component with Variants

```go
// components/button.templ
package components

import "fmt"

type ButtonProps struct {
    Text     string
    Variant  string
    Size     string
    Disabled bool
    OnClick  string
}

templ Button(props ButtonProps) {
    <button 
        class={ getButtonClasses(props) }
        disabled?={ props.Disabled }
        onclick={ templ.SafeScript(props.OnClick) }
    >
        { props.Text }
    </button>
}

func getButtonClasses(props ButtonProps) string {
    classes := "btn"
    if props.Variant != "" {
        classes += " btn-" + props.Variant
    }
    if props.Size != "" {
        classes += " btn-" + props.Size
    }
    if props.Disabled {
        classes += " btn-disabled"
    }
    return classes
}
```

### Card Component with Slots

```go
// components/card.templ
package components

type CardProps struct {
    Title       string
    Subtitle    string
    ImageUrl    string
    Padding     string
    Shadow      bool
}

templ Card(props CardProps) {
    <div class={ getCardClasses(props) }>
        if props.ImageUrl != "" {
            <img src={ props.ImageUrl } alt={ props.Title } class="card-image"/>
        }
        <div class="card-content">
            if props.Title != "" {
                <h3 class="card-title">{ props.Title }</h3>
            }
            if props.Subtitle != "" {
                <p class="card-subtitle">{ props.Subtitle }</p>
            }
            <div class="card-body">
                { children... }
            </div>
        </div>
    </div>
}

func getCardClasses(props CardProps) string {
    classes := "card"
    if props.Shadow {
        classes += " card-shadow"
    }
    if props.Padding != "" {
        classes += " padding-" + props.Padding
    }
    return classes
}
```

### Form Component with Validation

```go
// components/form.templ
package components

type FormFieldProps struct {
    Name        string
    Type        string
    Label       string
    Placeholder string
    Required    bool
    Value       string
    Error       string
}

templ FormField(props FormFieldProps) {
    <div class="form-field">
        <label for={ props.Name } class="form-label">
            { props.Label }
            if props.Required {
                <span class="required">*</span>
            }
        </label>
        <input
            type={ props.Type }
            id={ props.Name }
            name={ props.Name }
            placeholder={ props.Placeholder }
            value={ props.Value }
            required?={ props.Required }
            class={ getInputClasses(props) }
        />
        if props.Error != "" {
            <span class="form-error">{ props.Error }</span>
        }
    </div>
}

func getInputClasses(props FormFieldProps) string {
    classes := "form-input"
    if props.Error != "" {
        classes += " form-input-error"
    }
    return classes
}
```

## CLI Usage Examples

### Project Initialization

```bash
# Initialize with default template
templar init

# Initialize with minimal setup
templar init --minimal

# Initialize with specific template
templar init --template blog

# Initialize in specific directory
templar init my-project
cd my-project
```

### Development Server

```bash
# Start development server
templar serve

# Custom port and host
templar serve --port 3000 --host 0.0.0.0

# Disable auto-opening browser
templar serve --no-open

# Enable verbose logging
templar serve --verbose
```

### Component Management

```bash
# List all components
templar list

# List with detailed information
templar list --verbose

# List in JSON format
templar list --format json

# List components with properties
templar list --with-props
```

### Component Preview

```bash
# Preview component with default props
templar preview Button

# Preview with custom props
templar preview Card --props '{"title":"Test Card","subtitle":"Example"}'

# Preview with mock data file
templar preview ProductList --mock ./mocks/products.json

# Preview with custom wrapper
templar preview Button --wrapper layouts/minimal.templ
```

### Build and Watch

```bash
# Build all components once
templar build

# Build for production
templar build --production

# Watch for changes and rebuild
templar watch

# Watch specific patterns
templar watch --include "**/*.templ" --exclude "*_test.templ"
```

## API Integration Examples

### REST API Usage

```bash
# Health check
curl http://localhost:8080/api/health

# List components
curl http://localhost:8080/api/components

# Get component details
curl http://localhost:8080/api/components/Button

# Preview component
curl -X POST http://localhost:8080/api/preview \
  -H "Content-Type: application/json" \
  -d '{"component":"Button","props":{"text":"API Test","variant":"success"}}'

# Build component
curl -X POST http://localhost:8080/api/build/Button

# Get build status
curl http://localhost:8080/api/build/status
```

### WebSocket Live Reload

```javascript
// Connect to live reload
const ws = new WebSocket('ws://localhost:8080/ws/reload');

ws.onopen = function() {
    console.log('Connected to live reload');
};

ws.onmessage = function(event) {
    const data = JSON.parse(event.data);
    console.log('Reload event:', data);
    
    if (data.type === 'component_updated') {
        // Reload specific component
        location.reload();
    }
};

ws.onclose = function() {
    console.log('Live reload disconnected');
    // Attempt to reconnect
    setTimeout(() => {
        location.reload();
    }, 1000);
};
```

## Testing Examples

### Component Testing

```go
// components/button_test.go
package components

import (
    "context"
    "strings"
    "testing"
)

func TestButton(t *testing.T) {
    tests := []struct {
        name     string
        props    ButtonProps
        contains []string
    }{
        {
            name: "basic button",
            props: ButtonProps{
                Text:    "Click me",
                Variant: "primary",
            },
            contains: []string{
                "Click me",
                "btn-primary",
                "<button",
            },
        },
        {
            name: "disabled button",
            props: ButtonProps{
                Text:     "Disabled",
                Disabled: true,
            },
            contains: []string{
                "disabled",
                "btn-disabled",
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var buf strings.Builder
            err := Button(tt.props).Render(context.Background(), &buf)
            if err != nil {
                t.Fatalf("Failed to render: %v", err)
            }

            html := buf.String()
            for _, want := range tt.contains {
                if !strings.Contains(html, want) {
                    t.Errorf("Expected HTML to contain %q, got: %s", want, html)
                }
            }
        })
    }
}
```

### Integration Testing

```bash
#!/bin/bash
# test/integration.sh

# Start templar server in background
templar serve --port 8081 &
SERVER_PID=$!

# Wait for server to start
sleep 2

# Test health endpoint
if ! curl -f http://localhost:8081/api/health; then
    echo "Health check failed"
    kill $SERVER_PID
    exit 1
fi

# Test component listing
if ! curl -f http://localhost:8081/api/components | jq '.components | length'; then
    echo "Component listing failed"
    kill $SERVER_PID
    exit 1
fi

# Test live reload WebSocket
node -e "
const WebSocket = require('ws');
const ws = new WebSocket('ws://localhost:8081/ws/reload');
ws.on('open', () => {
    console.log('WebSocket connected');
    ws.close();
    process.exit(0);
});
ws.on('error', (err) => {
    console.error('WebSocket failed:', err);
    process.exit(1);
});
"

# Cleanup
kill $SERVER_PID
echo "Integration tests passed"
```

## Performance Examples

### Benchmark Testing

```go
// performance/benchmark_test.go
package performance

import (
    "context"
    "strings"
    "testing"
    "templar/components"
)

func BenchmarkButtonRender(b *testing.B) {
    props := components.ButtonProps{
        Text:    "Benchmark",
        Variant: "primary",
        Size:    "large",
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        var buf strings.Builder
        _ = components.Button(props).Render(context.Background(), &buf)
    }
}

func BenchmarkConcurrentRender(b *testing.B) {
    props := components.ButtonProps{Text: "Test", Variant: "primary"}
    
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            var buf strings.Builder
            _ = components.Button(props).Render(context.Background(), &buf)
        }
    })
}
```

### Load Testing

```bash
#!/bin/bash
# test/load.sh

# Start server
templar serve --port 8082 &
SERVER_PID=$!
sleep 2

# Install hey if not available
if ! command -v hey &> /dev/null; then
    go install github.com/rakyll/hey@latest
fi

# Load test preview endpoint
hey -n 1000 -c 10 -m POST \
    -H "Content-Type: application/json" \
    -d '{"component":"Button","props":{"text":"Load test"}}' \
    http://localhost:8082/api/preview

# Load test component listing
hey -n 500 -c 5 http://localhost:8082/api/components

# Cleanup
kill $SERVER_PID
```

## Docker Examples

### Dockerfile for Production

```dockerfile
# Dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go generate ./...
RUN CGO_ENABLED=0 GOOS=linux go build -o templar .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/templar .
COPY --from=builder /app/components ./components/
COPY --from=builder /app/static ./static/

EXPOSE 8080
CMD ["./templar", "serve", "--host", "0.0.0.0"]
```

### Docker Compose for Development

```yaml
# docker-compose.yml
version: '3.8'

services:
  templar:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./components:/app/components
      - ./static:/app/static
    environment:
      - TEMPLAR_DEV=true
      - TEMPLAR_HOT_RELOAD=true
    command: ["./templar", "serve", "--host", "0.0.0.0"]

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
    depends_on:
      - templar
```

## CI/CD Examples

### GitHub Actions Workflow

```yaml
# .github/workflows/test.yml
name: Test Templar

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
        with:
          go-version: '1.24'
      
      - name: Install dependencies
        run: go mod tidy
      
      - name: Generate templates
        run: go generate ./...
      
      - name: Run tests
        run: go test -v ./...
      
      - name: Run benchmarks
        run: go test -bench=. ./...
      
      - name: Test templar CLI
        run: |
          go build -o templar .
          ./templar init test-project
          cd test-project
          ../templar list
          timeout 10s ../templar serve --no-open || true
```

### Deployment Script

```bash
#!/bin/bash
# deploy.sh

set -e

echo "Building Templar..."
go generate ./...
go build -o templar .

echo "Running tests..."
go test ./...

echo "Building Docker image..."
docker build -t templar:latest .

echo "Deploying to production..."
docker tag templar:latest registry.example.com/templar:$(git rev-parse --short HEAD)
docker push registry.example.com/templar:$(git rev-parse --short HEAD)

echo "Updating deployment..."
kubectl set image deployment/templar templar=registry.example.com/templar:$(git rev-parse --short HEAD)
kubectl rollout status deployment/templar

echo "Deployment complete!"
```

These examples demonstrate Templar's flexibility and power for rapid Go templ development with enterprise-grade features.