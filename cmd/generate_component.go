package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/conneroisu/templar/internal/scaffolding"
	"github.com/spf13/cobra"
)

var componentCmd = &cobra.Command{
	Use:   "component",
	Short: "Generate component scaffolding",
	Long: `Generate component scaffolding from built-in templates.

This command provides subcommands for:
- Creating new components from templates
- Listing available templates
- Creating complete project scaffolds
- Generating component sets

Examples:
  templar component create Button --template button
  templar component create MyCard --template card --with-tests --with-styles
  templar component list                        # List available templates
  templar component scaffold                    # Create project scaffold`,
}

var componentCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new component from template",
	Long: `Create a new component from a built-in template.

Available templates include:
- button: Interactive button with variants
- card: Content card component  
- form: Form with validation
- layout: Page layout structure
- modal: Modal dialog
- navigation: Navigation menu
- table: Data table
- And many more...

Examples:
  templar component create Button --template button
  templar component create UserCard --template card --output ./components
  templar component create ContactForm --template form --with-tests --with-docs
  templar component create AppLayout --template layout --with-styles`,
	Args: cobra.ExactArgs(1),
	RunE: runComponentCreate,
}

var componentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available component templates",
	Long: `List all available component templates with descriptions and categories.

This shows built-in templates that can be used to generate components,
including their category, description, and parameter count.

Examples:
  templar component list                        # List all templates
  templar component list --category layout     # List templates in specific category
  templar component list --format table        # Show in table format`,
	RunE: runComponentList,
}

var componentScaffoldCmd = &cobra.Command{
	Use:   "scaffold",
	Short: "Create complete project scaffold",
	Long: `Create a complete project scaffold with essential components and structure.

This generates:
- Directory structure (components, views, styles, docs)
- Essential components (Button, Card, Layout, Form, Navigation)
- Base styles and utilities
- Component documentation
- Test files

Examples:
  templar component scaffold                    # Scaffold in current directory
  templar component scaffold --output ./my-app # Scaffold in specific directory`,
	RunE: runComponentScaffold,
}

var (
	componentTemplate   string
	componentOutput     string
	componentPackage    string
	componentWithTests  bool
	componentWithDocs   bool
	componentWithStyles bool
	componentCategory   string
	componentFormat     string
	componentAuthor     string
	componentProject    string
)

func init() {
	rootCmd.AddCommand(componentCmd)

	// Add subcommands
	componentCmd.AddCommand(componentCreateCmd)
	componentCmd.AddCommand(componentListCmd)
	componentCmd.AddCommand(componentScaffoldCmd)

	// Create command flags
	componentCreateCmd.Flags().
		StringVarP(&componentTemplate, "template", "t", "", "Template to use (required)")
	componentCreateCmd.Flags().
		StringVarP(&componentOutput, "output", "o", "./components", "Output directory")
	componentCreateCmd.Flags().
		StringVarP(&componentPackage, "package", "p", "components", "Package name")
	componentCreateCmd.Flags().
		BoolVar(&componentWithTests, "with-tests", false, "Generate test files")
	componentCreateCmd.Flags().
		BoolVar(&componentWithDocs, "with-docs", false, "Generate documentation")
	componentCreateCmd.Flags().
		BoolVar(&componentWithStyles, "with-styles", false, "Generate CSS styles")
	componentCreateCmd.Flags().StringVar(&componentAuthor, "author", "", "Component author")
	componentCreateCmd.Flags().StringVar(&componentProject, "project", "", "Project name")
	componentCreateCmd.MarkFlagRequired("template")

	// List command flags
	componentListCmd.Flags().StringVar(&componentCategory, "category", "", "Filter by category")
	componentListCmd.Flags().
		StringVar(&componentFormat, "format", "list", "Output format (list, table, json)")

	// Scaffold command flags
	componentScaffoldCmd.Flags().
		StringVarP(&componentOutput, "output", "o", ".", "Output directory")
	componentScaffoldCmd.Flags().
		StringVarP(&componentPackage, "package", "p", "components", "Package name")
	componentScaffoldCmd.Flags().StringVar(&componentAuthor, "author", "", "Project author")
	componentScaffoldCmd.Flags().StringVar(&componentProject, "project", "", "Project name")
}

func runComponentCreate(cmd *cobra.Command, args []string) error {
	componentName := args[0]

	// Validate component name
	if err := scaffolding.ValidateComponentName(componentName); err != nil {
		return fmt.Errorf("invalid component name: %w", err)
	}

	// Get current directory if project name not specified
	if componentProject == "" {
		cwd, err := os.Getwd()
		if err == nil {
			componentProject = filepath.Base(cwd)
		}
	}

	// Create generator
	generator := scaffolding.NewComponentGenerator(
		componentOutput,
		componentPackage,
		componentProject,
		componentAuthor,
	)

	// Check if template exists
	if _, exists := generator.GetTemplate(componentTemplate); !exists {
		fmt.Printf("❌ Template '%s' not found.\n\n", componentTemplate)
		fmt.Println("Available templates:")
		templates := generator.ListTemplates()
		for _, tmpl := range templates {
			fmt.Printf("  • %s - %s (%s)\n", tmpl.Name, tmpl.Description, tmpl.Category)
		}
		return fmt.Errorf("template not found")
	}

	// Generate component
	opts := scaffolding.GenerateOptions{
		Name:        componentName,
		Template:    componentTemplate,
		OutputDir:   componentOutput,
		PackageName: componentPackage,
		ProjectName: componentProject,
		Author:      componentAuthor,
		WithTests:   componentWithTests,
		WithDocs:    componentWithDocs,
		WithStyles:  componentWithStyles,
	}

	fmt.Printf("🏗️  Generating component: %s\n", componentName)
	fmt.Printf("   Template: %s\n", componentTemplate)
	fmt.Printf("   Output: %s\n", componentOutput)
	fmt.Printf("   Package: %s\n", componentPackage)

	if err := generator.Generate(opts); err != nil {
		return fmt.Errorf("failed to generate component: %w", err)
	}

	fmt.Printf("\n🎉 Component '%s' generated successfully!\n", componentName)

	// Show next steps
	fmt.Println("\nNext steps:")
	fmt.Printf("  1. Review the generated files in %s\n", componentOutput)
	if componentWithTests {
		fmt.Println("  2. Run tests: go test ./...")
	}
	if componentWithStyles {
		fmt.Println("  3. Include CSS in your project")
	}
	fmt.Println("  4. Import and use in your templates")

	return nil
}

func runComponentList(cmd *cobra.Command, args []string) error {
	generator := scaffolding.NewComponentGenerator("", "", "", "")

	switch componentFormat {
	case "table":
		return listTemplatesTable(generator)
	case "json":
		return listTemplatesJSON(generator)
	default:
		return listTemplatesList(generator)
	}
}

func listTemplatesList(generator *scaffolding.ComponentGenerator) error {
	fmt.Println("📋 Available Component Templates")
	fmt.Println("==============================")

	categories := generator.GetTemplatesByCategory()

	for category, templates := range categories {
		if componentCategory != "" && category != componentCategory {
			continue
		}

		fmt.Printf("\n🏷️  %s\n", strings.ToUpper(category))
		fmt.Println(strings.Repeat("-", len(category)+4))

		for _, tmpl := range templates {
			fmt.Printf("  • %-15s %s\n", tmpl.Name, tmpl.Description)
			if tmpl.Parameters > 0 {
				fmt.Printf("    └─ %d parameters\n", tmpl.Parameters)
			}
		}
	}

	fmt.Println("\nUsage:")
	fmt.Println("  templar component create <name> --template <template>")
	fmt.Println("\nExamples:")
	fmt.Println("  templar component create MyButton --template button")
	fmt.Println("  templar component create UserCard --template card --with-tests")

	return nil
}

func listTemplatesTable(generator *scaffolding.ComponentGenerator) error {
	fmt.Printf("%-15s %-12s %-10s %s\n", "NAME", "CATEGORY", "PARAMS", "DESCRIPTION")
	fmt.Println(strings.Repeat("-", 80))

	templates := generator.ListTemplates()
	for _, tmpl := range templates {
		if componentCategory != "" && tmpl.Category != componentCategory {
			continue
		}
		fmt.Printf("%-15s %-12s %-10d %s\n",
			tmpl.Name,
			tmpl.Category,
			tmpl.Parameters,
			tmpl.Description)
	}

	return nil
}

func listTemplatesJSON(generator *scaffolding.ComponentGenerator) error {
	templates := generator.ListTemplates()

	// Filter by category if specified
	if componentCategory != "" {
		filtered := []scaffolding.TemplateInfo{}
		for _, tmpl := range templates {
			if tmpl.Category == componentCategory {
				filtered = append(filtered, tmpl)
			}
		}
		templates = filtered
	}

	// Simple JSON output (for production, use json.Marshal)
	fmt.Println("[")
	for i, tmpl := range templates {
		comma := ","
		if i == len(templates)-1 {
			comma = ""
		}
		fmt.Printf("  {\n")
		fmt.Printf("    \"name\": \"%s\",\n", tmpl.Name)
		fmt.Printf("    \"description\": \"%s\",\n", tmpl.Description)
		fmt.Printf("    \"category\": \"%s\",\n", tmpl.Category)
		fmt.Printf("    \"parameters\": %d\n", tmpl.Parameters)
		fmt.Printf("  }%s\n", comma)
	}
	fmt.Println("]")

	return nil
}

func runComponentScaffold(cmd *cobra.Command, args []string) error {
	// Get current directory if project name not specified
	if componentProject == "" {
		cwd, err := os.Getwd()
		if err == nil {
			componentProject = filepath.Base(cwd)
		}
	}

	// Create generator
	generator := scaffolding.NewComponentGenerator(
		componentOutput,
		componentPackage,
		componentProject,
		componentAuthor,
	)

	fmt.Printf("🏗️  Creating project scaffold in: %s\n", componentOutput)
	fmt.Printf("   Package: %s\n", componentPackage)
	fmt.Printf("   Project: %s\n", componentProject)

	if err := generator.CreateProjectScaffold(componentOutput); err != nil {
		return fmt.Errorf("failed to create project scaffold: %w", err)
	}

	fmt.Printf("\n🎉 Project scaffold created successfully!\n")
	fmt.Printf("\nGenerated structure:\n")
	fmt.Printf("  %s/\n", componentOutput)
	fmt.Printf("  ├── components/\n")
	fmt.Printf("  │   ├── ui/       (Button, Card)\n")
	fmt.Printf("  │   ├── layout/   (Layout, Navigation)\n")
	fmt.Printf("  │   └── forms/    (Form)\n")
	fmt.Printf("  ├── views/\n")
	fmt.Printf("  ├── styles/\n")
	fmt.Printf("  ├── docs/\n")
	fmt.Printf("  └── examples/\n")

	fmt.Println("\nNext steps:")
	fmt.Printf("  1. cd %s\n", componentOutput)
	fmt.Println("  2. templar serve")
	fmt.Println("  3. Start building your components!")

	return nil
}
