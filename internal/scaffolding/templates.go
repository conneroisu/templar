package scaffolding

// Note: fmt, strings, time are used in template content generation

// ComponentTemplate represents a template for generating components
type ComponentTemplate struct {
	Name        string
	Description string
	Category    string
	Parameters  []TemplateParameter
	Content     string
	StylesCSS   string
	TestContent string
	DocContent  string
}

// TemplateParameter represents a parameter in a component template
type TemplateParameter struct {
	Name         string
	Type         string
	DefaultValue string
	Description  string
	Required     bool
}

// TemplateContext holds the context for template generation
type TemplateContext struct {
	ComponentName string
	PackageName   string
	Parameters    []TemplateParameter
	Author        string
	Date          string
	ProjectName   string
	Imports       []string
	CustomProps   map[string]interface{}
}

// GetBuiltinTemplates returns all built-in component templates
func GetBuiltinTemplates() map[string]ComponentTemplate {
	return map[string]ComponentTemplate{
		"button":     getButtonTemplate(),
		"card":       getCardTemplate(),
		"form":       getFormTemplate(),
		"layout":     getLayoutTemplate(),
		"navigation": getNavigationTemplate(),
		"modal":      getModalTemplate(),
		"table":      getTableTemplate(),
		"list":       getListTemplate(),
		"hero":       getHeroTemplate(),
		"footer":     getFooterTemplate(),
		"breadcrumb": getBreadcrumbTemplate(),
		"pagination": getPaginationTemplate(),
		"accordion":  getAccordionTemplate(),
		"tabs":       getTabsTemplate(),
		"carousel":   getCarouselTemplate(),
		"sidebar":    getSidebarTemplate(),
		"header":     getHeaderTemplate(),
		"alert":      getAlertTemplate(),
		"badge":      getBadgeTemplate(),
		"tooltip":    getTooltipTemplate(),
	}
}

func getButtonTemplate() ComponentTemplate {
	return ComponentTemplate{
		Name:        "button",
		Description: "Interactive button component with variants and states",
		Category:    "interaction",
		Parameters: []TemplateParameter{
			{Name: "text", Type: "string", DefaultValue: "Click me", Description: "Button text content", Required: true},
			{Name: "variant", Type: "string", DefaultValue: "primary", Description: "Button style variant", Required: false},
			{Name: "size", Type: "string", DefaultValue: "medium", Description: "Button size", Required: false},
			{Name: "disabled", Type: "bool", DefaultValue: "false", Description: "Whether button is disabled", Required: false},
			{Name: "onclick", Type: "string", DefaultValue: "", Description: "Click handler", Required: false},
		},
		Content: `package {{.PackageName}}

{{if .Imports}}{{range .Imports}}import "{{.}}"
{{end}}{{end}}

// {{.ComponentName}} renders an interactive button with configurable variants and states
templ {{.ComponentName}}(text string, variant string, size string, disabled bool, onclick string) {
	<button 
		class={ 
			"btn",
			"btn-" + variant,
			"btn-" + size,
			templ.KV("btn-disabled", disabled)
		}
		if !disabled && onclick != "" {
			onclick={ onclick }
		}
		disabled?={ disabled }
		type="button"
	>
		{ text }
	</button>
}

// {{.ComponentName}}Primary renders a primary button variant
templ {{.ComponentName}}Primary(text string) {
	@{{.ComponentName}}(text, "primary", "medium", false, "")
}

// {{.ComponentName}}Secondary renders a secondary button variant  
templ {{.ComponentName}}Secondary(text string) {
	@{{.ComponentName}}(text, "secondary", "medium", false, "")
}

// {{.ComponentName}}Danger renders a danger button variant
templ {{.ComponentName}}Danger(text string) {
	@{{.ComponentName}}(text, "danger", "medium", false, "")
}

// {{.ComponentName}}Small renders a small button
templ {{.ComponentName}}Small(text string, variant string) {
	@{{.ComponentName}}(text, variant, "small", false, "")
}

// {{.ComponentName}}Large renders a large button
templ {{.ComponentName}}Large(text string, variant string) {
	@{{.ComponentName}}(text, variant, "large", false, "")
}`,
		StylesCSS: `.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.5rem 1rem;
  font-size: 0.875rem;
  font-weight: 500;
  line-height: 1.25rem;
  border-radius: 0.375rem;
  border: 1px solid transparent;
  cursor: pointer;
  transition: all 0.15s ease-in-out;
  text-decoration: none;
  user-select: none;
}

.btn:focus {
  outline: 2px solid transparent;
  outline-offset: 2px;
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.5);
}

.btn-primary {
  background-color: #3b82f6;
  color: white;
}

.btn-primary:hover:not(.btn-disabled) {
  background-color: #2563eb;
}

.btn-secondary {
  background-color: #6b7280;
  color: white;
}

.btn-secondary:hover:not(.btn-disabled) {
  background-color: #4b5563;
}

.btn-danger {
  background-color: #ef4444;
  color: white;
}

.btn-danger:hover:not(.btn-disabled) {
  background-color: #dc2626;
}

.btn-small {
  padding: 0.25rem 0.75rem;
  font-size: 0.75rem;
}

.btn-large {
  padding: 0.75rem 1.5rem;
  font-size: 1rem;
}

.btn-disabled {
  opacity: 0.5;
  cursor: not-allowed;
}`,
		TestContent: `package {{.PackageName}}_test

import (
	"context"
	"strings"
	"testing"

	"{{.ProjectName}}/{{.PackageName}}"
)

func Test{{.ComponentName}}(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		variant  string
		size     string
		disabled bool
		want     []string
	}{
		{
			name:     "primary button",
			text:     "Click me",
			variant:  "primary",
			size:     "medium",
			disabled: false,
			want:     []string{"btn", "btn-primary", "btn-medium", "Click me"},
		},
		{
			name:     "disabled button",
			text:     "Disabled",
			variant:  "primary",
			size:     "medium",
			disabled: true,
			want:     []string{"btn-disabled", "disabled"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html, err := {{.PackageName}}.{{.ComponentName}}(tt.text, tt.variant, tt.size, tt.disabled, "").Render(context.Background())
			if err != nil {
				t.Fatalf("failed to render component: %v", err)
			}

			htmlStr := html.String()
			for _, want := range tt.want {
				if !strings.Contains(htmlStr, want) {
					t.Errorf("expected HTML to contain %q, got: %s", want, htmlStr)
				}
			}
		})
	}
}`,
		DocContent: `# {{.ComponentName}} Component

## Overview
The {{.ComponentName}} component provides an interactive button with configurable variants, sizes, and states.

## Usage

### Basic Usage
` + "```go" + `
@{{.ComponentName}}("Click me", "primary", "medium", false, "")
` + "```" + `

### Variants
- **Primary**: ` + "`" + `@{{.ComponentName}}Primary("Submit")` + "`" + `
- **Secondary**: ` + "`" + `@{{.ComponentName}}Secondary("Cancel")` + "`" + `
- **Danger**: ` + "`" + `@{{.ComponentName}}Danger("Delete")` + "`" + `

### Sizes
- **Small**: ` + "`" + `@{{.ComponentName}}Small("Small", "primary")` + "`" + `
- **Large**: ` + "`" + `@{{.ComponentName}}Large("Large", "primary")` + "`" + `

## Parameters
- **text** (string): Button text content
- **variant** (string): Style variant (primary, secondary, danger)
- **size** (string): Button size (small, medium, large)
- **disabled** (bool): Whether the button is disabled
- **onclick** (string): Click handler function

## Styling
The component uses CSS classes for styling. Include the provided CSS or customize as needed.

## Accessibility
- Proper button semantics with ` + "`" + `type="button"` + "`" + `
- Disabled state handling
- Focus management with keyboard navigation
- Screen reader friendly text content`,
	}
}

func getCardTemplate() ComponentTemplate {
	return ComponentTemplate{
		Name:        "card",
		Description: "Flexible card component for displaying content",
		Category:    "layout",
		Parameters: []TemplateParameter{
			{Name: "title", Type: "string", DefaultValue: "Card Title", Description: "Card title", Required: false},
			{Name: "content", Type: "string", DefaultValue: "Card content", Description: "Card body content", Required: false},
			{Name: "imageUrl", Type: "string", DefaultValue: "", Description: "Optional image URL", Required: false},
			{Name: "footer", Type: "string", DefaultValue: "", Description: "Optional footer content", Required: false},
		},
		Content: `package {{.PackageName}}

// {{.ComponentName}} renders a flexible card component
templ {{.ComponentName}}(title string, content string, imageUrl string, footer string) {
	<div class="card">
		if imageUrl != "" {
			<img src={ imageUrl } alt={ title } class="card-image"/>
		}
		<div class="card-body">
			if title != "" {
				<h3 class="card-title">{ title }</h3>
			}
			if content != "" {
				<div class="card-content">
					{ content }
				</div>
			}
			{ children... }
		</div>
		if footer != "" {
			<div class="card-footer">
				{ footer }
			</div>
		}
	</div>
}

// {{.ComponentName}}Simple renders a simple card with just title and content
templ {{.ComponentName}}Simple(title string, content string) {
	@{{.ComponentName}}(title, content, "", "")
}

// {{.ComponentName}}WithImage renders a card with an image
templ {{.ComponentName}}WithImage(title string, content string, imageUrl string) {
	@{{.ComponentName}}(title, content, imageUrl, "")
}`,
		StylesCSS: `.card {
  background: white;
  border-radius: 0.5rem;
  box-shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.1), 0 1px 2px 0 rgba(0, 0, 0, 0.06);
  overflow: hidden;
  transition: box-shadow 0.15s ease-in-out;
}

.card:hover {
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
}

.card-image {
  width: 100%;
  height: 12rem;
  object-fit: cover;
}

.card-body {
  padding: 1.5rem;
}

.card-title {
  font-size: 1.25rem;
  font-weight: 600;
  color: #1f2937;
  margin: 0 0 0.5rem 0;
}

.card-content {
  color: #6b7280;
  line-height: 1.5;
}

.card-footer {
  padding: 1rem 1.5rem;
  background-color: #f9fafb;
  border-top: 1px solid #e5e7eb;
}`,
	}
}

func getFormTemplate() ComponentTemplate {
	return ComponentTemplate{
		Name:        "form",
		Description: "Form component with validation and field management",
		Category:    "input",
		Parameters: []TemplateParameter{
			{Name: "action", Type: "string", DefaultValue: "", Description: "Form action URL", Required: false},
			{Name: "method", Type: "string", DefaultValue: "POST", Description: "HTTP method", Required: false},
			{Name: "title", Type: "string", DefaultValue: "", Description: "Form title", Required: false},
		},
		Content: `package {{.PackageName}}

// {{.ComponentName}} renders a form with validation support
templ {{.ComponentName}}(action string, method string, title string) {
	<form class="form" action={ action } method={ method }>
		if title != "" {
			<h2 class="form-title">{ title }</h2>
		}
		{ children... }
	</form>
}

// {{.ComponentName}}Field renders a form field with label and validation
templ {{.ComponentName}}Field(name string, label string, fieldType string, required bool, placeholder string, value string, errorMsg string) {
	<div class="form-field">
		<label for={ name } class="form-label">
			{ label }
			if required {
				<span class="form-required">*</span>
			}
		</label>
		<input
			type={ fieldType }
			id={ name }
			name={ name }
			class={ "form-input", templ.KV("form-input-error", errorMsg != "") }
			placeholder={ placeholder }
			value={ value }
			required?={ required }
		/>
		if errorMsg != "" {
			<div class="form-error">{ errorMsg }</div>
		}
	</div>
}

// {{.ComponentName}}Textarea renders a textarea field
templ {{.ComponentName}}Textarea(name string, label string, required bool, placeholder string, value string, rows int, errorMsg string) {
	<div class="form-field">
		<label for={ name } class="form-label">
			{ label }
			if required {
				<span class="form-required">*</span>
			}
		</label>
		<textarea
			id={ name }
			name={ name }
			class={ "form-textarea", templ.KV("form-input-error", errorMsg != "") }
			placeholder={ placeholder }
			required?={ required }
			rows={ fmt.Sprintf("%d", rows) }
		>{ value }</textarea>
		if errorMsg != "" {
			<div class="form-error">{ errorMsg }</div>
		}
	</div>
}

// {{.ComponentName}}Submit renders a form submit button
templ {{.ComponentName}}Submit(text string, variant string) {
	<button type="submit" class={ "btn", "btn-" + variant, "form-submit" }>
		{ text }
	</button>
}`,
		StylesCSS: `.form {
  max-width: 32rem;
  margin: 0 auto;
  background: white;
  padding: 2rem;
  border-radius: 0.5rem;
  box-shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.1);
}

.form-title {
  font-size: 1.5rem;
  font-weight: 600;
  color: #1f2937;
  margin: 0 0 1.5rem 0;
  text-align: center;
}

.form-field {
  margin-bottom: 1rem;
}

.form-label {
  display: block;
  font-size: 0.875rem;
  font-weight: 500;
  color: #374151;
  margin-bottom: 0.25rem;
}

.form-required {
  color: #ef4444;
}

.form-input,
.form-textarea {
  width: 100%;
  padding: 0.5rem 0.75rem;
  border: 1px solid #d1d5db;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  transition: border-color 0.15s ease-in-out, box-shadow 0.15s ease-in-out;
}

.form-input:focus,
.form-textarea:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.form-input-error {
  border-color: #ef4444;
}

.form-input-error:focus {
  border-color: #ef4444;
  box-shadow: 0 0 0 3px rgba(239, 68, 68, 0.1);
}

.form-error {
  font-size: 0.75rem;
  color: #ef4444;
  margin-top: 0.25rem;
}

.form-submit {
  width: 100%;
  margin-top: 1rem;
}`,
	}
}

func getLayoutTemplate() ComponentTemplate {
	return ComponentTemplate{
		Name:        "layout",
		Description: "Base layout component with header, main, and footer",
		Category:    "layout",
		Parameters: []TemplateParameter{
			{Name: "title", Type: "string", DefaultValue: "Page Title", Description: "Page title", Required: true},
			{Name: "description", Type: "string", DefaultValue: "", Description: "Page description", Required: false},
		},
		Content: `package {{.PackageName}}

// {{.ComponentName}} renders the base page layout
templ {{.ComponentName}}(title string, description string) {
	<!DOCTYPE html>
	<html lang="en">
		<head>
			<meta charset="UTF-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
			<title>{ title }</title>
			if description != "" {
				<meta name="description" content={ description }/>
			}
			<link rel="stylesheet" href="/static/css/styles.css"/>
		</head>
		<body>
			<div class="layout">
				<header class="layout-header">
					<nav class="layout-nav">
						<div class="layout-nav-brand">
							<h1>{ title }</h1>
						</div>
						<div class="layout-nav-links">
							{ children... }
						</div>
					</nav>
				</header>
				
				<main class="layout-main">
					{ children... }
				</main>
				
				<footer class="layout-footer">
					<p>&copy; { fmt.Sprintf("%d", time.Now().Year()) } { title }. All rights reserved.</p>
				</footer>
			</div>
		</body>
	</html>
}

// {{.ComponentName}}Content renders just the content area without full HTML structure
templ {{.ComponentName}}Content(title string) {
	<div class="layout-content">
		<header class="content-header">
			<h1>{ title }</h1>
		</header>
		<div class="content-body">
			{ children... }
		</div>
	</div>
}`,
		StylesCSS: `.layout {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.layout-header {
  background-color: #1f2937;
  color: white;
  padding: 1rem 0;
}

.layout-nav {
  max-width: 80rem;
  margin: 0 auto;
  padding: 0 1rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.layout-nav-brand h1 {
  margin: 0;
  font-size: 1.5rem;
  font-weight: 700;
}

.layout-nav-links {
  display: flex;
  gap: 1rem;
  align-items: center;
}

.layout-main {
  flex: 1;
  max-width: 80rem;
  margin: 0 auto;
  padding: 2rem 1rem;
  width: 100%;
}

.layout-footer {
  background-color: #f9fafb;
  padding: 1rem 0;
  text-align: center;
  border-top: 1px solid #e5e7eb;
}

.layout-content {
  max-width: 80rem;
  margin: 0 auto;
  padding: 0 1rem;
}

.content-header {
  margin-bottom: 2rem;
  padding-bottom: 1rem;
  border-bottom: 1px solid #e5e7eb;
}

.content-header h1 {
  margin: 0;
  font-size: 2rem;
  font-weight: 700;
  color: #1f2937;
}`,
	}
}

func getModalTemplate() ComponentTemplate {
	return ComponentTemplate{
		Name:        "modal",
		Description: "Modal dialog component with overlay and close functionality",
		Category:    "overlay",
		Parameters: []TemplateParameter{
			{Name: "title", Type: "string", DefaultValue: "Modal Title", Description: "Modal title", Required: false},
			{Name: "size", Type: "string", DefaultValue: "medium", Description: "Modal size", Required: false},
			{Name: "closable", Type: "bool", DefaultValue: "true", Description: "Whether modal can be closed", Required: false},
		},
		Content: `package {{.PackageName}}

// {{.ComponentName}} renders a modal dialog
templ {{.ComponentName}}(title string, size string, closable bool, isOpen bool) {
	<div class={ "modal", templ.KV("modal-open", isOpen) } id="modal">
		<div class="modal-overlay" if closable { onclick="closeModal()" }></div>
		<div class={ "modal-container", "modal-" + size }>
			<div class="modal-content">
				<div class="modal-header">
					if title != "" {
						<h2 class="modal-title">{ title }</h2>
					}
					if closable {
						<button type="button" class="modal-close" onclick="closeModal()" aria-label="Close">
							&times;
						</button>
					}
				</div>
				<div class="modal-body">
					{ children... }
				</div>
			</div>
		</div>
	</div>
}

// {{.ComponentName}}Confirm renders a confirmation modal
templ {{.ComponentName}}Confirm(title string, message string, confirmText string, cancelText string) {
	@{{.ComponentName}}(title, "small", true, false) {
		<div class="modal-confirm">
			<p>{ message }</p>
			<div class="modal-actions">
				<button type="button" class="btn btn-danger" onclick="confirmAction()">
					{ confirmText }
				</button>
				<button type="button" class="btn btn-secondary" onclick="closeModal()">
					{ cancelText }
				</button>
			</div>
		</div>
	}
}`,
		StylesCSS: `.modal {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: 1000;
  display: none;
  align-items: center;
  justify-content: center;
}

.modal-open {
  display: flex;
}

.modal-overlay {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(0, 0, 0, 0.5);
  cursor: pointer;
}

.modal-container {
  position: relative;
  background: white;
  border-radius: 0.5rem;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04);
  max-height: 90vh;
  overflow-y: auto;
}

.modal-small {
  max-width: 24rem;
}

.modal-medium {
  max-width: 32rem;
}

.modal-large {
  max-width: 48rem;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.5rem 1.5rem 0 1.5rem;
}

.modal-title {
  margin: 0;
  font-size: 1.25rem;
  font-weight: 600;
  color: #1f2937;
}

.modal-close {
  background: none;
  border: none;
  font-size: 1.5rem;
  color: #6b7280;
  cursor: pointer;
  padding: 0;
  width: 2rem;
  height: 2rem;
  display: flex;
  align-items: center;
  justify-content: center;
}

.modal-close:hover {
  color: #374151;
}

.modal-body {
  padding: 1.5rem;
}

.modal-confirm {
  text-align: center;
}

.modal-actions {
  display: flex;
  gap: 0.75rem;
  justify-content: center;
  margin-top: 1rem;
}`,
	}
}

// Additional template functions...
func getNavigationTemplate() ComponentTemplate {
	return ComponentTemplate{
		Name:        "navigation",
		Description: "Navigation component with responsive menu",
		Category:    "navigation",
		Content: `package {{.PackageName}}

// {{.ComponentName}} renders a responsive navigation bar
templ {{.ComponentName}}(brand string, links []NavLink) {
	<nav class="navigation">
		<div class="nav-container">
			<div class="nav-brand">
				<a href="/">{ brand }</a>
			</div>
			<div class="nav-menu">
				for _, link := range links {
					<a href={ link.URL } class={ "nav-link", templ.KV("nav-link-active", link.Active) }>
						{ link.Text }
					</a>
				}
			</div>
		</div>
	</nav>
}

type NavLink struct {
	Text   string
	URL    string
	Active bool
}`,
		StylesCSS: `.navigation {
  background-color: #1f2937;
  padding: 1rem 0;
}

.nav-container {
  max-width: 80rem;
  margin: 0 auto;
  padding: 0 1rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.nav-brand a {
  color: white;
  font-size: 1.25rem;
  font-weight: 700;
  text-decoration: none;
}

.nav-menu {
  display: flex;
  gap: 1rem;
}

.nav-link {
  color: #d1d5db;
  text-decoration: none;
  padding: 0.5rem 1rem;
  border-radius: 0.25rem;
  transition: all 0.15s ease-in-out;
}

.nav-link:hover,
.nav-link-active {
  color: white;
  background-color: rgba(255, 255, 255, 0.1);
}`,
	}
}

// Implement remaining template functions...
func getTableTemplate() ComponentTemplate {
	return ComponentTemplate{
		Name:        "table",
		Description: "Data table component with sorting and pagination",
		Category:    "data",
		Content:     "// Table template implementation...",
		StylesCSS:   "/* Table styles */",
	}
}

func getListTemplate() ComponentTemplate {
	return ComponentTemplate{
		Name:        "list",
		Description: "List component for displaying collections",
		Category:    "data",
		Content:     "// List template implementation...",
		StylesCSS:   "/* List styles */",
	}
}

func getHeroTemplate() ComponentTemplate {
	return ComponentTemplate{
		Name:        "hero",
		Description: "Hero section component for landing pages",
		Category:    "content",
		Content:     "// Hero template implementation...",
		StylesCSS:   "/* Hero styles */",
	}
}

func getFooterTemplate() ComponentTemplate {
	return ComponentTemplate{
		Name:        "footer",
		Description: "Footer component with links and information",
		Category:    "layout",
		Content:     "// Footer template implementation...",
		StylesCSS:   "/* Footer styles */",
	}
}

func getBreadcrumbTemplate() ComponentTemplate {
	return ComponentTemplate{
		Name:        "breadcrumb",
		Description: "Breadcrumb navigation component",
		Category:    "navigation",
		Content:     "// Breadcrumb template implementation...",
		StylesCSS:   "/* Breadcrumb styles */",
	}
}

func getPaginationTemplate() ComponentTemplate {
	return ComponentTemplate{
		Name:        "pagination",
		Description: "Pagination component for data navigation",
		Category:    "navigation",
		Content:     "// Pagination template implementation...",
		StylesCSS:   "/* Pagination styles */",
	}
}

func getAccordionTemplate() ComponentTemplate {
	return ComponentTemplate{
		Name:        "accordion",
		Description: "Collapsible accordion component",
		Category:    "interaction",
		Content:     "// Accordion template implementation...",
		StylesCSS:   "/* Accordion styles */",
	}
}

func getTabsTemplate() ComponentTemplate {
	return ComponentTemplate{
		Name:        "tabs",
		Description: "Tabbed content component",
		Category:    "interaction",
		Content:     "// Tabs template implementation...",
		StylesCSS:   "/* Tabs styles */",
	}
}

func getCarouselTemplate() ComponentTemplate {
	return ComponentTemplate{
		Name:        "carousel",
		Description: "Image/content carousel component",
		Category:    "media",
		Content:     "// Carousel template implementation...",
		StylesCSS:   "/* Carousel styles */",
	}
}

func getSidebarTemplate() ComponentTemplate {
	return ComponentTemplate{
		Name:        "sidebar",
		Description: "Sidebar navigation component",
		Category:    "layout",
		Content:     "// Sidebar template implementation...",
		StylesCSS:   "/* Sidebar styles */",
	}
}

func getHeaderTemplate() ComponentTemplate {
	return ComponentTemplate{
		Name:        "header",
		Description: "Page header component",
		Category:    "layout",
		Content:     "// Header template implementation...",
		StylesCSS:   "/* Header styles */",
	}
}

func getAlertTemplate() ComponentTemplate {
	return ComponentTemplate{
		Name:        "alert",
		Description: "Alert/notification component",
		Category:    "feedback",
		Content:     "// Alert template implementation...",
		StylesCSS:   "/* Alert styles */",
	}
}

func getBadgeTemplate() ComponentTemplate {
	return ComponentTemplate{
		Name:        "badge",
		Description: "Badge/label component for status indication",
		Category:    "feedback",
		Content:     "// Badge template implementation...",
		StylesCSS:   "/* Badge styles */",
	}
}

func getTooltipTemplate() ComponentTemplate {
	return ComponentTemplate{
		Name:        "tooltip",
		Description: "Tooltip component for contextual help",
		Category:    "feedback",
		Content:     "// Tooltip template implementation...",
		StylesCSS:   "/* Tooltip styles */",
	}
}
