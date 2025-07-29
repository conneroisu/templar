# CLI Documentation Style Guide

This guide defines standards for CLI help text, flag documentation, and user interface consistency across all Templar commands.

## Table of Contents

- [General Principles](#general-principles)
- [Command Structure](#command-structure)
- [Help Text Format](#help-text-format)
- [Flag Documentation](#flag-documentation)
- [Examples](#examples)
- [Error Messages](#error-messages)
- [Consistency Standards](#consistency-standards)

## General Principles

### User-Centric Design
- **Clarity over brevity**: Help text should be clear and actionable
- **Progressive disclosure**: Show essential information first, details second
- **Accessibility**: Use plain language and avoid technical jargon
- **Consistency**: Similar concepts should use similar language and formatting

### Information Hierarchy
1. **Short description**: One-line summary of what the command does
2. **Long description**: Detailed explanation with context
3. **Usage patterns**: Common use cases and workflows
4. **Examples**: Practical, real-world scenarios
5. **Advanced options**: Less common flags and configurations

## Command Structure

### Standard Command Definition
```go
var commandCmd = &cobra.Command{
    Use:     "command [arguments]",
    Aliases: []string{"c", "cmd"},
    Short:   "Brief one-line description of what this command does",
    Long:    `Detailed multi-line description following our format standards`,
    Args:    cobra.ExactArgs(1), // Or appropriate validation
    RunE:    runCommand,
}
```

### Command Naming
- Use **lowercase** for command names
- Use **kebab-case** for multi-word commands (e.g., `build-production`)
- Provide **meaningful aliases** for frequently used commands
- Keep aliases **short** but memorable (1-2 characters)

### Argument Patterns
- Use `<required>` for required arguments
- Use `[optional]` for optional arguments  
- Use `[file...]` for variable number of arguments
- Use `<component>` for specific types of arguments

## Help Text Format

### Short Description
- **Length**: One line, under 60 characters
- **Style**: Action-oriented, present tense
- **Format**: "Action description with brief context"
- **Examples**:
  - ✅ "Initialize a new Templar project with templates"
  - ✅ "Start development server with hot reload"
  - ❌ "This command initializes projects"
  - ❌ "Development server startup utility"

### Long Description Format
```
[Main description paragraph explaining what the command does and why you'd use it]

[Optional second paragraph with important context or workflow information]

Examples:
  command basic-usage              # Brief comment
  command --flag value             # Brief comment
  command complex --example       # Brief comment

[Optional sections:]
Available Options:
  option1    Description of option1
  option2    Description of option2

Pro Tips:
  • Use bullet points for helpful hints
  • Include workflow suggestions
  • Mention related commands

[Optional security/warning notes:]
Security Note:
  Important security considerations if applicable.
```

### Long Description Standards
- **First paragraph**: Core functionality and primary use case
- **Second paragraph** (if needed): Important context, workflow information, or prerequisites
- **Examples section**: Always include at least 3 practical examples
- **Use consistent formatting**: Spacing, bullet points, section headers
- **Include cross-references**: Mention related commands when relevant

## Flag Documentation

### Flag Definition Standards
```go
// Good: Clear, consistent flag definitions
cmd.Flags().StringP("output", "o", "", "output directory for generated files")
cmd.Flags().BoolP("verbose", "v", false, "enable verbose logging output")
cmd.Flags().IntP("port", "p", 8080, "server port number (1-65535)")

// Bad: Unclear or inconsistent descriptions
cmd.Flags().String("out", "", "output")
cmd.Flags().Bool("debug", false, "debug mode")
```

### Flag Description Format
- **Length**: Clear and concise, typically 40-80 characters
- **Style**: Descriptive phrase, not a complete sentence
- **Include constraints**: Value ranges, valid options, or formats
- **Use consistent terminology**: Same concepts = same words

### Flag Description Patterns
- **File/Directory flags**: "path to [description] (default: current directory)"
- **Boolean flags**: "enable [feature description]"
- **Numeric flags**: "number description (range: min-max)"
- **String flags**: "string description (options: opt1, opt2, opt3)"
- **List flags**: "comma-separated list of [items]"

### Flag Examples
```go
// File and directory flags
--config string    configuration file path (default: .templar.yml)
--output string    output directory for built files (default: dist)
--input string     input directory to scan (default: current directory)

// Boolean flags
--production       enable production build optimizations
--verbose          enable detailed logging output
--no-open          disable automatic browser opening

// Numeric flags
--port int         server port number (range: 1-65535, default: 8080)
--workers int      number of build workers (range: 1-16, default: 4)
--timeout int      operation timeout in seconds (default: 30)

// String flags with options
--format string    output format (options: table, json, yaml, csv, default: table)
--log-level string log level (options: debug, info, warn, error, default: info)
--template string  project template (options: minimal, blog, dashboard, default: minimal)

// List flags
--watch strings    file patterns to watch (default: **/*.templ)
--ignore strings   patterns to ignore during scanning
```

## Examples

### Example Standards
- **Minimum**: Include at least 3 examples per command
- **Progression**: Start simple, build complexity
- **Real-world**: Use practical scenarios, not abstract examples
- **Comments**: Include brief explanations for complex examples
- **Consistency**: Use same project names, file paths across commands

### Example Format
```
Examples:
  command basic-usage                     # What this basic usage does
  command --flag value                    # What this flag modification achieves  
  command complex --multiple --flags      # Complex scenario explanation
  command "path with spaces"              # Handle edge cases like spaces
  command --output /custom/path           # Show absolute paths when relevant
```

### Example Categories
1. **Basic usage**: Default behavior without flags
2. **Common variations**: Most frequently used flag combinations
3. **Advanced scenarios**: Complex workflows or edge cases
4. **Integration examples**: How command works with others
5. **Error prevention**: Show correct usage for common mistakes

### Practical Examples by Command Type

#### Initialize Commands
```
Examples:
  templar init                           # Initialize in current directory
  templar init my-project                # Create new project directory
  templar init --minimal                 # Minimal setup without examples
  templar init --template blog           # Use blog template with posts
  templar init --wizard                  # Interactive configuration setup
```

#### Server Commands
```
Examples:
  templar serve                          # Start server on localhost:8080
  templar serve --port 3000              # Use custom port
  templar serve --host 0.0.0.0           # Allow external connections
  templar serve --no-open                # Don't open browser automatically
  templar serve components/button.templ  # Serve specific component
```

#### Build Commands
```
Examples:
  templar build                          # Build all components
  templar build --output dist            # Build to custom directory
  templar build --production             # Production optimizations
  templar build --workers 8              # Use 8 parallel workers
  templar build --clean                  # Clean before building
```

## Error Messages

### Error Message Standards
- **Be specific**: Clearly identify what went wrong
- **Be actionable**: Tell users how to fix the problem
- **Be consistent**: Use standard error patterns
- **Be helpful**: Include relevant context and suggestions

### Error Message Format
```
Error: [Specific problem description]

Suggestion: [How to fix the problem]
Help: Run 'templar [command] --help' for usage information
```

### Common Error Patterns
- **File not found**: "Component 'ButtonComponent' not found. Run 'templar list' to see available components."
- **Invalid arguments**: "Invalid port '99999'. Port must be between 1 and 65535."
- **Permission errors**: "Permission denied writing to directory. Check directory permissions."
- **Configuration errors**: "Invalid configuration in .templar.yml line 15. Expected string, got number."

## Consistency Standards

### Terminology Dictionary
Use consistent terms across all commands:

| Concept | Standard Term | Avoid |
|---------|---------------|-------|
| Template files | components | templates, files |
| Development server | development server | dev server, server |
| File watching | file watching | watching, monitoring |
| Build process | build | compilation, generation |
| Configuration file | configuration file | config file, settings |
| Output directory | output directory | build dir, target dir |
| Mock data | mock data | fake data, test data |
| Hot reload | hot reload | live reload, auto-reload |

### Format Consistency
- **File paths**: Use forward slashes, even on Windows
- **URLs**: Always include protocol (http://, https://)
- **Commands**: Use `templar command` format in examples
- **Flags**: Use long form in examples, mention short form in descriptions
- **File extensions**: Include dots (.templ, .yml, .json)

### Cross-Reference Standards
- **Related commands**: "See also: templar list, templar preview"
- **Configuration**: "Configure with .templar.yml or --config flag"
- **Documentation**: "Documentation: https://github.com/conneroisu/templar"
- **Help references**: "Run 'templar [command] --help' for more information"

### Section Ordering
Standard order for help text sections:
1. Short description
2. Long description (main paragraph)
3. Context/workflow information (if needed)
4. Examples (always required)
5. Available options/templates (if applicable)
6. Pro tips (optional)
7. Security notes (if applicable)
8. Cross-references (see also)

## Validation Checklist

### Content Quality
- [ ] Short description is under 60 characters and action-oriented
- [ ] Long description explains both what and why
- [ ] At least 3 practical examples are included
- [ ] Examples progress from simple to complex
- [ ] All flags have clear, consistent descriptions
- [ ] Error scenarios are mentioned where relevant
- [ ] Cross-references to related commands are included

### Format Consistency
- [ ] Uses standard section headers and formatting
- [ ] Examples follow the comment format
- [ ] Flag descriptions follow naming patterns
- [ ] Terminology matches the style guide dictionary
- [ ] File paths and URLs are formatted consistently

### User Experience
- [ ] Help text is scannable with clear visual hierarchy
- [ ] Information is organized by frequency of use
- [ ] Common mistakes are addressed proactively
- [ ] Help leads to successful task completion
- [ ] Language is accessible to intended audience

### Technical Accuracy
- [ ] All examples work as shown
- [ ] Flag descriptions match actual behavior
- [ ] Default values are correct
- [ ] Constraints and validation rules are accurate
- [ ] Cross-references point to existing commands/docs

---

This style guide ensures consistent, user-friendly CLI documentation across all Templar commands. Following these standards improves discoverability, reduces user errors, and creates a professional command-line experience.