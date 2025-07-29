// Package apidocs provides automated API documentation generation for Templar's HTTP API.
//
// This package implements a comprehensive system for generating, versioning, and serving
// API documentation with the following components:
// - Code analysis and endpoint extraction from HTTP handlers
// - OpenAPI 3.0 specification generation
// - Interactive documentation interface with Swagger UI
// - Version tracking and change detection
// - CI/CD integration for automated doc updates
//
// The system emphasizes zero-maintenance documentation that stays synchronized with
// the actual implementation through automated analysis and generation.
package apidocs

import (
	"time"
)

// APIEndpoint represents a discovered HTTP endpoint with complete metadata.
type APIEndpoint struct {
	// Path is the URL path pattern (e.g., "/api/components/{id}")
	Path string `json:"path"`
	// Method is the HTTP method (GET, POST, PUT, DELETE, etc.)
	Method string `json:"method"`
	// Handler is the Go handler function name
	Handler string `json:"handler"`
	// Summary provides a brief description of the endpoint
	Summary string `json:"summary"`
	// Description contains detailed endpoint documentation
	Description string `json:"description"`
	// Parameters describes path, query, and header parameters
	Parameters []APIParameter `json:"parameters,omitempty"`
	// RequestBody describes the request body schema (for POST/PUT requests)
	RequestBody *APIRequestBody `json:"request_body,omitempty"`
	// Responses maps HTTP status codes to response descriptions
	Responses map[string]APIResponse `json:"responses"`
	// Tags categorize the endpoint for organization
	Tags []string `json:"tags,omitempty"`
	// Security specifies authentication requirements
	Security []SecurityRequirement `json:"security,omitempty"`
	// Deprecated indicates if the endpoint is deprecated
	Deprecated bool `json:"deprecated,omitempty"`
}

// APIParameter represents a parameter (path, query, header, or cookie).
type APIParameter struct {
	// Name is the parameter name
	Name string `json:"name"`
	// In specifies where the parameter appears (path, query, header, cookie)
	In string `json:"in"`
	// Description explains the parameter's purpose
	Description string `json:"description,omitempty"`
	// Required indicates if the parameter is mandatory
	Required bool `json:"required"`
	// Schema defines the parameter's data type and validation rules
	Schema *APISchema `json:"schema"`
	// Example provides a sample value
	Example interface{} `json:"example,omitempty"`
}

// APIRequestBody describes the request body for POST/PUT operations.
type APIRequestBody struct {
	// Description explains what the request body represents
	Description string `json:"description,omitempty"`
	// Content maps media types to their schemas
	Content map[string]APIMediaType `json:"content"`
	// Required indicates if the request body is mandatory
	Required bool `json:"required"`
}

// APIResponse represents an HTTP response with its schema and examples.
type APIResponse struct {
	// Description explains the response
	Description string `json:"description"`
	// Headers maps header names to their schemas
	Headers map[string]APIHeader `json:"headers,omitempty"`
	// Content maps media types to their schemas
	Content map[string]APIMediaType `json:"content,omitempty"`
}

// APIMediaType describes content for a specific media type.
type APIMediaType struct {
	// Schema defines the structure of the content
	Schema *APISchema `json:"schema,omitempty"`
	// Example provides a sample value
	Example interface{} `json:"example,omitempty"`
	// Examples provides multiple named examples
	Examples map[string]APIExample `json:"examples,omitempty"`
}

// APIHeader describes a response header.
type APIHeader struct {
	// Description explains the header's purpose
	Description string `json:"description,omitempty"`
	// Schema defines the header's data type
	Schema *APISchema `json:"schema"`
	// Required indicates if the header is always present
	Required bool `json:"required"`
}

// APIExample represents a named example value.
type APIExample struct {
	// Summary provides a brief description
	Summary string `json:"summary,omitempty"`
	// Description provides detailed explanation
	Description string `json:"description,omitempty"`
	// Value contains the example data
	Value interface{} `json:"value"`
}

// APISchema represents a data schema (JSON Schema subset).
type APISchema struct {
	// Type specifies the data type (string, integer, object, array, etc.)
	Type string `json:"type,omitempty"`
	// Format provides additional type information (date-time, email, etc.)
	Format string `json:"format,omitempty"`
	// Title provides a short description
	Title string `json:"title,omitempty"`
	// Description provides detailed explanation
	Description string `json:"description,omitempty"`
	// Properties defines object properties (for type: object)
	Properties map[string]*APISchema `json:"properties,omitempty"`
	// Items defines array element schema (for type: array)
	Items *APISchema `json:"items,omitempty"`
	// Required lists required property names (for type: object)
	Required []string `json:"required,omitempty"`
	// Example provides a sample value
	Example interface{} `json:"example,omitempty"`
	// Enum lists allowed values
	Enum []interface{} `json:"enum,omitempty"`
	// Default specifies the default value
	Default interface{} `json:"default,omitempty"`
	// Minimum/Maximum define numeric constraints
	Minimum *float64 `json:"minimum,omitempty"`
	Maximum *float64 `json:"maximum,omitempty"`
	// MinLength/MaxLength define string length constraints
	MinLength *int `json:"minLength,omitempty"`
	MaxLength *int `json:"maxLength,omitempty"`
	// Pattern defines a regex pattern for strings
	Pattern string `json:"pattern,omitempty"`
	// Ref references another schema component
	Ref string `json:"$ref,omitempty"`
}

// SecurityRequirement specifies authentication requirements for an endpoint.
type SecurityRequirement struct {
	// Type specifies the security scheme type (apiKey, http, oauth2, etc.)
	Type string `json:"type"`
	// Name is the security scheme name
	Name string `json:"name"`
	// Scopes lists required OAuth2 scopes (for oauth2 type)
	Scopes []string `json:"scopes,omitempty"`
}

// APISpecification represents the complete API documentation.
type APISpecification struct {
	// OpenAPI version (always "3.0.3" for this implementation)
	OpenAPI string `json:"openapi"`
	// Info contains API metadata
	Info APIInfo `json:"info"`
	// Servers lists available API servers
	Servers []APIServer `json:"servers,omitempty"`
	// Paths maps path patterns to their operations
	Paths map[string]map[string]APIEndpoint `json:"paths"`
	// Components contains reusable schema definitions
	Components *APIComponents `json:"components,omitempty"`
	// Security lists default security requirements
	Security []SecurityRequirement `json:"security,omitempty"`
	// Tags provide endpoint categorization
	Tags []APITag `json:"tags,omitempty"`
}

// APIInfo contains general API information.
type APIInfo struct {
	// Title is the API name
	Title string `json:"title"`
	// Description explains the API's purpose
	Description string `json:"description"`
	// Version is the API version
	Version string `json:"version"`
	// Contact provides maintainer information
	Contact *APIContact `json:"contact,omitempty"`
	// License specifies the API license
	License *APILicense `json:"license,omitempty"`
	// TermsOfService provides a URL to the terms of service
	TermsOfService string `json:"termsOfService,omitempty"`
}

// APIContact contains API maintainer contact information.
type APIContact struct {
	// Name is the contact person/organization name
	Name string `json:"name,omitempty"`
	// URL is the contact URL
	URL string `json:"url,omitempty"`
	// Email is the contact email address
	Email string `json:"email,omitempty"`
}

// APILicense contains API license information.
type APILicense struct {
	// Name is the license name
	Name string `json:"name"`
	// URL is the license URL
	URL string `json:"url,omitempty"`
}

// APIServer describes an API server.
type APIServer struct {
	// URL is the server URL
	URL string `json:"url"`
	// Description explains the server's purpose
	Description string `json:"description,omitempty"`
	// Variables defines URL template variables
	Variables map[string]APIServerVariable `json:"variables,omitempty"`
}

// APIServerVariable describes a server URL variable.
type APIServerVariable struct {
	// Default is the default value
	Default string `json:"default"`
	// Description explains the variable
	Description string `json:"description,omitempty"`
	// Enum lists allowed values
	Enum []string `json:"enum,omitempty"`
}

// APIComponents contains reusable API components.
type APIComponents struct {
	// Schemas contains reusable data schemas
	Schemas map[string]*APISchema `json:"schemas,omitempty"`
	// Responses contains reusable response definitions
	Responses map[string]APIResponse `json:"responses,omitempty"`
	// Parameters contains reusable parameter definitions
	Parameters map[string]APIParameter `json:"parameters,omitempty"`
	// RequestBodies contains reusable request body definitions
	RequestBodies map[string]APIRequestBody `json:"requestBodies,omitempty"`
	// Headers contains reusable header definitions
	Headers map[string]APIHeader `json:"headers,omitempty"`
	// SecuritySchemes defines authentication schemes
	SecuritySchemes map[string]APISecurityScheme `json:"securitySchemes,omitempty"`
}

// APISecurityScheme defines an authentication mechanism.
type APISecurityScheme struct {
	// Type specifies the security scheme type
	Type string `json:"type"`
	// Description explains the security scheme
	Description string `json:"description,omitempty"`
	// Name is the header/query parameter name (for apiKey type)
	Name string `json:"name,omitempty"`
	// In specifies where the API key appears (header, query, cookie)
	In string `json:"in,omitempty"`
	// Scheme specifies the HTTP authorization scheme (for http type)
	Scheme string `json:"scheme,omitempty"`
	// BearerFormat describes the bearer token format
	BearerFormat string `json:"bearerFormat,omitempty"`
}

// APITag provides endpoint categorization.
type APITag struct {
	// Name is the tag identifier
	Name string `json:"name"`
	// Description explains the tag's purpose
	Description string `json:"description,omitempty"`
	// ExternalDocs provides external documentation
	ExternalDocs *APIExternalDocs `json:"externalDocs,omitempty"`
}

// APIExternalDocs references external documentation.
type APIExternalDocs struct {
	// Description explains the external documentation
	Description string `json:"description,omitempty"`
	// URL is the documentation URL
	URL string `json:"url"`
}

// GenerationConfig configures API documentation generation.
type GenerationConfig struct {
	// OutputDir specifies where to write generated documentation
	OutputDir string `json:"output_dir"`
	// Version is the API version to generate
	Version string `json:"version"`
	// IncludeInternal determines if internal endpoints are documented
	IncludeInternal bool `json:"include_internal"`
	// ServeEnabled determines if the documentation server should start
	ServeEnabled bool `json:"serve_enabled"`
	// ServePort specifies the documentation server port
	ServePort int `json:"serve_port"`
	// AutoReload enables automatic regeneration on code changes
	AutoReload bool `json:"auto_reload"`
	// Formats specifies output formats (openapi, html, markdown)
	Formats []string `json:"formats"`
}

// GenerationResult contains the results of documentation generation.
type GenerationResult struct {
	// Specification is the generated OpenAPI specification
	Specification *APISpecification `json:"specification"`
	// EndpointsFound is the number of endpoints discovered
	EndpointsFound int `json:"endpoints_found"`
	// SchemasGenerated is the number of schemas created
	SchemasGenerated int `json:"schemas_generated"`
	// Errors contains any generation errors
	Errors []error `json:"errors,omitempty"`
	// Warnings contains generation warnings
	Warnings []string `json:"warnings,omitempty"`
	// GeneratedAt is the generation timestamp
	GeneratedAt time.Time `json:"generated_at"`
	// Version is the API version that was documented
	Version string `json:"version"`
}

// ComponentAnalysis represents analysis results for a Go type.
type ComponentAnalysis struct {
	// Name is the type name
	Name string `json:"name"`
	// Package is the Go package name
	Package string `json:"package"`
	// Schema is the generated JSON schema
	Schema *APISchema `json:"schema"`
	// Dependencies lists other types this type depends on
	Dependencies []string `json:"dependencies"`
	// IsExported indicates if the type is exported
	IsExported bool `json:"is_exported"`
	// Documentation contains extracted comments
	Documentation string `json:"documentation,omitempty"`
}

// EndpointAnalysis represents analysis results for an HTTP handler.
type EndpointAnalysis struct {
	// FunctionName is the handler function name
	FunctionName string `json:"function_name"`
	// Path is the extracted URL path pattern
	Path string `json:"path"`
	// Method is the HTTP method
	Method string `json:"method"`
	// RequestType is the request body type (if any)
	RequestType string `json:"request_type,omitempty"`
	// ResponseType is the response body type
	ResponseType string `json:"response_type,omitempty"`
	// Parameters lists extracted parameters
	Parameters []string `json:"parameters,omitempty"`
	// Documentation contains extracted comments
	Documentation string `json:"documentation,omitempty"`
	// Security contains security requirements
	Security []string `json:"security,omitempty"`
}