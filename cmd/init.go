package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/conneroisu/templar/internal/config"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:     "init [name]",
	Aliases: []string{"i"},
	Short:   "Initialize a new templar project with templates and smart configuration",
	Long: `Initialize a new templar project with the necessary directory structure
and configuration files. If no name is provided, initializes in the current directory.

The wizard provides smart defaults based on your project structure and helps
you choose the right template for your use case.

Examples:
  templar init                         # Initialize in current directory with examples
  templar init my-project              # Initialize in new directory 'my-project'
  templar init --minimal               # Minimal setup without examples
  templar init --wizard                # Interactive configuration wizard (recommended)
  templar init --template=blog         # Use blog template with posts and layouts
  templar init --template=dashboard    # Use dashboard template with sidebar and cards  
  templar init --template=landing      # Use landing page template with hero and features
  templar init --template=ecommerce    # Use e-commerce template with products and cart
  templar init --template=documentation # Use documentation template with navigation

Available Templates:
  minimal        Basic component setup
  blog          Blog posts, layouts, and content management
  dashboard     Admin dashboard with sidebar navigation and data cards
  landing       Marketing landing page with hero sections and feature lists
  ecommerce     Product listings, shopping cart, and purchase flows
  documentation Technical documentation with navigation and code blocks

Pro Tips:
  • Use --wizard for project-specific smart defaults
  • Templates include production-ready components and styling
  • All templates work with the development server and live preview`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

var (
	initMinimal  bool
	initExample  bool
	initTemplate string
	initWizard   bool
)

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().BoolVar(&initMinimal, "minimal", false, "Minimal setup without examples")
	initCmd.Flags().BoolVar(&initExample, "example", false, "Include example components")
	initCmd.Flags().StringVarP(&initTemplate, "template", "t", "", "Project template to use")
	initCmd.Flags().BoolVar(&initWizard, "wizard", false, "Run configuration wizard during initialization")
}

func runInit(cmd *cobra.Command, args []string) error {
	var projectDir string

	if len(args) == 0 {
		// Initialize in current directory
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		projectDir = cwd
	} else {
		// Create new directory
		projectDir = args[0]
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			return fmt.Errorf("failed to create project directory: %w", err)
		}
	}

	fmt.Printf("Initializing templar project in %s\n", projectDir)

	// Create directory structure
	if err := createDirectoryStructure(projectDir); err != nil {
		return fmt.Errorf("failed to create directory structure: %w", err)
	}

	// Create configuration file
	if initWizard {
		if err := createConfigWithWizard(projectDir); err != nil {
			return fmt.Errorf("failed to create configuration with wizard: %w", err)
		}
	} else {
		if err := createConfigFile(projectDir); err != nil {
			return fmt.Errorf("failed to create configuration file: %w", err)
		}
	}

	// Create Go module if it doesn't exist
	if err := createGoModule(projectDir); err != nil {
		return fmt.Errorf("failed to create Go module: %w", err)
	}

	// Create example components if requested
	if initExample || (!initMinimal && initTemplate == "") {
		if err := createExampleComponents(projectDir); err != nil {
			return fmt.Errorf("failed to create example components: %w", err)
		}
	}

	// Create template files if template is specified
	if initTemplate != "" {
		if err := createFromTemplate(projectDir, initTemplate); err != nil {
			return fmt.Errorf("failed to create from template: %w", err)
		}
	}

	fmt.Println("✓ Project initialized successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. cd " + projectDir)
	fmt.Println("  2. templar serve")
	fmt.Println("  3. Open http://localhost:8080 in your browser")

	return nil
}

func createDirectoryStructure(projectDir string) error {
	dirs := []string{
		"components",
		"views",
		"examples",
		"static",
		"static/css",
		"static/js",
		"static/images",
		"mocks",
		"preview",
		".templar",
		".templar/cache",
	}

	for _, dir := range dirs {
		dirPath := filepath.Join(projectDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

func createConfigFile(projectDir string) error {
	configPath := filepath.Join(projectDir, ".templar.yml")

	// Don't overwrite existing config
	if _, err := os.Stat(configPath); err == nil {
		fmt.Println("⚠ Configuration file already exists, skipping")
		return nil
	}

	configContent := `# Templar configuration file
server:
  port: 8080
  host: localhost
  open: true
  middleware:
    - cors
    - logger

build:
  command: "templ generate"
  watch:
    - "**/*.templ"
    - "**/*.go"
  ignore:
    - "*_test.go"
    - "vendor/**"
    - ".git/**"
    - "node_modules/**"
  cache_dir: ".templar/cache"

preview:
  mock_data: "./mocks"
  wrapper: "./preview/wrapper.templ"
  auto_props: true
  
components:
  scan_paths:
    - "./components"
    - "./views"
    - "./examples"
  exclude_patterns:
    - "*_test.templ"
    - "*.example.templ"

development:
  hot_reload: true
  css_injection: true
  state_preservation: true
  error_overlay: true
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Println("✓ Created .templar.yml configuration file")
	return nil
}

func createGoModule(projectDir string) error {
	goModPath := filepath.Join(projectDir, "go.mod")

	// Don't overwrite existing go.mod
	if _, err := os.Stat(goModPath); err == nil {
		fmt.Println("⚠ go.mod already exists, skipping")
		return nil
	}

	// Use directory name as module name
	projectName := filepath.Base(projectDir)
	if projectName == "." {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		projectName = filepath.Base(cwd)
	}

	// Clean up project name for module
	projectName = strings.ToLower(projectName)
	projectName = strings.ReplaceAll(projectName, " ", "-")
	projectName = strings.ReplaceAll(projectName, "_", "-")

	goModContent := fmt.Sprintf(`module %s

go 1.24

require (
	github.com/a-h/templ v0.2.778
)
`, projectName)

	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		return fmt.Errorf("failed to write go.mod: %w", err)
	}

	fmt.Println("✓ Created go.mod file")
	return nil
}

func createExampleComponents(projectDir string) error {
	// Create a simple button component
	buttonPath := filepath.Join(projectDir, "components", "button.templ")
	buttonContent := `package components

templ Button(text string, variant string) {
	<button class={ "btn", "btn-" + variant }>
		{ text }
	</button>
}

templ PrimaryButton(text string) {
	@Button(text, "primary")
}

templ SecondaryButton(text string) {
	@Button(text, "secondary")
}
`

	if err := os.WriteFile(buttonPath, []byte(buttonContent), 0644); err != nil {
		return fmt.Errorf("failed to create button component: %w", err)
	}

	// Create a card component
	cardPath := filepath.Join(projectDir, "components", "card.templ")
	cardContent := `package components

templ Card(title string, content string) {
	<div class="card">
		<div class="card-header">
			<h3 class="card-title">{ title }</h3>
		</div>
		<div class="card-content">
			<p>{ content }</p>
		</div>
	</div>
}

templ CardWithImage(title string, content string, imageUrl string) {
	<div class="card">
		<img src={ imageUrl } alt={ title } class="card-image"/>
		<div class="card-header">
			<h3 class="card-title">{ title }</h3>
		</div>
		<div class="card-content">
			<p>{ content }</p>
		</div>
	</div>
}
`

	if err := os.WriteFile(cardPath, []byte(cardContent), 0644); err != nil {
		return fmt.Errorf("failed to create card component: %w", err)
	}

	// Create a layout component
	layoutPath := filepath.Join(projectDir, "views", "layout.templ")
	layoutContent := `package views

templ Layout(title string) {
	<!DOCTYPE html>
	<html lang="en">
		<head>
			<meta charset="UTF-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
			<title>{ title }</title>
			<link rel="stylesheet" href="/static/css/styles.css"/>
		</head>
		<body>
			<header class="header">
				<h1>{ title }</h1>
			</header>
			<main class="main">
				{ children... }
			</main>
			<footer class="footer">
				<p>&copy; 2024 Templar Project</p>
			</footer>
		</body>
	</html>
}
`

	if err := os.WriteFile(layoutPath, []byte(layoutContent), 0644); err != nil {
		return fmt.Errorf("failed to create layout component: %w", err)
	}

	// Create example page
	examplePath := filepath.Join(projectDir, "examples", "demo.templ")
	exampleContent := `package examples

import "` + filepath.Base(projectDir) + `/components"
import "` + filepath.Base(projectDir) + `/views"

templ DemoPage() {
	@views.Layout("Demo Page") {
		<div class="demo-container">
			<h2>Component Demo</h2>
			
			<section class="demo-section">
				<h3>Buttons</h3>
				@components.PrimaryButton("Primary Button")
				@components.SecondaryButton("Secondary Button")
				@components.Button("Custom Button", "warning")
			</section>
			
			<section class="demo-section">
				<h3>Cards</h3>
				@components.Card("Simple Card", "This is a simple card component with just title and content.")
				@components.CardWithImage("Card with Image", "This card includes an image along with the title and content.", "/static/images/placeholder.jpg")
			</section>
		</div>
	}
}
`

	if err := os.WriteFile(examplePath, []byte(exampleContent), 0644); err != nil {
		return fmt.Errorf("failed to create example page: %w", err)
	}

	// Create basic CSS
	cssPath := filepath.Join(projectDir, "static", "css", "styles.css")
	cssContent := `/* Basic styles for templar project */
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  line-height: 1.6;
  color: #333;
  background-color: #f8f9fa;
}

.header {
  background-color: #007bff;
  color: white;
  padding: 1rem;
  text-align: center;
}

.main {
  max-width: 1200px;
  margin: 0 auto;
  padding: 2rem;
}

.footer {
  text-align: center;
  padding: 1rem;
  border-top: 1px solid #eee;
  margin-top: 2rem;
}

/* Button styles */
.btn {
  display: inline-block;
  padding: 0.5rem 1rem;
  margin: 0.25rem;
  border: none;
  border-radius: 0.25rem;
  cursor: pointer;
  font-size: 1rem;
  text-decoration: none;
  transition: all 0.2s;
}

.btn-primary {
  background-color: #007bff;
  color: white;
}

.btn-primary:hover {
  background-color: #0056b3;
}

.btn-secondary {
  background-color: #6c757d;
  color: white;
}

.btn-secondary:hover {
  background-color: #545b62;
}

.btn-warning {
  background-color: #ffc107;
  color: #212529;
}

.btn-warning:hover {
  background-color: #e0a800;
}

/* Card styles */
.card {
  background: white;
  border-radius: 0.5rem;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
  margin: 1rem 0;
  overflow: hidden;
}

.card-image {
  width: 100%;
  height: 200px;
  object-fit: cover;
}

.card-header {
  padding: 1rem;
  border-bottom: 1px solid #eee;
}

.card-title {
  margin: 0;
  font-size: 1.25rem;
  font-weight: 600;
}

.card-content {
  padding: 1rem;
}

/* Demo styles */
.demo-container {
  max-width: 800px;
  margin: 0 auto;
}

.demo-section {
  margin: 2rem 0;
  padding: 1rem;
  background: white;
  border-radius: 0.5rem;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

.demo-section h3 {
  margin-bottom: 1rem;
  color: #007bff;
}
`

	if err := os.WriteFile(cssPath, []byte(cssContent), 0644); err != nil {
		return fmt.Errorf("failed to create CSS file: %w", err)
	}

	// Create wrapper template for previews
	wrapperPath := filepath.Join(projectDir, "preview", "wrapper.templ")
	wrapperContent := `package preview

templ Wrapper(title string) {
	<!DOCTYPE html>
	<html lang="en">
		<head>
			<meta charset="UTF-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
			<title>{ title } - Preview</title>
			<link rel="stylesheet" href="/static/css/styles.css"/>
			<style>
				body { margin: 2rem; }
				.preview-container { 
					max-width: 800px; 
					margin: 0 auto; 
					background: white; 
					padding: 2rem; 
					border-radius: 0.5rem; 
					box-shadow: 0 2px 4px rgba(0,0,0,0.1); 
				}
			</style>
		</head>
		<body>
			<div class="preview-container">
				{ children... }
			</div>
		</body>
	</html>
}
`

	if err := os.WriteFile(wrapperPath, []byte(wrapperContent), 0644); err != nil {
		return fmt.Errorf("failed to create wrapper template: %w", err)
	}

	// Create placeholder image
	placeholderPath := filepath.Join(projectDir, "static", "images", ".gitkeep")
	if err := os.WriteFile(placeholderPath, []byte(""), 0644); err != nil {
		return fmt.Errorf("failed to create placeholder file: %w", err)
	}

	fmt.Println("✓ Created example components and assets")
	return nil
}

func createFromTemplate(projectDir string, templateName string) error {
	// Available templates with enhanced validation
	validTemplates := []string{"minimal", "blog", "dashboard", "landing", "ecommerce", "documentation"}
	
	// Use enhanced validation with fuzzy suggestions
	if err := ValidateTemplateWithSuggestion(templateName, validTemplates); err != nil {
		return err
	}

	switch templateName {
	case "minimal":
		return createMinimalTemplate(projectDir)
	case "blog":
		return createBlogTemplate(projectDir)
	case "dashboard":
		return createDashboardTemplate(projectDir)
	case "landing":
		return createLandingTemplate(projectDir)
	case "ecommerce":
		return createEcommerceTemplate(projectDir)
	case "documentation":
		return createDocumentationTemplate(projectDir)
	default:
		// This should not happen due to validation above, but keep for safety
		return fmt.Errorf("unknown template: %s. Available templates: %s", 
			templateName, strings.Join(validTemplates, ", "))
	}
}

func createMinimalTemplate(projectDir string) error {
	// Just create a basic component
	componentPath := filepath.Join(projectDir, "components", "hello.templ")
	componentContent := `package components

templ Hello(name string) {
	<h1>Hello, { name }!</h1>
}
`

	if err := os.WriteFile(componentPath, []byte(componentContent), 0644); err != nil {
		return fmt.Errorf("failed to create hello component: %w", err)
	}

	fmt.Println("✓ Created minimal template")
	return nil
}

func createBlogTemplate(projectDir string) error {
	// Create blog-specific components
	postPath := filepath.Join(projectDir, "components", "post.templ")
	postContent := `package components

import "time"

type Post struct {
	Title   string
	Content string
	Author  string
	Date    time.Time
}

templ PostCard(post Post) {
	<article class="post-card">
		<h2 class="post-title">{ post.Title }</h2>
		<div class="post-meta">
			<span class="post-author">By { post.Author }</span>
			<span class="post-date">{ post.Date.Format("January 2, 2006") }</span>
		</div>
		<div class="post-content">
			{ post.Content }
		</div>
	</article>
}

templ PostList(posts []Post) {
	<div class="post-list">
		for _, post := range posts {
			@PostCard(post)
		}
	</div>
}
`

	if err := os.WriteFile(postPath, []byte(postContent), 0644); err != nil {
		return fmt.Errorf("failed to create post component: %w", err)
	}

	fmt.Println("✓ Created blog template")
	return nil
}

func createDashboardTemplate(projectDir string) error {
	// Create dashboard components
	sidebarPath := filepath.Join(projectDir, "components", "sidebar.templ")
	sidebarContent := `package components

type NavItem struct {
	Label string
	Icon  string
	URL   string
	Active bool
}

templ Sidebar(items []NavItem) {
	<aside class="sidebar">
		<div class="sidebar-header">
			<h2>Dashboard</h2>
		</div>
		<nav class="sidebar-nav">
			for _, item := range items {
				<a href={ templ.URL(item.URL) } class={ "nav-item", templ.KV("active", item.Active) }>
					<i class={ "icon", item.Icon }></i>
					<span>{ item.Label }</span>
				</a>
			}
		</nav>
	</aside>
}

templ DashboardCard(title string, value string, trend string) {
	<div class="dashboard-card">
		<div class="card-header">
			<h3 class="card-title">{ title }</h3>
		</div>
		<div class="card-content">
			<div class="card-value">{ value }</div>
			<div class="card-trend">{ trend }</div>
		</div>
	</div>
}
`
	if err := os.WriteFile(sidebarPath, []byte(sidebarContent), 0644); err != nil {
		return fmt.Errorf("failed to create sidebar component: %w", err)
	}

	fmt.Println("✓ Created dashboard template")
	return nil
}

func createLandingTemplate(projectDir string) error {
	// Create landing page components
	heroPath := filepath.Join(projectDir, "components", "hero.templ")
	heroContent := `package components

templ Hero(title string, subtitle string, ctaText string, ctaLink string) {
	<section class="hero">
		<div class="hero-container">
			<h1 class="hero-title">{ title }</h1>
			<p class="hero-subtitle">{ subtitle }</p>
			<div class="hero-actions">
				<a href={ templ.URL(ctaLink) } class="btn btn-primary">{ ctaText }</a>
			</div>
		</div>
	</section>
}

type Feature struct {
	Icon        string
	Title       string
	Description string
}

templ Features(features []Feature) {
	<section class="features">
		<div class="features-container">
			<h2>Features</h2>
			<div class="features-grid">
				for _, feature := range features {
					<div class="feature-card">
						<div class="feature-icon">
							<i class={ feature.Icon }></i>
						</div>
						<h3 class="feature-title">{ feature.Title }</h3>
						<p class="feature-description">{ feature.Description }</p>
					</div>
				}
			</div>
		</div>
	</section>
}
`
	if err := os.WriteFile(heroPath, []byte(heroContent), 0644); err != nil {
		return fmt.Errorf("failed to create hero component: %w", err)
	}

	fmt.Println("✓ Created landing page template")
	return nil
}

func createEcommerceTemplate(projectDir string) error {
	// Create e-commerce components
	productPath := filepath.Join(projectDir, "components", "product.templ")
	productContent := `package components

type Product struct {
	ID          string
	Name        string
	Price       string
	Image       string
	Description string
	InStock     bool
}

templ ProductCard(product Product) {
	<div class="product-card">
		<div class="product-image">
			<img src={ product.Image } alt={ product.Name }/>
			if !product.InStock {
				<div class="out-of-stock-overlay">Out of Stock</div>
			}
		</div>
		<div class="product-info">
			<h3 class="product-name">{ product.Name }</h3>
			<p class="product-description">{ product.Description }</p>
			<div class="product-footer">
				<span class="product-price">{ product.Price }</span>
				if product.InStock {
					<button class="btn btn-primary">Add to Cart</button>
				} else {
					<button class="btn btn-secondary" disabled>Notify Me</button>
				}
			</div>
		</div>
	</div>
}

templ ProductGrid(products []Product) {
	<div class="product-grid">
		for _, product := range products {
			@ProductCard(product)
		}
	</div>
}
`
	if err := os.WriteFile(productPath, []byte(productContent), 0644); err != nil {
		return fmt.Errorf("failed to create product component: %w", err)
	}

	fmt.Println("✓ Created e-commerce template")
	return nil
}

func createDocumentationTemplate(projectDir string) error {
	// Create documentation components
	docPath := filepath.Join(projectDir, "components", "docs.templ")
	docContent := `package components

type DocSection struct {
	ID       string
	Title    string
	Content  string
	Children []DocSection
}

templ DocNav(sections []DocSection) {
	<nav class="doc-nav">
		<ul class="nav-list">
			for _, section := range sections {
				<li class="nav-item">
					<a href={ templ.URL("#" + section.ID) } class="nav-link">
						{ section.Title }
					</a>
					if len(section.Children) > 0 {
						<ul class="nav-sublist">
							for _, child := range section.Children {
								<li class="nav-subitem">
									<a href={ templ.URL("#" + child.ID) } class="nav-sublink">
										{ child.Title }
									</a>
								</li>
							}
						</ul>
					}
				</li>
			}
		</ul>
	</nav>
}

templ CodeBlock(language string, code string) {
	<div class="code-block">
		<div class="code-header">
			<span class="code-language">{ language }</span>
			<button class="copy-button">Copy</button>
		</div>
		<pre class="code-content"><code class={ language }>{ code }</code></pre>
	</div>
}

templ Alert(type_ string, message string) {
	<div class={ "alert", "alert-" + type_ }>
		<div class="alert-content">{ message }</div>
	</div>
}
`
	if err := os.WriteFile(docPath, []byte(docContent), 0644); err != nil {
		return fmt.Errorf("failed to create docs component: %w", err)
	}

	fmt.Println("✓ Created documentation template")
	return nil
}

func createConfigWithWizard(projectDir string) error {
	fmt.Println("\n🧙 Running Configuration Wizard")
	fmt.Println("==============================")

	// Create wizard with project directory for smart defaults
	wizard := config.NewConfigWizardWithProjectDir(projectDir)

	cfg, err := wizard.Run()
	if err != nil {
		return fmt.Errorf("configuration wizard failed: %w", err)
	}

	// Validate the generated configuration
	validation := config.ValidateConfigWithDetails(cfg)
	if validation.HasErrors() {
		fmt.Println("\n❌ Configuration validation failed:")
		fmt.Print(validation.String())
		return fmt.Errorf("generated configuration is invalid")
	}

	if validation.HasWarnings() {
		fmt.Println("\n⚠️  Configuration warnings:")
		fmt.Print(validation.String())
	}

	// Write configuration file
	configPath := filepath.Join(projectDir, ".templar.yml")
	if err := wizard.WriteConfigFile(configPath); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	return nil
}
