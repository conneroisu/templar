package plugins

// Error message constants.
const (
	// Generic error messages.
	ErrUnsupportedInstallMethod = "unsupported install method: %s"
	ErrConfigFileNotExist       = "config file does not exist: %s"
	ErrConfigMustExportObject   = "config file must export a configuration object"
	ErrFrameworkNotFound        = "framework %s not found"
	ErrActiveFrameworkNotFound  = "active framework %s not found"
	ErrNoConfigurationFound     = "no configuration found for framework %s"
	ErrPluginNotFound           = "plugin %s not found"
	ErrFailedShutdownPlugin     = "failed to shutdown plugin %s: %w"
	ErrCommandValidationFailed  = "command validation failed: %w"

	// File operation error messages.
	ErrFailedCreateOutputDirectory = "failed to create output directory: %w"
	ErrFailedCreateEntryPointDir   = "failed to create entry point directory: %w"
	ErrFailedWriteEntryPointFile   = "failed to write entry point file: %w"
	ErrFailedWriteCSSFile          = "failed to write CSS file: %w"
	ErrFailedReadConfigFile        = "failed to read config file: %w"
	ErrFailedWriteTempInputFile    = "failed to write temporary input file: %w"
	ErrFailedReadCompiledCSS       = "failed to read compiled CSS: %w"
	ErrFailedCreateEntryPoint      = "failed to create entry point: %w"
)

// File and directory constants.
const (
	OutputCSSFileName = "output.css"
	ModuleExportsStr  = "module.exports"
	SassExtension     = "sass"
	NpmCommand        = "npm"
	InstallArg        = "install"
	NpxCommand        = "npx"
	TailwindCSSBinary = "tailwindcss"
)

// Install methods.
const (
	InstallMethodNPM        = "npm"
	InstallMethodCDN        = "cdn"
	InstallMethodStandalone = "standalone"
)

// Plugin and framework identifiers.
const (
	PluginIdentifier  = "plugin"
	BuiltinSource     = "builtin"
	TailwindFramework = "tailwind"
)

// CSS and regex patterns.
const (
	CommentRegexPattern    = "/\\*.*?\\*/"
	WhitespaceRegexPattern = "\\s+"
	ClassRegexPattern      = "class=\"([^\"]*)\""
	CSSVarRegexPattern     = "--([a-zA-Z][a-zA-Z0-9_-]*)\\s*:\\s*([^;]+);"
	ImportURLTemplate      = "@import url('%s');\n"
)
