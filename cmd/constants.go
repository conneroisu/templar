package cmd

// Shared constants used across multiple cmd files

// Go type constants for component parameters.
const (
	TypeString  = "string"
	TypeInt     = "int"
	TypeInt32   = "int32"
	TypeInt64   = "int64"
	TypeFloat32 = "float32"
	TypeFloat64 = "float64"
	TypeBool    = "bool"
)

// Array/slice type constants.
const (
	TypeStringSlice = "[]string"
	TypeIntSlice    = "[]int"
	TypeSlicePrefix = "[]"
)

// TypeScript type constants.
const (
	TSString  = "string"
	TSNumber  = "number"
	TSBoolean = "boolean"
	TSAny     = "any"
)

// Common error message constants.
const (
	ErrorLoadConfig        = "failed to load config: %w"
	ErrorLoadConfiguration = "failed to load configuration: %w"
	ErrorCreateOutputDir   = "failed to create output directory: %w"
	ErrorWriteFile         = "failed to write file: %w"
	ErrorReadFile          = "failed to read file: %w"
	ErrorParseJSON         = "failed to parse JSON: %w"
	ErrorCreateServer      = "failed to create server: %w"
	ErrorRenderComponent   = "failed to render component: %w"
)

// File extension constants.
const (
	JSONExtension = ".json"
	GoExtension   = ".go"
	TSExtension   = ".ts"
	MDExtension   = ".md"
)

// Mock data value constants.
const (
	MockTextValue  = "Mock Text"
	MockValue      = "Mock Value"
	MockItem       = "Mock Item"
	MockSampleText = "Sample text"
)

// Mock data defaults for different types.
const (
	MockNumber = "123"
	MockFloat  = "3.14"
	MockTrue   = "true"
	MockFalse  = "false"
	MockNil    = "nil"
)

// Common directory and file names.
const (
	TemplarDir    = ".templar"
	TemplarConfig = ".templar.yml"
	PreviewHTML   = "preview.html"
)

// Optional value indicators.
const (
	OptionalYes = "Yes"
	OptionalNo  = "No"
)

// Status symbols.
const (
	SymbolCheckmark = "✅"
	SymbolCross     = "❌"
)

// User input response constants.
const (
	ResponseYes       = "yes"
	ResponseY         = "y"
	ResponseTrue      = "true"
	ResponseUnknown   = "unknown"
	StatusUnknown     = "unknown"
	OutputFormatJSON  = "json"
	OutputFormatYAML  = "yaml"
	OutputFormatYML   = "yml"
	OutputFormatTable = "table"
)

// Environment constants.
const (
	EnvironmentDevelopment = "development"
)

// HTTP method constants.
const (
	HTTPMethodGET  = "GET"
	HTTPMethodPOST = "POST"
)

// Status constants.
const (
	StatusSuccess = "success"
	StatusError   = "error"
	StatusHit     = "hit"
	StatusMiss    = "miss"
)

// Test constants.
const (
	TestTempDir       = "TestTempDir"
	TestRemoteAddr    = "192.168.1.100:1234"
	TestRateLimitAddr = "192.168.1.1:8080"
)

// File operation constants.
const (
	FileOpUnknown = "UNKNOWN"
)

// Performance test constants.
const (
	PerfTestType = "performance"
)

// Version constants.
const (
	VersionDev = "dev"
)

// Security constants.
const (
	UnsafeInline = "'unsafe-inline'"
	UnsafeEval   = "'unsafe-eval'"
)

// Error message constants.
const (
	ErrorInvalidFilePath = "Invalid file path"
)

// Validation scheme constants.
const (
	SchemeHTTP  = "http"
	SchemeHTTPS = "https"
)

// Command name constants.
const (
	CommandBuild = "build"
)

// Template constants.
const (
	TemplateMinimal = "minimal"
)

// Severity level constants.
const (
	SeverityInfo     = "info"
	SeverityWarning  = "warning"
	SeverityError    = "error"
	SeverityCritical = "critical"
	SeverityMajor    = "major"
)

// Mutation types.
const (
	MutationUnknown = "unknown"
)

// Configuration constants.
const (
	ConfigWrapperFormat   = "  wrapper: \"%s\"\n"
	ConfigHostFormat      = "  host: %s\n"
	ConfigAutoPropsFormat = "  auto_props: %t\n"
	ConfigCloseBrace      = "  }\n"
)

// Flag constants.
const (
	FlagNoOpen  = "no-open"
	FlagClean   = "clean"
	FlagVerbose = "verbose"
	FlagWrapper = "wrapper"
)

// Flag descriptions.
const (
	FlagDescVerbose = "Enable verbose/detailed output"
	FlagDescClean   = "Clean build artifacts before building"
	FlagDescWrapper = "Wrapper template path"
)

// Component constants.
const (
	ComponentName      = "components"
	ComponentDir       = "components"
	ComponentPackage   = "Package name"
	ComponentLanding   = "landing"
	ComponentDashboard = "dashboard"
	ComponentHomepage  = "homepage"
)

// Build and format constants.
const (
	FormatOption   = "format"
	ValidateOption = "validate"
	NPMCommand     = "npm"
	InstallOption  = "install"
)

// Plugin constants.
const (
	PluginEnableFormat = "🔌 Enabling plugin: %s\n"
)

// Count format constants.
const (
	ComponentCountFormat = "Found %d components\n"
)

// File names and config.
const (
	GitIgnoreFile     = ".gitignore"
	TemplarConfigFile = ".templar.yml"
)

// Input/Output messages.
const (
	InputErrorFormat    = "Failed to read input: %v\n"
	ConfigCancelMessage = "Configuration wizard cancelled."
)

// Port validation.
const (
	PortValidationError = "port must be between 1 and 65535, got %d"
)

// Path validation.
const (
	PathTraversalError = "path traversal attempt detected"
	PathSeparatorError = "path separators not allowed in component name"
)
