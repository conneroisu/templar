# Templar Error Code Reference

This document provides a comprehensive reference of all error codes used in Templar, their meanings, common causes, and resolution steps.

## Error Categories

Templar uses a structured error system with categorized error types for better debugging and user experience:

- **Validation Errors** (`validation`) - Recoverable errors from invalid user input or configuration
- **Security Errors** (`security`) - Non-recoverable security violations and protection triggers  
- **I/O Errors** (`io`) - File system and data access errors
- **Network Errors** (`network`) - Network communication and WebSocket errors
- **Build Errors** (`build`) - Component compilation and build pipeline errors
- **Configuration Errors** (`config`) - Invalid configuration or settings
- **Internal Errors** (`internal`) - System-level errors and unexpected conditions

## Error Code Format

Error codes follow the pattern: `ERR_<CATEGORY>_<OPERATION>` or use predefined constants for common errors.

Examples:
- `ERR_BUILD_COMPILE` - Build compilation error
- `ERR_NETWORK_WEBSOCKET_SEND_MESSAGE` - WebSocket message sending error
- `ERR_SECURITY_PATH_TRAVERSAL` - Path traversal security violation

---

## Core Error Codes

### Validation Errors

#### ERR_INVALID_PATH
**Category:** Validation  
**Recoverable:** Yes  
**Description:** Invalid file or directory path provided

**Example Message:**
```
[ERR_INVALID_PATH] invalid path: ../../../etc/passwd
```

**Common Causes:**
- Path contains invalid characters
- Path format is incorrect for the operating system
- Path is empty or malformed

**Resolution:**
1. Verify the path format is correct
2. Use absolute paths when possible
3. Ensure path follows OS conventions (forward/backward slashes)
4. Check for typos in directory or file names

---

#### ERR_COMPONENT_NOT_FOUND
**Category:** Validation  
**Recoverable:** Yes  
**Description:** Requested component could not be found

**Example Message:**
```
[ERR_COMPONENT_NOT_FOUND] component not found: ButtonGroup
```

**Common Causes:**
- Component name is misspelled
- Component file doesn't exist in scan paths
- Component hasn't been registered yet

**Resolution:**
1. Check component name spelling
2. Verify component exists in configured scan paths
3. Run `templar list` to see available components
4. Ensure component file has `.templ` extension

---

#### ERR_VALIDATION_FAILED
**Category:** Validation  
**Recoverable:** Yes  
**Description:** General validation failure with multiple field errors

**Example Message:**
```
[ERR_VALIDATION_FAILED] validation failed with 2 errors
```

**Common Causes:**
- Multiple form fields have invalid values
- Configuration contains several invalid settings
- Command arguments are malformed

**Resolution:**
1. Check the detailed error message for specific field errors
2. Validate each field mentioned in the error
3. Use `--help` flag to see valid options
4. Review configuration file syntax

---

### Security Errors

#### ERR_PATH_TRAVERSAL
**Category:** Security  
**Recoverable:** No  
**Description:** Path traversal attack attempt detected

**Example Message:**
```
[ERR_PATH_TRAVERSAL] path traversal attempt: ../../../etc/passwd
```

**Common Causes:**
- Malicious path injection attempt
- Incorrect path construction in code
- User input contains `../` sequences

**Resolution:**
1. **DO NOT** override this error - it's a security protection
2. Use only trusted, validated paths
3. Avoid user-controlled path inputs
4. Report if you believe this is a false positive

---

#### ERR_COMMAND_INJECTION
**Category:** Security  
**Recoverable:** No  
**Description:** Command injection attempt detected

**Example Message:**
```
[ERR_COMMAND_INJECTION] command injection attempt: templ && rm -rf /
```

**Common Causes:**
- Malicious command injection attempt
- Unsafe command construction
- User input contains shell metacharacters

**Resolution:**
1. **DO NOT** override this error - it's a security protection
2. Use only allowed commands from the allowlist
3. Avoid user-controlled command inputs
4. Report if you believe this is a false positive

---

#### ERR_INVALID_ORIGIN
**Category:** Security  
**Recoverable:** No  
**Description:** Invalid origin detected in WebSocket or HTTP request

**Example Message:**
```
[ERR_INVALID_ORIGIN] invalid origin: https://malicious-site.com
```

**Common Causes:**
- Request from unauthorized domain
- CORS policy violation
- Potential CSRF attack

**Resolution:**
1. Ensure requests come from authorized origins
2. Check browser console for CORS errors
3. Verify development server is accessed from correct URL
4. For local development, use `localhost` or `127.0.0.1`

---

### Build Errors

#### ERR_BUILD_FAILED
**Category:** Build  
**Recoverable:** Yes  
**Description:** Component build or compilation failed

**Example Message:**
```
[ERR_BUILD_FAILED] component:Button build failed for component: Button
```

**Common Causes:**
- Syntax errors in templ files
- Missing dependencies
- Invalid Go code in component
- Template compilation errors

**Resolution:**
1. Check the component file for syntax errors
2. Verify all imports are available
3. Run `templ generate` manually to see detailed errors
4. Check Go syntax in code blocks within templates

---

#### ERR_PIPELINE_NOT_STARTED
**Category:** Build  
**Recoverable:** Yes  
**Description:** Build pipeline is not running

**Example Message:**
```
[ERR_PIPELINE_NOT_STARTED] pipeline is not started
```

**Common Causes:**
- Build pipeline hasn't been initialized
- Pipeline was stopped unexpectedly
- System resource constraints

**Resolution:**
1. Restart the development server with `templar serve`
2. Check system resources (memory, CPU)
3. Review logs for pipeline startup errors
4. Try `templar build` to test build system

---

#### ERR_PIPELINE_ALREADY_STARTED
**Category:** Build  
**Recoverable:** Yes  
**Description:** Attempting to start an already running pipeline

**Example Message:**
```
[ERR_PIPELINE_ALREADY_STARTED] pipeline is already started
```

**Common Causes:**
- Multiple server instances running
- Race condition in startup code
- Previous shutdown didn't complete

**Resolution:**
1. Stop existing server instances
2. Check for running `templar serve` processes
3. Wait a moment and retry
4. Use `templar doctor` to check system state

---

### Configuration Errors

#### ERR_CONFIG_INVALID
**Category:** Configuration  
**Recoverable:** No  
**Description:** Configuration file contains invalid settings

**Example Message:**
```
[ERR_CONFIG_INVALID] invalid configuration for port: must be between 1024 and 65535
```

**Common Causes:**
- Invalid YAML syntax in `.templar.yml`
- Configuration values outside valid ranges
- Missing required configuration fields
- Incompatible configuration options

**Resolution:**
1. Validate YAML syntax with a YAML validator
2. Check configuration values against documentation
3. Use `templar config` to see current configuration
4. Compare with example configuration files

---

### Network Errors

#### ERR_NETWORK_WEBSOCKET_SEND_MESSAGE
**Category:** Network  
**Recoverable:** Yes  
**Description:** Failed to send WebSocket message

**Example Message:**
```
[ERR_NETWORK_WEBSOCKET_SEND_MESSAGE] network WEBSOCKET_SEND_MESSAGE failed for client123: connection closed
```

**Common Causes:**
- WebSocket connection was closed
- Network connectivity issues
- Client disconnected
- Message too large

**Resolution:**
1. Check network connectivity
2. Refresh browser to reconnect
3. Check browser console for WebSocket errors
4. Verify server is still running

---

#### ERR_NETWORK_SERVER_BIND_PORT
**Category:** Network  
**Recoverable:** Yes  
**Description:** Failed to bind to network port

**Example Message:**
```
[ERR_NETWORK_SERVER_BIND_PORT] network SERVER_BIND_PORT failed for localhost: port 8080 already in use
```

**Common Causes:**
- Port already in use by another process
- Insufficient permissions to bind port
- Invalid port number
- Firewall blocking port

**Resolution:**
1. Use a different port with `--port` flag
2. Stop process using the port: `lsof -ti:8080 | xargs kill`
3. Use port > 1024 to avoid permission issues
4. Check firewall settings

---

### I/O Errors

#### ERR_FILE_NOT_FOUND
**Category:** I/O  
**Recoverable:** No  
**Description:** Required file could not be found

**Example Message:**
```
[ERR_FILE_NOT_FOUND] file not found: components/button.templ
```

**Common Causes:**
- File was deleted or moved
- Incorrect file path
- File permissions prevent access
- Working directory changed

**Resolution:**
1. Verify file exists at specified path
2. Check file permissions (readable)
3. Ensure working directory is correct
4. Use absolute paths if relative paths fail

---

#### ERR_PERMISSION_DENIED
**Category:** I/O  
**Recoverable:** No  
**Description:** Insufficient permissions to access file or directory

**Example Message:**
```
[ERR_PERMISSION_DENIED] permission denied: /root/components/
```

**Common Causes:**
- File/directory permissions too restrictive
- Running as wrong user
- SELinux or AppArmor restrictions
- File locked by another process

**Resolution:**
1. Check file/directory permissions with `ls -la`
2. Ensure user has read/write access
3. Use `chmod` to fix permissions if needed
4. Run with appropriate user privileges

---

### CLI Command Errors

#### ERR_CLI_SERVE
**Category:** Validation  
**Recoverable:** Yes  
**Description:** Error in serve command execution

**Example Message:**
```
[ERR_CLI_SERVE] command 'serve' failed: invalid port configuration
```

**Common Causes:**
- Invalid command flags
- Missing required arguments
- Configuration conflicts
- System constraints

**Resolution:**
1. Check command usage with `templar serve --help`
2. Verify all required flags are provided
3. Check configuration file for conflicts
4. Use `templar doctor` to diagnose issues

---

### Component and Registry Errors

#### ERR_COMPONENT_PARSE
**Category:** Build  
**Recoverable:** Yes  
**Description:** Failed to parse component file

**Example Message:**
```
[ERR_COMPONENT_PARSE] component Button PARSE failed: syntax error at line 15
```

**Common Causes:**
- Invalid templ syntax
- Malformed HTML in template
- Incorrect Go code in template
- Missing closing tags

**Resolution:**
1. Check component file syntax at reported line
2. Validate HTML structure
3. Verify Go code in template blocks
4. Use editor with templ syntax highlighting

---

#### ERR_COMPONENT_SCAN_DIRECTORY
**Category:** Build  
**Recoverable:** Yes  
**Description:** Failed to scan directory for components

**Example Message:**
```
[ERR_COMPONENT_SCAN_DIRECTORY] component scanner SCAN_DIRECTORY failed: permission denied
```

**Common Causes:**
- Directory permissions prevent access
- Directory doesn't exist
- Network file system issues
- Symbolic link loops

**Resolution:**
1. Check directory exists and is readable
2. Verify scan path configuration
3. Check directory permissions
4. Avoid circular symbolic links

---

#### ERR_COMPONENT_REGISTRY_REGISTER
**Category:** Build  
**Recoverable:** Yes  
**Description:** Failed to register component in registry

**Example Message:**
```
[ERR_COMPONENT_REGISTRY_REGISTER] component Button REGISTRY_REGISTER failed: duplicate component name
```

**Common Causes:**
- Component name conflicts
- Registry corruption
- Concurrent registration attempts
- Invalid component metadata

**Resolution:**
1. Ensure component names are unique
2. Restart development server to reset registry
3. Check for duplicate component files
4. Verify component file is valid

---

### Field Validation Errors

#### ERR_FIELD_PORT
**Category:** Validation  
**Recoverable:** Yes  
**Description:** Invalid port number in field validation

**Example Message:**
```
[ERR_FIELD_PORT] validation error in field 'port': must be between 1024 and 65535
```

**Common Causes:**
- Port number out of valid range
- Non-numeric port value
- Reserved port attempted

**Resolution:**
1. Use port number between 1024-65535
2. Ensure value is numeric
3. Avoid system reserved ports (0-1023)

---

#### ERR_FIELD_PROJECT_NAME
**Category:** Validation  
**Recoverable:** Yes  
**Description:** Invalid project name in field validation

**Example Message:**
```
[ERR_FIELD_PROJECT_NAME] validation error in field 'project_name': contains invalid characters
```

**Common Causes:**
- Project name contains special characters
- Name is too long or too short
- Reserved name used

**Resolution:**
1. Use alphanumeric characters and hyphens only
2. Keep name length reasonable (3-50 characters)
3. Avoid reserved names like "templar", "test"

---

### Internal Errors

#### ERR_INTERNAL
**Category:** Internal  
**Recoverable:** No  
**Description:** Unexpected internal system error

**Example Message:**
```
[ERR_INTERNAL] unexpected error occurred: nil pointer dereference
```

**Common Causes:**
- Programming errors (bugs)
- System resource exhaustion
- Unexpected system state
- Race conditions

**Resolution:**
1. Report the error with full context
2. Restart the application
3. Check system resources
4. Review recent changes that might cause the issue

---

#### ERR_MULTIPLE_ERRORS
**Category:** Internal  
**Recoverable:** No  
**Description:** Multiple errors occurred simultaneously

**Example Message:**
```
[ERR_MULTIPLE_ERRORS] multiple errors occurred: 3 errors
```

**Common Causes:**
- Cascading failures
- System instability
- Resource exhaustion
- Multiple concurrent operations failing

**Resolution:**
1. Check individual errors in the error details
2. Address root cause of failures
3. Restart system if errors persist
4. Review system logs for patterns

---

## Dynamic Error Codes

Some error codes are generated dynamically based on operations:

### Service Layer Patterns
- `ERR_INIT_<OPERATION>` - Initialization service errors
- `ERR_BUILD_<OPERATION>` - Build service errors  
- `ERR_SERVE_<OPERATION>` - Serve service errors

### Data Layer Patterns
- `ERR_DATA_<OPERATION>` - Data access errors (READ, WRITE, DELETE)

### Network Patterns
- `ERR_NETWORK_<OPERATION>` - Network operation errors

### Component Patterns
- `ERR_COMPONENT_<OPERATION>` - Component-specific errors

### Security Patterns
- `ERR_SECURITY_<OPERATION>` - Security violation errors

---

## Troubleshooting Tips

### General Debug Steps

1. **Enable Verbose Logging**
   ```bash
   templar serve --verbose
   ```

2. **Check System Health**
   ```bash
   templar doctor
   ```

3. **Validate Configuration**
   ```bash
   templar config --validate
   ```

4. **Review Available Components**
   ```bash
   templar list --detailed
   ```

### Security Error Responses

Security errors are designed to protect the system and should not be bypassed:

- **DO NOT** ignore security errors
- Report false positives through proper channels
- Review security documentation before making changes
- Use security testing tools to validate fixes

### Recovery Strategies

1. **For Recoverable Errors:**
   - Fix the underlying issue
   - Retry the operation
   - Use alternative approaches if available

2. **For Non-Recoverable Errors:**
   - Restart the application
   - Fix system configuration
   - Address security issues immediately

### Getting Help

- Use `templar --help` for command information
- Check logs in `logs/` directory
- Review documentation in `docs/` directory
- Run `templar doctor` for system diagnostics

---

## Error Context Information

Templar errors include rich context information:

- **Component**: Which component was being processed
- **File Path**: Location of the error with line/column numbers
- **Context**: Additional debugging information
- **Error Chain**: Full error causation chain
- **Type**: Error category for appropriate handling
- **Recoverable**: Whether the error can be recovered from

This context helps in debugging and provides actionable information for resolution.