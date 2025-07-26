package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/conneroisu/templar/internal/config"
	"github.com/conneroisu/templar/internal/plugins/css"
	"github.com/spf13/cobra"
)

// cssCmd represents the css command for managing CSS frameworks.
var cssCmd = &cobra.Command{
	Use:   "css",
	Short: "Manage CSS framework integration",
	Long: `Manage CSS framework integration for Templar projects.

The css command provides subcommands to:
- List available CSS frameworks
- Setup and configure CSS frameworks
- Generate style guides
- Manage theming and variables`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help() // nolint:errcheck
	},
}

// cssListCmd lists available CSS frameworks.
var cssListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available CSS frameworks",
	Long:  `List all available CSS frameworks that can be integrated with Templar.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		manager := css.NewFrameworkManager(cfg, ".")

		ctx := context.Background()
		if err := manager.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize framework manager: %w", err)
		}

		frameworks := manager.GetAvailableFrameworks()

		if len(frameworks) == 0 {
			fmt.Println("No CSS frameworks available")

			return nil
		}

		fmt.Println("Available CSS Frameworks:")
		fmt.Println("========================")

		for _, fw := range frameworks {
			fmt.Printf("\n%s (%s)\n", fw.DisplayName, fw.Name)
			if fw.Description != "" {
				fmt.Printf("  Description: %s\n", fw.Description)
			}
			if fw.Version != "" {
				fmt.Printf("  Version: %s\n", fw.Version)
			}
			if len(fw.SupportedVersions) > 0 {
				fmt.Printf("  Supported Versions: %s\n", strings.Join(fw.SupportedVersions, ", "))
			}
			if len(fw.InstallMethods) > 0 {
				fmt.Printf("  Install Methods: %s\n", strings.Join(fw.InstallMethods, ", "))
			}
			if fw.Website != "" {
				fmt.Printf("  Website: %s\n", fw.Website)
			}
		}

		return nil
	},
}

// cssSetupCmd sets up a CSS framework.
var cssSetupCmd = &cobra.Command{
	Use:   "setup [framework]",
	Short: "Setup a CSS framework",
	Long: `Setup and configure a CSS framework for your Templar project.

Available frameworks: tailwind, bootstrap, bulma

Examples:
  templar css setup tailwind
  templar css setup bootstrap --method npm
  templar css setup bulma --method cdn --version 1.0.2`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		frameworkName := args[0]

		// Get flags
		method, _ := cmd.Flags().GetString("method")
		version, _ := cmd.Flags().GetString("version")
		outputPath, _ := cmd.Flags().GetString("output")
		cdnUrl, _ := cmd.Flags().GetString("cdn-url")
		force, _ := cmd.Flags().GetBool("force")
		generateConfig, _ := cmd.Flags().GetBool("config")

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		manager := css.NewFrameworkManager(cfg, ".")

		ctx := context.Background()
		if err := manager.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize framework manager: %w", err)
		}

		// Create setup configuration
		setupConfig := css.FrameworkSetupConfig{
			InstallMethod:  method,
			Version:        version,
			CDNUrl:         cdnUrl,
			OutputPath:     outputPath,
			SourcePaths:    []string{"src/**/*.{templ,html,js,ts}", "components/**/*.{templ,html}"},
			GenerateConfig: generateConfig,
			Force:          force,
			Options:        make(map[string]interface{}),
		}

		// Setup the framework
		fmt.Printf("Setting up %s CSS framework...\n", frameworkName)
		if err := manager.SetupFramework(ctx, frameworkName, setupConfig); err != nil {
			return fmt.Errorf("failed to setup framework %s: %w", frameworkName, err)
		}

		fmt.Printf("✅ Successfully setup %s CSS framework\n", frameworkName)

		// Show next steps
		fmt.Println("\nNext steps:")
		fmt.Printf("1. Run 'templar serve' to start the development server\n")
		fmt.Printf("2. Edit your components to include %s classes\n", frameworkName)
		fmt.Printf("3. Use 'templar css styleguide' to generate a style guide\n")

		return nil
	},
}

// cssStyleguideCmd generates a style guide.
var cssStyleguideCmd = &cobra.Command{
	Use:   "styleguide",
	Short: "Generate a CSS framework style guide",
	Long:  `Generate a comprehensive style guide for the active CSS framework.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		outputPath, _ := cmd.Flags().GetString("output")
		if outputPath == "" {
			outputPath = "styleguide.html"
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		manager := css.NewFrameworkManager(cfg, ".")

		ctx := context.Background()
		if err := manager.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize framework manager: %w", err)
		}

		activeFramework := manager.GetActiveFramework()
		if activeFramework == "" {
			return errors.New("no active CSS framework found. Run 'templar css setup <framework>' first")
		}

		fmt.Printf("Generating style guide for %s...\n", activeFramework)

		styleGuide, err := manager.GenerateStyleGuide(ctx)
		if err != nil {
			return fmt.Errorf("failed to generate style guide: %w", err)
		}

		if err := os.WriteFile(outputPath, styleGuide, 0644); err != nil {
			return fmt.Errorf("failed to write style guide to %s: %w", outputPath, err)
		}

		fmt.Printf("✅ Style guide generated: %s\n", outputPath)

		return nil
	},
}

// cssThemeCmd manages theming.
var cssThemeCmd = &cobra.Command{
	Use:   "theme",
	Short: "Manage CSS theming and variables",
	Long:  `Manage CSS theming, variables, and custom styling for the active framework.`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help() // nolint:errcheck
	},
}

// cssThemeExtractCmd extracts CSS variables.
var cssThemeExtractCmd = &cobra.Command{
	Use:   "extract",
	Short: "Extract CSS variables from framework",
	Long:  `Extract CSS variables from the active framework for customization.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		outputPath, _ := cmd.Flags().GetString("output")
		if outputPath == "" {
			outputPath = "theme-variables.json"
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		manager := css.NewFrameworkManager(cfg, ".")

		ctx := context.Background()
		if err := manager.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize framework manager: %w", err)
		}

		activeFramework := manager.GetActiveFramework()
		if activeFramework == "" {
			return errors.New("no active CSS framework found. Run 'templar css setup <framework>' first")
		}

		fmt.Printf("Extracting variables from %s...\n", activeFramework)

		// Read current CSS if it exists
		cssPath := "dist/styles.css"
		var cssContent []byte
		if _, err := os.Stat(cssPath); err == nil {
			cssContent, _ = os.ReadFile(cssPath)
		}

		variables, err := manager.ExtractVariables(cssContent)
		if err != nil {
			return fmt.Errorf("failed to extract variables: %w", err)
		}

		if len(variables) == 0 {
			fmt.Println("No variables found to extract")

			return nil
		}

		// Format as JSON for easy editing
		var jsonContent strings.Builder
		jsonContent.WriteString("{\n")

		i := 0
		for name, value := range variables {
			if i > 0 {
				jsonContent.WriteString(",\n")
			}
			jsonContent.WriteString(fmt.Sprintf("  \"%s\": \"%s\"", name, value))
			i++
		}

		jsonContent.WriteString("\n}")

		if err := os.WriteFile(outputPath, []byte(jsonContent.String()), 0644); err != nil {
			return fmt.Errorf("failed to write variables to %s: %w", outputPath, err)
		}

		fmt.Printf("✅ Variables extracted to: %s\n", outputPath)
		fmt.Printf("Found %d variables\n", len(variables))

		return nil
	},
}

// cssThemeGenerateCmd generates a custom theme.
var cssThemeGenerateCmd = &cobra.Command{
	Use:   "generate [variables-file]",
	Short: "Generate a custom theme from variables",
	Long: `Generate a custom theme CSS file from a variables JSON file.

The variables file should be in JSON format with variable names and values:
{
  "primary": "#3b82f6",
  "secondary": "#6b7280",
  "success": "#10b981"
}`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		variablesFile := args[0]
		outputPath, _ := cmd.Flags().GetString("output")
		if outputPath == "" {
			outputPath = "custom-theme.css"
		}

		// Read variables file
		variablesContent, err := os.ReadFile(variablesFile)
		if err != nil {
			return fmt.Errorf("failed to read variables file %s: %w", variablesFile, err)
		}

		// Parse JSON (simplified - in practice, you'd use encoding/json)
		variables := make(map[string]string)

		// Simple JSON parsing for the demo
		content := string(variablesContent)
		content = strings.Trim(content, " \n\t{}")

		lines := strings.Split(content, ",")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.Trim(strings.TrimSpace(parts[0]), "\"")
				value := strings.Trim(strings.TrimSpace(parts[1]), "\"")
				variables[key] = value
			}
		}

		if len(variables) == 0 {
			return fmt.Errorf("no variables found in file %s", variablesFile)
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		manager := css.NewFrameworkManager(cfg, ".")

		ctx := context.Background()
		if err := manager.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize framework manager: %w", err)
		}

		activeFramework := manager.GetActiveFramework()
		if activeFramework == "" {
			return errors.New("no active CSS framework found. Run 'templar css setup <framework>' first")
		}

		fmt.Printf("Generating custom theme for %s...\n", activeFramework)

		themeCSS, err := manager.GenerateTheme(variables)
		if err != nil {
			return fmt.Errorf("failed to generate theme: %w", err)
		}

		if err := os.WriteFile(outputPath, themeCSS, 0644); err != nil {
			return fmt.Errorf("failed to write theme to %s: %w", outputPath, err)
		}

		fmt.Printf("✅ Custom theme generated: %s\n", outputPath)
		fmt.Printf("Applied %d custom variables\n", len(variables))

		return nil
	},
}

// cssValidateCmd validates CSS framework configuration.
var cssValidateCmd = &cobra.Command{
	Use:   "validate [framework]",
	Short: "Validate CSS framework configuration",
	Long:  `Validate the configuration and setup of a CSS framework.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		manager := css.NewFrameworkManager(cfg, ".")

		ctx := context.Background()
		if err := manager.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize framework manager: %w", err)
		}

		var frameworkName string
		if len(args) > 0 {
			frameworkName = args[0]
		} else {
			frameworkName = manager.GetActiveFramework()
			if frameworkName == "" {
				return errors.New("no framework specified and no active framework found")
			}
		}

		fmt.Printf("Validating %s configuration...\n", frameworkName)

		if err := manager.ValidateFramework(frameworkName); err != nil {
			fmt.Printf("❌ Validation failed: %v\n", err)

			return nil
		}

		fmt.Printf("✅ %s configuration is valid\n", frameworkName)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(cssCmd)

	// Add subcommands
	cssCmd.AddCommand(cssListCmd)
	cssCmd.AddCommand(cssSetupCmd)
	cssCmd.AddCommand(cssStyleguideCmd)
	cssCmd.AddCommand(cssThemeCmd)
	cssCmd.AddCommand(cssValidateCmd)

	// Add theme subcommands
	cssThemeCmd.AddCommand(cssThemeExtractCmd)
	cssThemeCmd.AddCommand(cssThemeGenerateCmd)

	// Setup command flags
	cssSetupCmd.Flags().StringP("method", "m", "npm", "Install method (npm, cdn, standalone)")
	cssSetupCmd.Flags().StringP("version", "v", "", "Framework version")
	cssSetupCmd.Flags().StringP("output", "o", "dist/styles.css", "CSS output path")
	cssSetupCmd.Flags().String("cdn-url", "", "Custom CDN URL")
	cssSetupCmd.Flags().BoolP("force", "f", false, "Force reinstall if already exists")
	cssSetupCmd.Flags().Bool("config", true, "Generate framework configuration file")

	// Styleguide command flags
	cssStyleguideCmd.Flags().
		StringP("output", "o", "styleguide.html", "Output path for style guide")

	// Theme extract flags
	cssThemeExtractCmd.Flags().
		StringP("output", "o", "theme-variables.json", "Output path for variables")

	// Theme generate flags
	cssThemeGenerateCmd.Flags().
		StringP("output", "o", "custom-theme.css", "Output path for custom theme")
}
