package cmd

// Shared constants used across multiple cmd files

// Go type constants for component parameters
const (
	TypeString  = "string"
	TypeInt     = "int"
	TypeInt32   = "int32"
	TypeInt64   = "int64"
	TypeFloat32 = "float32"
	TypeFloat64 = "float64"
	TypeBool    = "bool"
)

// Array/slice type constants
const (
	TypeStringSlice = "[]string"
	TypeIntSlice    = "[]int"
	TypeSlicePrefix = "[]"
)

// TypeScript type constants
const (
	TSString  = "string"
	TSNumber  = "number"
	TSBoolean = "boolean"
	TSAny     = "any"
)

// Common error message constants
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

// File extension constants
const (
	JSONExtension = ".json"
	GoExtension   = ".go"
	TSExtension   = ".ts"
	MDExtension   = ".md"
)

// Mock data value constants
const (
	MockTextValue  = "Mock Text"
	MockValue      = "Mock Value"
	MockItem       = "Mock Item"
	MockSampleText = "Sample text"
)

// Mock data defaults for different types
const (
	MockNumber = "123"
	MockFloat  = "3.14"
	MockTrue   = "true"
	MockFalse  = "false"
	MockNil    = "nil"
)

// Common directory and file names
const (
	TemplarDir    = ".templar"
	TemplarConfig = ".templar.yml"
	PreviewHTML   = "preview.html"
)

// Optional value indicators
const (
	OptionalYes = "Yes"
	OptionalNo  = "No"
)

// Status symbols
const (
	SymbolCheckmark = "✅"
	SymbolCross     = "❌"
)
