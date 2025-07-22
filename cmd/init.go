package cmd

import (
	"fmt"
	"os"

	"github.com/conneroisu/templar/internal/errors"
	"github.com/conneroisu/templar/internal/services"
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
			return errors.CLIError("INIT", "failed to get current directory", err)
		}
		projectDir = cwd
	} else if len(args) == 1 {
		// Initialize in new directory
		projectDir = args[0]
	} else {
		// Too many arguments
		return errors.ArgumentError("project_name", "too many arguments provided", args)
	}

	fmt.Printf("Initializing templar project in %s\n", projectDir)

	// Create initialization service
	initService := services.NewInitService()

	// Configure initialization options
	opts := services.InitOptions{
		ProjectDir: projectDir,
		Minimal:    initMinimal,
		Example:    initExample,
		Template:   initTemplate,
		Wizard:     initWizard,
	}

	// Initialize the project using the service
	if err := initService.InitProject(opts); err != nil {
		return err
	}

	fmt.Println("✓ Project initialized successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. cd " + projectDir)
	fmt.Println("  2. templar serve")
	fmt.Println("  3. Open http://localhost:8080 in your browser")

	return nil
}