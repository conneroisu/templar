# Troubleshooting Guide

This guide covers common issues you might encounter while using Templar and how to resolve them.

## Quick Diagnostics

Before diving into specific issues, try these quick diagnostic steps:

```bash
# Check Templar version and installation
templar version

# Validate your configuration
templar config validate

# List discovered components
templar list

# Check if templ is installed and working
templ version
templ generate
```

## Installation Issues

### "templar: command not found"

**Cause**: Templar is not installed or not in your PATH.

**Solutions**:
1. **Reinstall Templar**:
   ```bash
   go install github.com/conneroisu/templar@latest
   ```

2. **Check your GOPATH**:
   ```bash
   echo $GOPATH
   echo $PATH
   # Ensure $GOPATH/bin is in your PATH
   export PATH=$PATH:$(go env GOPATH)/bin
   ```

3. **Use full path**:
   ```bash
   $(go env GOPATH)/bin/templar version
   ```

### "go: cannot find main module"

**Cause**: Trying to install from a directory that's not a Go module.

**Solution**:
```bash
# Install from anywhere
go install github.com/conneroisu/templar@latest

# Or initialize a module first if working locally
go mod init templar-dev
go install github.com/conneroisu/templar@latest
```

## Project Setup Issues

### "No configuration file found"

**Cause**: Missing `.templar.yml` configuration file.

**Solutions**:
1. **Initialize a new project**:
   ```bash
   templar init
   ```

2. **Create minimal configuration**:
   ```yaml
   # .templar.yml
   server:
     port: 8080
   components:
     scan_paths:
       - "./components"
   ```

3. **Use default configuration**:
   ```bash
   templar serve --use-defaults
   ```

### "Configuration validation failed"

**Cause**: Invalid YAML syntax or configuration values.

**Debug steps**:
```bash
# Validate YAML syntax
python -c "import yaml; yaml.safe_load(open('.templar.yml'))"

# Or use a YAML linter
yamllint .templar.yml

# Check Templar's validation
templar config validate --verbose
```

**Common YAML issues**:
- **Tabs instead of spaces**: Use only spaces for indentation
- **Missing colons**: Ensure `key: value` format
- **Incorrect nesting**: Check indentation levels
- **Unquoted special characters**: Quote strings with special characters

## Component Discovery Issues

### "No components found"

**Cause**: Templar can't find any templ components in the scanned directories.

**Debug steps**:
```bash
# Check if directories exist
ls -la components/
ls -la views/

# Check for .templ files
find . -name "*.templ" -type f

# Verify scan paths in config
grep -A5 "scan_paths" .templar.yml

# List with verbose output
templar list --verbose
```

**Solutions**:
1. **Create a test component**:
   ```go
   // components/test.templ
   package components
   
   templ Test() {
       <div>Hello, World!</div>
   }
   ```

2. **Update scan paths**:
   ```yaml
   components:
     scan_paths:
       - "./components"
       - "./internal/views"
       - "./pkg/templates"
   ```

3. **Check exclude patterns**:
   ```yaml
   components:
     exclude_patterns:
       - "*_test.templ"  # Make sure your files aren't excluded
   ```

### "Component 'X' not found"

**Cause**: Component exists but Templar can't find it.

**Debug steps**:
```bash
# Check exact filename and function name
grep -r "templ ComponentName" components/

# Verify the component compiles
templ generate

# Check if it's in excluded patterns
templar list | grep ComponentName
```

**Solutions**:
1. **Ensure function name matches**:
   ```go
   // File: components/button.templ
   templ Button() { ... }  // Function name must match
   ```

2. **Check package declaration**:
   ```go
   package components  // Must be correct package
   ```

3. **Verify file location**:
   ```bash
   # Component should be in scanned directory
   ls -la components/button.templ
   ```

## Build Issues

### "templ generate failed"

**Cause**: Syntax errors in templ files.

**Debug steps**:
```bash
# Run templ generate manually to see errors
cd components/
templ generate

# Check specific file
templ generate button.templ
```

**Common syntax errors**:
- **Missing braces**: `{ variable }` not `{{ variable }}`
- **Invalid Go syntax**: Check variable declarations and types
- **Missing imports**: Import required packages
- **Package mismatch**: Ensure package name matches directory

**Example fix**:
```go
// ❌ Wrong
templ Button(text string) {
    <button>{{ text }}</button>  // Wrong syntax
}

// ✅ Correct
templ Button(text string) {
    <button>{ text }</button>     // Correct syntax
}
```

### "Build command failed"

**Cause**: Custom build command in configuration is failing.

**Debug steps**:
```bash
# Run the build command manually
make generate  # or whatever your build command is

# Check build configuration
grep -A5 "build:" .templar.yml

# Test with default build command
templar build --command "templ generate"
```

**Solutions**:
1. **Use absolute paths in build commands**
2. **Ensure all tools are installed**
3. **Check working directory**

## Server Issues

### "Port already in use"

**Cause**: Another process is using the specified port.

**Solutions**:
```bash
# Find what's using the port
lsof -i :8080

# Kill the process
lsof -ti :8080 | xargs kill

# Use a different port
templar serve --port 3000

# Or configure in .templar.yml
server:
  port: 3000
```

### "Permission denied" when starting server

**Cause**: Trying to bind to a privileged port (<1024) without root access.

**Solutions**:
```bash
# Use unprivileged port (>1024)
templar serve --port 8080

# Or run with sudo (not recommended)
sudo templar serve --port 80
```

### "Server starts but browser shows 'connection refused'"

**Cause**: Server is binding to localhost but accessed from different host.

**Solutions**:
```bash
# Bind to all interfaces
templar serve --host 0.0.0.0

# Or specific interface
templar serve --host 192.168.1.100
```

## Hot Reload Issues

### "Changes not reflected in browser"

**Cause**: Hot reload not working properly.

**Debug steps**:
```bash
# Check browser console for WebSocket errors
# Open browser DevTools > Console

# Check configuration
grep -A5 "development:" .templar.yml

# Test WebSocket connection
curl -H "Upgrade: websocket" http://localhost:8080/ws
```

**Solutions**:
1. **Enable hot reload**:
   ```yaml
   development:
     hot_reload: true
   ```

2. **Check WebSocket connection**:
   - Browser console should show WebSocket connection
   - No firewall blocking WebSocket traffic

3. **Manual refresh**:
   ```bash
   # If hot reload fails, try manual refresh
   # Ctrl+F5 or Cmd+Shift+R
   ```

### "WebSocket connection failed"

**Cause**: WebSocket upgrade or connection issues.

**Debug steps**:
```bash
# Check if WebSocket endpoint exists
curl -I http://localhost:8080/ws

# Check for proxy interference
# Disable browser extensions
# Try incognito/private mode
```

**Solutions**:
1. **Check proxy settings**: Disable HTTP proxies
2. **Try different browser**: Test in Chrome/Firefox/Safari
3. **Check firewall**: Ensure WebSocket traffic is allowed

## Component Preview Issues

### "Component renders blank"

**Cause**: Component requires props but none provided.

**Solutions**:
```bash
# Provide required props
templar preview Button --props '{"text": "Click me"}'

# Use mock data
templar preview UserCard --mock ./mocks/user.json

# Check component requirements
templar list --with-props
```

### "Invalid props format"

**Cause**: JSON props are malformed.

**Debug steps**:
```bash
# Validate JSON
echo '{"text": "test"}' | jq .

# Check expected prop types
templar list UserCard --with-props
```

**Solutions**:
1. **Use proper JSON format**:
   ```bash
   # ✅ Correct
   templar preview Button --props '{"text": "Click me", "disabled": false}'
   
   # ❌ Wrong
   templar preview Button --props '{text: "Click me"}'  # Missing quotes
   ```

2. **Use mock files for complex props**:
   ```json
   // mocks/button.json
   {
     "text": "Click me",
     "variant": "primary",
     "disabled": false
   }
   ```

## Performance Issues

### "Slow component discovery"

**Cause**: Scanning too many directories or large files.

**Solutions**:
```yaml
# Optimize scan configuration
components:
  scan_paths:
    - "./components"  # Be specific
  exclude_patterns:
    - "node_modules/**"
    - "*.bak"
    - "vendor/**"
  scan_depth: 3  # Limit depth
  ignore_large_files: true
```

### "Slow build times"

**Cause**: Building too many components or inefficient build process.

**Solutions**:
```yaml
# Enable build caching
build:
  cache_enabled: true
  cache_dir: ".templar/cache"
  workers: 4  # Parallel builds

# Watch fewer file patterns
  watch_patterns:
    - "**/*.templ"  # Only watch what's needed
```

## Development Workflow Issues

### "Components not updating after changes"

**Cause**: File watcher not detecting changes or caching issues.

**Solutions**:
```bash
# Clear cache
rm -rf .templar/cache

# Restart server
# Ctrl+C to stop, then templar serve

# Force rebuild
templar build --force

# Check file watcher
templar watch --verbose
```

### "Error overlay not showing"

**Cause**: Error overlay disabled or JavaScript errors.

**Solutions**:
```yaml
# Enable error overlay
development:
  error_overlay: true
```

```bash
# Check browser console for JavaScript errors
# Ensure no ad blockers are interfering
```

## Environment-Specific Issues

### Docker/Container Issues

**Solutions**:
```dockerfile
# Ensure proper port exposure
EXPOSE 8080

# Use correct host binding
CMD ["templar", "serve", "--host", "0.0.0.0"]
```

### Windows-Specific Issues

**File watching issues**:
```bash
# Use polling instead of native file events
templar serve --poll

# Or configure in .templar.yml
build:
  watch_method: "poll"
```

**Path separator issues**:
```yaml
# Use forward slashes even on Windows
components:
  scan_paths:
    - "./components"  # Not ".\components"
```

### macOS-Specific Issues

**Permission issues**:
```bash
# Grant terminal full disk access
# System Preferences > Security & Privacy > Privacy > Full Disk Access

# Or use different directory
mkdir ~/templar-projects
cd ~/templar-projects
templar init
```

## Getting More Help

### Enable Verbose Logging

```bash
# Run commands with verbose output
templar serve --verbose
templar build --verbose
templar list --verbose

# Enable debug logging
export TEMPLAR_LOG_LEVEL=debug
templar serve
```

### Collect Diagnostic Information

```bash
# System information
go version
templ version
templar version

# Project information
cat .templar.yml
ls -la components/
templar list
templar config validate

# Log files
cat ~/.templar/logs/templar.log
```

### Report Issues

When reporting bugs, include:

1. **System information**: OS, Go version, Templar version
2. **Configuration**: Your `.templar.yml` file
3. **Steps to reproduce**: Exact commands run
4. **Expected vs actual behavior**
5. **Error messages**: Full error output
6. **Logs**: Relevant log entries

```bash
# Create a bug report
templar debug report > debug-report.txt
```

### Community Resources

- **GitHub Issues**: [Report bugs](https://github.com/conneroisu/templar/issues)
- **Discussions**: [Ask questions](https://github.com/conneroisu/templar/discussions)
- **Documentation**: [Read the docs](https://templar.dev)
- **Examples**: [Browse examples](https://github.com/conneroisu/templar/tree/main/examples)

## Prevention Tips

### Best Practices

1. **Use version control**: Track your `.templar.yml` file
2. **Validate configuration**: Run `templar config validate` regularly
3. **Keep dependencies updated**: Update Go, templ, and Templar regularly
4. **Monitor logs**: Check for warnings in verbose output
5. **Test in clean environment**: Use containers or VMs for testing

### Common Pitfalls

1. **Don't use tabs in YAML**: Always use spaces
2. **Quote special characters**: Use quotes for strings with colons, etc.
3. **Keep paths relative**: Use `./components` not `/absolute/paths`
4. **Watch file limits**: Be careful with recursive watching
5. **Check port conflicts**: Use `lsof` to check port usage

---

Still having issues? [Open an issue](https://github.com/conneroisu/templar/issues) with your diagnostic information!