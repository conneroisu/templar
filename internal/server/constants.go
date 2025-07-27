package server

// HTTP Content Types.
const (
	ContentTypeJSON = "application/json"
	ContentTypeHTML = "text/html"
)

// Common Host Values.
const (
	LocalhostHost = "localhost"
)

// JSON Response Messages.
const (
	JSONStatusRendered    = `{"status": "rendered", "message": "Playground render complete"}`
	JSONStatusEditorReady = `{"status": "ok", "message": "Editor API ready"}`
	JSONStatusFileReady   = `{"status": "ok", "message": "File API ready"}`
	JSONStatusInlineReady = `{"status": "ok", "message": "Inline editor ready"}`
	JSONStatusCacheReady  = `{"status": "ok", "message": "Build cache management ready"}`
)

// HTTP Header Names.
const (
	HeaderContentType = "Content-Type"
)

// Common Error Messages.
const (
	ErrComponentNameRequired = "Component name required"
	ErrComponentNotFound     = "Component not found"
	ErrMethodNotAllowed      = "Method not allowed"
)

// URL Path Prefixes.
const (
	ComponentPathPrefix = "/component/"
	RenderPathPrefix    = "/render/"
	EditorPathPrefix    = "/editor/"
)

// JSON Status Messages.
const (
	StatusOK       = "ok"
	StatusRendered = "rendered"
)

// Test Configuration Constants.
const (
	TestLocalhost     = "localhost"
	TestHost3000      = "localhost:3000"
	TestHost8080      = "localhost:8080"
	TestRemoteAddr1   = "192.168.1.100:1234"
	TestRateLimitAddr = "192.168.1.1:8080"
)

// Security constants.
const (
	UnsafeInline = "'unsafe-inline'"
)

// Error messages.
const (
	ErrorInvalidFilePath = "Invalid file path"
	ErrorLevel           = "error"
)

// Go type constants.
const (
	TypeString      = "string"
	TypeInt         = "int"
	TypeInt32       = "int32"
	TypeInt64       = "int64"
	TypeBool        = "bool"
	TypeStringSlice = "[]string"
)

// Environment constants.
const (
	EnvironmentDevelopment = "development"
	EnvironmentProduction  = "production"
)

// Scheme constants.
const (
	SchemeHTTPS = "https"
)
