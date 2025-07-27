// Package build constants for repeated strings extracted from goconst analysis.
package build

const (
	// HTML Template Constants.
	HTMLDoctype      = "<!DOCTYPE html>\n"
	HTMLLangOpen     = "<html lang=\"en\">\n"
	HTMLHeadOpen     = "<head>\n"
	HTMLHeadClose    = "</head>\n"
	HTMLBodyOpen     = "<body>\n"
	HTMLBodyClose    = "</body>\n"
	HTMLMainOpen     = "  <main>\n"
	HTMLMainClose    = "  </main>\n"
	HTMLClose        = "</html>\n"
	HTMLCharsetMeta  = "  <meta charset=\"UTF-8\">\n"
	HTMLViewportMeta = "  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n"
	HTMLDescMeta     = "  <meta name=\"description\" content=\"%s\">\n"
	HTMLMainCSS      = "  <link rel=\"stylesheet\" href=\"%s/css/main.css\">\n"
	HTMLAssetsCSS    = "  <link rel=\"stylesheet\" href=\"/assets/css/main.css\">\n"
	HTMLHeading1     = "    <h1>%s</h1>\n"
	HTMLParagraph    = "    <p>%s</p>\n"
	HTMLDivClose     = "    </div>\n"

	// Format String Constants.
	FormatFilenameLine       = "%s:%d"
	FormatFilenameLineColumn = "%s:%d:%d"

	// Error Message Constants.
	ErrorTemplTimeoutFmt   = "templ generate timed out: %w"
	ErrorTemplFailedFmt    = "templ generate failed: %w\nOutput: %s"
	ErrorCmdValidationFmt  = "command validation failed: %w"
	ErrorParsedPrefix      = "Parsed errors:"
	ErrorCompilationFailed = "component compilation failed"
	ErrorAssetProcessFmt   = "failed to process asset %s: %w"
	ErrorPageDirCreateFmt  = "failed to create page directory: %w"

	// Asset Type Constants.
	AssetTypeJavaScript = "javascript"
	AssetTypeCSS        = "css"
	AssetTypeImage      = "image"
	AssetTypeFont       = "font"
	AssetTypeHTML       = "html"
	AssetTypeData       = "data"
	AssetTypeMain       = "main"

	// File Extension Constants.
	ExtHTML = ".html"

	// Build Status Constants.
	StatusPipelineNotStarted = "pipeline_not_started"

	// JavaScript Module Constants.
	JSFromPrefix    = "from "
	JSRequirePrefix = "require("

	// URL Constants.
	DefaultBaseURL = "https://example.com"

	// Directory Constants.
	DirDocker     = "docker"
	DirImages     = "images"
	DirFonts      = "fonts"
	DirReports    = "reports"
	DirDeployment = "deployment"

	// Test Structure Constants.
	StructureFlat   = "flat"
	StructureNested = "nested"
	StructureMixed  = "mixed"
)
