# Button Component

## Overview
The Button component provides an interactive button with configurable variants, sizes, and states.

## Usage

### Basic Usage
```go
@Button("Click me", "primary", "medium", false, "")
```

### Variants
- **Primary**: `@ButtonPrimary("Submit")`
- **Secondary**: `@ButtonSecondary("Cancel")`
- **Danger**: `@ButtonDanger("Delete")`

### Sizes
- **Small**: `@ButtonSmall("Small", "primary")`
- **Large**: `@ButtonLarge("Large", "primary")`

## Parameters
- **text** (string): Button text content
- **variant** (string): Style variant (primary, secondary, danger)
- **size** (string): Button size (small, medium, large)
- **disabled** (bool): Whether the button is disabled
- **onclick** (string): Click handler function

## Styling
The component uses CSS classes for styling. Include the provided CSS or customize as needed.

## Accessibility
- Proper button semantics with `type="button"`
- Disabled state handling
- Focus management with keyboard navigation
- Screen reader friendly text content