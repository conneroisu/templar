package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// StandardFlags provides consistent flag definitions across commands
type StandardFlags struct {
	// Server flags
	Port           int    `flag:"port,p" desc:"Port to serve on" default:"8080"`
	Host           string `flag:"host" desc:"Host to bind to" default:"localhost"`
	DisableBrowser bool   `flag:"disable-browser" desc:"Don't open browser automatically" default:"false"`

	// Component flags
	Props     string `flag:"props" desc:"Component properties (JSON or @file.json)" default:""`
	PropsFile string `flag:"props-file,f" desc:"Properties file (JSON)" default:""`
	MockData  string `flag:"mock,m" desc:"Mock data file or pattern" default:""`
	Wrapper   string `flag:"wrapper,w" desc:"Wrapper template" default:""`

	// Build flags
	WatchPattern string `flag:"watch" desc:"File watch pattern" default:"**/*.templ"`
	BuildCmd     string `flag:"build-cmd" desc:"Build command to run" default:"templ generate"`

	// Output flags
	OutputFormat string `flag:"output,o" desc:"Output format (table|json|yaml)" default:"table"`
	Verbose      bool   `flag:"verbose,v" desc:"Enable verbose output" default:"false"`
	Quiet        bool   `flag:"quiet,q" desc:"Suppress output" default:"false"`
}

// AddStandardFlags adds standard flags to a command
func AddStandardFlags(cmd *cobra.Command, flagTypes ...string) *StandardFlags {
	flags := &StandardFlags{}

	for _, flagType := range flagTypes {
		switch flagType {
		case "server":
			addServerFlags(cmd, flags)
		case "component":
			addComponentFlags(cmd, flags)
		case "build":
			addBuildFlags(cmd, flags)
		case "output":
			addOutputFlags(cmd, flags)
		}
	}

	return flags
}

func addServerFlags(cmd *cobra.Command, flags *StandardFlags) {
	cmd.Flags().IntVarP(&flags.Port, "port", "p", 8080, "Port to serve on")
	cmd.Flags().StringVar(&flags.Host, "host", "localhost", "Host to bind to")
	cmd.Flags().BoolVar(&flags.DisableBrowser, "disable-browser", false, "Don't open browser automatically")
}

func addComponentFlags(cmd *cobra.Command, flags *StandardFlags) {
	cmd.Flags().StringVar(&flags.Props, "props", "", "Component properties (JSON or @file.json)")
	cmd.Flags().StringVarP(&flags.PropsFile, "props-file", "f", "", "Properties file (JSON)")
	cmd.Flags().StringVarP(&flags.MockData, "mock", "m", "", "Mock data file or pattern")
	cmd.Flags().StringVarP(&flags.Wrapper, "wrapper", "w", "", "Wrapper template")
}

func addBuildFlags(cmd *cobra.Command, flags *StandardFlags) {
	cmd.Flags().StringVar(&flags.WatchPattern, "watch", "**/*.templ", "File watch pattern")
	cmd.Flags().StringVar(&flags.BuildCmd, "build-cmd", "templ generate", "Build command to run")
}

func addOutputFlags(cmd *cobra.Command, flags *StandardFlags) {
	cmd.Flags().StringVarP(&flags.OutputFormat, "output", "o", "table", "Output format (table|json|yaml)")
	cmd.Flags().BoolVarP(&flags.Verbose, "verbose", "v", false, "Enable verbose output")
	cmd.Flags().BoolVarP(&flags.Quiet, "quiet", "q", false, "Suppress output")
}

// ParseProps parses component properties with support for file references
func (f *StandardFlags) ParseProps() (map[string]interface{}, error) {
	var props map[string]interface{}

	// If PropsFile is specified, use it
	if f.PropsFile != "" {
		data, err := os.ReadFile(f.PropsFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read props file %s: %w", f.PropsFile, err)
		}

		if err := json.Unmarshal(data, &props); err != nil {
			return nil, fmt.Errorf("invalid JSON in props file %s: %w", f.PropsFile, err)
		}

		return props, nil
	}

	// If Props starts with @, treat as file reference
	if strings.HasPrefix(f.Props, "@") {
		filename := strings.TrimPrefix(f.Props, "@")
		data, err := os.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to read props file %s: %w", filename, err)
		}

		if err := json.Unmarshal(data, &props); err != nil {
			return nil, fmt.Errorf("invalid JSON in props file %s: %w", filename, err)
		}

		return props, nil
	}

	// Parse as inline JSON
	if f.Props != "" {
		if err := json.Unmarshal([]byte(f.Props), &props); err != nil {
			return nil, fmt.Errorf("invalid JSON in props: %w", err)
		}

		return props, nil
	}

	return make(map[string]interface{}), nil
}

// ShouldOpenBrowser returns whether to open browser based on flags
func (f *StandardFlags) ShouldOpenBrowser() bool {
	return !f.DisableBrowser
}

// ValidateFlags validates flag combinations and values
func (f *StandardFlags) ValidateFlags() error {
	// Port validation
	if f.Port < 1 || f.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", f.Port)
	}

	// Host validation
	if f.Host == "" {
		return fmt.Errorf("host cannot be empty")
	}

	// Props validation
	if f.Props != "" && f.PropsFile != "" {
		return fmt.Errorf("cannot specify both --props and --props-file")
	}

	// Output format validation
	validFormats := []string{"table", "json", "yaml"}
	if f.OutputFormat != "" {
		valid := false
		for _, format := range validFormats {
			if f.OutputFormat == format {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid output format %s, must be one of: %s",
				f.OutputFormat, strings.Join(validFormats, ", "))
		}
	}

	// Quiet and verbose are mutually exclusive
	if f.Quiet && f.Verbose {
		return fmt.Errorf("cannot specify both --quiet and --verbose")
	}

	return nil
}

// SetViperBindings binds flags to viper configuration keys
func SetViperBindings(cmd *cobra.Command, bindings map[string]string) {
	for flagName, configKey := range bindings {
		if flag := cmd.Flags().Lookup(flagName); flag != nil {
			// This would require viper import, but we keep it simple for now
			_ = configKey // Placeholder for viper binding
		}
	}
}

// AddFlagValidation adds validation for a specific flag
func AddFlagValidation(cmd *cobra.Command, flagName string, validator func(string) error) {
	flag := cmd.Flags().Lookup(flagName)
	if flag == nil {
		return
	}

	// Store original value setter
	originalSet := flag.Value.Set

	// Create wrapper that validates
	flag.Value = &validatingValue{
		Value:       flag.Value,
		validator:   validator,
		originalSet: originalSet,
	}
}

type validatingValue struct {
	pflag.Value
	validator   func(string) error
	originalSet func(string) error
}

func (v *validatingValue) Set(val string) error {
	if v.validator != nil {
		if err := v.validator(val); err != nil {
			return err
		}
	}
	return v.originalSet(val)
}

// Port validation helper
func ValidatePort(portStr string) error {
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port number: %s", portStr)
	}

	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", port)
	}

	return nil
}

// File existence validation helper
func ValidateFileExists(filename string) error {
	if filename == "" {
		return nil // Empty is valid for optional files
	}

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filename)
	}

	return nil
}

// JSON validation helper
func ValidateJSON(jsonStr string) error {
	if jsonStr == "" {
		return nil
	}

	var temp interface{}
	if err := json.Unmarshal([]byte(jsonStr), &temp); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	return nil
}
