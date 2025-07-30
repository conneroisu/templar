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
	Short:   "Initialize a new Templar project with templates and configuration",
	Long: `Initialize a new Templar project with the necessary directory structure
and configuration files. If no name is provided, initializes in the current directory.

The initialization process creates a project structure optimized for templ component
development with smart defaults and optional templates for common use cases.

Examples:
  templar init                         # Initialize in current directory
  templar init my-project              # Create new project directory
  templar init --minimal               # Minimal setup without examples
  templar init --wizard                # Interactive configuration wizard
  templar init --template blog         # Use blog template with posts and layouts
  templar init --template dashboard    # Use dashboard template with navigation
  templar init --template landing      # Use landing page template with sections
  templar init --template ecommerce    # Use e-commerce template with cart
  templar init --template docs         # Use documentation template

Available Templates:
  minimal        Basic component setup (default)
  blog          Blog posts, layouts, and content management
  dashboard     Admin dashboard with sidebar navigation and data cards
  landing       Marketing landing page with hero sections and feature lists
  ecommerce     Product listings, shopping cart, and purchase flows
  docs          Technical documentation with navigation and code blocks

Pro Tips:
  • Use --wizard for interactive project setup with smart defaults
  • All templates include production-ready components and styling
  • Templates work seamlessly with the development server and live preview
  • Generated projects include .templar.yml configuration for customization

See also: templar serve, templar list, templar build`,
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
	initCmd.Flags().
		BoolVar(&initWizard, "wizard", false, "Run configuration wizard during initialization")
}

func runInit(cmd *cobra.Command, args []string) error {
	var projectDir string

	switch len(args) {
	case 0:
		// Initialize in current directory
		cwd, err := os.Getwd()
		if err != nil {
			return errors.CLIError("INIT", "failed to get current directory", err)
		}
		projectDir = cwd
	case 1:
		// Initialize in new directory - validate the argument first
		if err := validateArgument(args[0]); err != nil {
			return errors.ValidationFailure("project_name", err.Error(), args[0], "Use a safe project name without special characters or path traversal sequences")
		}
		projectDir = args[0]
	default:
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
