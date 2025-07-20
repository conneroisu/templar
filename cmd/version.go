package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/conneroisu/templar/internal/version"
	"github.com/spf13/cobra"
)

var (
	versionFormat string
	versionShort  bool
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long: `Display version information for templar including:

- Semantic version number
- Git commit hash
- Build timestamp
- Go version used for compilation
- Target platform (OS/architecture)
- Build user (if available)

Examples:
  templar version              # Show short version
  templar version --detailed   # Show detailed version info
  templar version --format json # Output as JSON`,
	RunE: runVersionCommand,
}

func init() {
	rootCmd.AddCommand(versionCmd)

	versionCmd.Flags().StringVarP(&versionFormat, "format", "f", "text", "Output format (text, json)")
	versionCmd.Flags().BoolVar(&versionShort, "short", false, "Show short version only")
	versionCmd.Flags().Bool("detailed", false, "Show detailed version information")
}

func runVersionCommand(cmd *cobra.Command, args []string) error {
	detailed, _ := cmd.Flags().GetBool("detailed")

	switch versionFormat {
	case "json":
		return outputVersionJSON()
	case "text":
		if versionShort {
			return outputVersionShort()
		} else if detailed {
			return outputVersionDetailed()
		} else {
			return outputVersionDefault()
		}
	default:
		return fmt.Errorf("unsupported format: %s (supported: text, json)", versionFormat)
	}
}

func outputVersionShort() error {
	fmt.Println(version.GetShortVersion())
	return nil
}

func outputVersionDefault() error {
	info := version.GetBuildInfo()

	fmt.Printf("templar %s", info.Version)

	if info.GitCommit != "unknown" && len(info.GitCommit) >= 7 {
		fmt.Printf(" (%s)", info.GitCommit[:7])
	}

	if version.IsDirty() {
		fmt.Print(" (dirty)")
	}

	fmt.Println()

	if !info.BuildTime.IsZero() {
		fmt.Printf("Built: %s\n", info.BuildTime.Format("2006-01-02 15:04:05 UTC"))
	}

	fmt.Printf("Go: %s\n", info.GoVersion)
	fmt.Printf("Platform: %s\n", info.Platform)

	return nil
}

func outputVersionDetailed() error {
	fmt.Println(version.GetDetailedVersion())

	if version.IsDirty() {
		fmt.Println("Working directory: dirty")
	}

	if version.IsRelease() {
		fmt.Println("Build type: release")
	} else {
		fmt.Println("Build type: development")
	}

	return nil
}

func outputVersionJSON() error {
	info := version.GetBuildInfo()

	// Add extra fields for JSON output
	jsonInfo := map[string]interface{}{
		"version":    info.Version,
		"git_commit": info.GitCommit,
		"build_time": info.BuildTime,
		"go_version": info.GoVersion,
		"platform":   info.Platform,
		"build_user": info.BuildUser,
		"is_release": version.IsRelease(),
		"is_dirty":   version.IsDirty(),
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(jsonInfo)
}
