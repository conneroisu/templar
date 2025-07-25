# Templar

**Rapid prototyping and development toolkit for Go templ components**

Templar is a powerful CLI tool designed to streamline the development of [templ](https://templ.guide/) components. It provides real-time browser preview, hot reload, component scaffolding, and a comprehensive development server - everything you need to build and iterate on templ components quickly.

## ✨ Features

- 🚀 **Instant Preview** - See your components in the browser with live reload
- 🔥 **Hot Reload** - Changes reflect immediately without manual refresh
- 📁 **Component Discovery** - Automatically finds and catalogs your templ components
- 🎨 **Mock Data Support** - Test components with realistic data
- 🔧 **Built-in Dev Server** - Production-ready HTTP server with WebSocket support
- 📦 **Project Scaffolding** - Get started quickly with templates and examples
- 🎯 **Smart Error Handling** - Detailed error messages with suggestions
- ⚡ **Performance Optimized** - Fast builds with caching and worker pools

## 🚀 Quick Start

### Installation

```bash
# Install from source (requires Go 1.21+)
go install github.com/conneroisu/templar@latest

# Or download from releases
curl -L https://github.com/conneroisu/templar/releases/latest/download/templar-linux-amd64 -o templar
chmod +x templar && sudo mv templar /usr/local/bin/
```

### Create Your First Project

```bash
# Create a new project with examples
templar init my-components

# Or start minimal
templar init my-components --minimal

# Navigate to your project
cd my-components
```

### Start Development Server

```bash
# Start the development server (opens browser automatically)
templar serve

# Or specify a custom port
templar serve --port 3000
```

That's it! Your development server is running at `http://localhost:8080` with live reload enabled.

## 📖 User Guide

### Project Structure

After running `templar init`, you'll have a structure like this:

```
my-components/
├── .templar.yml          # Configuration file
├── components/           # Your templ components
│   ├── button.templ
│   └── card.templ
├── examples/             # Generated preview examples
├── static/              # Static assets (CSS, JS, images)
└── README.md
```

### Creating Components

Create a new templ component in the `components/` directory:

```go
// components/hello.templ
package components

templ Hello(name string) {
    <div class="hello">
        <h1>Hello, { name }!</h1>
        <p>Welcome to Templar</p>
    </div>
}
```

Templar will automatically discover this component and make it available for preview.

### Previewing Components

```bash
# List all discovered components
templar list

# Preview a specific component
templar preview Hello

# Preview with props
templar preview Hello --props '{"name": "World"}'

# Preview with mock data file
templar preview Hello --mock ./mocks/hello.json
```

### Configuration

Edit `.templar.yml` to customize your setup:

```yaml
# .templar.yml
server:
  port: 8080
  host: localhost
  auto_open: true

components:
  scan_paths:
    - "./components"
    - "./views"
  exclude_patterns:
    - "*_test.templ"

build:
  command: "templ generate"
  watch_patterns:
    - "**/*.templ"
    - "**/*.go"

development:
  hot_reload: true
  css_injection: true
  error_overlay: true
```

## 🔧 Commands Reference

### Project Management

| Command | Description | Example |
|---------|-------------|---------|
| `templar init [name]` | Create new project | `templar init my-app` |
| `templar init --minimal` | Create minimal project | `templar init --minimal` |
| `templar init --template blog` | Use specific template | `templar init --template blog` |

### Development Server

| Command | Description | Example |
|---------|-------------|---------|
| `templar serve` | Start development server | `templar serve` |
| `templar serve --port 3000` | Use custom port | `templar serve --port 3000` |
| `templar serve --no-open` | Don't open browser | `templar serve --no-open` |

### Component Management

| Command | Description | Example |
|---------|-------------|---------|
| `templar list` | List all components | `templar list` |
| `templar list --json` | Output as JSON | `templar list --json` |
| `templar list --with-props` | Include component props | `templar list --with-props` |

### Component Preview

| Command | Description | Example |
|---------|-------------|---------|
| `templar preview Button` | Preview component | `templar preview Button` |
| `templar preview Card --props '{...}'` | Preview with props | `templar preview Card --props '{"title":"Test"}'` |
| `templar preview Card --mock file.json` | Preview with mock data | `templar preview Card --mock ./mocks/card.json` |

### Build & Watch

| Command | Description | Example |
|---------|-------------|---------|
| `templar build` | Build all components | `templar build` |
| `templar build --production` | Production build | `templar build --production` |
| `templar watch` | Watch for changes | `templar watch` |

## 🎯 Common Workflows

### Developing a New Component

1. **Create the component**:
   ```bash
   # Create components/button.templ
   ```

2. **Start development server**:
   ```bash
   templar serve
   ```

3. **Preview your component**:
   ```bash
   templar preview Button --props '{"text": "Click me", "variant": "primary"}'
   ```

4. **Make changes** - the browser will automatically reload

### Working with Mock Data

Create mock data files to test your components:

```json
// mocks/user-card.json
{
  "user": {
    "name": "John Doe",
    "email": "john@example.com",
    "avatar": "https://via.placeholder.com/150"
  },
  "isOnline": true
}
```

```bash
templar preview UserCard --mock ./mocks/user-card.json
```

### Building for Production

```bash
# Build optimized version
templar build --production

# The generated files are ready for deployment
```

## 🛠️ Configuration Guide

### Server Configuration

```yaml
server:
  port: 8080                    # Server port
  host: "localhost"             # Server host
  auto_open: true               # Open browser automatically
  middleware: ["cors", "logging"] # Enable middleware
```

### Component Discovery

```yaml
components:
  scan_paths:                   # Directories to scan
    - "./components"
    - "./views"
    - "./examples"
  exclude_patterns:             # Patterns to ignore
    - "*_test.templ"
    - "*.bak"
    - "node_modules/**"
```

### Build Configuration

```yaml
build:
  command: "templ generate"     # Build command
  watch_patterns:               # Files to watch
    - "**/*.templ"
    - "**/*.go"
    - "**/*.css"
  cache_dir: ".templar/cache"   # Cache directory
```

### Development Features

```yaml
development:
  hot_reload: true              # Enable hot reload
  css_injection: true           # Inject CSS changes
  error_overlay: true           # Show errors in browser
  source_maps: true             # Generate source maps
```

## 🔍 Troubleshooting

### Common Issues

**Server won't start**
- Check if port is already in use: `lsof -i :8080`
- Try a different port: `templar serve --port 3000`

**Components not found**
- Verify component paths in `.templar.yml`
- Check component syntax with `templ generate`
- Run `templar list` to see discovered components

**Hot reload not working**
- Ensure WebSocket connection is established
- Check browser console for errors
- Verify `hot_reload: true` in configuration

**Build errors**
- Check component syntax: `templ generate`
- Review error overlay in browser
- Verify all dependencies are installed

### Getting Help

```bash
# Show help for any command
templar help
templar serve --help
templar preview --help

# Check configuration
templar config validate

# View build logs
templar build --verbose
```

### Error Messages

Templar provides detailed error messages with suggestions:

```
Error: Component 'Button' not found

Suggestions:
  • Check if the component file exists in the scanned directories
  • Verify the component name matches the function name
  • Run 'templar list' to see all discovered components
  • Check your .templar.yml scan_paths configuration

Available components: Card, Header, Footer
```

## 🏗️ Advanced Usage

### Custom Templates

Create project templates for common patterns:

```bash
# Create from custom template
templar init --template https://github.com/user/templar-template

# Or use local template
templar init --template ./my-template
```

### Integration with Build Tools

Use Templar in your existing build pipeline:

```bash
# In package.json
{
  "scripts": {
    "dev": "templar serve",
    "build": "templar build --production",
    "preview": "templar preview"
  }
}
```

### Performance Optimization

For large projects, optimize Templar's performance:

```yaml
# .templar.yml
build:
  workers: 4                    # Parallel build workers
  cache_enabled: true           # Enable build caching
  
components:
  scan_depth: 3                 # Limit directory scan depth
  ignore_large_files: true      # Skip files >1MB
```

## 📚 Examples

### Basic Component
```go
// components/alert.templ
package components

type AlertType string

const (
    AlertSuccess AlertType = "success"
    AlertWarning AlertType = "warning"
    AlertError   AlertType = "error"
)

templ Alert(message string, alertType AlertType) {
    <div class={ "alert", "alert-" + string(alertType) }>
        { message }
    </div>
}
```

### Component with Children
```go
// components/card.templ
package components

templ Card(title string) {
    <div class="card">
        <div class="card-header">
            <h3>{ title }</h3>
        </div>
        <div class="card-body">
            { children... }
        </div>
    </div>
}
```

### Using Context
```go
// components/user-info.templ
package components

import "context"

templ UserInfo() {
    @Header("User Information")
    <div>
        if user := ctx.Value("user"); user != nil {
            <p>Welcome, { user.(string) }</p>
        } else {
            <p>Please log in</p>
        }
    </div>
}
```

## 🤝 Contributing

We welcome contributions! See our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Setup

```bash
# Clone the repository
git clone https://github.com/conneroisu/templar.git
cd templar

# Install dependencies
go mod tidy

# Run tests
make test

# Start development
make dev
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [templ](https://templ.guide/) - The amazing Go templating language
- [Cobra](https://cobra.dev/) - CLI framework
- All our [contributors](https://github.com/conneroisu/templar/contributors)

---

**Made with ❤️ for the Go and templ community**

For more examples and detailed documentation, visit our [Documentation Site](https://templar.dev) or check out the [examples directory](./examples).