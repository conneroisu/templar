# TestButton Component

## Overview
The TestButton component provides an interactive button with configurable variants, sizes, and states.

## Usage

### Basic Usage
```go
@TestButton("Click me", "primary", "medium", false, "")
```

### Variants
- **Primary**: `@TestButtonPrimary("Submit")`
- **Secondary**: `@TestButtonSecondary("Cancel")`
- **Danger**: `@TestButtonDanger("Delete")`

### Sizes
- **Small**: `@TestButtonSmall("Small", "primary")`
- **Large**: `@TestButtonLarge("Large", "primary")`

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