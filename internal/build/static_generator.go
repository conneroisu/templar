package build

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/types"
)

// StaticSiteGenerator handles generation of static HTML files from templ components
type StaticSiteGenerator struct {
	config        *config.Config
	outputDir     string
	templateCache map[string]string
	layoutCache   map[string]string
}

// StaticGenerationOptions configures static site generation
type StaticGenerationOptions struct {
	Prerendering bool   `json:"prerendering"`
	CriticalCSS  bool   `json:"critical_css"`
	CDNPath      string `json:"cdn_path,omitempty"`
	Environment  string `json:"environment"`
	BaseURL      string `json:"base_url,omitempty"`
	OutputFormat string `json:"output_format"` // "html", "json", "both"

	// SEO and metadata
	GenerateSitemap bool              `json:"generate_sitemap"`
	GenerateRobots  bool              `json:"generate_robots"`
	MetaDefaults    map[string]string `json:"meta_defaults,omitempty"`

	// Performance
	InlineCSS      bool `json:"inline_css"`
	MinifyHTML     bool `json:"minify_html"`
	OptimizeImages bool `json:"optimize_images"`

	// Custom pages
	CustomPages []CustomPage      `json:"custom_pages,omitempty"`
	ErrorPages  map[string]string `json:"error_pages,omitempty"`

	// Build context
	BuildTime time.Time `json:"build_time"`
	GitCommit string    `json:"git_commit,omitempty"`
	Version   string    `json:"version,omitempty"`
}

// CustomPage represents a custom static page to generate
type CustomPage struct {
	Path        string                 `json:"path"`
	Template    string                 `json:"template"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Layout      string                 `json:"layout,omitempty"`
}

// StaticPage represents a generated static page
type StaticPage struct {
	Path         string            `json:"path"`
	Title        string            `json:"title"`
	Description  string            `json:"description"`
	Component    string            `json:"component"`
	Size         int64             `json:"size"`
	GeneratedAt  time.Time         `json:"generated_at"`
	Hash         string            `json:"hash"`
	Dependencies []string          `json:"dependencies"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// SitemapEntry represents an entry in the sitemap
type SitemapEntry struct {
	URL          string    `json:"url"`
	LastModified time.Time `json:"last_modified"`
	ChangeFreq   string    `json:"change_freq"`
	Priority     float64   `json:"priority"`
}

// NewStaticSiteGenerator creates a new static site generator
func NewStaticSiteGenerator(cfg *config.Config, outputDir string) *StaticSiteGenerator {
	return &StaticSiteGenerator{
		config:        cfg,
		outputDir:     outputDir,
		templateCache: make(map[string]string),
		layoutCache:   make(map[string]string),
	}
}

// Generate creates static HTML files from templ components
func (s *StaticSiteGenerator) Generate(
	ctx context.Context, 
	components []*types.ComponentInfo, 
	options StaticGenerationOptions,
) ([]string, error) {
	generatedFiles := make([]string, 0)

	// Ensure output directory exists
	if err := os.MkdirAll(s.outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate component pages
	for _, component := range components {
		if component.IsExported && component.IsRenderable {
			pageFiles, err := s.generateComponentPage(ctx, component, options)
			if err != nil {
				return nil, fmt.Errorf("failed to generate page for component %s: %w", component.Name, err)
			}
			generatedFiles = append(generatedFiles, pageFiles...)
		}
	}

	// Generate custom pages
	for _, customPage := range options.CustomPages {
		pageFile, err := s.generateCustomPage(ctx, customPage, options)
		if err != nil {
			return nil, fmt.Errorf("failed to generate custom page %s: %w", customPage.Path, err)
		}
		generatedFiles = append(generatedFiles, pageFile)
	}

	// Generate error pages
	for code, templatePath := range options.ErrorPages {
		errorPageFile, err := s.generateErrorPage(ctx, code, templatePath, options)
		if err != nil {
			return nil, fmt.Errorf("failed to generate error page %s: %w", code, err)
		}
		generatedFiles = append(generatedFiles, errorPageFile)
	}

	// Generate sitemap
	if options.GenerateSitemap {
		sitemapFile, err := s.generateSitemap(ctx, generatedFiles, options)
		if err != nil {
			return nil, fmt.Errorf("failed to generate sitemap: %w", err)
		}
		generatedFiles = append(generatedFiles, sitemapFile)
	}

	// Generate robots.txt
	if options.GenerateRobots {
		robotsFile, err := s.generateRobotsTxt(ctx, options)
		if err != nil {
			return nil, fmt.Errorf("failed to generate robots.txt: %w", err)
		}
		generatedFiles = append(generatedFiles, robotsFile)
	}

	// Generate index file
	indexFile, err := s.generateIndexPage(ctx, components, options)
	if err != nil {
		return nil, fmt.Errorf("failed to generate index page: %w", err)
	}
	generatedFiles = append(generatedFiles, indexFile)

	return generatedFiles, nil
}

// generateComponentPage creates a static HTML page for a component
func (s *StaticSiteGenerator) generateComponentPage(
	ctx context.Context, 
	component *types.ComponentInfo, 
	options StaticGenerationOptions,
) ([]string, error) {
	generatedFiles := make([]string, 0)

	// Generate main component page
	pagePath := filepath.Join(s.outputDir, s.getComponentPagePath(component))

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(pagePath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create page directory: %w", err)
	}

	// Generate HTML content
	htmlContent, err := s.renderComponentHTML(component, options)
	if err != nil {
		return nil, fmt.Errorf("failed to render component HTML: %w", err)
	}

	// Apply optimizations
	if options.MinifyHTML {
		htmlContent = s.minifyHTML(htmlContent)
	}

	// Write HTML file
	if err := os.WriteFile(pagePath, []byte(htmlContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write HTML file: %w", err)
	}
	generatedFiles = append(generatedFiles, pagePath)

	// Generate JSON representation if requested
	if options.OutputFormat == "json" || options.OutputFormat == "both" {
		jsonPath := strings.TrimSuffix(pagePath, ".html") + ".json"
		jsonContent, err := s.generateComponentJSON(component, options)
		if err != nil {
			return nil, fmt.Errorf("failed to generate JSON: %w", err)
		}

		if err := os.WriteFile(jsonPath, []byte(jsonContent), 0644); err != nil {
			return nil, fmt.Errorf("failed to write JSON file: %w", err)
		}
		generatedFiles = append(generatedFiles, jsonPath)
	}

	// Generate component variants if they exist
	if len(component.Examples) > 0 {
		for _, example := range component.Examples {
			variantPath := strings.TrimSuffix(pagePath, ".html") + "-" + s.sanitizeFileName(example.Name) + ".html"
			variantHTML, err := s.renderComponentVariant(component, example, options)
			if err != nil {
				return nil, fmt.Errorf("failed to render variant %s: %w", example.Name, err)
			}

			if err := os.WriteFile(variantPath, []byte(variantHTML), 0644); err != nil {
				return nil, fmt.Errorf("failed to write variant file: %w", err)
			}
			generatedFiles = append(generatedFiles, variantPath)
		}
	}

	return generatedFiles, nil
}

// generateCustomPage creates a custom static page
func (s *StaticSiteGenerator) generateCustomPage(ctx context.Context, page CustomPage, options StaticGenerationOptions) (string, error) {
	pagePath := filepath.Join(s.outputDir, page.Path)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(pagePath), 0755); err != nil {
		return "", fmt.Errorf("failed to create page directory: %w", err)
	}

	// Render page content
	htmlContent, err := s.renderCustomPageHTML(page, options)
	if err != nil {
		return "", fmt.Errorf("failed to render custom page: %w", err)
	}

	// Apply optimizations
	if options.MinifyHTML {
		htmlContent = s.minifyHTML(htmlContent)
	}

	// Write file
	if err := os.WriteFile(pagePath, []byte(htmlContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write custom page: %w", err)
	}

	return pagePath, nil
}

// generateErrorPage creates error pages (404, 500, etc.)
func (s *StaticSiteGenerator) generateErrorPage(
	ctx context.Context, 
	errorCode, templatePath string, 
	options StaticGenerationOptions,
) (string, error) {
	pagePath := filepath.Join(s.outputDir, errorCode+".html")

	// Create basic error page HTML
	htmlContent := s.generateErrorPageHTML(errorCode, templatePath, options)

	// Apply optimizations
	if options.MinifyHTML {
		htmlContent = s.minifyHTML(htmlContent)
	}

	// Write file
	if err := os.WriteFile(pagePath, []byte(htmlContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write error page: %w", err)
	}

	return pagePath, nil
}

// generateSitemap creates an XML sitemap
func (s *StaticSiteGenerator) generateSitemap(
	ctx context.Context, 
	generatedFiles []string, 
	options StaticGenerationOptions,
) (string, error) {
	sitemapPath := filepath.Join(s.outputDir, "sitemap.xml")

	baseURL := options.BaseURL
	if baseURL == "" {
		baseURL = "https://example.com" // Default base URL
	}

	var sitemap strings.Builder
	sitemap.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	sitemap.WriteString("\n")
	sitemap.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	sitemap.WriteString("\n")

	// Add entries for generated HTML files
	for _, file := range generatedFiles {
		if strings.HasSuffix(file, ".html") {
			relPath, err := filepath.Rel(s.outputDir, file)
			if err != nil {
				continue
			}

			url := strings.TrimSuffix(baseURL, "/") + "/" + strings.TrimPrefix(relPath, "/")

			sitemap.WriteString("  <url>\n")
			sitemap.WriteString(fmt.Sprintf("    <loc>%s</loc>\n", url))
			sitemap.WriteString(fmt.Sprintf("    <lastmod>%s</lastmod>\n", time.Now().Format("2006-01-02")))
			sitemap.WriteString("    <changefreq>weekly</changefreq>\n")
			sitemap.WriteString("    <priority>0.8</priority>\n")
			sitemap.WriteString("  </url>\n")
		}
	}

	sitemap.WriteString("</urlset>\n")

	// Write sitemap
	if err := os.WriteFile(sitemapPath, []byte(sitemap.String()), 0644); err != nil {
		return "", fmt.Errorf("failed to write sitemap: %w", err)
	}

	return sitemapPath, nil
}

// generateRobotsTxt creates a robots.txt file
func (s *StaticSiteGenerator) generateRobotsTxt(ctx context.Context, options StaticGenerationOptions) (string, error) {
	robotsPath := filepath.Join(s.outputDir, "robots.txt")

	robotsContent := "User-agent: *\n"
	robotsContent += "Allow: /\n"

	if options.GenerateSitemap {
		baseURL := options.BaseURL
		if baseURL == "" {
			baseURL = "https://example.com"
		}
		robotsContent += fmt.Sprintf("Sitemap: %s/sitemap.xml\n", strings.TrimSuffix(baseURL, "/"))
	}

	// Write robots.txt
	if err := os.WriteFile(robotsPath, []byte(robotsContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write robots.txt: %w", err)
	}

	return robotsPath, nil
}

// generateIndexPage creates the main index page
func (s *StaticSiteGenerator) generateIndexPage(
	ctx context.Context, 
	components []*types.ComponentInfo, 
	options StaticGenerationOptions,
) (string, error) {
	indexPath := filepath.Join(s.outputDir, "index.html")

	// Create component catalog HTML
	htmlContent := s.generateComponentCatalogHTML(components, options)

	// Apply optimizations
	if options.MinifyHTML {
		htmlContent = s.minifyHTML(htmlContent)
	}

	// Write index file
	if err := os.WriteFile(indexPath, []byte(htmlContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write index page: %w", err)
	}

	return indexPath, nil
}

// HTML Generation Methods

// renderComponentHTML generates HTML for a component page
func (s *StaticSiteGenerator) renderComponentHTML(component *types.ComponentInfo, options StaticGenerationOptions) (string, error) {
	var html strings.Builder

	html.WriteString("<!DOCTYPE html>\n")
	html.WriteString("<html lang=\"en\">\n")
	html.WriteString("<head>\n")
	html.WriteString(fmt.Sprintf("  <title>%s - Component</title>\n", component.Name))
	html.WriteString("  <meta charset=\"UTF-8\">\n")
	html.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")

	// Add meta description
	if component.Description != "" {
		html.WriteString(fmt.Sprintf("  <meta name=\"description\" content=\"%s\">\n", component.Description))
	}

	// Add CSS
	if options.CDNPath != "" {
		html.WriteString(fmt.Sprintf("  <link rel=\"stylesheet\" href=\"%s/css/main.css\">\n", options.CDNPath))
	} else {
		html.WriteString("  <link rel=\"stylesheet\" href=\"/assets/css/main.css\">\n")
	}

	// Add critical CSS inline if enabled
	if options.CriticalCSS && options.InlineCSS {
		criticalCSS := s.extractCriticalCSS(component)
		if criticalCSS != "" {
			html.WriteString("  <style>\n")
			html.WriteString(criticalCSS)
			html.WriteString("  </style>\n")
		}
	}

	html.WriteString("</head>\n")
	html.WriteString("<body>\n")

	// Add component content
	html.WriteString("  <main>\n")
	html.WriteString(fmt.Sprintf("    <h1>%s</h1>\n", component.Name))

	if component.Description != "" {
		html.WriteString(fmt.Sprintf("    <p class=\"description\">%s</p>\n", component.Description))
	}

	// Add component preview
	html.WriteString("    <div class=\"component-preview\">\n")
	html.WriteString("      <!-- Component would be rendered here -->\n")
	html.WriteString(fmt.Sprintf("      <div class=\"placeholder\">%s Component Preview</div>\n", component.Name))
	html.WriteString("    </div>\n")

	// Add component documentation
	if len(component.Parameters) > 0 {
		html.WriteString("    <section class=\"component-props\">\n")
		html.WriteString("      <h2>Properties</h2>\n")
		html.WriteString("      <table>\n")
		html.WriteString("        <thead>\n")
		html.WriteString("          <tr><th>Name</th><th>Type</th><th>Required</th><th>Description</th></tr>\n")
		html.WriteString("        </thead>\n")
		html.WriteString("        <tbody>\n")

		for _, param := range component.Parameters {
			required := "Yes"
			if param.Optional {
				required = "No"
			}
			html.WriteString(fmt.Sprintf("          <tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>\n",
				param.Name, param.Type, required, param.Description))
		}

		html.WriteString("        </tbody>\n")
		html.WriteString("      </table>\n")
		html.WriteString("    </section>\n")
	}

	html.WriteString("  </main>\n")

	// Add JavaScript if needed
	if options.CDNPath != "" {
		html.WriteString(fmt.Sprintf("  <script src=\"%s/js/main.js\"></script>\n", options.CDNPath))
	} else {
		html.WriteString("  <script src=\"/assets/js/main.js\"></script>\n")
	}

	html.WriteString("</body>\n")
	html.WriteString("</html>\n")

	return html.String(), nil
}

// renderComponentVariant generates HTML for a component variant/example
func (s *StaticSiteGenerator) renderComponentVariant(
	component *types.ComponentInfo, 
	example types.ComponentExample, 
	options StaticGenerationOptions,
) (string, error) {
	var html strings.Builder

	html.WriteString("<!DOCTYPE html>\n")
	html.WriteString("<html lang=\"en\">\n")
	html.WriteString("<head>\n")
	html.WriteString(fmt.Sprintf("  <title>%s - %s Variant</title>\n", component.Name, example.Name))
	html.WriteString("  <meta charset=\"UTF-8\">\n")
	html.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")

	// Add CSS
	if options.CDNPath != "" {
		html.WriteString(fmt.Sprintf("  <link rel=\"stylesheet\" href=\"%s/css/main.css\">\n", options.CDNPath))
	} else {
		html.WriteString("  <link rel=\"stylesheet\" href=\"/assets/css/main.css\">\n")
	}

	html.WriteString("</head>\n")
	html.WriteString("<body>\n")
	html.WriteString("  <main>\n")
	html.WriteString(fmt.Sprintf("    <h1>%s - %s</h1>\n", component.Name, example.Name))

	if example.Description != "" {
		html.WriteString(fmt.Sprintf("    <p>%s</p>\n", example.Description))
	}

	html.WriteString("    <div class=\"variant-preview\">\n")
	html.WriteString(fmt.Sprintf("      <!-- %s variant would be rendered here -->\n", example.Name))
	html.WriteString("    </div>\n")
	html.WriteString("  </main>\n")
	html.WriteString("</body>\n")
	html.WriteString("</html>\n")

	return html.String(), nil
}

// renderCustomPageHTML generates HTML for a custom page
func (s *StaticSiteGenerator) renderCustomPageHTML(page CustomPage, options StaticGenerationOptions) (string, error) {
	var html strings.Builder

	html.WriteString("<!DOCTYPE html>\n")
	html.WriteString("<html lang=\"en\">\n")
	html.WriteString("<head>\n")
	html.WriteString(fmt.Sprintf("  <title>%s</title>\n", page.Title))
	html.WriteString("  <meta charset=\"UTF-8\">\n")
	html.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")

	if page.Description != "" {
		html.WriteString(fmt.Sprintf("  <meta name=\"description\" content=\"%s\">\n", page.Description))
	}

	// Add CSS
	if options.CDNPath != "" {
		html.WriteString(fmt.Sprintf("  <link rel=\"stylesheet\" href=\"%s/css/main.css\">\n", options.CDNPath))
	} else {
		html.WriteString("  <link rel=\"stylesheet\" href=\"/assets/css/main.css\">\n")
	}

	html.WriteString("</head>\n")
	html.WriteString("<body>\n")
	html.WriteString("  <main>\n")
	html.WriteString(fmt.Sprintf("    <h1>%s</h1>\n", page.Title))

	if page.Description != "" {
		html.WriteString(fmt.Sprintf("    <p>%s</p>\n", page.Description))
	}

	html.WriteString("    <div class=\"custom-content\">\n")
	html.WriteString("      <!-- Custom page content would be rendered here -->\n")
	html.WriteString("    </div>\n")
	html.WriteString("  </main>\n")
	html.WriteString("</body>\n")
	html.WriteString("</html>\n")

	return html.String(), nil
}

// generateErrorPageHTML creates HTML for error pages
func (s *StaticSiteGenerator) generateErrorPageHTML(errorCode, templatePath string, options StaticGenerationOptions) string {
	var html strings.Builder

	html.WriteString("<!DOCTYPE html>\n")
	html.WriteString("<html lang=\"en\">\n")
	html.WriteString("<head>\n")
	html.WriteString(fmt.Sprintf("  <title>Error %s</title>\n", errorCode))
	html.WriteString("  <meta charset=\"UTF-8\">\n")
	html.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	html.WriteString("  <link rel=\"stylesheet\" href=\"/assets/css/main.css\">\n")
	html.WriteString("</head>\n")
	html.WriteString("<body>\n")
	html.WriteString("  <main class=\"error-page\">\n")
	html.WriteString(fmt.Sprintf("    <h1>Error %s</h1>\n", errorCode))

	switch errorCode {
	case "404":
		html.WriteString("    <p>The page you're looking for could not be found.</p>\n")
	case "500":
		html.WriteString("    <p>Internal server error occurred.</p>\n")
	default:
		html.WriteString(fmt.Sprintf("    <p>An error occurred (Code: %s).</p>\n", errorCode))
	}

	html.WriteString("    <a href=\"/\">Return to Home</a>\n")
	html.WriteString("  </main>\n")
	html.WriteString("</body>\n")
	html.WriteString("</html>\n")

	return html.String()
}

// generateComponentCatalogHTML creates the main component catalog page
func (s *StaticSiteGenerator) generateComponentCatalogHTML(components []*types.ComponentInfo, options StaticGenerationOptions) string {
	var html strings.Builder

	html.WriteString("<!DOCTYPE html>\n")
	html.WriteString("<html lang=\"en\">\n")
	html.WriteString("<head>\n")
	html.WriteString("  <title>Component Catalog</title>\n")
	html.WriteString("  <meta charset=\"UTF-8\">\n")
	html.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	html.WriteString("  <meta name=\"description\" content=\"Templar component catalog and documentation\">\n")
	html.WriteString("  <link rel=\"stylesheet\" href=\"/assets/css/main.css\">\n")
	html.WriteString("</head>\n")
	html.WriteString("<body>\n")
	html.WriteString("  <main>\n")
	html.WriteString("    <h1>Component Catalog</h1>\n")
	html.WriteString("    <p>Browse and explore all available components.</p>\n")

	html.WriteString("    <div class=\"component-grid\">\n")

	for _, component := range components {
		if component.IsExported && component.IsRenderable {
			html.WriteString("      <div class=\"component-card\">\n")
			html.WriteString(fmt.Sprintf("        <h3><a href=\"%s\">%s</a></h3>\n", s.getComponentPagePath(component), component.Name))

			if component.Description != "" {
				html.WriteString(fmt.Sprintf("        <p>%s</p>\n", component.Description))
			}

			html.WriteString(fmt.Sprintf("        <small>%s</small>\n", component.Package))
			html.WriteString("      </div>\n")
		}
	}

	html.WriteString("    </div>\n")
	html.WriteString("  </main>\n")
	html.WriteString("</body>\n")
	html.WriteString("</html>\n")

	return html.String()
}

// Helper methods

// getComponentPagePath generates the page path for a component
func (s *StaticSiteGenerator) getComponentPagePath(component *types.ComponentInfo) string {
	// Create a clean URL path
	sanitizedName := s.sanitizeFileName(component.Name)
	if component.Package != "" && component.Package != "main" {
		return fmt.Sprintf("%s/%s.html", component.Package, sanitizedName)
	}
	return fmt.Sprintf("%s.html", sanitizedName)
}

// sanitizeFileName creates a safe filename from a string
func (s *StaticSiteGenerator) sanitizeFileName(name string) string {
	// Convert to lowercase and replace non-alphanumeric with hyphens
	sanitized := strings.ToLower(name)
	sanitized = strings.ReplaceAll(sanitized, " ", "-")
	sanitized = strings.ReplaceAll(sanitized, "_", "-")

	// Remove any non-alphanumeric characters except hyphens
	var result strings.Builder
	for _, r := range sanitized {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	return strings.Trim(result.String(), "-")
}

// minifyHTML performs basic HTML minification
func (s *StaticSiteGenerator) minifyHTML(html string) string {
	// Basic minification - remove extra whitespace
	lines := strings.Split(html, "\n")
	var minified strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			minified.WriteString(trimmed)
			minified.WriteString(" ")
		}
	}

	result := strings.TrimSpace(minified.String())

	// Remove space between tags
	result = strings.ReplaceAll(result, "> <", "><")

	return result
}

// extractCriticalCSS extracts critical CSS for a component (placeholder)
func (s *StaticSiteGenerator) extractCriticalCSS(component *types.ComponentInfo) string {
	// Placeholder implementation
	// In a real implementation, this would analyze the component and extract
	// the minimal CSS needed for above-the-fold rendering
	return "/* Critical CSS for " + component.Name + " */"
}

// generateComponentJSON creates a JSON representation of a component
func (s *StaticSiteGenerator) generateComponentJSON(component *types.ComponentInfo, options StaticGenerationOptions) (string, error) {
	componentData := map[string]interface{}{
		"name":         component.Name,
		"package":      component.Package,
		"description":  component.Description,
		"parameters":   component.Parameters,
		"examples":     component.Examples,
		"metadata":     component.Metadata,
		"generated_at": time.Now(),
		"build_info": map[string]interface{}{
			"environment": options.Environment,
			"version":     options.Version,
			"git_commit":  options.GitCommit,
		},
	}

	jsonData, err := json.MarshalIndent(componentData, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal component JSON: %w", err)
	}

	return string(jsonData), nil
}
