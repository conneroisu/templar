// Package cmd provides the command-line interface for Templar with comprehensive
// configuration management supporting multiple configuration sources.
//
// Configuration System:
//
//	The CLI supports flexible configuration through multiple sources with clear precedence:
//	1. Command-line flags (--config, --port, etc.) - highest priority
//	2. TEMPLAR_CONFIG_FILE environment variable - custom config file path
//	3. Individual environment variables (TEMPLAR_SERVER_PORT, etc.)
//	4. Configuration files (.templar.yml) - lowest priority
//
// Environment Variables:
//
//	TEMPLAR_CONFIG_FILE: Path to custom configuration file
//	TEMPLAR_SERVER_PORT: Override server port
//	TEMPLAR_SERVER_HOST: Override server host
//	TEMPLAR_DEVELOPMENT_HOT_RELOAD: Enable/disable hot reload
//	And many more following the TEMPLAR_<SECTION>_<OPTION> pattern
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "templar",
	Short: "A rapid prototyping CLI helper tool for Go templ",
	Long: `Templar is a rapid prototyping CLI helper tool for Go templ that provides
browser preview functionality, hot reload capability, and streamlined development workflows.

Key Features:
  • Component discovery and scanning
  • Hot reload development server
  • Component isolation and preview
  • Build pipeline integration
  • Mock data generation
  • WebSocket-based live updates

Quick Start:
  templar init                    Initialize a new project
  templar serve                   Start development server
  templar list                    List all components
  templar build                   Build all components
  templar interactive             Interactive command menu

Command Aliases (for faster typing):
  init (i), serve (s), preview (p), build (b), list (l), watch (w), interactive (m)

Documentation: https://github.com/conneroisu/templar`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .templar.yml, can also use TEMPLAR_CONFIG_FILE env var)")
	rootCmd.PersistentFlags().StringP("log-level", "l", "info", "log level (debug, info, warn, error)")
	viper.BindPFlag("log-level", rootCmd.PersistentFlags().Lookup("log-level"))
}

// initConfig initializes the configuration system with support for multiple config sources.
//
// Configuration Loading Priority (highest to lowest):
//  1. --config flag: Explicitly specified config file path
//  2. TEMPLAR_CONFIG_FILE environment variable: Custom config file path
//  3. Default: .templar.yml in current directory
//
// Environment Variable Usage:
//
//	export TEMPLAR_CONFIG_FILE=/path/to/custom-config.yml
//	templar serve  # Uses custom-config.yml
//
//	export TEMPLAR_CONFIG_FILE=./configs/dev.yml
//	templar serve --config prod.yml  # Uses prod.yml (flag overrides env var)
//
// The function also enables automatic environment variable binding for all
// configuration values with the TEMPLAR_ prefix (e.g., TEMPLAR_SERVER_PORT=8080).
func initConfig() {
	// Priority 1: Use config file specified via --config flag (highest priority)
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else if envConfigFile := os.Getenv("TEMPLAR_CONFIG_FILE"); envConfigFile != "" {
		// Priority 2: Use config file specified via TEMPLAR_CONFIG_FILE environment variable
		// This allows users to set a project-specific config without modifying command line
		// Supports both relative paths (./custom-config.yml) and absolute paths
		viper.SetConfigFile(envConfigFile)
	} else {
		// Priority 3: Search for default .templar.yml in current directory (lowest priority)
		// This maintains backward compatibility with existing projects
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".templar")
	}

	// Enable automatic environment variable binding with TEMPLAR_ prefix
	// Examples: TEMPLAR_SERVER_PORT, TEMPLAR_SERVER_HOST, TEMPLAR_DEVELOPMENT_HOT_RELOAD
	viper.SetEnvPrefix("TEMPLAR")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Attempt to read the configuration file
	// If file doesn't exist or has errors, Viper will use defaults without failing
	// This ensures graceful degradation when config files are missing or malformed
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
