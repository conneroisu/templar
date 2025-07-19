package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
)

// BuildInfo contains version and build information
type BuildInfo struct {
	Version   string    `json:"version"`
	GitCommit string    `json:"git_commit"`
	BuildTime time.Time `json:"build_time"`
	GoVersion string    `json:"go_version"`
	Platform  string    `json:"platform"`
	BuildUser string    `json:"build_user,omitempty"`
}

// These variables are set at build time using -ldflags
var (
	// Version is the semantic version of the application
	Version = "dev"
	
	// GitCommit is the git commit hash when the binary was built
	GitCommit = "unknown"
	
	// BuildTime is the time when the binary was built (RFC3339 format)
	BuildTime = "unknown"
	
	// BuildUser is the user who built the binary
	BuildUser = "unknown"
)

// GetBuildInfo returns comprehensive build information
func GetBuildInfo() *BuildInfo {
	buildTime := parseISOTime(BuildTime)
	
	return &BuildInfo{
		Version:   GetVersion(),
		GitCommit: GetGitCommit(),
		BuildTime: buildTime,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		BuildUser: BuildUser,
	}
}

// GetVersion returns the application version
func GetVersion() string {
	if Version != "" && Version != "dev" {
		return Version
	}
	
	// Try to get version from debug build info
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "(devel)" && info.Main.Version != "" {
			return info.Main.Version
		}
		
		// Look for version in build settings
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" && len(setting.Value) >= 7 {
				return fmt.Sprintf("dev-%s", setting.Value[:7])
			}
		}
	}
	
	return "dev"
}

// GetGitCommit returns the git commit hash
func GetGitCommit() string {
	if GitCommit != "" && GitCommit != "unknown" {
		return GitCommit
	}
	
	// Try to get commit from debug build info
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				return setting.Value
			}
		}
	}
	
	return "unknown"
}

// GetBuildTime returns the build time
func GetBuildTime() time.Time {
	return parseISOTime(BuildTime)
}

// GetShortVersion returns a short version string suitable for display
func GetShortVersion() string {
	version := GetVersion()
	commit := GetGitCommit()
	
	if commit != "unknown" && len(commit) >= 7 {
		shortCommit := commit[:7]
		if version != "dev" {
			return fmt.Sprintf("%s (%s)", version, shortCommit)
		}
		return fmt.Sprintf("dev-%s", shortCommit)
	}
	
	return version
}

// GetDetailedVersion returns a detailed version string with all build info
func GetDetailedVersion() string {
	info := GetBuildInfo()
	
	var parts []string
	parts = append(parts, fmt.Sprintf("Version: %s", info.Version))
	
	if info.GitCommit != "unknown" {
		parts = append(parts, fmt.Sprintf("Commit: %s", info.GitCommit))
	}
	
	if !info.BuildTime.IsZero() {
		parts = append(parts, fmt.Sprintf("Built: %s", info.BuildTime.Format(time.RFC3339)))
	}
	
	parts = append(parts, fmt.Sprintf("Go: %s", info.GoVersion))
	parts = append(parts, fmt.Sprintf("Platform: %s", info.Platform))
	
	if info.BuildUser != "unknown" && info.BuildUser != "" {
		parts = append(parts, fmt.Sprintf("User: %s", info.BuildUser))
	}
	
	return strings.Join(parts, "\n")
}

// IsRelease returns true if this is a release build (not dev)
func IsRelease() bool {
	version := GetVersion()
	return version != "dev" && !strings.HasPrefix(version, "dev-")
}

// IsDirty returns true if the working directory was dirty when built
func IsDirty() bool {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.modified" {
				return setting.Value == "true"
			}
		}
	}
	return false
}

// parseISOTime parses an ISO 8601 time string, returns zero time on error
func parseISOTime(timeStr string) time.Time {
	if timeStr == "" || timeStr == "unknown" {
		return time.Time{}
	}
	
	// Try RFC3339 format first (ISO 8601)
	if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
		return t
	}
	
	// Try RFC3339 without timezone
	if t, err := time.Parse("2006-01-02T15:04:05", timeStr); err == nil {
		return t
	}
	
	// Try other common formats
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
	}
	
	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t
		}
	}
	
	return time.Time{}
}