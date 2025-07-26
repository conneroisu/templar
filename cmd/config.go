package cmd

import (
	"fmt"
	"os"

	"github.com/conneroisu/templar/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage Templar configuration",
	Long: `Manage Templar configuration files and settings.

This command provides subcommands for:
- Creating new configuration through an interactive wizard
- Validating existing configuration files
- Showing current configuration values

Examples:
  templar config wizard                # Run interactive configuration wizard
  templar config validate              # Validate current configuration
  templar config show                  # Show current configuration
  templar config validate --file .templar.yml  # Validate specific config file`,
}

var configWizardCmd = &cobra.Command{
	Use:   "wizard",
	Short: "Run interactive configuration wizard",
	Long: `Run an interactive configuration wizard to set up your Templar project.

The wizard will guide you through all configuration options including:
- Server settings (port, host, environment)
- Component scanning paths and exclusions
- Build configuration and watch patterns
- Development features (hot reload, error overlay)
- Preview settings and mock data
- Plugin configuration

Examples:
  templar config wizard                # Run wizard and save to .templar.yml
  templar config wizard --output config.yml  # Save to custom file`,
	RunE: runConfigWizard,
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration file",
	Long: `Validate a Templar configuration file for correctness and best practices.

This command checks for:
- Required fields and proper data types
- Valid port ranges and hostnames
- Proper file paths and directory structure
- Security issues in configuration
- Performance recommendations

Examples:
  templar config validate              # Validate .templar.yml in current directory
  templar config validate --file config.yml  # Validate specific file
  templar config validate --strict    # Enable strict validation with warnings as errors`,
	RunE: runConfigValidate,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long: `Display the current Templar configuration including all resolved values.

This shows the final configuration after:
- Loading from configuration file
- Applying environment variable overrides
- Setting default values
- Processing command-line flags

Examples:
  templar config show                  # Show all configuration
  templar config show --format yaml   # Show in YAML format
  templar config show --format json   # Show in JSON format`,
	RunE: runConfigShow,
}

var (
	configOutput   string
	configFile     string
	configFormat   string
	configStrict   bool
	configNoWizard bool
)

func init() {
	rootCmd.AddCommand(configCmd)

	// Add subcommands
	configCmd.AddCommand(configWizardCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configShowCmd)

	// Wizard flags
	configWizardCmd.Flags().
		StringVarP(&configOutput, "output", "o", ".templar.yml", "Output configuration file")

	// Validate flags
	configValidateCmd.Flags().
		StringVarP(&configFile, "file", "f", "", "Configuration file to validate (default: .templar.yml)")
	configValidateCmd.Flags().BoolVar(&configStrict, "strict", false, "Treat warnings as errors")

	// Show flags
	configShowCmd.Flags().StringVar(&configFormat, "format", "yaml", "Output format (yaml, json)")

	// Main config command flags
	configCmd.Flags().BoolVar(&configNoWizard, "no-wizard", false, "Skip wizard and use defaults")
}

func runConfigWizard(cmd *cobra.Command, args []string) error {
	fmt.Println("🧙 Starting Templar Configuration Wizard")
	fmt.Println("========================================")

	// Check if output file already exists
	if _, err := os.Stat(configOutput); err == nil {
		fmt.Printf("⚠️  Configuration file %s already exists.\n", configOutput)
		fmt.Print("Do you want to overwrite it? (y/N): ")

		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			fmt.Printf("Failed to read input: %v\n", err)

			return err
		}

		if response != "y" && response != "Y" && response != "yes" && response != "Yes" {
			fmt.Println("Configuration wizard cancelled.")

			return nil
		}
	}

	// Create and run wizard
	wizard := config.NewConfigWizard()

	cfg, err := wizard.Run()
	if err != nil {
		return fmt.Errorf("configuration wizard failed: %w", err)
	}

	// Validate the generated configuration
	validation := config.ValidateConfigWithDetails(cfg)
	if validation.HasErrors() {
		fmt.Println("\n❌ Configuration validation failed:")
		fmt.Print(validation.String())

		return errors.New("generated configuration is invalid")
	}

	if validation.HasWarnings() {
		fmt.Println("\n⚠️  Configuration warnings:")
		fmt.Print(validation.String())
		fmt.Print("Continue anyway? (y/N): ")

		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			fmt.Printf("Failed to read input: %v\n", err)

			return err
		}

		if response != "y" && response != "Y" && response != "yes" && response != "Yes" {
			fmt.Println("Configuration wizard cancelled.")

			return nil
		}
	}

	// Write configuration file
	if err := wizard.WriteConfigFile(configOutput); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	fmt.Printf("\n🎉 Configuration wizard completed successfully!\n")
	fmt.Printf("Configuration saved to: %s\n", configOutput)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  1. Review the configuration file\n")
	fmt.Printf("  2. Run 'templar serve' to start the development server\n")
	fmt.Printf("  3. Open http://localhost:8080 in your browser\n")

	return nil
}

func runConfigValidate(cmd *cobra.Command, args []string) error {
	// Determine config file to validate
	targetFile := configFile
	if targetFile == "" {
		// Look for .templar.yml in current directory
		if _, err := os.Stat(".templar.yml"); err == nil {
			targetFile = ".templar.yml"
		} else {
			return errors.New("no configuration file found. Use --file to specify a config file " +
				"or run 'templar config wizard' to create one")
		}
	}

	// Check if file exists
	if _, err := os.Stat(targetFile); os.IsNotExist(err) {
		return fmt.Errorf("configuration file %s does not exist", targetFile)
	}

	fmt.Printf("🔍 Validating configuration file: %s\n", targetFile)
	fmt.Println("=====================================")

	// Load the configuration using Viper
	v := viper.New()
	v.SetConfigFile(targetFile)

	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read configuration file: %w", err)
	}

	// Unmarshal into config struct
	var cfg config.Config
	if err := v.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("failed to parse configuration: %w", err)
	}

	// Run detailed validation
	validation := config.ValidateConfigWithDetails(&cfg)

	if validation.Valid && !validation.HasWarnings() {
		fmt.Println("✅ Configuration is valid!")
		fmt.Println("No errors or warnings found.")

		return nil
	}

	// Print validation results
	if validation.HasErrors() {
		fmt.Print(validation.String())

		return fmt.Errorf("configuration validation failed with %d errors", len(validation.Errors))
	}

	if validation.HasWarnings() {
		fmt.Print(validation.String())

		if configStrict {
			return fmt.Errorf(
				"configuration validation failed in strict mode with %d warnings",
				len(validation.Warnings),
			)
		}

		fmt.Println("✅ Configuration is valid with warnings.")
		fmt.Printf(
			"Found %d warnings. Use --strict to treat warnings as errors.\n",
			len(validation.Warnings),
		)
	}

	return nil
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	fmt.Println("📋 Current Templar Configuration")
	fmt.Println("===============================")

	// Load current configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Show configuration in requested format
	switch configFormat {
	case "yaml", "yml":
		return showConfigYAML(cfg)
	case "json":
		return showConfigJSON(cfg)
	default:
		return fmt.Errorf("unsupported format: %s (supported: yaml, json)", configFormat)
	}
}

func showConfigYAML(cfg *config.Config) error {
	fmt.Println("# Current Templar Configuration")
	fmt.Println("# Resolved from all sources (file, env vars, defaults)")
	fmt.Println()

	// Server configuration
	fmt.Println("server:")
	fmt.Printf("  port: %d\n", cfg.Server.Port)
	fmt.Printf("  host: %s\n", cfg.Server.Host)
	fmt.Printf("  open: %t\n", cfg.Server.Open)
	fmt.Printf("  environment: %s\n", cfg.Server.Environment)
	if len(cfg.Server.Middleware) > 0 {
		fmt.Println("  middleware:")
		for _, middleware := range cfg.Server.Middleware {
			fmt.Printf("    - %s\n", middleware)
		}
	}
	if len(cfg.Server.AllowedOrigins) > 0 {
		fmt.Println("  allowed_origins:")
		for _, origin := range cfg.Server.AllowedOrigins {
			fmt.Printf("    - %s\n", origin)
		}
	}
	fmt.Println()

	// Build configuration
	fmt.Println("build:")
	fmt.Printf("  command: \"%s\"\n", cfg.Build.Command)
	if len(cfg.Build.Watch) > 0 {
		fmt.Println("  watch:")
		for _, pattern := range cfg.Build.Watch {
			fmt.Printf("    - \"%s\"\n", pattern)
		}
	}
	if len(cfg.Build.Ignore) > 0 {
		fmt.Println("  ignore:")
		for _, pattern := range cfg.Build.Ignore {
			fmt.Printf("    - \"%s\"\n", pattern)
		}
	}
	fmt.Printf("  cache_dir: \"%s\"\n", cfg.Build.CacheDir)
	fmt.Println()

	// Preview configuration
	fmt.Println("preview:")
	fmt.Printf("  mock_data: \"%s\"\n", cfg.Preview.MockData)
	fmt.Printf("  wrapper: \"%s\"\n", cfg.Preview.Wrapper)
	fmt.Printf("  auto_props: %t\n", cfg.Preview.AutoProps)
	fmt.Println()

	// Components configuration
	fmt.Println("components:")
	if len(cfg.Components.ScanPaths) > 0 {
		fmt.Println("  scan_paths:")
		for _, path := range cfg.Components.ScanPaths {
			fmt.Printf("    - \"%s\"\n", path)
		}
	}
	if len(cfg.Components.ExcludePatterns) > 0 {
		fmt.Println("  exclude_patterns:")
		for _, pattern := range cfg.Components.ExcludePatterns {
			fmt.Printf("    - \"%s\"\n", pattern)
		}
	}
	fmt.Println()

	// Development configuration
	fmt.Println("development:")
	fmt.Printf("  hot_reload: %t\n", cfg.Development.HotReload)
	fmt.Printf("  css_injection: %t\n", cfg.Development.CSSInjection)
	fmt.Printf("  state_preservation: %t\n", cfg.Development.StatePreservation)
	fmt.Printf("  error_overlay: %t\n", cfg.Development.ErrorOverlay)
	fmt.Println()

	// Plugins configuration
	if len(cfg.Plugins.Enabled) > 0 || len(cfg.Plugins.DiscoveryPaths) > 0 {
		fmt.Println("plugins:")
		if len(cfg.Plugins.Enabled) > 0 {
			fmt.Println("  enabled:")
			for _, plugin := range cfg.Plugins.Enabled {
				fmt.Printf("    - %s\n", plugin)
			}
		}
		if len(cfg.Plugins.Disabled) > 0 {
			fmt.Println("  disabled:")
			for _, plugin := range cfg.Plugins.Disabled {
				fmt.Printf("    - %s\n", plugin)
			}
		}
		if len(cfg.Plugins.DiscoveryPaths) > 0 {
			fmt.Println("  discovery_paths:")
			for _, path := range cfg.Plugins.DiscoveryPaths {
				fmt.Printf("    - \"%s\"\n", path)
			}
		}
	}

	return nil
}

func showConfigJSON(cfg *config.Config) error {
	fmt.Println("// Current Templar Configuration")
	fmt.Println("// Resolved from all sources (file, env vars, defaults)")
	fmt.Println("{")

	// This is a simplified JSON output - for production, you'd use json.Marshal
	fmt.Printf("  \"server\": {\n")
	fmt.Printf("    \"port\": %d,\n", cfg.Server.Port)
	fmt.Printf("    \"host\": \"%s\",\n", cfg.Server.Host)
	fmt.Printf("    \"open\": %t,\n", cfg.Server.Open)
	fmt.Printf("    \"environment\": \"%s\"\n", cfg.Server.Environment)
	fmt.Printf("  },\n")

	fmt.Printf("  \"build\": {\n")
	fmt.Printf("    \"command\": \"%s\",\n", cfg.Build.Command)
	fmt.Printf("    \"cache_dir\": \"%s\"\n", cfg.Build.CacheDir)
	fmt.Printf("  },\n")

	fmt.Printf("  \"development\": {\n")
	fmt.Printf("    \"hot_reload\": %t,\n", cfg.Development.HotReload)
	fmt.Printf("    \"css_injection\": %t,\n", cfg.Development.CSSInjection)
	fmt.Printf("    \"error_overlay\": %t\n", cfg.Development.ErrorOverlay)
	fmt.Printf("  }\n")

	fmt.Println("}")

	return nil
}
