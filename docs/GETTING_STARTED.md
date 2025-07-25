# Getting Started with Templar

This guide will walk you through everything you need to know to get started with Templar, from installation to building your first components.

## Prerequisites

- **Go 1.21 or later** - [Install Go](https://golang.org/doc/install)
- **templ CLI** - Install with `go install github.com/a-h/templ/cmd/templ@latest`
- **Basic familiarity with Go and HTML** - Understanding of Go syntax and HTML structure

## Installation

### Option 1: Install from Source (Recommended)

```bash
go install github.com/conneroisu/templar@latest
```

### Option 2: Download Binary

```bash
# Linux
curl -L https://github.com/conneroisu/templar/releases/latest/download/templar-linux-amd64 -o templar
chmod +x templar && sudo mv templar /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/conneroisu/templar/releases/latest/download/templar-darwin-amd64 -o templar
chmod +x templar && sudo mv templar /usr/local/bin/

# macOS (Apple Silicon)
curl -L https://github.com/conneroisu/templar/releases/latest/download/templar-darwin-arm64 -o templar
chmod +x templar && sudo mv templar /usr/local/bin/

# Windows
# Download templar-windows-amd64.exe from releases page
```

### Verify Installation

```bash
templar version
# Should output version information
```

## Your First Project

### Step 1: Create a New Project

```bash
# Create a new project with examples
templar init my-first-project

# Navigate to the project
cd my-first-project
```

This creates a project structure like:

```
my-first-project/
├── .templar.yml              # Configuration file
├── components/               # Your templ components
│   ├── button.templ         # Example button component
│   ├── card.templ           # Example card component
│   └── layout.templ         # Example layout component
├── examples/                # Generated example files
├── static/                  # Static assets (CSS, JS, images)
│   └── style.css           # Basic CSS styles
├── mocks/                   # Mock data for testing
│   └── example.json        # Example mock data
└── README.md               # Project-specific README
```

### Step 2: Explore the Example Components

Let's look at the generated button component:

```go
// components/button.templ
package components

type ButtonVariant string

const (
    ButtonPrimary   ButtonVariant = "primary"
    ButtonSecondary ButtonVariant = "secondary"
    ButtonDanger    ButtonVariant = "danger"
)

templ Button(text string, variant ButtonVariant, onclick string) {
    <button 
        class={ "btn", "btn-" + string(variant) }
        onclick={ onclick }
        type="button"
    >
        { text }
    </button>
}
```

### Step 3: Start the Development Server

```bash
templar serve
```

This will:
- Start a development server on `http://localhost:8080`
- Automatically open your browser
- Enable hot reload for instant updates
- Set up WebSocket connection for live updates

### Step 4: Preview Your Components

In your browser, you'll see the Templar dashboard with:
- List of discovered components
- Preview buttons for each component
- Links to component source files

Or use the CLI:

```bash
# List all components
templar list

# Preview the button component
templar preview Button

# Preview with custom props
templar preview Button --props '{"text": "Click Me!", "variant": "primary", "onclick": "alert(\"Hello!\")"}'
```

## Creating Your First Component

Let's create a simple user profile component from scratch:

### Step 1: Create the Component File

```bash
# Create components/user-profile.templ
```

```go
// components/user-profile.templ
package components

type User struct {
    Name     string `json:"name"`
    Email    string `json:"email"`
    Avatar   string `json:"avatar"`
    Role     string `json:"role"`
    IsOnline bool   `json:"isOnline"`
}

templ UserProfile(user User) {
    <div class="user-profile">
        <div class="user-avatar">
            <img src={ user.Avatar } alt={ user.Name } />
            if user.IsOnline {
                <span class="status-indicator online"></span>
            } else {
                <span class="status-indicator offline"></span>
            }
        </div>
        <div class="user-info">
            <h3 class="user-name">{ user.Name }</h3>
            <p class="user-email">{ user.Email }</p>
            <span class="user-role">{ user.Role }</span>
        </div>
    </div>
}
```

### Step 2: Add CSS Styles

```css
/* static/style.css - Add these styles */
.user-profile {
    display: flex;
    align-items: center;
    padding: 1rem;
    border: 1px solid #e2e8f0;
    border-radius: 8px;
    background: white;
    max-width: 300px;
}

.user-avatar {
    position: relative;
    margin-right: 1rem;
}

.user-avatar img {
    width: 60px;
    height: 60px;
    border-radius: 50%;
    object-fit: cover;
}

.status-indicator {
    position: absolute;
    bottom: 0;
    right: 0;
    width: 16px;
    height: 16px;
    border-radius: 50%;
    border: 2px solid white;
}

.status-indicator.online {
    background-color: #10b981;
}

.status-indicator.offline {
    background-color: #6b7280;
}

.user-info {
    flex: 1;
}

.user-name {
    margin: 0 0 0.25rem 0;
    font-size: 1.125rem;
    font-weight: 600;
    color: #1f2937;
}

.user-email {
    margin: 0 0 0.5rem 0;
    font-size: 0.875rem;
    color: #6b7280;
}

.user-role {
    font-size: 0.75rem;
    font-weight: 500;
    color: #3b82f6;
    background-color: #dbeafe;
    padding: 0.25rem 0.5rem;
    border-radius: 9999px;
}
```

### Step 3: Create Mock Data

```json
// mocks/user-profile.json
{
  "name": "Alice Johnson",
  "email": "alice@example.com",
  "avatar": "https://images.unsplash.com/photo-1494790108755-2616b612c341?w=150&h=150&fit=crop&crop=face",
  "role": "Senior Developer",
  "isOnline": true
}
```

### Step 4: Preview Your Component

Save your files, and Templar will automatically detect the new component. You can then:

```bash
# Preview with mock data
templar preview UserProfile --mock ./mocks/user-profile.json

# Or preview with inline props
templar preview UserProfile --props '{
  "name": "Bob Smith",
  "email": "bob@example.com", 
  "avatar": "https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=150&h=150&fit=crop&crop=face",
  "role": "Designer",
  "isOnline": false
}'
```

## Understanding Component Development

### Hot Reload

As you save changes to your components, Templar automatically:
1. **Detects file changes** using file system watchers
2. **Rebuilds components** using `templ generate`
3. **Sends updates** via WebSocket to connected browsers
4. **Refreshes the preview** without losing component state

### Component Props

Templar automatically detects component parameters and their types:

```go
// Simple types
templ Button(text string, disabled bool) { ... }

// Struct types
templ UserCard(user User, showActions bool) { ... }

// Optional parameters with defaults
templ Alert(message string, variant AlertType) {
    // Use variant or default to "info"
}
```

### Mock Data Workflow

1. **Create mock files** in the `mocks/` directory
2. **Use realistic data** that matches your component props
3. **Test edge cases** with different mock scenarios
4. **Validate JSON** with `templar config validate`

## Configuration

### Basic Configuration

The `.templar.yml` file controls Templar's behavior:

```yaml
# .templar.yml
server:
  port: 8080
  auto_open: true

components:
  scan_paths:
    - "./components"
  exclude_patterns:
    - "*_test.templ"

build:
  command: "templ generate"
  watch_patterns:
    - "**/*.templ"
    - "**/*.go"

development:
  hot_reload: true
  error_overlay: true
```

### Custom Build Commands

If you have a custom build process:

```yaml
build:
  command: "make generate"  # Use your custom command
  pre_build: ["go mod tidy"]  # Commands to run before build
  post_build: ["go fmt ./..."]  # Commands to run after build
```

## Working with Existing Projects

### Adding Templar to an Existing Go Project

1. **Initialize Templar in your project**:
   ```bash
   templar init --minimal
   ```

2. **Update configuration** to point to your component directories:
   ```yaml
   components:
     scan_paths:
       - "./internal/views"
       - "./pkg/components"
       - "./web/templates"
   ```

3. **Start the development server**:
   ```bash
   templar serve
   ```

### Integration with Existing Build Tools

#### With Make

```makefile
# Makefile
.PHONY: dev
dev:
	templar serve

.PHONY: components
components:
	templar build

.PHONY: preview
preview:
	templar preview $(COMPONENT)
```

#### With npm/yarn

```json
{
  "scripts": {
    "dev": "templar serve",
    "build:components": "templar build",
    "preview": "templar preview"
  }
}
```

## Next Steps

Now that you have Templar set up and running:

1. **Explore the CLI commands** - Run `templar help` to see all available commands
2. **Read the Configuration Guide** - Learn about advanced configuration options
3. **Check out the Examples** - Look at the `examples/` directory for more complex components
4. **Set up your workflow** - Integrate Templar with your existing development tools
5. **Build real components** - Start creating the components you need for your project

## Common First-Time Issues

### Component Not Found

**Problem**: "Component 'MyComponent' not found"

**Solutions**:
- Ensure the templ function name matches the component name
- Check that the file is in a scanned directory
- Verify the component compiles with `templ generate`
- Run `templar list` to see discovered components

### Hot Reload Not Working

**Problem**: Changes don't appear in the browser

**Solutions**:
- Check browser console for WebSocket errors
- Ensure `hot_reload: true` in configuration
- Verify no firewall is blocking WebSocket connections
- Try refreshing the browser manually

### Build Errors

**Problem**: Components fail to build

**Solutions**:
- Run `templ generate` manually to see detailed errors
- Check Go syntax in your templ files
- Ensure all imports are valid
- Review the error overlay in the browser

### Port Already in Use

**Problem**: "Port 8080 already in use"

**Solutions**:
- Use a different port: `templar serve --port 3000`
- Check what's using the port: `lsof -i :8080`
- Kill the conflicting process or choose another port

## Getting Help

- **CLI Help**: `templar help [command]`
- **Configuration Help**: `templar config --help`
- **GitHub Issues**: [Report bugs or request features](https://github.com/conneroisu/templar/issues)
- **Discussions**: [Ask questions and share ideas](https://github.com/conneroisu/templar/discussions)

Welcome to the Templar community! 🎉