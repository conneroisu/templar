// Package types provides common type definitions used throughout the Templar CLI.
// This package contains shared types to avoid circular dependencies between packages.
package types

import "time"

// ComponentInfo contains comprehensive metadata about a discovered templ component,
// including its structure, dependencies, and runtime information used by the
// scanner, registry, and build pipeline.
type ComponentInfo struct {
	// Name is the component identifier (e.g., "Button", "CardHeader")
	Name string
	// Package is the Go package name where the component is defined
	Package string
	// FilePath is the absolute path to the .templ file containing the component
	FilePath string
	// Parameters describes the component's input parameters and their types
	Parameters []ParameterInfo
	// Imports lists Go packages imported by the component template
	Imports []string
	// LastMod tracks the last modification time for change detection
	LastMod time.Time
	// Hash provides a CRC32 checksum for efficient change detection
	Hash string
	// Dependencies lists other components or files this component depends on
	Dependencies []string
	// Metadata stores plugin-specific or custom component information
	Metadata map[string]interface{}
	// IsExported indicates if the component function is exported (public)
	IsExported bool
	// IsRenderable indicates if the component can be rendered independently
	IsRenderable bool
	// Description provides human-readable documentation for the component
	Description string
	// Examples contains sample usage scenarios for the component
	Examples []ComponentExample
}

// ParameterInfo describes a component parameter extracted from the templ
// function signature during AST analysis.
type ParameterInfo struct {
	// Name is the parameter name as declared in the templ function
	Name string
	// Type is the Go type of the parameter (e.g., "string", "*User", "[]Item")
	Type string
	// Optional indicates if the parameter has a default value or is pointer type
	Optional bool
	// Default stores the default value if one is specified (may be nil)
	Default interface{}
	// Description provides documentation for the parameter
	Description string
}

// ComponentExample represents a usage example for a component.
type ComponentExample struct {
	// Name is the example identifier
	Name string
	// Description explains what this example demonstrates
	Description string
	// Props contains the example parameter values
	Props map[string]interface{}
	// Code contains the example templ code
	Code string
}

// EventType represents the type of component change event.
type EventType string

const (
	EventTypeAdded   EventType = "added"
	EventTypeUpdated EventType = "updated"
	EventTypeRemoved EventType = "removed"
)

// ComponentEvent represents a change in the component registry, used for
// real-time notifications to watchers like the development server and UI.
type ComponentEvent struct {
	// Type indicates the kind of change (added, updated, removed)
	Type EventType
	// Component contains the component information (may be nil for removed events)
	Component *ComponentInfo
	// Timestamp records when the event occurred for ordering and filtering
	Timestamp time.Time
}
